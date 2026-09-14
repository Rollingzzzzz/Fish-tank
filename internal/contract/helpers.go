// G0.2: small helpers shared by every package (stdlib only).
package contract

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"math/rand"
	"strings"
)

// Clamp constrains v to [lo, hi].
func Clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// RandSeed returns a deterministic RNG for the given seed (D5).
func RandSeed(seed int64) *rand.Rand { return rand.New(rand.NewSource(seed)) }

// Slugify converts s to an ASCII slug valid as an ID: ^[a-z0-9-]{3,32}$.
// Letters/digits are kept, everything else collapses to '-'; result is trimmed
// to 32 chars and padded/sanitized to at least 3 chars with "x".
func Slugify(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	out := b.String()
	for strings.Contains(out, "--") {
		out = strings.ReplaceAll(out, "--", "-")
	}
	out = strings.Trim(out, "-")
	if len(out) > 32 {
		out = strings.Trim(out[:32], "-")
	}
	if len(out) < 3 {
		out = strings.TrimRight(out+"xxx", "-")
		if len(out) < 3 {
			out = "xxx"
		}
	}
	return out
}

// Fingerprint returns the first 16 hex chars of SHA-256(kind + ":" + canonical).
// Used by the uniqueness gate (C6).
func Fingerprint(kind, canonical string) string {
	sum := sha256.Sum256([]byte(kind + ":" + canonical))
	return hex.EncodeToString(sum[:])[:16]
}

// Quantize snaps v to the nearest multiple of q (used by fingerprinting).
func Quantize(v, q float64) float64 {
	if q == 0 {
		return v
	}
	return math.Round(v/q) * q
}

// ValidHex reports whether s looks like "#rrggbb".
func ValidHex(s string) bool {
	if len(s) != 7 || s[0] != '#' {
		return false
	}
	for _, c := range s[1:] {
		if !strings.ContainsRune("0123456789abcdefABCDEF", c) {
			return false
		}
	}
	return true
}

// HexToRGB parses "#rrggbb" to 8-bit components; returns black on error.
func HexToRGB(s string) (r, g, b uint8) {
	if !ValidHex(s) {
		return 0, 0, 0
	}
	_, _ = fmt.Sscanf(strings.ToLower(s), "#%02x%02x%02x", &r, &g, &b)
	return r, g, b
}
