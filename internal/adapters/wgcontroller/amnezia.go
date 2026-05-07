// Package wgcontroller — AmneziaController scaffolding (F0).
//
// This file is the F0 SCAFFOLDING for AmneziaWG support. It only provides
// the type, constructor, and InterfaceController interface satisfaction with
// "not implemented" return values. The actual kernel-talk logic (shell-out
// to `awg` / `awg-quick` + `vishvananda/netlink` for link/addr/route) lands
// in F1 (BNet-dr23).
//
// AmneziaWG is a fork of WireGuard adding traffic obfuscation parameters
// (Jc, Jmin, Jmax, S1-S4, H1-H4, I1-I5) on top of the standard WG protocol.
// It uses its own kernel module (amneziawg.ko) and userspace tool (`awg`).
// At V2 (the version installed on iad1-vpn-01 as of 2026-05-07), 16
// obfuscation params are supported — see domain.AmneziaInterfaceExtras.
package wgcontroller

import (
	"context"
	"errors"
	"os/exec"

	"github.com/h44z/wg-portal/internal/config"
	"github.com/h44z/wg-portal/internal/domain"
)

// errAmneziaNotImplemented is returned by every AmneziaController method
// during F0 scaffolding. F1 replaces these with real implementations.
var errAmneziaNotImplemented = errors.New(
	"AmneziaController: not yet implemented (F0 scaffolding stub — landing in F1)")

// AmneziaController manages amneziawg.ko interfaces by shelling out to
// `awg` / `awg-quick` from amneziawg-tools, plus using netlink directly
// for link/addr/route lifecycle (mirrors LocalController's split between
// wgctrl-go for protocol params and netlink for plumbing).
//
// F0 carries only the struct + constructor + interface-method stubs.
type AmneziaController struct {
	coreCfg *config.Config

	// awgBinary is the absolute path to the `awg` userspace binary.
	// Resolved at construction time via exec.LookPath; if unavailable,
	// NewAmneziaController returns an error and ControllerManager skips
	// registration so wg-portal still boots cleanly on hosts without
	// amneziawg-tools installed.
	awgBinary      string
	awgQuickBinary string
}

// NewAmneziaController constructs an AmneziaController if `awg` and
// `awg-quick` are present in PATH. Returns an error if either tool is
// missing — the controller_manager treats that as "skip registration"
// rather than fatal so wg-portal can boot on hosts without amneziawg.
func NewAmneziaController(coreCfg *config.Config) (*AmneziaController, error) {
	awg, err := exec.LookPath("awg")
	if err != nil {
		return nil, err
	}
	awgQuick, err := exec.LookPath("awg-quick")
	if err != nil {
		return nil, err
	}

	return &AmneziaController{
		coreCfg:        coreCfg,
		awgBinary:      awg,
		awgQuickBinary: awgQuick,
	}, nil
}

// GetId reports the backend identifier this controller registers under.
// The constant is shared with config.AmneziawgBackendName so frontend
// dropdowns and the controller_manager agree on the spelling.
func (c *AmneziaController) GetId() domain.InterfaceBackend {
	return domain.InterfaceBackend(config.AmneziawgBackendName)
}

func (c *AmneziaController) GetInterfaces(_ context.Context) ([]domain.PhysicalInterface, error) {
	return nil, errAmneziaNotImplemented
}

func (c *AmneziaController) GetInterface(_ context.Context, _ domain.InterfaceIdentifier) (
	*domain.PhysicalInterface, error,
) {
	return nil, errAmneziaNotImplemented
}

func (c *AmneziaController) GetPeers(_ context.Context, _ domain.InterfaceIdentifier) (
	[]domain.PhysicalPeer, error,
) {
	return nil, errAmneziaNotImplemented
}

func (c *AmneziaController) SaveInterface(
	_ context.Context,
	_ domain.InterfaceIdentifier,
	_ func(pi *domain.PhysicalInterface) (*domain.PhysicalInterface, error),
) error {
	return errAmneziaNotImplemented
}

func (c *AmneziaController) DeleteInterface(_ context.Context, _ domain.InterfaceIdentifier) error {
	return errAmneziaNotImplemented
}

func (c *AmneziaController) SavePeer(
	_ context.Context,
	_ domain.InterfaceIdentifier,
	_ domain.PeerIdentifier,
	_ func(pp *domain.PhysicalPeer) (*domain.PhysicalPeer, error),
) error {
	return errAmneziaNotImplemented
}

func (c *AmneziaController) DeletePeer(
	_ context.Context, _ domain.InterfaceIdentifier, _ domain.PeerIdentifier,
) error {
	return errAmneziaNotImplemented
}

func (c *AmneziaController) PingAddresses(_ context.Context, _ string) (*domain.PingerResult, error) {
	return nil, errAmneziaNotImplemented
}
