package wgcontroller

import (
	"strings"
	"testing"
)

// TestParseAwgInterfaceDumpLine_V2 covers the 20-field V2 dump line
// emitted by `awg show <iface> dump` on AmneziaWG ≥1.0.20251009 (the
// version installed on iad1-vpn-01 as of 2026-05-07).
//
// Sample captured live:
//
//	QAEZ7+ozQfr0XQHC0AyjC8R5jxg6lrPhBP8LjqCqcGw=  KJGxC...=  41199
//	5  50  500  50  100  100  200  12345  23456  34567  45678
//	deadbeef  cafebabe  12345678  87654321  abcdef00  off
func TestParseAwgInterfaceDumpLine_V2(t *testing.T) {
	line := strings.Join([]string{
		"QAEZ7+ozQfr0XQHC0AyjC8R5jxg6lrPhBP8LjqCqcGw=", // private key
		"KJGxCJXDf5WOgdiY8OboQqkywEIduPaD+xbx0KxZ5Fc=", // public key
		"41199",      // listen port
		"5",          // jc
		"50",         // jmin
		"500",        // jmax
		"50",         // s1
		"100",        // s2
		"100",        // s3
		"200",        // s4
		"12345",      // h1
		"23456",      // h2
		"34567",      // h3
		"45678",      // h4
		"deadbeef",   // i1
		"cafebabe",   // i2
		"12345678",   // i3
		"87654321",   // i4
		"abcdef00",   // i5
		"off",        // fwmark
	}, "\t")

	d, err := parseAwgInterfaceDumpLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if d.PrivateKey != "QAEZ7+ozQfr0XQHC0AyjC8R5jxg6lrPhBP8LjqCqcGw=" {
		t.Errorf("private key mismatch: got %q", d.PrivateKey)
	}
	if d.PublicKey != "KJGxCJXDf5WOgdiY8OboQqkywEIduPaD+xbx0KxZ5Fc=" {
		t.Errorf("public key mismatch: got %q", d.PublicKey)
	}
	if d.ListenPort != 41199 {
		t.Errorf("listen port = %d, want 41199", d.ListenPort)
	}
	if d.FwMark != 0 {
		t.Errorf("fwmark = %d, want 0 (off)", d.FwMark)
	}

	tests := []struct {
		name string
		got  any
		want any
	}{
		{"jc", d.Jc, 5}, {"jmin", d.Jmin, 50}, {"jmax", d.Jmax, 500},
		{"s1", d.S1, 50}, {"s2", d.S2, 100}, {"s3", d.S3, 100}, {"s4", d.S4, 200},
		{"h1", d.H1, uint32(12345)}, {"h2", d.H2, uint32(23456)},
		{"h3", d.H3, uint32(34567)}, {"h4", d.H4, uint32(45678)},
		{"i1", d.I1, "deadbeef"}, {"i2", d.I2, "cafebabe"},
		{"i3", d.I3, "12345678"}, {"i4", d.I4, "87654321"}, {"i5", d.I5, "abcdef00"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
		}
	}
}

// TestParseAwgInterfaceDumpLine_V1Fallback covers the legacy 4-field
// dump that older amneziawg-tools (or vanilla wg-tools symlinked to
// awg) might produce. Should still parse, with all V2 params zero.
func TestParseAwgInterfaceDumpLine_V1Fallback(t *testing.T) {
	line := "abc=\tdef=\t51820\toff"
	d, err := parseAwgInterfaceDumpLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.PrivateKey != "abc=" || d.PublicKey != "def=" || d.ListenPort != 51820 {
		t.Errorf("V1 fields wrong: %+v", d)
	}
	if d.Jc != 0 || d.S3 != 0 || d.I1 != "" {
		t.Errorf("V2 fields should be zero in V1 fallback, got: %+v", d)
	}
}

// TestParseAwgInterfaceDumpLine_FwMarkHex covers a hex-encoded fwmark
// (awg outputs "0x4e24" style for non-zero marks).
func TestParseAwgInterfaceDumpLine_FwMarkHex(t *testing.T) {
	line := strings.Join([]string{
		"abc=", "def=", "51820",
		"0", "0", "0", "0", "0", "0", "0",
		"0", "0", "0", "0",
		"", "", "", "", "",
		"0x4e24",
	}, "\t")
	d, err := parseAwgInterfaceDumpLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.FwMark != 0x4e24 {
		t.Errorf("fwmark = %#x, want 0x4e24", d.FwMark)
	}
}

