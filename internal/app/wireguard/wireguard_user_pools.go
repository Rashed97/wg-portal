package wireguard

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/netip"

	"github.com/h44z/wg-portal/internal/app"
	"github.com/h44z/wg-portal/internal/domain"
)

// GetUserInterfacePools returns all pools for the given user, one row
// per interface where the user has been allocated.
func (m Manager) GetUserInterfacePools(
	ctx context.Context,
	user domain.UserIdentifier,
) ([]domain.UserInterfacePool, error) {
	if err := domain.ValidateAdminAccessRights(ctx); err != nil {
		// non-admins can only read their own
		currentUser := domain.GetUserInfo(ctx)
		if currentUser == nil || currentUser.Id != user {
			return nil, err
		}
	}

	ifaces, err := m.db.GetAllInterfaces(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list interfaces: %w", err)
	}

	var out []domain.UserInterfacePool
	for _, iface := range ifaces {
		p, err := m.db.GetUserInterfacePool(ctx, user, iface.Identifier)
		if err != nil {
			return nil, fmt.Errorf("get pool for %s: %w", iface.Identifier, err)
		}
		if p != nil {
			out = append(out, *p)
		}
	}
	return out, nil
}

// GetUserInterfacePool returns one pool row, or (nil, nil) if not allocated.
func (m Manager) GetUserInterfacePool(
	ctx context.Context,
	user domain.UserIdentifier,
	iface domain.InterfaceIdentifier,
) (*domain.UserInterfacePool, error) {
	if err := domain.ValidateAdminAccessRights(ctx); err != nil {
		currentUser := domain.GetUserInfo(ctx)
		if currentUser == nil || currentUser.Id != user {
			return nil, err
		}
	}
	return m.db.GetUserInterfacePool(ctx, user, iface)
}

// SetUserInterfacePool sets (or updates) the pool for (user × iface),
// validates that each new CIDR fits within the interface's supernet
// and doesn't overlap reserved or other-user pools, then optionally
// renumbers existing peer addresses to land in the new pool.
//
// The renumber preserves the host bits: a peer with .2 in the old pool
// becomes .2 in the new pool. If the new pool is smaller than the old
// (e.g. /27 → /29) and a peer's host bit doesn't fit, returns an error
// without making any changes.
//
// Set skipRenumber=true to write the pool row only — useful when the
// old pool was unused or the operator is doing a coordinated migration
// outside the auto-update path.
func (m Manager) SetUserInterfacePool(
	ctx context.Context,
	user domain.UserIdentifier,
	iface domain.InterfaceIdentifier,
	newPool *domain.UserInterfacePool,
	skipRenumber bool,
) (*domain.UserInterfacePool, error) {
	if err := domain.ValidateAdminAccessRights(ctx); err != nil {
		return nil, err
	}

	ifaceRow, err := m.db.GetInterface(ctx, iface)
	if err != nil {
		return nil, fmt.Errorf("get interface %s: %w", iface, err)
	}

	// Validate each family: parse CIDR, check it fits supernet, check
	// it's outside reserved, check no overlap with another user's pool.
	if err := m.validateUserPool(ctx, user, ifaceRow, newPool); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
	}

	// Find old pool for diff (renumber path).
	oldPool, _ := m.db.GetUserInterfacePool(ctx, user, iface)

	newPool.UserIdentifier = user
	newPool.InterfaceIdentifier = iface

	// Renumber peer addresses if pool changed and not skipped.
	if !skipRenumber && oldPool != nil {
		peers, err := m.db.GetUserPeers(ctx, user)
		if err != nil {
			return nil, fmt.Errorf("get user peers: %w", err)
		}

		// Plan renumbers for each (peer × family).
		type renumber struct {
			peerID  domain.PeerIdentifier
			ifaceID domain.InterfaceIdentifier
			oldAddr domain.Cidr
			newAddr domain.Cidr
		}
		var changes []renumber
		for _, p := range peers {
			if p.InterfaceIdentifier != iface {
				continue
			}
			for _, addr := range p.Interface.Addresses {
				newAddr, ok, err := remapAddress(addr, oldPool, newPool)
				if err != nil {
					return nil, fmt.Errorf("renumber peer %s: %w", p.Identifier, err)
				}
				if !ok {
					continue // not in any family that changed
				}
				changes = append(changes, renumber{
					peerID:  p.Identifier,
					ifaceID: iface,
					oldAddr: addr,
					newAddr: newAddr,
				})
			}
		}

		// Apply renumbers via SavePeer (so all the GORM hooks fire).
		for _, c := range changes {
			err := m.db.SavePeer(ctx, c.peerID,
				func(p *domain.Peer) (*domain.Peer, error) {
					updated := false
					for i, addr := range p.Interface.Addresses {
						if addr.String() == c.oldAddr.String() {
							p.Interface.Addresses[i] = c.newAddr
							updated = true
						}
					}
					if updated {
						slog.InfoContext(ctx, "renumbered peer address",
							"peer", c.peerID, "from", c.oldAddr.String(), "to", c.newAddr.String())
					}
					return p, nil
				})
			if err != nil {
				return nil, fmt.Errorf("save renumbered peer %s: %w", c.peerID, err)
			}
		}

		// Trigger route resync: fetch interface + peers + republish to event bus.
		if i, peers, err := m.db.GetInterfaceAndPeers(ctx, iface); err == nil {
			m.bus.Publish(app.TopicRouteUpdate, domain.RoutingTableInfo{
				Interface:  *i,
				AllowedIps: i.GetAllowedIPs(peers),
				FwMark:     i.FirewallMark,
				Table:      i.GetRoutingTable(),
				TableStr:   i.RoutingTable,
			})
		}
	}

	// Persist the pool row last (after all peer changes succeed).
	if err := m.db.SaveUserInterfacePool(ctx, newPool); err != nil {
		return nil, fmt.Errorf("save pool: %w", err)
	}
	return newPool, nil
}

