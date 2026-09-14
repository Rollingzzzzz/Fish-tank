// G3.6: registry + gate tests — cap drop, hue-shift duplication, name
// uniqueness, recent fingerprints and registry merge dedupe.
package content

import (
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// hslHex is a test-only HSL to #rrggbb converter (mirrors the simulate
// generator's helper) used to craft exact-hue palettes.
func hslHex(h, s, l float64) string {
	h = math.Mod(math.Mod(h, 360)+360, 360)
	c := (1 - math.Abs(2*l-1)) * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := l - c/2
	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	to := func(v float64) int { return int(math.Round((v + m) * 255)) }
	return fmt.Sprintf("#%02x%02x%02x", to(r), to(g), to(b))
}

func hexFromHue(h float64) string { return hslHex(h, 1, 0.5) }

func palAtHue(h float64) contract.Palette {
	return contract.Palette{
		Body:   hexFromHue(h),
		Belly:  hexFromHue(h + 30),
		Accent: hexFromHue(h + 180),
		Glow:   hexFromHue(h + 200),
	}
}

func pat() contract.Pattern { return contract.Pattern{Type: "stripe", Density: 0.5, Size: 0.5} }

func TestGateHueShift(t *testing.T) {
	s := newTestStore(t)
	base := palAtHue(100)
	if err := s.WriteSpecies(&contract.Species{
		ID: "first-fish", Name: "First Fish", Pattern: pat(), Palette: base,
	}); err != nil {
		t.Fatal(err)
	}
	// 2 degree hue shift stays inside the 10 degree bucket -> duplicate.
	ok, conflicts := s.Gate(KindSpecies, "second-fish", "Second Fish", pat(), palAtHue(102))
	if ok {
		t.Fatal("2 degree shift must be rejected as duplicate")
	}
	if len(conflicts) == 0 {
		t.Fatal("conflicts must list the colliding hash")
	}
	// 20 degree shift moves at least the body hue bucket -> accepted.
	ok, _ = s.Gate(KindSpecies, "second-fish", "Second Fish", pat(), palAtHue(120))
	if !ok {
		t.Fatal("20 degree shift must be accepted")
	}
}

func TestGateNameCaseInsensitive(t *testing.T) {
	s := newTestStore(t)
	sp := &contract.Species{
		ID: "alpha-fish", Name: "Alpha Fish", Pattern: pat(), Palette: palAtHue(10),
	}
	if err := s.WriteSpecies(sp); err != nil {
		t.Fatal(err)
	}
	ok, conflicts := s.Gate(KindSpecies, "beta-fish", "ALPHA FISH", pat(), palAtHue(200))
	if ok {
		t.Fatal("case-insensitive duplicate name must be rejected")
	}
	if len(conflicts) == 0 || conflicts[0] != "name:alpha fish" {
		t.Fatalf("name conflict token wrong: %v", conflicts)
	}
	// Different name, different hue: accepted.
	if ok, _ = s.Gate(KindSpecies, "gamma-fish", "Gamma Fish", pat(), palAtHue(220)); !ok {
		t.Fatal("distinct artifact must pass the gate")
	}
	// Re-gating the identical registered species: name passes (self), but the
	// fingerprint is already registered, so it must be rejected (C6).
	ok, conflicts = s.Gate(KindSpecies, "alpha-fish", "Alpha Fish", pat(), palAtHue(10))
	if ok {
		t.Fatal("registered fingerprint must be rejected even for the same ID")
	}
	if len(conflicts) != 1 || strings.HasPrefix(conflicts[0], "name:") {
		t.Fatalf("expected only the fingerprint conflict, got %v", conflicts)
	}
}

func TestGateRecipe(t *testing.T) {
	s := newTestStore(t)
	r := &contract.PatternRecipe{ID: "r-one", Stage: "fry", Name: "R One",
		SatMul: 0.8, AlphaMul: 0.6, GlowMul: 1.0,
		Pattern: contract.Pattern{Type: "spot", Density: 0.4, Size: 0.5}}
	if err := s.WriteRecipe(r); err != nil {
		t.Fatal(err)
	}
	dup := *r
	dup.ID = "r-two"
	dup.Name = "R Two"
	if ok, _ := s.GateRecipe(&dup); ok {
		t.Fatal("identical recipe fingerprint must be rejected")
	}
	dup.SatMul += 0.3
	dup.AlphaMul += 0.2
	dup.GlowMul += 0.4
	if ok, _ := s.GateRecipe(&dup); !ok {
		t.Fatal("materially different recipe must pass")
	}
}

func TestRegistryCapDropsOldest(t *testing.T) {
	s := newTestStore(t)
	reg := s.reg
	for i := 0; i < contract.RegistryCap; i++ {
		reg.appendEntry(RegistryEntry{Hash: "h" + string(rune('a'+i%26)) + itoa(i), Kind: "species", ID: itoa(i)})
	}
	if reg.Len() != contract.RegistryCap {
		t.Fatalf("len = %d, want cap", reg.Len())
	}
	first := reg.Entries()[0].Hash
	reg.appendEntry(RegistryEntry{Hash: "newest", Kind: "species", ID: "newest"})
	if reg.Len() != contract.RegistryCap {
		t.Fatalf("cap overflow: len = %d", reg.Len())
	}
	if reg.Entries()[0].Hash == first {
		t.Fatal("oldest entry must be dropped on overflow")
	}
	if reg.Entries()[reg.Len()-1].Hash != "newest" {
		t.Fatal("newest entry must be last")
	}
	reg.Save()
}

func TestRecentFingerprints(t *testing.T) {
	s := newTestStore(t)
	s.reg.Append(KindSpecies, "a", "hash-a")
	s.reg.Append(KindSpecies, "b", "hash-b")
	got := s.RecentFingerprints(2)
	if len(got) != 2 || got[0] != "hash-a" || got[1] != "hash-b" {
		t.Fatalf("RecentFingerprints order wrong: %v", got)
	}
	got = s.RecentFingerprints(1)
	if len(got) != 1 || got[0] != "hash-b" {
		t.Fatalf("RecentFingerprints(1) wrong: %v", got)
	}
	if got = s.RecentFingerprints(0); got != nil {
		t.Fatalf("n=0 must return nil, got %v", got)
	}
}

func TestRegistryMergeDedupes(t *testing.T) {
	r, err := LoadRegistry(filepath.Join(t.TempDir(), "registry.json"))
	if err != nil {
		t.Fatal(err)
	}
	r.Append(KindSpecies, "a", "hash-a")
	added := r.Merge([]RegistryEntry{
		{Hash: "hash-a", Kind: KindSpecies, ID: "a"}, // duplicate
		{Hash: "hash-c", Kind: KindWater, ID: "c"},
	})
	if added != 1 {
		t.Fatalf("merge added %d, want 1 (identical hashes dedupe)", added)
	}
	if !r.Has("hash-a") || !r.Has("hash-c") {
		t.Fatal("merged entries missing")
	}
	// Re-load from disk must keep merged entries (persistence check).
	r2, err := LoadRegistry(r.path)
	if err != nil {
		t.Fatal(err)
	}
	if !r2.Has("hash-c") {
		t.Fatal("merge not persisted")
	}
}
