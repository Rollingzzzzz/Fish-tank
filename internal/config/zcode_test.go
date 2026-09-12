// G3.5: ZCode key import tests — fixture dirs with config.json/credentials.json
// in a temp dir; ordering, field-name matching, implausibility filters, masking.
package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestFindZCodeKeyInConfig(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "config.json"),
		`{"client":{"theme":"dark","apiKey":"gsk_abcdefghijklmnopqrstuvwxyz123456"},"other":{"token":"short"}}`)
	key, ok := FindZCodeKey(dir)
	if !ok {
		t.Fatal("key in nested apiKey field not found")
	}
	if key != "gsk_abcdefghijklmnopqrstuvwxyz123456" {
		t.Fatalf("wrong key found: %q", MaskKey(key))
	}
}

func TestFindZCodeKeyFallbackCredentials(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "credentials.json"),
		`{"authtoken":"ABCDEFGHIJKLMNOPQRSTUVWX","note":"primary session"}`)
	key, ok := FindZCodeKey(dir)
	if !ok || key != "ABCDEFGHIJKLMNOPQRSTUVWX" {
		t.Fatalf("credentials.json authtoken not found: %v %q", ok, MaskKey(key))
	}
}

func TestFindZCodeKeyConfigWinsOverCredentials(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "config.json"), `{"token":"CONFIGCONFIGCONFIGCONFIG12"}`)
	writeFile(t, filepath.Join(dir, "credentials.json"), `{"token":"CREDCREDCREDCREDCRED12"}`)
	key, _ := FindZCodeKey(dir)
	if key != "CONFIGCONFIGCONFIGCONFIG12" {
		t.Fatalf("config.json must be scanned first, got %q", MaskKey(key))
	}
}

func TestFindZCodeKeyFilters(t *testing.T) {
	dir := t.TempDir()
	// Too short, contains whitespace, wrong field name, non-string: all skipped;
	// the api_key (underscore + mixed case) variant must still match.
	writeFile(t, filepath.Join(dir, "config.json"), `{
		"apiKeyShort": "gsk_short",
		"description": "a long sentence with whitespace inside",
		"keyId": "abcdefghijklmnopqrst",
		"retries": 12345,
		"api_key": "gsk_ABCDEFGHIJKLMNOPQRSTUVWX"
	}`)
	key, ok := FindZCodeKey(dir)
	if !ok || key != "gsk_ABCDEFGHIJKLMNOPQRSTUVWX" {
		t.Fatalf("implausible values must be skipped; got %v %q", ok, MaskKey(key))
	}
}

func TestFindZCodeKeyArrayAndDeepNesting(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "credentials.json"), `{
		"sessions": [{"name":"main","authorization":"BearerBearerBearer123456"}],
		"provider": {"auth": {"KEY": "nestednestednested12345678"}}
	}`)
	key, ok := FindZCodeKey(dir)
	if !ok || key != "BearerBearerBearer123456" {
		t.Fatalf("first match in document order expected; got %v %q", ok, MaskKey(key))
	}
}

func TestFindZCodeKeyMissing(t *testing.T) {
	dir := t.TempDir()
	if _, ok := FindZCodeKey(dir); ok {
		t.Fatal("empty dir must not find a key")
	}
	if _, ok := FindZCodeKey(filepath.Join(dir, "does-not-exist")); ok {
		t.Fatal("missing dir must not find a key")
	}
	if _, ok := FindZCodeKey(""); ok {
		t.Fatal("empty dir argument must not find a key")
	}
}

func TestFindZCodeKeyCorruptJSON(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "config.json"), `{not valid json at all`)
	if _, ok := FindZCodeKey(dir); ok {
		t.Fatal("corrupt JSON must not find a key (and must not crash)")
	}
}

func TestMaskKey(t *testing.T) {
	full := "gsk_supersecretvalue_9abc"
	masked := MaskKey(full)
	if !strings.HasSuffix(masked, "9abc") {
		t.Fatalf("masked key must keep last 4 chars: %q", masked)
	}
	if strings.Contains(masked, "supersecret") {
		t.Fatal("masked key leaks the secret body")
	}
	if MaskKey("abc") != "****" {
		t.Fatal("short keys must be fully masked")
	}
}
