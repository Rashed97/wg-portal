package configfile

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"text/template"

	"github.com/h44z/wg-portal/internal/domain"
)

//go:embed tpl_files/*
var TemplateFiles embed.FS

// TemplateHandler is responsible for rendering the WireGuard configuration files
// based on the provided templates.
type TemplateHandler struct {
	templates *template.Template
}

func newTemplateHandler() (*TemplateHandler, error) {
	tplFuncs := template.FuncMap{
		"CidrsToString": domain.CidrsToString,
	}

	templateCache, err := template.New("WireGuard").Funcs(tplFuncs).ParseFS(TemplateFiles, "tpl_files/*.tpl")
	if err != nil {
		return nil, err
	}

	handler := &TemplateHandler{
		templates: templateCache,
	}

	return handler, nil
}

// GetInterfaceConfig returns the rendered configuration file for a WireGuard interface.
func (c TemplateHandler) GetInterfaceConfig(cfg *domain.Interface, peers []domain.Peer) (io.Reader, error) {
	var tplBuff bytes.Buffer

	err := c.templates.ExecuteTemplate(&tplBuff, "wg_interface.tpl", map[string]any{
		"Interface": cfg,
		"Peers":     peers,
		"Portal": map[string]any{
			"Version": "unknown",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to execute interface template for %s: %w", cfg.Identifier, err)
	}

	return &tplBuff, nil
}

// GetPeerConfig returns the rendered configuration file for a WireGuard peer.
//
// serverIface is the wg-portal-managed Interface that this peer connects TO
// (i.e., the server side). It's needed for AmneziaWG-style configs so the
// SERVER's obfuscation params (Jc/Jmin/Jmax/S1-S4/H1-H4/I1-I5) get embedded
// in the peer's downloaded .conf — both ends must agree on those values to
// successfully handshake. For non-AWG styles serverIface can be nil.
func (c TemplateHandler) GetPeerConfig(peer *domain.Peer, serverIface *domain.Interface, style string) (
	io.Reader, error,
) {
	var tplBuff bytes.Buffer

	ctx := map[string]any{
		"Style":           style,
		"Peer":            peer,
		"ServerInterface": serverIface,
		"Portal": map[string]any{
			"Version": "unknown",
		},
	}

	// Pre-extract AmneziaInterfaceExtras into a top-level template var so the
	// .tpl can reference {{ .AmneziaExtras.Jc }} etc. without runtime type
	// assertions (Go's text/template can't downcast `any`).
	if serverIface != nil {
		// Round-trip via the PhysicalInterface extras path. The Interface
		// domain object doesn't carry extras directly — they live on
		// PhysicalInterface — so we re-derive via the interface's Backend.
		// Callers (manager.GetPeerConfig) are expected to have populated
		// the AmneziaExtras on the interface before calling. As a fallback
		// for the AWG style we pass a zero-value AmneziaInterfaceExtras
		// so the template's `{{ if eq .Style "amneziawg" }}` guard still
		// renders the field block (with zeros, which AWG kernel treats as
		// "param disabled").
		if amx := serverIface.AmneziaExtras; amx != nil {
			ctx["AmneziaExtras"] = *amx
		} else if style == domain.ConfigStyleAmneziaWG {
			ctx["AmneziaExtras"] = domain.AmneziaInterfaceExtras{}
		}
	}

	err := c.templates.ExecuteTemplate(&tplBuff, "wg_peer.tpl", ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to execute peer template for %s: %w", peer.Identifier, err)
	}

	return &tplBuff, nil
}