// TestParseAwgPeerDumpLine covers the 8-field peer dump line emitted by
// `awg show <iface> dump` for each peer on the interface. Field order
// matches `wg show <iface> dump` exactly:
//
//	pubkey  psk  endpoint  allowed-ips  latest-handshake  rx  tx  keepalive
func TestParseAwgPeerDumpLine(t *testing.T) {
	line := strings.Join([]string{
		"WUlq4bz95xyFUa1sTLy0yKHQY44eWYeOFj4CfAP5pzs=", // public key
		"(none)",                          // preshared key (none)
		"70.21.35.10:63596",               // endpoint
		"10.66.0.2/32,2602:f481:0:cc::2/128", // allowed-ips
		"1730000000",                      // latest-handshake epoch
		"123456",                          // rx bytes
		"7890123",                         // tx bytes
		"25",                              // persistent-keepalive
	}, "\t")

	p, err := parseAwgPeerDumpLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.PublicKey != "WUlq4bz95xyFUa1sTLy0yKHQY44eWYeOFj4CfAP5pzs=" {
		t.Errorf("public key mismatch: %q", p.PublicKey)
	}
	if p.PresharedKey != "" {
		t.Errorf("preshared key should be empty for '(none)', got %q", p.PresharedKey)
	}
	if p.Endpoint != "70.21.35.10:63596" {
		t.Errorf("endpoint = %q", p.Endpoint)
	}
	if len(p.AllowedIPs) != 2 ||
		p.AllowedIPs[0] != "10.66.0.2/32" ||
		p.AllowedIPs[1] != "2602:f481:0:cc::2/128" {
		t.Errorf("allowed-ips = %v", p.AllowedIPs)
	}
	if p.LatestHandshake.Unix() != 1730000000 {
		t.Errorf("latest-handshake = %v", p.LatestHandshake)
	}
	if p.RxBytes != 123456 || p.TxBytes != 7890123 {
		t.Errorf("rx/tx bytes wrong: rx=%d tx=%d", p.RxBytes, p.TxBytes)
	}
	if p.PersistentKeepalive != 25 {
		t.Errorf("persistent-keepalive = %d", p.PersistentKeepalive)
	}
}

// TestParseAwgPeerDumpLine_OffKeepalive covers a peer with keepalive
// disabled — `awg show dump` emits the literal string "off" in that
// position.
func TestParseAwgPeerDumpLine_OffKeepalive(t *testing.T) {
	line := strings.Join([]string{
		"abc=", "(none)", "(none)", "(none)", "0", "0", "0", "off",
	}, "\t")
	p, err := parseAwgPeerDumpLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.PersistentKeepalive != 0 {
		t.Errorf("keepalive should be 0 for 'off', got %d", p.PersistentKeepalive)
	}
	if !p.LatestHandshake.IsZero() {
		t.Errorf("latest-handshake should be zero, got %v", p.LatestHandshake)
	}
	if p.Endpoint != "" {
		t.Errorf("endpoint should be empty for '(none)', got %q", p.Endpoint)
	}
	if len(p.AllowedIPs) != 0 {
		t.Errorf("allowed-ips should be empty for '(none)', got %v", p.AllowedIPs)
	}
}

// TestAppendAwgParam covers the helper that omits zero-valued AWG
// params (zero == "param disabled" in AWG kernel, no need to noisy-set).
func TestAppendAwgParam(t *testing.T) {
	args := appendAwgParam([]string{"set", "awg0"}, "jc", 0)
	if len(args) != 2 {
		t.Errorf("zero value should not be appended: %v", args)
	}
	args = appendAwgParam(args, "jc", 5)
	if len(args) != 4 || args[2] != "jc" || args[3] != "5" {
		t.Errorf("non-zero should append key+value: %v", args)
	}
	args = appendAwgParamUint(args, "h1", 0)
	if len(args) != 4 {
		t.Errorf("zero uint should not be appended: %v", args)
	}
	args = appendAwgParamUint(args, "h1", 12345)
	if len(args) != 6 || args[4] != "h1" || args[5] != "12345" {
		t.Errorf("non-zero uint should append: %v", args)
	}
}
