// Package wgcontroller — AmneziaController (F1).
//
// AmneziaWG is a fork of WireGuard adding traffic-obfuscation parameters
// (Jc, Jmin, Jmax, S1-S4, H1-H4, I1-I5) on top of standard WG. It uses
// its own kernel module (amneziawg.ko) and userspace tool (`awg`).
//
// This controller manages amneziawg interfaces by:
//   - shelling out to `awg` (and `awg-quick` for one-time bring-up
//     helpers if needed) for protocol-level config (private key, peers,
//     listen-port, fwmark, AWG obfuscation params).
//   - using vishvananda/netlink directly for link / addr / route
//     plumbing — exactly mirroring LocalController's split between
//     wgctrl-go (protocol) and netlink (link layer).
//
// We do NOT use `wgctrl-go` because that library only knows the standard
// WireGuard genl_ops keys; AmneziaWG's V2 obfuscation params are silently
// dropped if pushed via wgctrl. The shell-out path naturally rides on
// whatever AWG protocol version `awg` supports (V2 as of 2026-05-07 on
// iad1-vpn-01 with kernel module 1.0.20251009).
//
// Tradeoffs of shell-out vs direct netlink encoding (see BNet-qsd9
// for the future-work bead): per-call fork/exec is ~5-10ms which is
// negligible for our CRUD-rate workload but would be measurable at
// 1000+ peers per interface or sub-second stats polling.
package wgcontroller

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	probing "github.com/prometheus-community/pro-bing"
	"github.com/vishvananda/netlink"

	"github.com/h44z/wg-portal/internal/config"
	"github.com/h44z/wg-portal/internal/domain"
	"github.com/h44z/wg-portal/internal/lowlevel"
)

// AmneziaController manages amneziawg.ko interfaces. See package doc.
type AmneziaController struct {
	coreCfg *config.Config

	// awgBinary / awgQuickBinary are absolute paths resolved at construction
	// time via exec.LookPath. If either is missing on the host,
	// NewAmneziaController returns an error and ControllerManager skips
	// registration so wg-portal still boots cleanly on hosts without
	// amneziawg-tools.
	awgBinary      string
	awgQuickBinary string

	// nl handles link/addr/route/rule lifecycle for amneziawg interfaces.
	// Same lowlevel package LocalController uses — addresses, routes,
	// MTU, link-up/down all work identically because amneziawg.ko exposes
	// the standard netlink LinkType "amneziawg" and inherits WG's L2
	// configuration surface.
	nl *lowlevel.NetlinkManager
}

// NewAmneziaController constructs an AmneziaController if `awg` and
// `awg-quick` are present in PATH. Returns an error if either is missing.
func NewAmneziaController(coreCfg *config.Config) (*AmneziaController, error) {
	awg, err := exec.LookPath("awg")
	if err != nil {
		return nil, fmt.Errorf("awg binary not found in PATH: %w", err)
	}
	awgQuick, err := exec.LookPath("awg-quick")
	if err != nil {
		return nil, fmt.Errorf("awg-quick binary not found in PATH: %w", err)
	}

	return &AmneziaController{
		coreCfg:        coreCfg,
		awgBinary:      awg,
		awgQuickBinary: awgQuick,
		nl:             &lowlevel.NetlinkManager{},
	}, nil
}

// GetId reports the backend identifier this controller registers under.
func (c *AmneziaController) GetId() domain.InterfaceBackend {
	return domain.InterfaceBackend(config.AmneziawgBackendName)
}

// region awg shell-out helpers

