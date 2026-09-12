// N9: species variety + Chosen supremacy tests — every normal seed stays
// under the contract.Normal* caps (so no elder can ever reach the Chosen's
// 64*1.4 elder length), the new seeds validate and pass the FD8 lilac gate,
// and role-aware clamping keeps the full range for the core Chosen alone.
package content

import (
	"strings"
	"testing"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// elderChosenLen is the Chosen's maximum body length (base px x her 1.4 size x
// the 1.04 elder growth factor).
const elderChosenLen = 64 * 1.4

// TestSeedSpeciesSupremacy: 9 seeds, unique names, and every normal species
// strictly below the Chosen's scale/finery for its whole life.
func TestSeedSpeciesSupremacy(t *testing.T) {
	s := newTestStore(t)
	if err := s.EnsureSeed(); err != nil {
		t.Fatalf("EnsureSeed: %v", err)
	}
	list := s.Species()
	if len(list) != 9 {
		t.Fatalf("seed species = %d, want 9", len(list))
	}
	seen := map[string]bool{}
	chosen := 0
	for _, sp := range list {
		key := strings.ToLower(sp.Name)
		if seen[key] {
			t.Fatalf("duplicate species name %q", sp.Name)
		}
		seen[key] = true
		if sp.Role == contract.RoleChosen {
			chosen++
			if sp.Size != spSizeMax || sp.Fin != spFinMax || sp.Tail != spTailMax {
				t.Fatalf("chosen %s must keep the full range, got size=%v fin=%v tail=%v",
					sp.ID, sp.Size, sp.Fin, sp.Tail)
			}
			continue
		}
		if sp.Size > contract.NormalSizeMax {
			t.Fatalf("%s size %v exceeds NormalSizeMax %v", sp.ID, sp.Size, contract.NormalSizeMax)
		}
		if sp.Fin > contract.NormalFinMax {
			t.Fatalf("%s fin %v exceeds NormalFinMax %v", sp.ID, sp.Fin, contract.NormalFinMax)
		}
		if sp.Tail > contract.NormalTailMax {
			t.Fatalf("%s tail %v exceeds NormalTailMax %v", sp.ID, sp.Tail, contract.NormalTailMax)
		}
		if sp.Behavior.Speed > contract.NormalSpeedMax { // F23: pace supremacy
			t.Fatalf("%s speed %v exceeds NormalSpeedMax %v", sp.ID, sp.Behavior.Speed, contract.NormalSpeedMax)
		}
		// max elder length: base x clamped size x 1.04 elder growth
		elder := 64 * contract.Clamp(sp.Size, spSizeMin, contract.NormalSizeMax) * 1.04
		if elder >= elderChosenLen {
			t.Fatalf("%s elder length %.2f not below the Chosen's %.2f", sp.ID, elder, elderChosenLen)
		}
	}
	if chosen != 1 {
		t.Fatalf("seed must contain exactly one chosen species, got %d", chosen)
	}
}

// TestNewSeedSpeciesValidateAndGate: the three N9 seeds survive strict
// validation, wear no lilac, and pass the species gate on a fresh registry.
func TestNewSeedSpeciesValidateAndGate(t *testing.T) {
	s := newTestStore(t)
	if err := s.EnsureSeed(); err != nil {
		t.Fatalf("EnsureSeed: %v", err)
	}
	fresh, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("fresh Load: %v", err)
	}
	for _, id := range []string{"ribbon-streamer", "puff-orbit", "dart-spindle"} {
		sp := s.SpeciesByID(id)
		if sp == nil {
			t.Fatalf("seed species %s missing", id)
		}
		if err := s.ValidateSpecies(sp); err != nil {
			t.Fatalf("%s invalid: %v", id, err)
		}
		if lilacReserved(sp.Palette) {
			t.Fatalf("%s wears reserved lilac hues %v", id, sp.Palette)
		}
		if ok, conflicts := fresh.GateSpecies(sp); !ok {
			t.Fatalf("%s rejected by gate on fresh store: %v", id, conflicts)
		}
	}
}

// TestClampRoleAware: normal species and non-core "chosen" get the supremacy
// caps; only the core-seeded Chosen keeps the full range on load and write.
func TestClampRoleAware(t *testing.T) {
	pal := contract.Palette{Body: "#101010", Belly: "#202020", Accent: "#303030", Glow: "#404040"}
	pat := contract.Pattern{Type: "stripe", Density: 0.5, Size: 0.5}

	s := newTestStore(t)
	if err := s.EnsureSeed(); err != nil {
		t.Fatalf("EnsureSeed: %v", err)
	}
	normal := &contract.Species{ID: "giant-normal", Name: "Giant Normal", Size: 9, Fin: 9, Tail: 9,
		Palette: pal, Pattern: pat, Source: "user"}
	if err := s.WriteSpecies(normal); err != nil {
		t.Fatal(err)
	}
	if got := s.SpeciesByID("giant-normal"); got.Size != contract.NormalSizeMax ||
		got.Fin != contract.NormalFinMax || got.Tail != contract.NormalTailMax {
		t.Fatalf("normal not capped: %+v", got)
	}

	fakeChosen := &contract.Species{ID: "fake-chosen", Name: "Fake Chosen", Role: contract.RoleChosen,
		Size: 9, Fin: 9, Tail: 9, Palette: pal, Pattern: pat, Source: "species-agent"}
	if err := s.WriteSpecies(fakeChosen); err != nil {
		t.Fatal(err)
	}
	if got := s.SpeciesByID("fake-chosen"); got.Size != contract.NormalSizeMax {
		t.Fatalf("non-core chosen must be capped, got size %v", got.Size)
	}

	// The core Chosen round-trips through WriteSpecies with her full range.
	core := s.SpeciesByID("chosen-lilastar")
	if core == nil {
		t.Fatal("chosen-lilastar missing after seed")
	}
	clone := *core
	clone.Name = "Chosen Relabeled" // unique name so the write is not a name clash
	if err := s.WriteSpecies(&clone); err != nil {
		t.Fatal(err)
	}
	if got := s.SpeciesByID("chosen-lilastar"); got.Size != spSizeMax || got.Fin != spFinMax {
		t.Fatalf("core chosen must keep full range on rewrite: %+v", got)
	}
}
