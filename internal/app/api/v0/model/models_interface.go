package model

import (
	"time"

	"github.com/h44z/wg-portal/internal"
	"github.com/h44z/wg-portal/internal/config"
	"github.com/h44z/wg-portal/internal/domain"
)

type Interface struct {
	Identifier        string `json:"Identifier" example:"wg0"`      // device name, for example: wg0
	DisplayName       string `json:"DisplayName"`                   // a nice display name/ description for the interface
	Mode              string `json:"Mode" example:"server"`         // the interface type, either 'server', 'client' or 'any'
	Backend           string `json:"Backend" example:"local"`       // the backend used for this interface e.g., local, mikrotik, ...
	PrivateKey        string `json:"PrivateKey" example:"abcdef=="` // private Key of the server interface
	PublicKey         string `json:"PublicKey" example:"abcdef=="`  // public Key of the server interface
	Disabled          bool   `json:"Disabled"`                      // flag that specifies if the interface is enabled (up) or not (down)
	DisabledReason    string `json:"DisabledReason"`                // the reason why the interface has been disabled
	SaveConfig        bool   `json:"SaveConfig"`                    // automatically persist config changes to the wgX.conf file
	CreateDefaultPeer bool   `json:"CreateDefaultPeer"`             // if true, default peers will be created for this interface

	ListenPort   int      `json:"ListenPort"`   // the listening port, for example: 51820
	Addresses    []string `json:"Addresses"`    // the interface ip addresses
	Dns          []string `json:"Dns"`          // the dns server that should be set if the interface is up, comma separated
	DnsSearch    []string `json:"DnsSearch"`    // the dns search option string that should be set if the interface is up, will be appended to DnsStr
	Mtu          int      `json:"Mtu"`          // the device MTU
	FirewallMark uint32   `json:"FirewallMark"` // a firewall mark
	RoutingTable string   `json:"RoutingTable"` // the routing table

	PreUp    string `json:"PreUp"`    // action that is executed before the device is up
	PostUp   string `json:"PostUp"`   // action that is executed after the device is up
	PreDown  string `json:"PreDown"`  // action that is executed before the device is down
	PostDown string `json:"PostDown"` // action that is executed after the device is down

	PeerDefNetwork             []string `json:"PeerDefNetwork"`             // the default subnets from which peers will get their IP addresses, comma seperated
	PeerDefDns                 []string `json:"PeerDefDns"`                 // the default dns server for the peer
	PeerDefDnsSearch           []string `json:"PeerDefDnsSearch"`           // the default dns search options for the peer
	PeerDefEndpoint            string   `json:"PeerDefEndpoint"`            // the default endpoint for the peer
	PeerDefAllowedIPs          []string `json:"PeerDefAllowedIPs"`          // the default allowed IP string for the peer
	PeerDefMtu                 int      `json:"PeerDefMtu"`                 // the default device MTU
	PeerDefPersistentKeepalive int      `json:"PeerDefPersistentKeepalive"` // the default persistent keep-alive Value
	PeerDefFirewallMark        uint32   `json:"PeerDefFirewallMark"`        // default firewall mark
	PeerDefRoutingTable        string   `json:"PeerDefRoutingTable"`        // the default routing table

	PeerDefPreUp    string `json:"PeerDefPreUp"`    // default action that is executed before the device is up
	PeerDefPostUp   string `json:"PeerDefPostUp"`   // default action that is executed after the device is up
	PeerDefPreDown  string `json:"PeerDefPreDown"`  // default action that is executed before the device is down
	PeerDefPostDown string `json:"PeerDefPostDown"` // default action that is executed after the device is down

	// AmneziaWG V2 obfuscation parameters (only meaningful when Backend ==
	// "amneziawg"). Always emitted; client renders an extra panel when
	// Backend is amneziawg and these are editable.
	AmneziaWG *AmneziaWGParams `json:"AmneziaWG,omitempty"`

	// Calculated values

	EnabledPeers int    `json:"EnabledPeers"`
	TotalPeers   int    `json:"TotalPeers"`
	Filename     string `json:"Filename"` // the filename of the config file, for example: wg0.conf
}

