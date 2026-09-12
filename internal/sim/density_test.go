// v0.3.1: the screen IS the tank — density acceptance. A big canvas is a
// big aquarium: more fish, more plants, more critters; a small canvas keeps
// the classic budget. Fish always still outnumber plants (F14 holds at any
// size).
package sim

import (
	"testing"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

func mkSpecies(n int) []*contract.Species {
	out := make([]*contract.Species, 0, n)
	for i := 0; i < n; i++ {
		sp := testSpecies(float64(i % 2))
		sp.ID = sp.ID + "-d" + string(rune('0'+i))
		out = append(out, sp)
	}
	return out
}

func TestBigTankIsABiggerAquarium(t *testing.T) {
	sp := mkSpecies(4)
	cfg := contract.Config{MaxFish: 24, DaySeconds: 60}
	big := NewWorld(3440, 1440, cfg, sp, nil, nil)
	small := NewWorld(1280, 720, cfg, sp, nil, nil)
	if big.density < 3 || big.density > contract.DensityMax {
		t.Fatalf("21:9 1440p density = %.2f, want ~3.4 (clamped ≤%.0f)", big.density, contract.DensityMax)
	}
	if small.density != 1 {
		t.Fatalf("reference canvas density = %.2f, want 1", small.density)
	}
	if big.popCap <= small.popCap {
		t.Fatalf("popCap must grow with the tank: big=%d small=%d", big.popCap, small.popCap)
	}
	// v1: the initial school fills to the slider value in both tanks (test
	// worlds carry no embedded Chosen → MaxFish-1) — the area scale shows in
	// popCap (the breeding ceiling), not the boot count
	if len(big.fishes) != 23 || len(small.fishes) != 23 {
		t.Fatalf("initial school must fill to MaxFish-1: big=%d small=%d, want 23", len(big.fishes), len(small.fishes))
	}
	if big.popCap > contract.PopCapMax {
		t.Fatalf("popCap %d exceeds the absolute ceiling %d", big.popCap, contract.PopCapMax)
	}
}

func TestPlantCapScalesButFishStillOutnumber(t *testing.T) {
	sp := mkSpecies(4)
	cfg := contract.Config{MaxFish: 24, DaySeconds: 60}
	big := NewWorld(3440, 1440, cfg, sp, nil, nil)
	for i := 0; i < 40; i++ {
		big.AddPlant(&contract.PlantDesign{ID: "p" + string(rune('a'+i%26)), Fronds: 5, Height: 0.3, Sway: 0.5, Colors: []string{"#123456", "#654321"}})
	}
	if got := len(big.Plants()); got > contract.PlantCapMax || got > len(big.aliveFishes())-1 {
		t.Fatalf("plant budget broke: plants=%d fish=%d ceiling=%d", got, len(big.aliveFishes()), contract.PlantCapMax)
	}
}

func TestRestoreRescalesOldPositions(t *testing.T) {
	sp := mkSpecies(1)
	cfg := contract.Config{MaxFish: 24, DaySeconds: 60}
	w := NewWorld(3440, 1440, cfg, sp, nil, nil)
	s := contract.Save{SchemaVersion: 2, Fish: []contract.SavedFish{{
		SpeciesID: sp[0].ID, Seed: 7, AgeDays: 6, Pos: contract.Vec2{X: 640, Y: 360},
	}}}
	if err := w.Restore(s); err != nil {
		t.Fatal(err)
	}
	f := w.fishes[0]
	if f.Pos.X < 1700 || f.Pos.Y < 700 { // 640,360 × ~2.68 must land mid-tank
		t.Fatalf("restored position not rescaled into the big tank: (%.0f, %.0f)", f.Pos.X, f.Pos.Y)
	}
}

// F29: up to 100 — the popCap honors the configured MaxFish
// at the reference canvas, and the absolute ceiling is the frozen contract.
func TestPopCapHonorsUserMax(t *testing.T) {
	sp := mkSpecies(4)
	w := NewWorld(1280, 720, contract.Config{MaxFish: 100, DaySeconds: 60}, sp, nil, nil)
	if w.popCap != 100 {
		t.Fatalf("MaxFish=100 at 720p: popCap = %d, want 100 (no silent clamp)", w.popCap)
	}
	big := NewWorld(3440, 1440, contract.Config{MaxFish: 100, DaySeconds: 60}, sp, nil, nil)
	if big.popCap != contract.PopCapMax {
		t.Fatalf("MaxFish=100 at 3440x1440: popCap = %d, want the ceiling %d", big.popCap, contract.PopCapMax)
	}
	// the hatch gate lets the population GROW past the old 30: drop 40
	// hatch-ready eggs (the care economy would breed these over time) and
	// confirm the cap admits well beyond 30 alive fish
	for i := 0; i < 40 && len(w.fishes) < 40; i++ {
		w.SpawnEgg(w.fishes[i%len(w.fishes)].Sp.ID)
	}
	for i := 0; i < 60*8; i++ { // 8 s: eggs hatch in 6 s
		w.Update(1/60.0, Input{})
	}
	if len(w.fishes) <= 30 {
		t.Fatalf("population capped at the old limit: %d alive", len(w.fishes))
	}
}