// awgRun runs `awg <args...>` and returns stdout. Stdin can be supplied
// for commands that read keys via /dev/stdin (private-key, preshared-key)
// to keep secrets off the command line where they'd appear in
// /proc/<pid>/cmdline.
func (c *AmneziaController) awgRun(stdin string, args ...string) (string, error) {
	cmd := exec.Command(c.awgBinary, args...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("awg %s: %w (stderr: %s)",
			strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

// awgShowInterfaces returns the list of amneziawg interface names
// currently present in the kernel. Equivalent to `awg show interfaces`.
func (c *AmneziaController) awgShowInterfaces() ([]string, error) {
	out, err := c.awgRun("", "show", "interfaces")
	if err != nil {
		return nil, err
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return nil, nil
	}
	// Output is space-separated interface names on one line.
	return strings.Fields(out), nil
}

// awgInterfaceDump holds the parsed first line of `awg show <iface> dump`.
// The dump format is tab-separated; for AWG V2 the interface line carries
// 20 fields:
//
//	private-key  public-key  listen-port  jc  jmin  jmax  s1  s2  s3  s4
//	h1  h2  h3  h4  i1  i2  i3  i4  i5  fwmark
//
// When V1-only, S3/S4/I1..I5 may be empty strings or "0"; we tolerate either.
type awgInterfaceDump struct {
	PrivateKey string
	PublicKey  string
	ListenPort int
	FwMark     uint32

	// AWG params
	Jc, Jmin, Jmax int
	S1, S2, S3, S4 int
	H1, H2, H3, H4 uint32
	I1, I2, I3, I4, I5 string
}

// awgPeerDump holds one parsed peer line from `awg show <iface> dump`.
// 8 tab-separated fields (same as `wg show <iface> dump` peer line):
//
//	public-key  preshared-key  endpoint  allowed-ips
//	latest-handshake  rx-bytes  tx-bytes  persistent-keepalive
//
// Empty / placeholder values appear as "(none)" or "0" depending on field.
type awgPeerDump struct {
	PublicKey      string
	PresharedKey   string
	Endpoint       string
	AllowedIPs     []string
	LatestHandshake time.Time
	RxBytes        uint64
	TxBytes        uint64
	PersistentKeepalive int
}

// awgShowDump runs `awg show <iface> dump` and parses the output.
// Returns the interface dump + any peer dumps (in declaration order).
func (c *AmneziaController) awgShowDump(name string) (
	*awgInterfaceDump, []awgPeerDump, error,
) {
	out, err := c.awgRun("", "show", name, "dump")
	if err != nil {
		// distinguish "no such device" from other errors so callers can
		// translate to os.ErrNotExist for the getOrCreate flow.
		if strings.Contains(err.Error(), "No such device") ||
			strings.Contains(err.Error(), "Unable to access interface") {
			return nil, nil, os.ErrNotExist
		}
		return nil, nil, err
	}

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) == 0 {
		return nil, nil, fmt.Errorf("empty dump for %s", name)
	}

	iface, err := parseAwgInterfaceDumpLine(lines[0])
	if err != nil {
		return nil, nil, fmt.Errorf("parse interface line: %w", err)
	}

	peers := make([]awgPeerDump, 0, len(lines)-1)
	for i, line := range lines[1:] {
		p, err := parseAwgPeerDumpLine(line)
		if err != nil {
			return nil, nil, fmt.Errorf("parse peer line %d: %w", i+1, err)
		}
		peers = append(peers, p)
	}

	return iface, peers, nil
}

func parseAwgInterfaceDumpLine(line string) (*awgInterfaceDump, error) {
	f := strings.Split(line, "\t")
	// Tolerate either V1 (4 fields: priv, pub, port, fwmark) or full V2
	// (20 fields: priv, pub, port, jc, jmin, jmax, s1, s2, s3, s4,
	//  h1, h2, h3, h4, i1, i2, i3, i4, i5, fwmark).
	// V1 layout (no AWG params returned) shouldn't happen on a host running
	// amneziawg.ko, but guard anyway.
	if len(f) < 4 {
		return nil, fmt.Errorf("expected at least 4 fields, got %d (%q)", len(f), line)
	}

	d := &awgInterfaceDump{
		PrivateKey: f[0],
		PublicKey:  f[1],
	}
	port, err := strconv.Atoi(f[2])
	if err != nil {
		return nil, fmt.Errorf("listen port: %w", err)
	}
	d.ListenPort = port

	if len(f) == 4 {
		// V1-style — fwmark is at index 3
		d.FwMark = parseFwMark(f[3])
		return d, nil
	}

	// V2-style — 20 fields, fwmark is at index 19
	if len(f) < 20 {
		return nil, fmt.Errorf("expected 20 V2 fields, got %d (%q)", len(f), line)
	}
	d.Jc = parseIntOrZero(f[3])
	d.Jmin = parseIntOrZero(f[4])
	d.Jmax = parseIntOrZero(f[5])
	d.S1 = parseIntOrZero(f[6])
	d.S2 = parseIntOrZero(f[7])
	d.S3 = parseIntOrZero(f[8])
	d.S4 = parseIntOrZero(f[9])
	d.H1 = parseUint32OrZero(f[10])
	d.H2 = parseUint32OrZero(f[11])
	d.H3 = parseUint32OrZero(f[12])
	d.H4 = parseUint32OrZero(f[13])
	d.I1 = f[14]
	d.I2 = f[15]
	d.I3 = f[16]
	d.I4 = f[17]
	d.I5 = f[18]
	d.FwMark = parseFwMark(f[19])

	return d, nil
}

func parseAwgPeerDumpLine(line string) (awgPeerDump, error) {
	f := strings.Split(line, "\t")
	if len(f) < 8 {
		return awgPeerDump{}, fmt.Errorf("expected 8 peer fields, got %d (%q)", len(f), line)
	}

	p := awgPeerDump{
		PublicKey: f[0],
	}
	if f[1] != "(none)" && f[1] != "" {
		p.PresharedKey = f[1]
	}
	if f[2] != "(none)" && f[2] != "" {
		p.Endpoint = f[2]
	}
	if f[3] != "(none)" && f[3] != "" {
		for _, a := range strings.Split(f[3], ",") {
			a = strings.TrimSpace(a)
			if a != "" {
				p.AllowedIPs = append(p.AllowedIPs, a)
			}
		}
	}
	if f[4] != "0" && f[4] != "" {
		if ts, err := strconv.ParseInt(f[4], 10, 64); err == nil && ts > 0 {
			p.LatestHandshake = time.Unix(ts, 0)
		}
	}
	if v, err := strconv.ParseUint(f[5], 10, 64); err == nil {
		p.RxBytes = v
	}
	if v, err := strconv.ParseUint(f[6], 10, 64); err == nil {
		p.TxBytes = v
	}
	if f[7] != "off" && f[7] != "" {
		if v, err := strconv.Atoi(f[7]); err == nil {
			p.PersistentKeepalive = v
		}
	}
	return p, nil
}

func parseIntOrZero(s string) int {
	if s == "" || s == "off" {
		return 0
	}
	v, _ := strconv.Atoi(s)
	return v
}

func parseUint32OrZero(s string) uint32 {
	if s == "" || s == "off" {
		return 0
	}
	v, _ := strconv.ParseUint(s, 10, 32)
	return uint32(v)
}

func parseFwMark(s string) uint32 {
	if s == "" || s == "off" {
		return 0
	}
	// awg outputs fwmark as hex with optional 0x prefix
	if strings.HasPrefix(s, "0x") {
		v, _ := strconv.ParseUint(s[2:], 16, 32)
		return uint32(v)
	}
	v, _ := strconv.ParseUint(s, 10, 32)
	return uint32(v)
}

// endregion awg shell-out helpers

// region interface read

func (c *AmneziaController) GetInterfaces(_ context.Context) ([]domain.PhysicalInterface, error) {
	names, err := c.awgShowInterfaces()
	if err != nil {
		return nil, fmt.Errorf("awg interface list: %w", err)
	}

	ifaces := make([]domain.PhysicalInterface, 0, len(names))
	for _, name := range names {
		pi, err := c.getInterface(domain.InterfaceIdentifier(name))
		if err != nil {
			return nil, fmt.Errorf("get %s: %w", name, err)
		}
		ifaces = append(ifaces, *pi)
	}
	return ifaces, nil
}

func (c *AmneziaController) GetInterface(_ context.Context, id domain.InterfaceIdentifier) (
	*domain.PhysicalInterface, error,
) {
	return c.getInterface(id)
}

func (c *AmneziaController) getInterface(id domain.InterfaceIdentifier) (*domain.PhysicalInterface, error) {
	dump, _, err := c.awgShowDump(string(id))
	if err != nil {
		return nil, err
	}
	return c.convertAmneziaInterface(string(id), dump)
}

func (c *AmneziaController) convertAmneziaInterface(name string, dump *awgInterfaceDump) (
	*domain.PhysicalInterface, error,
) {
	iface := &domain.PhysicalInterface{
		Identifier: domain.InterfaceIdentifier(name),
		KeyPair: domain.KeyPair{
			PrivateKey: dump.PrivateKey,
			PublicKey:  dump.PublicKey,
		},
		ListenPort:   dump.ListenPort,
		FirewallMark: dump.FwMark,
		ImportSource: domain.ControllerTypeAmnezia,
		DeviceType:   "amneziawg",
	}

	// Carry the V2 obfuscation params via the backend extras so the
	// service layer (and ultimately the UI / .conf renderer) can
	// round-trip them.
	iface.SetExtras(domain.AmneziaInterfaceExtras{
		Id:   name,
		Jc:   dump.Jc,
		Jmin: dump.Jmin,
		Jmax: dump.Jmax,
		S1:   dump.S1,
		S2:   dump.S2,
		S3:   dump.S3,
		S4:   dump.S4,
		H1:   dump.H1,
		H2:   dump.H2,
		H3:   dump.H3,
		H4:   dump.H4,
		I1:   dump.I1,
		I2:   dump.I2,
		I3:   dump.I3,
		I4:   dump.I4,
		I5:   dump.I5,
	})

	// Read link-layer state (addresses, MTU, oper state, byte stats) from
	// netlink — same as LocalController.
	link, err := c.nl.LinkByName(name)
	if err != nil {
		return nil, fmt.Errorf("netlink link by name %s: %w", name, err)
	}
	addrs, err := c.nl.AddrList(link)
	if err != nil {
		return nil, fmt.Errorf("netlink addr list %s: %w", name, err)
	}
	for _, a := range addrs {
		iface.Addresses = append(iface.Addresses, domain.CidrFromNetlinkAddr(a))
	}
	iface.Mtu = link.Attrs().MTU
	// amneziawg.ko (like wireguard.ko) reports OperUnknown when the link
	// is up. We mirror LocalController's check.
	iface.DeviceUp = link.Attrs().OperState == netlink.OperUnknown
	if stats := link.Attrs().Statistics; stats != nil {
		iface.BytesUpload = stats.TxBytes
		iface.BytesDownload = stats.RxBytes
	}

	return iface, nil
}

// endregion interface read

// region peer read

func (c *AmneziaController) GetPeers(_ context.Context, deviceId domain.InterfaceIdentifier) (
	[]domain.PhysicalPeer, error,
) {
	_, peerDumps, err := c.awgShowDump(string(deviceId))
	if err != nil {
		return nil, fmt.Errorf("awg show dump %s: %w", deviceId, err)
	}

	peers := make([]domain.PhysicalPeer, 0, len(peerDumps))
	for _, p := range peerDumps {
		converted, err := c.convertAmneziaPeer(p)
		if err != nil {
			return nil, fmt.Errorf("convert peer %s: %w", p.PublicKey, err)
		}
		peers = append(peers, converted)
	}
	return peers, nil
}

func (c *AmneziaController) convertAmneziaPeer(p awgPeerDump) (domain.PhysicalPeer, error) {
	peer := domain.PhysicalPeer{
		Identifier:          domain.PeerIdentifier(p.PublicKey),
		Endpoint:            p.Endpoint,
		PersistentKeepalive: p.PersistentKeepalive,
		LastHandshake:       p.LatestHandshake,
		BytesUpload:         p.RxBytes,
		BytesDownload:       p.TxBytes,
		ImportSource:        domain.ControllerTypeAmnezia,
		KeyPair: domain.KeyPair{
			PublicKey: p.PublicKey,
		},
	}
	if p.PresharedKey != "" {
		peer.PresharedKey = domain.PreSharedKey(p.PresharedKey)
	}
	for _, a := range p.AllowedIPs {
		_, ipnet, err := net.ParseCIDR(a)
		if err != nil {
			return domain.PhysicalPeer{}, fmt.Errorf("invalid allowed-ip %q: %w", a, err)
		}
		peer.AllowedIPs = append(peer.AllowedIPs, domain.CidrFromIpNet(*ipnet))
	}
	peer.SetExtras(domain.AmneziaPeerExtras{Disabled: false})
	return peer, nil
}

// endregion peer read

// region interface write

func (c *AmneziaController) SaveInterface(
	_ context.Context,
	id domain.InterfaceIdentifier,
	updateFunc func(pi *domain.PhysicalInterface) (*domain.PhysicalInterface, error),
) error {
	pi, err := c.getOrCreateInterface(id)
	if err != nil {
		return err
	}

	if updateFunc != nil {
		pi, err = updateFunc(pi)
		if err != nil {
			return err
		}
	}

	if err := c.updateLowLevelInterface(pi); err != nil {
		return err
	}
	if err := c.updateAmneziaWGInterface(pi); err != nil {
		return err
	}
	return nil
}

func (c *AmneziaController) getOrCreateInterface(id domain.InterfaceIdentifier) (*domain.PhysicalInterface, error) {
	pi, err := c.getInterface(id)
	if err == nil {
		return pi, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("get interface %s: %w", id, err)
	}

	if err := c.createLowLevelInterface(id); err != nil {
		return nil, err
	}
	pi, err = c.getInterface(id)
	if err != nil {
		return nil, fmt.Errorf("get interface after create %s: %w", id, err)
	}
	return pi, nil
}

func (c *AmneziaController) createLowLevelInterface(id domain.InterfaceIdentifier) error {
	link := &netlink.GenericLink{
		LinkAttrs: netlink.LinkAttrs{Name: string(id)},
		LinkType:  "amneziawg",
	}
	if err := c.nl.LinkAdd(link); err != nil {
		return fmt.Errorf("amneziawg link add %s: %w", id, err)
	}
	return nil
}

// updateLowLevelInterface mirrors LocalController.updateLowLevelInterface
// exactly — addresses, MTU, link-up/down via netlink. The order-of-operations
// matters for the auto-route install (see comment block; same fix as the
// pending upstream patch on this branch).
func (c *AmneziaController) updateLowLevelInterface(pi *domain.PhysicalInterface) error {
	link, err := c.nl.LinkByName(string(pi.Identifier))
	if err != nil {
		return err
	}
	if pi.Mtu != 0 {
		if err := c.nl.LinkSetMTU(link, pi.Mtu); err != nil {
			return fmt.Errorf("mtu set: %w", err)
		}
	}

	// Bring up BEFORE setting addresses so the kernel auto-installs the
	// proto-kernel link-scope route for each prefix. Same gotcha as
	// LocalController — see commit 3bab7b3 on this branch.
	if pi.DeviceUp {
		if err := c.nl.LinkSetUp(link); err != nil {
			return fmt.Errorf("link up: %w", err)
		}
	}

	for _, addr := range pi.Addresses {
		if err := c.nl.AddrReplace(link, addr.NetlinkAddr()); err != nil {
			return fmt.Errorf("addr replace %s: %w", addr.String(), err)
		}
	}

	rawAddrs, err := c.nl.AddrList(link)
	if err != nil {
		return fmt.Errorf("addr list: %w", err)
	}
	for _, raw := range rawAddrs {
		want := domain.CidrFromNetlinkAddr(raw)
		keep := false
		for _, a := range pi.Addresses {
			if a == want {
				keep = true
				break
			}
		}
		if keep {
			continue
		}
		if err := c.nl.AddrDel(link, &raw); err != nil {
			return fmt.Errorf("addr del %s: %w", want.String(), err)
		}
	}

	if !pi.DeviceUp {
		if err := c.nl.LinkSetDown(link); err != nil {
			return fmt.Errorf("link down: %w", err)
		}
	}
	return nil
}

// updateAmneziaWGInterface pushes the WG-protocol-level settings (private
// key, listen-port, fwmark) AND the AWG V2 obfuscation params to the
// kernel via `awg set`. Equivalent to LocalController.updateWireGuardInterface
// but via shell-out because wgctrl-go doesn't know AWG keys.
func (c *AmneziaController) updateAmneziaWGInterface(pi *domain.PhysicalInterface) error {
	args := []string{"set", string(pi.Identifier)}

	// listen-port + fwmark (always emit; "0" / "off" disables fwmark)
	args = append(args, "listen-port", strconv.Itoa(pi.ListenPort))
	if pi.FirewallMark != 0 {
		args = append(args, "fwmark", strconv.FormatUint(uint64(pi.FirewallMark), 10))
	} else {
		args = append(args, "fwmark", "off")
	}

	// AWG V2 obfuscation params from extras (zero values disable the param)
	if extras, ok := pi.GetExtras().(domain.AmneziaInterfaceExtras); ok {
		args = appendAwgParam(args, "jc", extras.Jc)
		args = appendAwgParam(args, "jmin", extras.Jmin)
		args = appendAwgParam(args, "jmax", extras.Jmax)
		args = appendAwgParam(args, "s1", extras.S1)
		args = appendAwgParam(args, "s2", extras.S2)
		args = appendAwgParam(args, "s3", extras.S3)
		args = appendAwgParam(args, "s4", extras.S4)
		args = appendAwgParamUint(args, "h1", extras.H1)
		args = appendAwgParamUint(args, "h2", extras.H2)
		args = appendAwgParamUint(args, "h3", extras.H3)
		args = appendAwgParamUint(args, "h4", extras.H4)
		if extras.I1 != "" {
			args = append(args, "i1", extras.I1)
		}
		if extras.I2 != "" {
			args = append(args, "i2", extras.I2)
		}
		if extras.I3 != "" {
			args = append(args, "i3", extras.I3)
		}
		if extras.I4 != "" {
			args = append(args, "i4", extras.I4)
		}
		if extras.I5 != "" {
			args = append(args, "i5", extras.I5)
		}
	}

	// Private key always last so it lands on /dev/stdin without conflicting
	// with positional args.
	args = append(args, "private-key", "/dev/stdin")
	if _, err := c.awgRun(pi.KeyPair.PrivateKey+"\n", args...); err != nil {
		return fmt.Errorf("awg set %s: %w", pi.Identifier, err)
	}
	return nil
}

// appendAwgParam appends "<key> <val>" only when val > 0. AWG treats zero
// as "param disabled" — we match that semantic to avoid noisy "set s3 0"
// calls when the operator hasn't configured V2 params.
func appendAwgParam(args []string, key string, val int) []string {
	if val == 0 {
		return args
	}
	return append(args, key, strconv.Itoa(val))
}

func appendAwgParamUint(args []string, key string, val uint32) []string {
	if val == 0 {
		return args
	}
	return append(args, key, strconv.FormatUint(uint64(val), 10))
}

func (c *AmneziaController) DeleteInterface(_ context.Context, id domain.InterfaceIdentifier) error {
	link, err := c.nl.LinkByName(string(id))
	if err != nil {
		var notFound netlink.LinkNotFoundError
		if errors.As(err, &notFound) {
			return nil
		}
		return fmt.Errorf("link by name %s: %w", id, err)
	}
	if err := c.nl.LinkDel(link); err != nil {
		return fmt.Errorf("link del %s: %w", id, err)
	}
	return nil
}

// endregion interface write

// region peer write

func (c *AmneziaController) SavePeer(
	_ context.Context,
	deviceId domain.InterfaceIdentifier,
	id domain.PeerIdentifier,
	updateFunc func(pp *domain.PhysicalPeer) (*domain.PhysicalPeer, error),
) error {
	pp, err := c.getOrCreatePeer(deviceId, id)
	if err != nil {
		return err
	}

	pp, err = updateFunc(pp)
	if err != nil {
		return err
	}

	// Mirror LocalController: a peer flagged as Disabled in the extras is
	// removed from the kernel rather than left in a dormant state, since
	// AWG (like WG) has no per-peer "disabled" knob.
	if extras, ok := pp.GetExtras().(domain.AmneziaPeerExtras); ok && extras.Disabled {
		return c.deletePeer(deviceId, id)
	}

	if err := c.updatePeer(deviceId, pp); err != nil {
		return err
	}
	return nil
}

func (c *AmneziaController) getOrCreatePeer(
	deviceId domain.InterfaceIdentifier, id domain.PeerIdentifier,
) (*domain.PhysicalPeer, error) {
	pp, err := c.getPeer(deviceId, id)
	if err == nil {
		return pp, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("peer get: %w", err)
	}

	// Create the peer with no further config (allowed-ips empty etc.)
	// — the subsequent updatePeer call sets the real values.
	if _, err := c.awgRun("", "set", string(deviceId),
		"peer", string(id), "allowed-ips", ""); err != nil {
		return nil, fmt.Errorf("create peer %s on %s: %w", id, deviceId, err)
	}
	pp, err = c.getPeer(deviceId, id)
	if err != nil {
		return nil, fmt.Errorf("get peer after create: %w", err)
	}
	return pp, nil
}

func (c *AmneziaController) getPeer(
	deviceId domain.InterfaceIdentifier, id domain.PeerIdentifier,
) (*domain.PhysicalPeer, error) {
	if !id.IsPublicKey() {
		return nil, errors.New("invalid public key")
	}
	_, peerDumps, err := c.awgShowDump(string(deviceId))
	if err != nil {
		return nil, err
	}
	for _, pd := range peerDumps {
		if pd.PublicKey != string(id) {
			continue
		}
		pp, err := c.convertAmneziaPeer(pd)
		if err != nil {
			return nil, err
		}
		return &pp, nil
	}
	return nil, os.ErrNotExist
}

func (c *AmneziaController) updatePeer(
	deviceId domain.InterfaceIdentifier, pp *domain.PhysicalPeer,
) error {
	args := []string{"set", string(deviceId), "peer", pp.KeyPair.PublicKey}

	// Endpoint
	if pp.Endpoint != "" {
		args = append(args, "endpoint", pp.Endpoint)
	}

	// Persistent keepalive (0 disables)
	if pp.PersistentKeepalive > 0 {
		args = append(args, "persistent-keepalive", strconv.Itoa(pp.PersistentKeepalive))
	} else {
		args = append(args, "persistent-keepalive", "0")
	}

	// Allowed IPs (always emit; blank string = no allowed-ips)
	allowed := make([]string, 0, len(pp.AllowedIPs))
	for _, a := range pp.AllowedIPs {
		allowed = append(allowed, a.String())
	}
	args = append(args, "allowed-ips", strings.Join(allowed, ","))

	// Preshared key on stdin to keep it off cmdline. If empty, explicitly
	// clear via /dev/stdin <<< "" (awg accepts an empty psk to remove).
	args = append(args, "preshared-key", "/dev/stdin")
	stdin := string(pp.PresharedKey) + "\n"

	if _, err := c.awgRun(stdin, args...); err != nil {
		return fmt.Errorf("awg set peer %s on %s: %w", pp.KeyPair.PublicKey, deviceId, err)
	}
	return nil
}

func (c *AmneziaController) DeletePeer(
	_ context.Context, deviceId domain.InterfaceIdentifier, id domain.PeerIdentifier,
) error {
	if !id.IsPublicKey() {
		return errors.New("invalid public key")
	}
	return c.deletePeer(deviceId, id)
}

func (c *AmneziaController) deletePeer(
	deviceId domain.InterfaceIdentifier, id domain.PeerIdentifier,
) error {
	if _, err := c.awgRun("", "set", string(deviceId),
		"peer", string(id), "remove"); err != nil {
		return fmt.Errorf("awg remove peer %s on %s: %w", id, deviceId, err)
	}
	return nil
}

// endregion peer write

// region ping

// PingAddresses is identical to LocalController's implementation —
// pro-bing has no concept of which kernel module backs the underlying
// interface, it just opens an ICMP socket.
func (c *AmneziaController) PingAddresses(
	ctx context.Context, addr string,
) (*domain.PingerResult, error) {
	pinger, err := probing.NewPinger(addr)
	if err != nil {
		return nil, fmt.Errorf("pinger new %s: %w", addr, err)
	}
	pinger.SetPrivileged(!c.coreCfg.Statistics.PingUnprivileged)
	pinger.Count = 1
	pinger.Timeout = 2 * time.Second
	if err := pinger.RunWithContext(ctx); err != nil {
		return nil, fmt.Errorf("ping %s: %w", addr, err)
	}
	stats := pinger.Statistics()
	return &domain.PingerResult{
		PacketsRecv: stats.PacketsRecv,
		PacketsSent: stats.PacketsSent,
		Rtts:        stats.Rtts,
	}, nil
}

// endregion ping

// staticAssertInterface ensures we satisfy the InterfaceController contract
// at compile time. Removed at link time; serves only as a build-break if
// the upstream interface evolves.
var _ domain.InterfaceController = (*AmneziaController)(nil)

// silence unused-import warnings if a build path strips a dependency.
var _ = slog.Default
