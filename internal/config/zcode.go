// G3.5: import the GLM API key from the local ZCode CLI configuration
// (read-only, on user request). The only allowed external read per C1:
// %USERPROFILE%\.zcode\v2\config.json and credentials.json.
package config

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// zcodeKeyFields are the field names (normalized: lowercase, _ and - stripped)
// under which a key-like value is accepted.
var zcodeKeyFields = map[string]bool{
	"apikey":        true,
	"authtoken":     true,
	"token":         true,
	"key":           true,
	"authorization": true,
}

// zcodeFiles are scanned in order; the first plausible key wins.
var zcodeFiles = []string{"config.json", "credentials.json"}

// FindZCodeKey scans dir (normally %USERPROFILE%\.zcode\v2) for the GLM API
// key and returns the first plausible secret found. The key is NEVER logged;
// callers must use MaskKey for any user-visible rendering (D9).
func FindZCodeKey(dir string) (string, bool) {
	if strings.TrimSpace(dir) == "" {
		return "", false
	}
	for _, name := range zcodeFiles {
		f, err := os.Open(filepath.Join(dir, name))
		if err != nil {
			continue // missing file is normal, not an error
		}
		key, ok := scanJSONForKey(f)
		f.Close()
		if ok {
			return key, true
		}
	}
	return "", false
}

// FindZCodeKeyDefault scans %USERPROFILE%\.zcode\v2.
func FindZCodeKeyDefault() (string, bool) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false
	}
	return FindZCodeKey(filepath.Join(home, ".zcode", "v2"))
}

// MaskKey renders a key for logs/UI: only the last 4 characters stay visible.
func MaskKey(key string) string {
	if len(key) <= 4 {
		return "****"
	}
	return strings.Repeat("*", 12) + key[len(key)-4:]
}

// plausibleSecret accepts strings of length >= 20 without any whitespace.
func plausibleSecret(s string) bool {
	if len(s) < 20 {
		return false
	}
	for _, r := range s {
		if unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

// scanJSONForKey walks the JSON document in document order and returns the
// first string value under a key-like field name that passes plausibleSecret.
// A small phase machine distinguishes keys from values inside objects; corrupt
// files yield "not found", never an error (D4).
func scanJSONForKey(r io.Reader) (string, bool) {
	dec := json.NewDecoder(r)
	type frame struct {
		object  bool // object (keys possible) vs array
		wantKey bool // next string token in this object is a key
	}
	var stack []frame
	currentKey := ""

	valueDone := func() {
		if n := len(stack); n > 0 && stack[n-1].object {
			stack[n-1].wantKey = true // a completed value is followed by a key or '}'
		}
	}
	for {
		tok, err := dec.Token()
		if err != nil {
			return "", false // includes io.EOF
		}
		switch t := tok.(type) {
		case json.Delim:
			switch t {
			case '{':
				stack = append(stack, frame{object: true, wantKey: true})
			case '[':
				stack = append(stack, frame{object: false})
			default: // '}' or ']': the parent finished one composite value
				if n := len(stack); n > 0 {
					stack = stack[:n-1]
				}
				valueDone()
			}
		case string:
			n := len(stack)
			if n > 0 && stack[n-1].object && stack[n-1].wantKey {
				currentKey = t
				stack[n-1].wantKey = false
				continue
			}
			if zcodeKeyFields[normalizeField(currentKey)] && plausibleSecret(t) {
				return t, true
			}
			valueDone()
		default: // numbers, booleans, null: complete a value
			valueDone()
		}
	}
}

// normalizeField lowercases and strips separators for tolerant matching.
func normalizeField(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "_", "")
	s = strings.ReplaceAll(s, "-", "")
	return s
}
