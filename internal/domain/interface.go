package domain

import (
	"fmt"
	"log/slog"
	"math"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"

	"github.com/h44z/wg-portal/internal"
	"github.com/h44z/wg-portal/internal/config"
)

const (
	InterfaceTypeServer InterfaceType = "server"
	InterfaceTypeClient InterfaceType = "client"
	InterfaceTypeAny    InterfaceType = "any"
)

var allowedFileNameRegex = regexp.MustCompile("[^a-zA-Z0-9-_]+")

type InterfaceIdentifier string
type InterfaceType string
type InterfaceBackend string

type Interface struct {
	BaseModel

	// WireGuard specific (for the [interface] section of the config file)

	Identifier InterfaceIdentifier `gorm:"primaryKey"` // device name, for example: wg0
	KeyPair                        // private/public Key of the server interface
	ListenPort int                 // the listening port, for example: 51820

	Addresses    []Cidr `gorm:"many2many:interface_addresses;"` // the interface ip addresses
	DnsStr       string // the dns server that should be set if the interface is up, comma separated
	DnsSearchStr string // the dns search option string that should be set if the interface is up, will be appended to DnsStr

	Mtu          int    // the device MTU
	FirewallMark uint32 // a firewall mark
	RoutingTable string // the routing table number or "off" if the routing table should not be managed

	PreUp    string // action that is executed before the device is up
	PostUp   string // action that is executed after the device is up
	PreDown  string // action that is executed before the device is down
	PostDown string // action that is executed after the device is down

	SaveConfig bool // automatically persist config changes to the wgX.conf file

	// WG Portal specific
	DisplayName       string           // a nice display name/ description for the interface
	Type              InterfaceType    // the interface type, either InterfaceTypeServer or InterfaceTypeClient
	CreateDefaultPeer bool             // if true, default peers will be created for this interface
	Backend           InterfaceBackend // the backend that is used to manage the interface (wgctrl, mikrotik, ...)
	DriverType        string           // the interface driver type (linux, software, ...)
	Disabled          *time.Time       `gorm:"index"` // flag that specifies if the interface is enabled (up) or not (down)
	DisabledReason    string           // the reason why the interface has been disabled

	// Default settings for the peer, used for new peers, those settings will be published to ConfigOption options of
	// the peer config

	PeerDefNetworkStr          string // the default subnets from which peers will get their IP addresses, comma seperated
	PeerDefDnsStr              string // the default dns server for the peer
	PeerDefDnsSearchStr        string // the default dns search options for the peer
	PeerDefEndpoint            string // the default endpoint for the peer
	PeerDefAllowedIPsStr       string // the default allowed IP string for the peer
	PeerDefMtu                 int    // the default device MTU
	PeerDefPersistentKeepalive int    // the default persistent keep-alive Value
	PeerDefFirewallMark        uint32 // default firewall mark
	PeerDefRoutingTable        string // the default routing table

	PeerDefPreUp    string // default action that is executed before the device is up
	PeerDefPostUp   string // default action that is executed after the device is up
	PeerDefPreDown  string // default action that is executed before the device is down
	PeerDefPostDown string // default action that is executed after the device is down

	// Per-interface user pool config (BNet-2ya4 / BNet-5ag6). Each user
	// gets a /UserPoolSizeV4 slice of UserPoolSupernetV4 on this
	// interface — auto-allocated on first peer creation; persisted in
	// the user_interface_pools table. CIDRs in UserPoolReservedV4 are
	// skipped during allocation (e.g. "10.66.0.0/24" reserved for the
	// wg0 interface IP + future system peers).
	//
	// Empty SupernetV4 → legacy behavior (peers drawn directly from
	// PeerDefNetworkStr without per-user scoping).
	UserPoolSupernetV4    string `gorm:"column:user_pool_supernet_v4"`
	UserPoolSizeV4        int    `gorm:"column:user_pool_size_v4"`
	UserPoolReservedV4    string `gorm:"column:user_pool_reserved_v4"` // comma-separated CIDRs to skip
	UserPoolSupernetV6Ula string `gorm:"column:user_pool_supernet_v6_ula"`
	UserPoolSizeV6Ula     int    `gorm:"column:user_pool_size_v6_ula"`
	UserPoolReservedV6Ula string `gorm:"column:user_pool_reserved_v6_ula"`
	UserPoolSupernetV6Pi  string `gorm:"column:user_pool_supernet_v6_pi"`
	UserPoolSizeV6Pi      int    `gorm:"column:user_pool_size_v6_pi"`
	UserPoolReservedV6Pi  string `gorm:"column:user_pool_reserved_v6_pi"`

	// Self-provisioning access control
	LdapAllowedUsers map[string][]UserIdentifier `gorm:"serializer:json"` // Materialised during LDAP sync, keyed by ProviderName

	// AmneziaWG V2 obfuscation params (Jc, Jmin, Jmax, S1-S4, H1-H4, I1-I5),
	// only meaningful when Backend == AmneziawgBackendName. Persisted as
	// JSON so future protocol-version additions don't require schema
	// migrations. Operator-editable in the InterfaceEditModal; embedded
	// into peer .conf downloads at GetPeerConfig time so client and
	// server share the same obfuscation pattern.
	AmneziaExtras *AmneziaInterfaceExtras `gorm:"serializer:json"`
}