// AmneziaWGParams is the JSON-friendly view of domain.AmneziaInterfaceExtras.
type AmneziaWGParams struct {
	Jc   int `json:"Jc"`
	Jmin int `json:"Jmin"`
	Jmax int `json:"Jmax"`

	S1 int `json:"S1"`
	S2 int `json:"S2"`
	S3 int `json:"S3"`
	S4 int `json:"S4"`

	H1 uint32 `json:"H1"`
	H2 uint32 `json:"H2"`
	H3 uint32 `json:"H3"`
	H4 uint32 `json:"H4"`

	I1 string `json:"I1"`
	I2 string `json:"I2"`
	I3 string `json:"I3"`
	I4 string `json:"I4"`
	I5 string `json:"I5"`
}

func NewInterface(src *domain.Interface, peers []domain.Peer) *Interface {
	iface := &Interface{
		Identifier:                 string(src.Identifier),
		DisplayName:                src.DisplayName,
		Mode:                       string(src.Type),
		Backend:                    string(src.Backend),
		PrivateKey:                 src.PrivateKey,
		PublicKey:                  src.PublicKey,
		Disabled:                   src.IsDisabled(),
		DisabledReason:             src.DisabledReason,
		SaveConfig:                 src.SaveConfig,
		CreateDefaultPeer:          src.CreateDefaultPeer,
		ListenPort:                 src.ListenPort,
		Addresses:                  domain.CidrsToStringSlice(src.Addresses),
		Dns:                        internal.SliceString(src.DnsStr),
		DnsSearch:                  internal.SliceString(src.DnsSearchStr),
		Mtu:                        src.Mtu,
		FirewallMark:               src.FirewallMark,
		RoutingTable:               src.RoutingTable,
		PreUp:                      src.PreUp,
		PostUp:                     src.PostUp,
		PreDown:                    src.PreDown,
		PostDown:                   src.PostDown,
		PeerDefNetwork:             internal.SliceString(src.PeerDefNetworkStr),
		PeerDefDns:                 internal.SliceString(src.PeerDefDnsStr),
		PeerDefDnsSearch:           internal.SliceString(src.PeerDefDnsSearchStr),
		PeerDefEndpoint:            src.PeerDefEndpoint,
		PeerDefAllowedIPs:          internal.SliceString(src.PeerDefAllowedIPsStr),
		PeerDefMtu:                 src.PeerDefMtu,
		PeerDefPersistentKeepalive: src.PeerDefPersistentKeepalive,
		PeerDefFirewallMark:        src.PeerDefFirewallMark,
		PeerDefRoutingTable:        src.PeerDefRoutingTable,
		PeerDefPreUp:               src.PeerDefPreUp,
		PeerDefPostUp:              src.PeerDefPostUp,
		PeerDefPreDown:             src.PeerDefPreDown,
		PeerDefPostDown:            src.PeerDefPostDown,

		EnabledPeers: 0,
		TotalPeers:   0,
		Filename:     src.GetConfigFileName(),
	}

	if iface.Backend == "" {
		iface.Backend = config.LocalBackendName // default to local backend
	}

	if src.AmneziaExtras != nil {
		// Defensive normalization for I1-I5: the awg-tools dump emits the
		// literal "(null)" for unset fields, and rows persisted by older
		// builds (before parseAwgHexField) have that literal stored in DB.
		// Translate at the DTO boundary so the UI never sees "(null)".
		nullToEmpty := func(s string) string {
			if s == "(null)" {
				return ""
			}
			return s
		}
		iface.AmneziaWG = &AmneziaWGParams{
			Jc:   src.AmneziaExtras.Jc,
			Jmin: src.AmneziaExtras.Jmin,
			Jmax: src.AmneziaExtras.Jmax,
			S1:   src.AmneziaExtras.S1,
			S2:   src.AmneziaExtras.S2,
			S3:   src.AmneziaExtras.S3,
			S4:   src.AmneziaExtras.S4,
			H1:   src.AmneziaExtras.H1,
			H2:   src.AmneziaExtras.H2,
			H3:   src.AmneziaExtras.H3,
			H4:   src.AmneziaExtras.H4,
			I1:   nullToEmpty(src.AmneziaExtras.I1),
			I2:   nullToEmpty(src.AmneziaExtras.I2),
			I3:   nullToEmpty(src.AmneziaExtras.I3),
			I4:   nullToEmpty(src.AmneziaExtras.I4),
			I5:   nullToEmpty(src.AmneziaExtras.I5),
		}
	}

	if len(peers) > 0 {
		iface.TotalPeers = len(peers)

		activePeers := 0
		for _, peer := range peers {
			if !peer.IsDisabled() {
				activePeers++
			}
		}
		iface.EnabledPeers = activePeers
	}

	return iface
}

