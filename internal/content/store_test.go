// G3.1: content store tests — load skip/clamp behavior, atomic writes,
// EnsureSeed idempotency and pack export/import round-trip.
package content

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Load(filepath.Join(t.TempDir(), "content"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return s
}

func TestLoadCreatesDirs(t *testing.T) {
	root := filepath.Join(t.TempDir(), "content")
	if _, err := Load(root); err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, d := range []string{DirSpecies, DirWater, DirPlants, DirPatterns, DirCorals} {
		if fi, err := os.Stat(filepath.Join(root, d)); err != nil || !fi.IsDir() {
			t.Fatalf("dir %s not created", d)
		}
	}
}

func TestLoadSkipsUnparseableJSON(t *testing.T) {
	root := t.TempDir()
	s, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	bad := filepath.Join(root, DirSpecies, "broken.json")
	if err := os.WriteFile(bad, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	s2, err := Load(root)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(s2.Species()) != 0 {
		t.Fatalf("broken file should be skipped, got %d species", len(s2.Species()))
	}
	_ = s
}

func TestLoadRejectsBadHexAndSlug(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, DirSpecies), 0o755)
	os.WriteFile(filepath.Join(root, DirSpecies, "badhex.json"),
		[]byte(`{"id":"badhex","name":"Bad Hex","palette":{"body":"zzz","belly":"#ffffff","accent":"#ffffff","glow":"#ffffff"},"pattern":{"type":"spot"}}`), 0o644)
	os.WriteFile(filepath.Join(root, DirSpecies, "Bad Slug!.json"),
		[]byte(`{"id":"Bad Slug!","name":"Bad Slug","palette":{"body":"#111111","belly":"#222222","accent":"#333333","glow":"#444444"},"pattern":{"type":"spot"}}`), 0o644)
	s, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(s.Species()) != 0 {
		t.Fatalf("invalid files must be skipped, got %d", len(s.Species()))
	}
}

func TestLoadClampsRanges(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, DirSpecies), 0o755)
	os.WriteFile(filepath.Join(root, DirSpecies, "huge.json"),
		[]byte(`{"id":"huge","name":"Huge","size":99,"width":-3,"fin":100,"tail":0.0,"palette":{"body":"#101010","belly":"#202020","accent":"#303030","glow":"#404040"},"pattern":{"type":"wave","density":7,"size":-1},"behavior":{"speed":42,"schooling":9,"curiosity":-2,"skittish":3,"depth":-9}}`), 0o644)
	s, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	sp := s.SpeciesByID("huge")
	if sp == nil {
		t.Fatal("clamped species should load, not be rejected")
	}
	// N9: no role in the file -> normal species -> supremacy caps apply.
	if sp.Size != contract.NormalSizeMax || sp.Width != spWidthMin || sp.Fin != contract.NormalFinMax || sp.Tail != spTailMin {
		t.Fatalf("species clamps wrong: %+v", sp)
	}
	if sp.Pattern.Density != 1 || sp.Pattern.Size != 0 || sp.Behavior.Speed != contract.NormalSpeedMax || sp.Behavior.Schooling != 1 {
		t.Fatalf("pattern/behavior clamps wrong: %+v", sp)
	}
}

