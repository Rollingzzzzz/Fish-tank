// N1/FD8: coral content tests — write/load roundtrip, clamps, strict kind
// enum, pack export, gate fingerprints, seed loading and the lilac band
// reservation for species.
package content

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

func coralFixture(id, name string) *contract.CoralDesign {
	return &contract.CoralDesign{
		ID: id, Name: name, Kind: "fan", Fronds: 5, Height: 0.2, Width: 1.0,
		Curve: 0.4, Sway: 0.4, Colors: []string{"#123456", "#654321", "#abcdef"},
		Glow: 0.5, Source: "water-agent",
	}
}

func TestCoralWriteLoadRoundTrip(t *testing.T) {
	root := filepath.Join(t.TempDir(), "content")
	s, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	in := coralFixture("test-fan", "Test Fan")
	if err := s.WriteCoral(in); err != nil {
		t.Fatalf("WriteCoral: %v", err)
	}
	if in.Fronds != 5 { // caller struct must stay untouched (clamped copy)
		t.Fatal("input mutated by WriteCoral")
	}
	s2, err := Load(root) // reload from disk
	if err != nil {
		t.Fatal(err)
	}
	got := s2.Corals()
	if len(got) != 1 || got[0].ID != "test-fan" || got[0].Name != "Test Fan" {
		t.Fatalf("roundtrip mismatch: %+v", got)
	}
	if got[0].Kind != "fan" || len(got[0].Colors) != 3 || got[0].Source != "water-agent" {
		t.Fatalf("fields not preserved: %+v", got[0])
	}
	if got[0].CreatedAt == "" {
		t.Fatal("CreatedAt must be stamped on write")
	}
}

func TestCoralClamps(t *testing.T) {
	s := newTestStore(t)
	in := &contract.CoralDesign{ID: "clampy", Name: "Clampy", Kind: "", Fronds: 99,
		Height: 5, Width: 0.01, Curve: -3, Sway: 7, Glow: 2,
		Colors: []string{"#111111", "#222222", "#333333", "#444444", "#555555"}}
	if err := s.WriteCoral(in); err != nil {
		t.Fatalf("WriteCoral: %v", err)
	}
	got := s.Corals()[0]
	if got.Kind != "fan" || got.Fronds != coFrondsMax || got.Height != coHeightMax ||
		got.Width != coWidthMin || got.Curve != 0 || got.Sway != 1 || got.Glow != 1 {
		t.Fatalf("clamps wrong: %+v", got)
	}
	if len(got.Colors) != 4 {
		t.Fatalf("colors clamp wrong: %v", got.Colors)
	}
	// One color is padded up to the two-stop minimum.
	in2 := &contract.CoralDesign{ID: "onecolor", Name: "One Color", Kind: "brain",
		Colors: []string{"#0a0b0c"}}
	if err := s.WriteCoral(in2); err != nil {
		t.Fatalf("WriteCoral one color: %v", err)
	}
	var one *contract.CoralDesign
	for _, c := range s.Corals() {
		if c.ID == "onecolor" {
			one = c
		}
	}
	if one == nil || len(one.Colors) != 2 {
		t.Fatalf("single color not padded to 2: %v", one)
	}
}

