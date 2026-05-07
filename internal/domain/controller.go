package domain

// ControllerType defines the type of controller used to manage interfaces.

const (
	ControllerTypeMikrotik = "mikrotik"
	ControllerTypeLocal    = "wgctrl"
	ControllerTypePfsense  = "pfsense"
	ControllerTypeAmnezia  = "amneziawg"
)

// Controller extras can be used to store additional information available for specific controllers only.

type MikrotikInterfaceExtras struct {
	Id       string // internal mikrotik ID
	Comment  string
	Disabled bool
}

type MikrotikPeerExtras struct {
	Id              string // internal mikrotik ID
	Name            string
	Comment         string
	IsResponder     bool
	Disabled        bool
	ClientEndpoint  string
	ClientAddress   string
	ClientDns       string
	ClientKeepalive int
}

type LocalPeerExtras struct {
	Disabled bool
}

type PfsenseInterfaceExtras struct {
	Id       string // internal pfSense ID
	Comment  string
	Disabled bool
}

type PfsensePeerExtras struct {
	Id              string // internal pfSense ID
	Name            string
	Comment         string
	Disabled        bool
	ClientEndpoint  string
	ClientAddress   string
	ClientDns       string
	ClientKeepalive int
}

// AmneziaInterfaceExtras holds the AmneziaWG V2 obfuscation parameters for an
// interface managed by an AmneziaController. These parameters are interface-
// wide (NOT per-peer) in current AmneziaWG kernel module versions.
//
// Param meanings (per AmneziaWG protocol spec):
//   Jc       — junk packet count (number of decoy packets sent at handshake start)
//   Jmin     — minimum size of each junk packet in bytes
//   Jmax     — maximum size of each junk packet in bytes
//   S1       — header size (init) for V1 obfuscation
//   S2       — header size (response) for V1 obfuscation
//   S3       — V2 only — header size (cookie) for V2 obfuscation
//   S4       — V2 only — header size (transport) for V2 obfuscation
//   H1..H4   — magic-header values for the four packet types
//                (H1=init, H2=response, H3=cookie, H4=transport)
//   I1..I5   — V2 only — extra protocol-init "junk" payload patterns
//
// V1-only deployments leave S3, S4, I1..I5 unset (zero / empty).
//
// All fields are kept simple strings (rather than ints / [16]byte etc.)
// because the awg CLI accepts hex/decimal strings for I1..I5 and decimal
// for everything else, and we want to round-trip whatever the kernel
// reports verbatim.
type AmneziaInterfaceExtras struct {
	Id       string // local interface name (e.g., "awg0")
	Disabled bool

	// V1 obfuscation parameters (also valid in V2)
	Jc   int    // junk packet count
	Jmin int    // junk packet min size
	Jmax int    // junk packet max size
	S1   int    // header size (init)
	S2   int    // header size (response)
	H1   uint32 // magic header (init)
	H2   uint32 // magic header (response)
	H3   uint32 // magic header (cookie)
	H4   uint32 // magic header (transport)

	// V2 obfuscation parameters (zero means "not set" — kernel uses default / disabled)
	S3 int    // V2: header size (cookie)
	S4 int    // V2: header size (transport)
	I1 string // V2: init junk pattern (hex-string accepted by awg)
	I2 string
	I3 string
	I4 string
	I5 string
}

// AmneziaPeerExtras is a placeholder for forward-compatibility. As of
// AmneziaWG protocol V2, all obfuscation parameters live on the Interface
// (above) — peer-level extras are limited to what standard WireGuard
// already exposes (no AWG-specific per-peer state).
type AmneziaPeerExtras struct {
	Disabled bool
}