func TestWriteSpeciesClampedCopy(t *testing.T) {
	s := newTestStore(t)
	in := &contract.Species{
		ID:      "test-fish",
		Name:    "Test Fish",
		Size:    55,
		Width:   -1,
		Palette: contract.Palette{Body: "#112233", Belly: "#445566", Accent: "#778899", Glow: "#aabbcc"},
		Pattern: contract.Pattern{Type: "koi", Density: 4, Size: -2},
		Behavior: contract.Behavior{
			Speed: 99, Schooling: 3, Curiosity: -1, Skittish: 0.5, Depth: 0.5,
		},
		Source: "species-agent",
	}
	if err := s.WriteSpecies(in); err != nil {
		t.Fatalf("WriteSpecies: %v", err)
	}
	if in.Size != 55 { // caller struct must stay untouched (clamped copy)
		t.Fatalf("input mutated: %+v", in)
	}
	got := s.SpeciesByID("test-fish")
	if got == nil || got.Size != contract.NormalSizeMax || got.Behavior.Speed != contract.NormalSpeedMax {
		t.Fatalf("stored copy not clamped: %+v", got)
	}
	if _, err := os.Stat(filepath.Join(s.Root(), DirSpecies, "test-fish.json")); err != nil {
		t.Fatalf("file not written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(s.Root(), DirSpecies, "test-fish.json.tmp")); !os.IsNotExist(err) {
		t.Fatalf("tmp file should be renamed away")
	}
}

func TestWriteRejectsBadSlugAndHex(t *testing.T) {
	s := newTestStore(t)
	err := s.WriteSpecies(&contract.Species{ID: "Bad!", Name: "x"})
	if err == nil || !strings.Contains(err.Error(), "slug") {
		t.Fatalf("expected slug error, got %v", err)
	}
	err = s.WriteSpecies(&contract.Species{ID: "ok-id", Name: "x",
		Palette: contract.Palette{Body: "nope", Belly: "#ffffff", Accent: "#ffffff", Glow: "#ffffff"}})
	if err == nil || !strings.Contains(err.Error(), "#rrggbb") {
		t.Fatalf("expected hex error, got %v", err)
	}
}

func TestEnsureSeedIdempotent(t *testing.T) {
	s := newTestStore(t)
	if err := s.EnsureSeed(); err != nil {
		t.Fatalf("EnsureSeed: %v", err)
	}
	first := map[string]int{
		KindSpecies: len(s.Species()), KindWater: len(s.Waters()),
		KindPlant: len(s.Plants()), KindPattern: len(s.Recipes()),
		KindCoral: len(s.Corals()),
	}
	want := map[string]int{KindSpecies: 11, KindWater: 3, KindPlant: 10, KindPattern: 4, KindCoral: 4} // v0.3.7 F30 flora, v1.1: +titan & shark
	for k, n := range want {
		if first[k] != n {
			t.Fatalf("seed %s = %d, want %d", k, first[k], n)
		}
	}
	regAfterFirst := s.reg.Len()
	if regAfterFirst != 32 { // v0.3.7 F30: 30 + the v1.1 titan & shark
		t.Fatalf("registry entries after seed = %d, want 30", regAfterFirst)
	}
	// Second run must be a no-op.
	if err := s.EnsureSeed(); err != nil {
		t.Fatalf("EnsureSeed 2: %v", err)
	}
	if len(s.Species()) != 11 || s.reg.Len() != regAfterFirst {
		t.Fatalf("EnsureSeed not idempotent: species=%d registry=%d", len(s.Species()), s.reg.Len())
	}
	for _, sp := range s.Species() {
		if sp.Source != "core" {
			t.Fatalf("seed species source = %q, want core", sp.Source)
		}
	}
	stages := map[string]bool{}
	for _, r := range s.Recipes() {
		stages[r.Stage] = true
	}
	for _, st := range []string{"fry", "juvenile", "adult", "elder"} {
		if !stages[st] {
			t.Fatalf("missing seed recipe for stage %s", st)
		}
	}
}

func TestExportImportRoundTrip(t *testing.T) {
	src := newTestStore(t)
	if err := src.EnsureSeed(); err != nil {
		t.Fatal(err)
	}
	extra := &contract.Species{
		ID: "extra-fish", Name: "Extra Fish", Size: 1,
		Palette: contract.Palette{Body: "#010203", Belly: "#040506", Accent: "#070809", Glow: "#0a0b0c"},
		Pattern: contract.Pattern{Type: "vein", Density: 0.3, Size: 0.4},
	}
	if err := src.WriteSpecies(extra); err != nil {
		t.Fatal(err)
	}
	zipPath := filepath.Join(t.TempDir(), "pack.zip")
	if err := src.ExportPack(zipPath); err != nil {
		t.Fatalf("ExportPack: %v", err)
	}

	dst, err := Load(filepath.Join(t.TempDir(), "dst"))
	if err != nil {
		t.Fatal(err)
	}
	if err := dst.ImportPack(zipPath); err != nil {
		t.Fatalf("ImportPack: %v", err)
	}
	if len(dst.Species()) != 12 || len(dst.Waters()) != 3 || len(dst.Plants()) != 10 || len(dst.Recipes()) != 4 || len(dst.Corals()) != 4 {
		t.Fatalf("import file set mismatch: sp=%d wa=%d pl=%d rc=%d co=%d",
			len(dst.Species()), len(dst.Waters()), len(dst.Plants()), len(dst.Recipes()), len(dst.Corals()))
	}
	if dst.SpeciesByID("extra-fish") == nil {
		t.Fatal("extra-fish missing after import")
	}
	// Registry merged: source had 31 entries (30 seed + extra), dst must have all.
	if got := dst.reg.Len(); got != src.reg.Len() {
		t.Fatalf("registry after import = %d, want %d", got, src.reg.Len())
	}
	// Re-import must dedupe (no registry growth).
	if err := dst.ImportPack(zipPath); err != nil {
		t.Fatal(err)
	}
	if got := dst.reg.Len(); got != src.reg.Len() {
		t.Fatalf("registry grew on re-import: %d != %d", got, src.reg.Len())
	}
}