// DeleteUserInterfacePool clears a user's pool row for an interface.
// Existing peer addresses are left in place (admin can renumber via
// PUT, or the auto-allocator will pick a fresh pool on next peer).
func (m Manager) DeleteUserInterfacePool(
	ctx context.Context,
	user domain.UserIdentifier,
	iface domain.InterfaceIdentifier,
) error {
	if err := domain.ValidateAdminAccessRights(ctx); err != nil {
		return err
	}
	// SaveUserInterfacePool with empty CIDRs effectively releases.
	// We don't have a hard delete on the table; clear the fields.
	return m.db.SaveUserInterfacePool(ctx, &domain.UserInterfacePool{
		UserIdentifier:      user,
		InterfaceIdentifier: iface,
	})
}

// validateUserPool ensures each family's CIDR is well-formed, sits
// within the iface's configured supernet, doesn't overlap reserved
// CIDRs, and doesn't collide with another user's pool on the same
// iface.
func (m Manager) validateUserPool(
	ctx context.Context,
	user domain.UserIdentifier,
	iface *domain.Interface,
	pool *domain.UserInterfacePool,
) error {
	all, err := m.db.GetUserInterfacePoolsForInterface(ctx, iface.Identifier)
	if err != nil {
		return err
	}

	type fam struct {
		label    string
		newPool  string
		supernet string
		reserved string
		size     int
		picker   func(p *domain.UserInterfacePool) string
	}
	families := []fam{
		{"v4", pool.PoolV4, iface.UserPoolSupernetV4, iface.UserPoolReservedV4, iface.UserPoolSizeV4,
			func(p *domain.UserInterfacePool) string { return p.PoolV4 }},
		{"v6 ULA", pool.PoolV6Ula, iface.UserPoolSupernetV6Ula, iface.UserPoolReservedV6Ula, iface.UserPoolSizeV6Ula,
			func(p *domain.UserInterfacePool) string { return p.PoolV6Ula }},
		{"v6 PI", pool.PoolV6Pi, iface.UserPoolSupernetV6Pi, iface.UserPoolReservedV6Pi, iface.UserPoolSizeV6Pi,
			func(p *domain.UserInterfacePool) string { return p.PoolV6Pi }},
	}
	for _, f := range families {
		if f.newPool == "" {
			continue // empty = unset for this family, no validation needed
		}
		newPrefix, err := netip.ParsePrefix(f.newPool)
		if err != nil {
			return fmt.Errorf("%s pool %q: %w", f.label, f.newPool, err)
		}
		if f.supernet == "" {
			return fmt.Errorf("%s supernet not configured on interface %s", f.label, iface.Identifier)
		}
		supernet, err := netip.ParsePrefix(f.supernet)
		if err != nil {
			return fmt.Errorf("invalid supernet on iface %s: %w", iface.Identifier, err)
		}
		if !supernet.Contains(newPrefix.Addr()) {
			return fmt.Errorf("%s pool %s not within supernet %s", f.label, newPrefix, supernet)
		}
		if f.size > 0 && newPrefix.Bits() != f.size {
			return fmt.Errorf("%s pool %s must be /%d (interface size)", f.label, newPrefix, f.size)
		}
		// Reserved overlap.
		for _, r := range splitCommas(f.reserved) {
			rPrefix, err := netip.ParsePrefix(r)
			if err != nil {
				continue
			}
			if prefixesOverlap(newPrefix, rPrefix) {
				return fmt.Errorf("%s pool %s overlaps reserved %s", f.label, newPrefix, rPrefix)
			}
		}
		// Other-user overlap.
		for _, other := range all {
			if other.UserIdentifier == user {
				continue
			}
			if other := f.picker(&other); other != "" {
				op, err := netip.ParsePrefix(other)
				if err != nil {
					continue
				}
				if prefixesOverlap(newPrefix, op) {
					return fmt.Errorf("%s pool %s overlaps another user's pool %s", f.label, newPrefix, op)
				}
			}
		}
	}
	return nil
}