// IsUserAllowed returns true if the interface has no filter, or if the user is in the allowed list.
func (i *Interface) IsUserAllowed(userId UserIdentifier, cfg *config.Config) bool {
	isRestricted := false
	for _, provider := range cfg.Auth.Ldap {
		if _, exists := provider.InterfaceFilter[string(i.Identifier)]; exists {
			isRestricted = true
			break
		}
	}

	if !isRestricted {
		return true // The interface is completely unrestricted by LDAP config
	}

	for _, allowedUsers := range i.LdapAllowedUsers {
		for _, uid := range allowedUsers {
			if uid == userId {
				return true
			}
		}
	}
	return false
}

// PublicInfo returns a copy of the interface with only the public information.
// Sensible information like keys are not included.
func (i *Interface) PublicInfo() Interface {
	return Interface{
		Identifier:  i.Identifier,
		DisplayName: i.DisplayName,
		Type:        i.Type,
		Backend:     i.Backend, // surfaces "amneziawg" vs "local" so the UI can badge AWG interfaces
		Disabled:    i.Disabled,
	}
}

// Validate performs checks to ensure that the interface is valid.
func (i *Interface) Validate() error {
	// validate peer default endpoint, add port if needed
	if i.PeerDefEndpoint != "" {
		host, port, err := net.SplitHostPort(i.PeerDefEndpoint)
		switch {
		case err != nil && !strings.Contains(err.Error(), "missing port in address"):
			return fmt.Errorf("invalid default endpoint: %w", err)
		case err != nil && strings.Contains(err.Error(), "missing port in address"):
			// In this case, the entire string is the host, and there's no port.
			host = i.PeerDefEndpoint
			port = strconv.Itoa(i.ListenPort)
		}

		i.PeerDefEndpoint = net.JoinHostPort(host, port)
	}

	return nil
}

func (i *Interface) IsDisabled() bool {
	if i == nil {
		return true
	}
	return i.Disabled != nil
}

func (i *Interface) AddressStr() string {
	return CidrsToString(i.Addresses)
}

func (i *Interface) CopyCalculatedAttributes(src *Interface) {
	i.BaseModel = src.BaseModel
}

func (i *Interface) GetConfigFileName() string {
	filename := allowedFileNameRegex.ReplaceAllString(string(i.Identifier), "")
	filename = internal.TruncateString(filename, 16)
	filename += ".conf"

	return filename
}

