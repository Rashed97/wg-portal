package handlers

import (
	"encoding/base64"
	"strings"
)

// Base64UrlDecode decodes a base64-encoded string that has been made URL-
// path-safe by substituting standard-base64 characters that need %-encoding
// in URLs. The substitution map (note: NOT RFC 4648 §5):
//
//	URL form  →  standard base64
//	    -     →     =
//	    _     →     /
//	    .     →     +
//
// In other words, callers (frontend or external API users) must:
//  1. Take the identifier they want to address (interface name like "wg0",
//     or a peer identifier which is itself the standard-base64 of the
//     32-byte WireGuard public key, etc.) and standard-base64-encode it
//     AS UTF-8 bytes — this gives a URL-safe alphabet plus '+', '/', '='.
//  2. Apply the substitution above so the result is safe in a URL path.
//
// Example: peer identifier "BOz1...HQ=" → b64-encode the literal string
// → "Qk96...UT0=" → substitute → "Qk96...UT0-" → use as path segment.
//
// Decode errors are silently swallowed and an empty string is returned;
// handlers must check for "" and return HTTP 400 themselves.
func Base64UrlDecode(in string) string {
	in = strings.ReplaceAll(in, "-", "=")
	in = strings.ReplaceAll(in, "_", "/")
	in = strings.ReplaceAll(in, ".", "+")

	output, _ := base64.StdEncoding.DecodeString(in)
	return string(output)
}
