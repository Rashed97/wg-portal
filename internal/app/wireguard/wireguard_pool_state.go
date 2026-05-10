package wireguard

import (
	"context"
	"fmt"
	"math/big"
	"net/netip"

	"github.com/h44z/wg-portal/internal/domain"
)

// GetInterfaceAllocatorState computes the read-only allocator summary
// for the given interface — used by the admin UI to show "next free
// /27" and pool usage at a glance.
func (m Manager) GetInterfaceAllocatorState(
	ctx context.Context,
	iface domain.InterfaceIdentifier,
) (*domain.PoolAllocatorState, error) {
	if err := domain.ValidateAdminAccessRights(ctx); err != nil {
		return nil, err
	}
	ifaceRow, err := m.db.GetInterface(ctx, iface)
	if err != nil {
		return nil, fmt.Errorf("get interface %s: %w", iface, err)
	}
	pools, err := m.db.GetUserInterfacePoolsForInterface(ctx, iface)
	if err != nil {
		return nil, fmt.Errorf("list pools: %w", err)
	}

	state := &domain.PoolAllocatorState{
		InterfaceIdentifier: iface,
		AllocatedCount:      len(pools),
	}

	state.NextFreeV4, state.SupernetV4Total, state.SupernetV4Reserved =
		probeNextFree(ifaceRow.UserPoolSupernetV4, ifaceRow.UserPoolReservedV4, ifaceRow.UserPoolSizeV4,
			extractAllocated(pools, "v4"))

	state.NextFreeV6Ula, state.SupernetV6UlaTotal, state.SupernetV6UlaReserved =
		probeNextFree(ifaceRow.UserPoolSupernetV6Ula, ifaceRow.UserPoolReservedV6Ula, ifaceRow.UserPoolSizeV6Ula,
			extractAllocated(pools, "ula"))

	state.NextFreeV6Pi, state.SupernetV6PiTotal, state.SupernetV6PiReserved =
		probeNextFree(ifaceRow.UserPoolSupernetV6Pi, ifaceRow.UserPoolReservedV6Pi, ifaceRow.UserPoolSizeV6Pi,
			extractAllocated(pools, "pi"))

	return state, nil
}

// probeNextFree walks the supernet in /size strides and returns the
// first free slice (string), the total slice count the supernet can
// hold, and how many of those slices are blocked by reservations.
//
// Supernet "" → no allocation possible: returns "", 0, 0.
// Cap iteration at 4096 slices for safety on absurd configs.
func probeNextFree(supernet, reservedStr string, size int, used map[string]struct{}) (next string, total, reservedCount int) {
	if supernet == "" || size <= 0 {
		return "", 0, 0
	}
	supernetPrefix, err := netip.ParsePrefix(supernet)
	if err != nil {
		return "", 0, 0
	}
	if size <= supernetPrefix.Bits() {
		return "", 0, 0
	}

	// Total: 2^(size - supernet_bits) but capped to keep loop bounded.
	bits := size - supernetPrefix.Bits()
	if bits > 12 {
		bits = 12 // cap at 4096 slices to keep this O(1) for huge supernets
	}
	total = 1 << bits

	var reserved []netip.Prefix
	for _, r := range splitCommas(reservedStr) {
		p, err := netip.ParsePrefix(r)
		if err == nil {
			reserved = append(reserved, p)
		}
	}

	// Use big.Int math to step from /size slice to /size slice in
	// constant time per iteration. Stepping address-by-address (as
	// nextSiblingPrefix does) would take 2^48 nexts per /80 — infinite.
	stride := new(big.Int).Lsh(big.NewInt(1), uint(supernetPrefix.Addr().BitLen()-size))
	cur := addrToBig(supernetPrefix.Masked().Addr())
	bitLen := supernetPrefix.Addr().BitLen()

	for i := 0; i < total; i++ {
		candidate := netip.PrefixFrom(bigToAddr(cur, bitLen, supernetPrefix.Addr().Is4()), size)
		if !supernetPrefix.Contains(candidate.Addr()) {
			break
		}
		s := candidate.String()
		blocked := false
		for _, rp := range reserved {
			if prefixesOverlap(rp, candidate) {
				blocked = true
				reservedCount++
				break
			}
		}
		if !blocked {
			if _, taken := used[s]; !taken && next == "" {
				next = s
			}
		}
		cur = new(big.Int).Add(cur, stride)
	}
	return
}

func addrToBig(a netip.Addr) *big.Int {
	b := a.As16()
	return new(big.Int).SetBytes(b[:])
}

func bigToAddr(n *big.Int, bitLen int, is4 bool) netip.Addr {
	var buf [16]byte
	bytes := n.Bytes()
	// Right-align bytes in the 16-byte buffer.
	if len(bytes) > 16 {
		bytes = bytes[len(bytes)-16:]
	}
	copy(buf[16-len(bytes):], bytes)
	if is4 {
		// Take the low 4 bytes.
		var v4 [4]byte
		copy(v4[:], buf[12:16])
		return netip.AddrFrom4(v4)
	}
	return netip.AddrFrom16(buf)
}


func extractAllocated(pools []domain.UserInterfacePool, family string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, p := range pools {
		var s string
		switch family {
		case "v4":
			s = p.PoolV4
		case "ula":
			s = p.PoolV6Ula
		case "pi":
			s = p.PoolV6Pi
		}
		if s != "" {
			out[s] = struct{}{}
		}
	}
	return out
}