// GetAllowedIPs returns the allowed IPs for the interface depending on the interface type and peers.
// For example, if the interface type is Server, the allowed IPs are the IPs of the peers.
// If the interface type is Client, the allowed IPs correspond to the AllowedIPsStr of the peers.
func (i *Interface) GetAllowedIPs(peers []Peer) []Cidr {
	var allowedCidrs []Cidr

	switch i.Type {
	case InterfaceTypeServer, InterfaceTypeAny:
		for _, peer := range peers {
			for _, ip := range peer.Interface.Addresses {
				allowedCidrs = append(allowedCidrs, ip.HostAddr())
			}
			if peer.ExtraAllowedIPsStr != "" {
				extraIPs, err := CidrsFromString(peer.ExtraAllowedIPsStr)
				if err == nil {
					allowedCidrs = append(allowedCidrs, extraIPs...)
				}
			}
		}
	case InterfaceTypeClient:
		for _, peer := range peers {
			allowedIPs, err := CidrsFromString(peer.AllowedIPsStr.GetValue())
			if err == nil {
				allowedCidrs = append(allowedCidrs, allowedIPs...)
			}
		}
	}

	return allowedCidrs
}

func (i *Interface) ManageRoutingTable() bool {
	routingTableStr := strings.ToLower(i.RoutingTable)
	return routingTableStr != "off"
}

// GetRoutingTable returns the routing table number or
//
//	-1 if RoutingTable was set to "off" or an error occurred
func (i *Interface) GetRoutingTable() int {

	routingTableStr := strings.ToLower(i.RoutingTable)
	switch {
	case routingTableStr == "":
		return 0
	case routingTableStr == "off":
		return -1
	case strings.HasPrefix(routingTableStr, "0x"):
		if i.Backend != config.LocalBackendName {
			return 0 // ignore numeric routing table numbers for non-local controllers
		}
		numberStr := strings.ReplaceAll(routingTableStr, "0x", "")
		routingTable, err := strconv.ParseUint(numberStr, 16, 64)
		if err != nil {
			slog.Error("failed to parse routing table number", "table", routingTableStr, "error", err)
			return -1
		}
		if routingTable > math.MaxInt32 {
			slog.Error("routing table number too large", "table", routingTable, "max", math.MaxInt32)
			return -1
		}
		return int(routingTable)
	default:
		if i.Backend != config.LocalBackendName {
			return 0 // ignore numeric routing table numbers for non-local controllers
		}
		routingTable, err := strconv.Atoi(routingTableStr)
		if err != nil {
			slog.Error("failed to parse routing table number", "table", routingTableStr, "error", err)
			return -1
		}
		if routingTable > math.MaxInt32 {
			slog.Error("routing table number too large", "table", routingTable, "max", math.MaxInt32)
			return -1
		}
		return routingTable
	}
}

type PhysicalInterface struct {
	Identifier InterfaceIdentifier // device name, for example: wg0
	KeyPair                        // private/public Key of the server interface
	ListenPort int                 // the listening port, for example: 51820

	Addresses []Cidr // the interface ip addresses

	Mtu          int    // the device MTU
	FirewallMark uint32 // a firewall mark

	DeviceUp bool // device status

	ImportSource string // import source (wgctrl, file, ...)
	DeviceType   string // device type (Linux kernel, userspace, ...)

	BytesUpload   uint64
	BytesDownload uint64

	backendExtras any // additional backend-specific extras, e.g., domain.MikrotikInterfaceExtras
}

func (p *PhysicalInterface) GetExtras() any {
	return p.backendExtras
}

func (p *PhysicalInterface) SetExtras(extras any) {
	switch extras.(type) {
	case MikrotikInterfaceExtras: // OK
	case PfsenseInterfaceExtras: // OK
	case AmneziaInterfaceExtras: // OK
	default:
		panic(fmt.Sprintf("unsupported interface backend extras type %T", extras))
	}

	p.backendExtras = extras
}