func splitCommas(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	cur := ""
	for _, ch := range s {
		if ch == ',' || ch == ' ' {
			if cur != "" {
				out = append(out, cur)
				cur = ""
			}
			continue
		}
		cur += string(ch)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

func prefixesOverlap(a, b netip.Prefix) bool {
	return a.Contains(b.Addr()) || b.Contains(a.Addr())
}

// remapAddress takes a peer's current /N address and the old + new
// pool CIDRs; returns the equivalent address in the new pool, or
// (_, false, nil) if the old address isn't in the changed family.
//
// Preserves host bits. So .2 in old /27 becomes .2 in new /27.
// Returns an error if the new pool can't fit the host bits (smaller
// prefix length than required).
func remapAddress(addr domain.Cidr, oldPool, newPool *domain.UserInterfacePool) (domain.Cidr, bool, error) {
	addrPrefix, err := netip.ParsePrefix(addr.String())
	if err != nil {
		return domain.Cidr{}, false, err
	}

	// Pick which family this address is in.
	var oldStr, newStr string
	switch {
	case addrPrefix.Addr().Is4():
		oldStr, newStr = oldPool.PoolV4, newPool.PoolV4
	default:
		// v6: ULA vs PI determined by whether the address sits inside the old ULA
		oldUla, _ := netip.ParsePrefix(oldPool.PoolV6Ula)
		if oldPool.PoolV6Ula != "" && oldUla.Contains(addrPrefix.Addr()) {
			oldStr, newStr = oldPool.PoolV6Ula, newPool.PoolV6Ula
		} else {
			oldStr, newStr = oldPool.PoolV6Pi, newPool.PoolV6Pi
		}
	}
	if oldStr == "" || newStr == "" {
		return domain.Cidr{}, false, nil
	}
	oldPrefix, err := netip.ParsePrefix(oldStr)
	if err != nil {
		return domain.Cidr{}, false, err
	}
	if !oldPrefix.Contains(addrPrefix.Addr()) {
		return domain.Cidr{}, false, nil
	}
	newPrefix, err := netip.ParsePrefix(newStr)
	if err != nil {
		return domain.Cidr{}, false, err
	}
	if oldPrefix.Bits() != newPrefix.Bits() {
		return domain.Cidr{}, false, errors.New("old and new pools must have same prefix length for renumber")
	}

	// Compute the new address: replace the prefix bits, keep the host bits.
	hostOffset, err := computeHostOffset(addrPrefix.Addr(), oldPrefix)
	if err != nil {
		return domain.Cidr{}, false, err
	}
	newAddr, err := addrAtOffset(newPrefix.Addr(), hostOffset)
	if err != nil {
		return domain.Cidr{}, false, err
	}
	out := domain.Cidr{
		Addr:      newAddr.String(),
		NetLength: addrPrefix.Bits(),
	}
	out.Cidr = netip.PrefixFrom(newAddr, addrPrefix.Bits()).String()
	return out, true, nil
}

// computeHostOffset returns the integer offset of addr within the
// /prefix.Bits() network. addr must be inside prefix.
func computeHostOffset(addr netip.Addr, prefix netip.Prefix) (uint64, error) {
	if !prefix.Contains(addr) {
		return 0, fmt.Errorf("addr %s not in prefix %s", addr, prefix)
	}
	addrBytes := addr.As16()
	netBytes := prefix.Masked().Addr().As16()
	var diff [16]byte
	for i := range addrBytes {
		diff[i] = addrBytes[i] - netBytes[i]
	}
	// Take the low 8 bytes (max we support is /64 host space — 8 bytes / 64 bits).
	var off uint64
	for _, b := range diff[8:] {
		off = (off << 8) | uint64(b)
	}
	return off, nil
}

func addrAtOffset(base netip.Addr, off uint64) (netip.Addr, error) {
	bytes := base.As16()
	// Add off to the low 8 bytes.
	for i := 15; i >= 8 && off > 0; i-- {
		sum := uint64(bytes[i]) + (off & 0xff)
		bytes[i] = byte(sum & 0xff)
		off = (off >> 8) + (sum >> 8)
	}
	if off != 0 {
		return netip.Addr{}, fmt.Errorf("offset overflow")
	}
	if base.Is4() {
		return netip.AddrFrom4([4]byte{bytes[12], bytes[13], bytes[14], bytes[15]}), nil
	}
	return netip.AddrFrom16(bytes), nil
}
