// v0.3.2: floor composition acceptance — the user's split: ~20% plants,
// ~40% rocks (enterable), ~20% corals+things, ~20% open water for the fish.
package sim

import (
	"testing"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

func compositionWorld(t *testing.T, w, h float64) *World {
	t.Helper()
	sp := mkSpecies(4)
	sp = append(sp, &contract.Species{ID: "chosen-lilastar", Name: "Lilastar",
		Role: contract.RoleChosen, Size: 1.4, Width: 1.1, Fin: 1.5, Tail: 1.4,
		Palette:  contract.Palette{Body: "#7a3fd6", Belly: "#2a0f4d", Accent: "#c77dff", Glow: "#c77dff"},
		Pattern:  contract.Pattern{Type: "vein", Density: 0.6, Size: 0.5},
		Behavior: contract.Behavior{Speed: 1, Schooling: 0.1, Curiosity: 0.7},
		Source:   "core"})
	cfg := contract.Config{MaxFish: 24, DaySeconds: 60}
	world := NewWorld(w, h, cfg, sp, plantDesigns(40), nil)
	world.SetRockBase(w * contract.FloorShareRocks)
	world.SetCorals(coralDesigns(24))
	return world
}

func TestFloorCompositionSplit(t *testing.T) {
	for _, sz := range [][2]float64{{1280, 720}, {1720, 720}, {3440, 1440}} {
		w := compositionWorld(t, sz[0], sz[1])
		pl, rk, co, op := w.FloorShares()
		if !near(pl, contract.FloorSharePlants, 0.07) {
			t.Errorf("%.0fx%.0f: plants share %.2f, want ~%.2f", sz[0], sz[1], pl, contract.FloorSharePlants)
		}
		if !near(rk, contract.FloorShareRocks, 0.02) {
			t.Errorf("%.0fx%.0f: rocks share %.2f, want ~%.2f", sz[0], sz[1], rk, contract.FloorShareRocks)
		}
		if !near(co, contract.FloorShareCorals, 0.07) {
			t.Errorf("%.0fx%.0f: corals share %.2f, want ~%.2f", sz[0], sz[1], co, contract.FloorShareCorals)
		}
		if !near(op, 0.20, 0.10) {
			t.Errorf("%.0fx%.0f: open share %.2f, want ~0.20", sz[0], sz[1], op)
		}
	}
}

func near(v, want, tol float64) bool { return v >= want-tol && v <= want+tol }

func plantDesigns(n int) []*contract.PlantDesign {
	out := make([]*contract.PlantDesign, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, &contract.PlantDesign{
			ID: "pl" + string(rune('a'+i)), Fronds: 5, Height: 0.3, Width: 1,
			Sway: 0.5, Colors: []string{"#123456", "#654321"},
		})
	}
	return out
}

func coralDesigns(n int) []*contract.CoralDesign {
	out := make([]*contract.CoralDesign, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, &contract.CoralDesign{
			ID: "co" + string(rune('a'+i)), Kind: "fan", Fronds: 5, Height: 0.2,
			Width: 1, Sway: 0.5, Colors: []string{"#123456", "#654321"},
		})
	}
	return out
}

// v0.3.5: the Chosen is one of a kind — SpawnEgg refuses her; her nest
// distance reads ~0 when she is home.
func TestChosenCannotBeSpawned(t *testing.T) {
	w := compositionWorld(t, 1280, 720)
	w.SpawnEgg("chosen-lilastar")
	if len(w.Eggs()) != 0 {
		t.Fatal("SpawnEgg must never create the Chosen")
	}
}

func TestChosenNestDistAtHome(t *testing.T) {
	w := compositionWorld(t, 1280, 720)
	home := v2(w.W*0.5, w.H*0.75)
	w.SetZones([]contract.Zone{{Center: home, Radius: contract.ZoneRadius, Owner: "chosen"}})
	for _, f := range w.Fishes() {
		if f.Sp.Role == contract.RoleChosen {
			f.Pos = home
		}
	}
	if len(w.Fishes()) == 0 {
		t.Fatal("world has no fish")
	}
	if d := w.ChosenNestDist(); d > 5 {
		t.Fatalf("chosen at home should read ~0, got %.1f", d)
	}
}