func ConvertPhysicalInterface(pi *PhysicalInterface) *Interface {
	networks := make([]Cidr, 0, len(pi.Addresses))
	for _, addr := range pi.Addresses {
		networks = append(networks, addr.NetworkAddr())
	}

	// create a new basic interface with the data from the physical interface
	iface := &Interface{
		Identifier:                 pi.Identifier,
		KeyPair:                    pi.KeyPair,
		ListenPort:                 pi.ListenPort,
		Addresses:                  pi.Addresses,
		DnsStr:                     "",
		DnsSearchStr:               "",
		Mtu:                        pi.Mtu,
		FirewallMark:               pi.FirewallMark,
		RoutingTable:               "",
		PreUp:                      "",
		PostUp:                     "",
		PreDown:                    "",
		PostDown:                   "",
		SaveConfig:                 false,
		DisplayName:                string(pi.Identifier),
		Type:                       InterfaceTypeAny,
		DriverType:                 pi.DeviceType,
		Disabled:                   nil,
		PeerDefNetworkStr:          CidrsToString(networks),
		PeerDefDnsStr:              "",
		PeerDefDnsSearchStr:        "",
		PeerDefEndpoint:            "",
		PeerDefAllowedIPsStr:       CidrsToString(networks),
		PeerDefMtu:                 pi.Mtu,
		PeerDefPersistentKeepalive: 0,
		PeerDefFirewallMark:        0,
		PeerDefRoutingTable:        "",
		PeerDefPreUp:               "",
		PeerDefPostUp:              "",
		PeerDefPreDown:             "",
		PeerDefPostDown:            "",
	}

	if pi.GetExtras() == nil {
		return iface
	}

	// enrich the data with controller-specific extras
	now := time.Now()
	switch pi.ImportSource {
	case ControllerTypeMikrotik:
		extras := pi.GetExtras().(MikrotikInterfaceExtras)
		iface.DisplayName = extras.Comment
		if extras.Disabled {
			iface.Disabled = &now
		} else {
			iface.Disabled = nil
		}
	case ControllerTypePfsense:
		extras := pi.GetExtras().(PfsenseInterfaceExtras)
		iface.DisplayName = extras.Comment
		if extras.Disabled {
			iface.Disabled = &now
		} else {
			iface.Disabled = nil
		}
	case ControllerTypeAmnezia:
		extras := pi.GetExtras().(AmneziaInterfaceExtras)
		// AmneziaWG has no comment field; display name keeps the interface
		// identifier the kernel reports. Disabled state mirrors WireGuard.
		if extras.Disabled {
			iface.Disabled = &now
		} else {
			iface.Disabled = nil
		}
		// Carry AWG obfuscation params onto the user-facing Interface so
		// they round-trip through the DB and end up in peer .conf
		// downloads.
		extrasCopy := extras
		iface.AmneziaExtras = &extrasCopy
	}

	return iface
}

func MergeToPhysicalInterface(pi *PhysicalInterface, i *Interface) {
	pi.Identifier = i.Identifier
	pi.PublicKey = i.PublicKey
	pi.PrivateKey = i.PrivateKey
	pi.ListenPort = i.ListenPort
	pi.Mtu = i.Mtu
	pi.FirewallMark = i.FirewallMark
	pi.DeviceUp = !i.IsDisabled()
	pi.Addresses = i.Addresses

	switch pi.ImportSource {
	case ControllerTypeMikrotik:
		extras := MikrotikInterfaceExtras{
			Comment:  i.DisplayName,
			Disabled: i.IsDisabled(),
		}
		pi.SetExtras(extras)
	case ControllerTypePfsense:
		extras := PfsenseInterfaceExtras{
			Comment:  i.DisplayName,
			Disabled: i.IsDisabled(),
		}
		pi.SetExtras(extras)
	case ControllerTypeAmnezia:
		// AWG params come from the user-editable Interface.AmneziaExtras
		// (which is what the operator set in the UI / DB). The Disabled
		// flag is also user intent. Anything else (e.g., Id) is kernel-
		// derived and may already be on PhysicalInterface.
		var extras AmneziaInterfaceExtras
		if i.AmneziaExtras != nil {
			extras = *i.AmneziaExtras
		}
		extras.Id = string(i.Identifier)
		extras.Disabled = i.IsDisabled()
		pi.SetExtras(extras)
	}
}

type RoutingTableInfo struct {
	Interface  Interface
	AllowedIps []Cidr
	FwMark     uint32
	Table      int
	TableStr   string // the routing table number as string (used by mikrotik, linux uses the numeric value)
	IsDeleted  bool   // true if the interface was deleted, false otherwise
}

func (r RoutingTableInfo) String() string {
	v4, v6 := CidrsPerFamily(r.AllowedIps)
	return fmt.Sprintf("%s: fwmark=%d; table=%d; routes_4=%d; routes_6=%d", r.Interface.Identifier, r.FwMark, r.Table,
		len(v4), len(v6))
}