func NewInterfaces(src []domain.Interface, srcPeers [][]domain.Peer) []Interface {
	results := make([]Interface, len(src))
	for i := range src {
		if srcPeers == nil {
			results[i] = *NewInterface(&src[i], nil)
		} else {
			results[i] = *NewInterface(&src[i], srcPeers[i])
		}
	}

	return results
}

func NewDomainInterface(src *Interface) *domain.Interface {
	now := time.Now()

	cidrs, _ := domain.CidrsFromArray(src.Addresses)

	res := &domain.Interface{
		BaseModel:  domain.BaseModel{},
		Identifier: domain.InterfaceIdentifier(src.Identifier),
		KeyPair: domain.KeyPair{
			PrivateKey: src.PrivateKey,
			PublicKey:  src.PublicKey,
		},
		ListenPort:                 src.ListenPort,
		Addresses:                  cidrs,
		DnsStr:                     internal.SliceToString(src.Dns),
		DnsSearchStr:               internal.SliceToString(src.DnsSearch),
		Mtu:                        src.Mtu,
		FirewallMark:               src.FirewallMark,
		RoutingTable:               src.RoutingTable,
		PreUp:                      src.PreUp,
		PostUp:                     src.PostUp,
		PreDown:                    src.PreDown,
		PostDown:                   src.PostDown,
		SaveConfig:                 src.SaveConfig,
		CreateDefaultPeer:          src.CreateDefaultPeer,
		DisplayName:                src.DisplayName,
		Type:                       domain.InterfaceType(src.Mode),
		Backend:                    domain.InterfaceBackend(src.Backend),
		DriverType:                 "",  // currently unused
		Disabled:                   nil, // set below
		DisabledReason:             src.DisabledReason,
		PeerDefNetworkStr:          internal.SliceToString(src.PeerDefNetwork),
		PeerDefDnsStr:              internal.SliceToString(src.PeerDefDns),
		PeerDefDnsSearchStr:        internal.SliceToString(src.PeerDefDnsSearch),
		PeerDefEndpoint:            src.PeerDefEndpoint,
		PeerDefAllowedIPsStr:       internal.SliceToString(src.PeerDefAllowedIPs),
		PeerDefMtu:                 src.PeerDefMtu,
		PeerDefPersistentKeepalive: src.PeerDefPersistentKeepalive,
		PeerDefFirewallMark:        src.PeerDefFirewallMark,
		PeerDefRoutingTable:        src.PeerDefRoutingTable,
		PeerDefPreUp:               src.PeerDefPreUp,
		PeerDefPostUp:              src.PeerDefPostUp,
		PeerDefPreDown:             src.PeerDefPreDown,
		PeerDefPostDown:            src.PeerDefPostDown,
	}

	if src.Disabled {
		res.Disabled = &now
	}

	if src.AmneziaWG != nil {
		res.AmneziaExtras = &domain.AmneziaInterfaceExtras{
			Id:   src.Identifier,
			Jc:   src.AmneziaWG.Jc,
			Jmin: src.AmneziaWG.Jmin,
			Jmax: src.AmneziaWG.Jmax,
			S1:   src.AmneziaWG.S1,
			S2:   src.AmneziaWG.S2,
			S3:   src.AmneziaWG.S3,
			S4:   src.AmneziaWG.S4,
			H1:   src.AmneziaWG.H1,
			H2:   src.AmneziaWG.H2,
			H3:   src.AmneziaWG.H3,
			H4:   src.AmneziaWG.H4,
			I1:   src.AmneziaWG.I1,
			I2:   src.AmneziaWG.I2,
			I3:   src.AmneziaWG.I3,
			I4:   src.AmneziaWG.I4,
			I5:   src.AmneziaWG.I5,
		}
	}

	return res
}