func TestCoralInvalidKindRejected(t *testing.T) {
	s := newTestStore(t)
	err := s.WriteCoral(&contract.CoralDesign{ID: "tree-coral", Name: "Tree", Kind: "tree",
		Colors: []string{"#101010", "#202020"}})
	if err == nil || !strings.Contains(err.Error(), "unknown kind") {
		t.Fatalf("invalid kind must be rejected, got %v", err)
	}
	err = s.ValidateCoral(&contract.CoralDesign{ID: "bad-id!", Name: "Bad", Kind: "fan",
		Colors: []string{"#101010", "#202020"}})
	if err == nil || !strings.Contains(err.Error(), "slug") {
		t.Fatalf("bad slug must be rejected, got %v", err)
	}
	// Unknown kind files on disk are skipped on load (never fatal, D4).
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, DirCorals), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, DirCorals, "wildkind.json"),
		[]byte(`{"id":"wildkind","name":"Wild","kind":"tree","colors":["#101010","#202020"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	s2, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(s2.Corals()) != 0 {
		t.Fatal("unknown kind file must be skipped on load")
	}
}

func TestCoralPackExportAndFingerprint(t *testing.T) {
	s := newTestStore(t)
	if err := s.EnsureSeed(); err != nil {
		t.Fatal(err)
	}
	extra := coralFixture("extra-coral", "Extra Coral")
	if err := s.WriteCoral(extra); err != nil {
		t.Fatal(err)
	}
	zipPath := filepath.Join(t.TempDir(), "pack.zip")
	if err := s.ExportPack(zipPath); err != nil {
		t.Fatalf("ExportPack: %v", err)
	}
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	coralEntries := 0
	hasSeed, hasExtra := false, false
	for _, f := range r.File {
		if !strings.HasPrefix(f.Name, DirCorals+"/") {
			continue
		}
		coralEntries++
		if strings.HasSuffix(f.Name, "violet-seafern.json") {
			hasSeed = true
		}
		if strings.HasSuffix(f.Name, "extra-coral.json") {
			hasExtra = true
		}
	}
	if coralEntries != 5 || !hasSeed || !hasExtra {
		t.Fatalf("pack corals = %d (seed %v, extra %v)", coralEntries, hasSeed, hasExtra)
	}
	// Gate: a fresh design passes; its exact duplicate is fingerprint-rejected.
	fresh := coralFixture("fresh-coral", "Fresh Coral")
	fresh.Kind, fresh.Fronds, fresh.Curve, fresh.Sway = "branch", 8, 0.8, 0.2
	fresh.Colors = []string{"#1a2b3c", "#3c2b1a", "#d0e0f0"}
	if ok, _ := s.GateCoral(fresh); !ok {
		t.Fatal("new coral must pass GateCoral")
	}
	if err := s.WriteCoral(fresh); err != nil {
		t.Fatalf("WriteCoral fresh: %v", err)
	}
	dup := coralFixture("dup-coral", "Dup Coral")
	dup.Kind, dup.Fronds, dup.Curve, dup.Sway = fresh.Kind, fresh.Fronds, fresh.Curve, fresh.Sway
	dup.Colors = append([]string(nil), fresh.Colors...)
	ok, conflicts := s.GateCoral(dup)
	if ok || len(conflicts) == 0 {
		t.Fatalf("identical coral design must be rejected, conflicts=%v", conflicts)
	}
}

func TestSeedCoralsAndSpeciesLoad(t *testing.T) {
	root := filepath.Join(t.TempDir(), "content")
	s, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.EnsureSeed(); err != nil {
		t.Fatal(err)
	}
	s2, err := Load(root) // reload: seeds must come back through the reader
	if err != nil {
		t.Fatal(err)
	}
	wantCorals := map[string]string{
		"violet-seafern": "fan", "magma-brain": "brain",
		"cyan-candlebranch": "branch", "rose-veilfan": "fan",
	}
	got := s2.Corals()
	if len(got) != len(wantCorals) {
		t.Fatalf("coral seeds = %d, want %d", len(got), len(wantCorals))
	}
	for _, c := range got {
		kind, ok := wantCorals[c.ID]
		if !ok || c.Kind != kind || c.Source != "core" {
			t.Fatalf("seed coral %s wrong (kind %s, source %s)", c.ID, c.Kind, c.Source)
		}
	}
	sps := s2.Species()
	if len(sps) != 11 { // N9: 6 original + ribbon-streamer, puff-orbit, dart-spindle + v1.1 titan & shark
		t.Fatalf("species seeds = %d, want 11", len(sps))
	}
	glass := s2.SpeciesByID("glass-sucker")
	if glass == nil || glass.Role != contract.RoleNormal || glass.Behavior.Attachment != 0.9 {
		t.Fatalf("glass-sucker wrong: %+v", glass)
	}
	chosen := s2.SpeciesByID("chosen-lilastar")
	if chosen == nil || chosen.Role != contract.RoleChosen {
		t.Fatalf("chosen-lilastar wrong: %+v", chosen)
	}
}

func TestLilacReservation(t *testing.T) {
	if !lilacReserved(contract.Palette{Body: "#b57edc"}) {
		t.Fatal("lilac body must be reserved")
	}
	if lilacReserved(contract.Palette{Body: "#5f7d8c", Belly: "#8fb3b6",
		Accent: "#9fe8d8", Glow: "#6fd8c8"}) {
		t.Fatal("teal-slate palette must not be reserved")
	}
	s := newTestStore(t)
	lilac := contract.Palette{Body: "#b57edc", Belly: "#d9c2f0", Accent: "#e6d6ff", Glow: "#c77dff"}
	// 1) A plain agent species wearing lilac is rejected for repair.
	sp := &contract.Species{ID: "copycat-fish", Name: "Copycat", Source: "species-agent",
		Pattern: pat(), Palette: lilac}
	if ok, conflicts := s.GateSpecies(sp); ok || len(conflicts) == 0 {
		t.Fatalf("lilac agent species must be rejected, conflicts=%v", conflicts)
	}
	// 2) Core source is exempt.
	sp2 := &contract.Species{ID: "core-fish", Name: "Core Fish", Source: "core",
		Pattern: pat(), Palette: lilac}
	if ok, _ := s.GateSpecies(sp2); !ok {
		t.Fatal("core species must be lilac-exempt")
	}
	// 3) The Chosen's own ID is exempt.
	sp3 := &contract.Species{ID: chosenSpeciesID, Name: "Chosen Lilastar", Source: "user",
		Pattern: pat(), Palette: lilac}
	if ok, _ := s.GateSpecies(sp3); !ok {
		t.Fatal("chosen species must be lilac-exempt")
	}
	// 4) Non-lilac agent species pass.
	sp4 := &contract.Species{ID: "teal-fish", Name: "Teal Fish", Source: "species-agent",
		Pattern: pat(), Palette: palAtHue(160)}
	if ok, _ := s.GateSpecies(sp4); !ok {
		t.Fatal("non-lilac species must pass")
	}
}