func (r RoutingTableInfo) ManagementEnabled() bool {
	if r.Table == -1 {
		return false
	}

	return true
}

func (r RoutingTableInfo) GetRoutingTable() int {
	if r.Table <= 0 {
		return int(r.FwMark) // use the dynamic routing table which has the same number as the firewall mark
	}

	return r.Table
}

type IpFamily int

const (
	IpFamilyIPv4 IpFamily = unix.AF_INET
	IpFamilyIPv6 IpFamily = unix.AF_INET6
)

func (f IpFamily) String() string {
	switch f {
	case IpFamilyIPv4:
		return "IPv4"
	case IpFamilyIPv6:
		return "IPv6"
	default:
		return "unknown"
	}
}

// RouteRule represents a routing table rule.
type RouteRule struct {
	InterfaceId InterfaceIdentifier
	IpFamily    IpFamily
	FwMark      uint32
	Table       int
	HasDefault  bool
}

// InterfaceSiteState carries the per-region state for a logical
// interface row (BNet-264h Option B anycast). The interface row in
// the `interfaces` table is shared across the whole fleet — exactly
// one row per anycast group ("wg0", "awg0", …). This sibling table
// holds the bits that DIFFER between regions: addresses, server
// keypair, listen port, etc. Each wg-portal instance reads the row
// for (interface, this_site_id) and applies it to its local kernel.
type InterfaceSiteState struct {
	BaseModel

	InterfaceIdentifier InterfaceIdentifier `gorm:"primaryKey;column:interface_identifier"`
	SiteId              string              `gorm:"primaryKey;column:site_id"`

	// Server-side address(es) on this region's kernel device. Stored
	// as the same comma-separated CIDR string used by Interface.AddressStr().
	AddressStr string `gorm:"column:address_str"`

	// Per-region keypair (each region has its own server identity so
	// clients use distinct PublicKey per region in their .conf).
	PrivateKey string `gorm:"column:private_key;serializer:encstr"`
	PublicKey  string `gorm:"column:public_key"`

	// Listen port on this region. Typically 41194 (wg) / 41195 (awg);
	// per-region override available for flexibility.
	ListenPort int `gorm:"column:listen_port"`
}

// Addresses returns AddressStr parsed into Cidr values; empty slice
// when AddressStr is blank.
func (s InterfaceSiteState) Addresses() []Cidr {
	if s.AddressStr == "" {
		return nil
	}
	out, _ := CidrsFromString(s.AddressStr)
	return out
}

// PeerKernelState gates per-(peer × site) kernel materialization
// (BNet-264h Option B anycast). Active=true means THIS region's
// kernel currently has the peer in wg0/awg0 AND the local sidecar
// is BGP-advertising the peer's /32. Each peer has one row per site
// that hosts the peer's interface. wg-portal flips `active` based
// on handshake events; the sidecar is the fast-path writer.
type PeerKernelState struct {
	BaseModel

	PeerIdentifier PeerIdentifier `gorm:"primaryKey;column:peer_identifier"`
	SiteId         string         `gorm:"primaryKey;column:site_id"`

	Active           bool       `gorm:"column:active;index"`
	LastHandshakeAt  *time.Time `gorm:"column:last_handshake_at"`
	LastSetActiveAt  *time.Time `gorm:"column:last_set_active_at"`
	LastSetInactiveAt *time.Time `gorm:"column:last_set_inactive_at"`
}

// PoolAllocatorState is a read-only summary of an interface's
// per-(user × interface) auto-allocator state (BNet-m76e QoL).
type PoolAllocatorState struct {
	InterfaceIdentifier InterfaceIdentifier
	AllocatedCount      int

	NextFreeV4    string
	NextFreeV6Ula string
	NextFreeV6Pi  string

	SupernetV4Total       int
	SupernetV6UlaTotal    int
	SupernetV6PiTotal     int
	SupernetV4Reserved    int
	SupernetV6UlaReserved int
	SupernetV6PiReserved  int
}
