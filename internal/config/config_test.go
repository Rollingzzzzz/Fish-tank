// F29: the config clamp admits a 100-fish stock — a hand-edited
// maxFish must survive Load and Save without snapping back to the old 30.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// v1: a fresh install wakes up self-sufficient — 60 fish, auto feed and
// auto care on, music off.
func TestFreshInstallDefaults(t *testing.T) {
	c := Default()
	if c.MaxFish != 60 {
		t.Fatalf("default MaxFish = %d, want 60", c.MaxFish)
	}
	if !c.AutoFeed || !c.AutoCare {
		t.Fatal("v1 defaults must ship with auto feed + auto care ON")
	}
	if c.MusicOn {
		t.Fatal("v1 default ships music OFF")
	}
	// an existing config keeps the player's explicit choices on upgrade
	b := []byte(`{"maxFish": 24, "autoFeed": false, "musicOn": true}`)
	cfg := Default()
	if err := json.Unmarshal(b, cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.MaxFish != 24 || cfg.AutoFeed || !cfg.MusicOn {
		t.Fatal("an explicit config must override the new defaults")
	}
}

func TestMaxFishClampAdmits100(t *testing.T) {
	cfg := Default()
	cfg.MaxFish = 100
	p := filepath.Join(t.TempDir(), "config.json")
	if err := Save(p, cfg); err != nil {
		t.Fatal(err)
	}
	got, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if got.MaxFish != 100 {
		t.Fatalf("maxFish 100 snapped back to %d — the clamp still caps at 30", got.MaxFish)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatal(err)
	}
}
