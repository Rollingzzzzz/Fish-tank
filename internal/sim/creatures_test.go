// v1.1: floor critter tests (G44–G46) — the area-scaled cap, the walk band,
// the absolute nest-circle exclusion, the lure steering, the edible-struggle
// pipeline, the burrow exit and the persistence guarantees (never saved;
// old snapshots with the retired Creature field still load cleanly).
package sim

import (
	"strings"
	"testing"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

func critterWorld(t *testing.T, normals int) *World {
	t.Helper()
	w := titanWorld(t, normals)
	w.creatures = w.creatures[:0]
	return w
}

// G44: emergence fills to the area-scaled cap, spread along the sand band,
// and never inside her circle — spawn checks plus the per-tick projection.
func TestCritterCapSpreadAndNestExclusion(t *testing.T) {
	w := critterWorld(t, 6)
	w.zones = []contract.Zone{{Center: v2(w.W*0.5, w.H*0.925), Radius: contract.ZoneRadius, Owner: "chosen"}}
	capN := w.creatureCap()
	for i := 0; i < 2000 && len(w.creatures) < capN; i++ {
		w.creatureT = 0 // spawn due now
		w.tickCreatures(0.05)
	}
	if len(w.creatures) != capN {
		t.Fatalf("critters %d never reached the area cap %d", len(w.creatures), capN)
	}
	for _, c := range w.creatures {
		if c.Kind != "crab" && c.Kind != "shrimp" {
			t.Fatalf("unknown critter kind %q", c.Kind)
		}
		if c.Pos.X < w.W*0.04 || c.Pos.X > w.W*0.96 {
			t.Fatalf("%s at x=%.0f outside the sand span", c.Kind, c.Pos.X)
		}
		if c.Pos.Y < w.H*0.90 || c.Pos.Y > w.H*0.98 {
			t.Fatalf("%s at y=%.0f off the sand band", c.Kind, c.Pos.Y)
		}
		for _, z := range w.zones {
			if z.Owner == "chosen" && hyp2(sub(c.Pos, z.Center)) < z.Radius+4 {
				t.Fatalf("%s at %.0f,%.0f stood inside her circle", c.Kind, c.Pos.X, c.Pos.Y)
			}
		}
	}
}

// G44: the life cycle — walk, then the burrow exit; no corpse is ever left.
func TestCritterBurrowExit(t *testing.T) {
	w := critterWorld(t, 4)
	c := &Creature{Kind: "crab", Pos: v2(w.W*0.2, w.H*0.925), Life: 0.5, Dir: 1}
	w.creatures = append(w.creatures, c)
	w.tickCreatures(0.1)
	if c.Burrow != 0 || len(w.creatures) != 1 {
		t.Fatal("a mid-life critter burrowed early or vanished")
	}
	w.tickCreatures(0.6) // Age passes Life → the burrow starts
	if c.Burrow <= 0 {
		t.Fatal("an aged critter never started to burrow")
	}
	for i := 0; i < 100 && len(w.creatures) > 0; i++ {
		w.tickCreatures(0.1)
	}
	if len(w.creatures) != 0 {
		t.Fatal("a burrowed critter never left the tank")
	}
}

// G45: a struggling critter out-pulls every other interest for hungry fish.
func TestCritterLureSteersHungryFish(t *testing.T) {
	w := critterWorld(t, 1)
	f := w.fishes[0]
	c := &Creature{Kind: "shrimp", Pos: add(f.Pos, v2(150, 0)), Life: 0.5} // vulnerable
	w.creatures = append(w.creatures, c)
	f.Satiety = 0.5
	f.steer(0.016, f.maxSpeed(0), 0, w)
	if f.seekBonus < 2 {
		t.Fatal("a hungry fish ignored the struggling critter")
	}
	// full-bellied fish feel nothing
	f.Satiety = 1
	f.steer(0.016, f.maxSpeed(0), 0, w)
	if f.seekBonus > 1 {
		t.Fatal("a full fish still chased the critter")
	}
}

// G45: the first hungry fish in reach takes the struggling critter — eat,
// care, sparkle, one log line, gone.
func TestCritterIsEatenWhenReached(t *testing.T) {
	w := critterWorld(t, 2)
	f := w.fishes[0]
	f.Pos = v2(w.W*0.5, w.H*0.5) // pinned: independent of the spawn draw
	f.Satiety = 0.5
	c := &Creature{Kind: "crab", Pos: f.Pos, Life: 0.5} // vulnerable
	w.creatures = append(w.creatures, c)
	care := w.Care
	logs := len(w.Log)
	w.tickCreatures(0.02)
	if len(w.creatures) != 0 {
		t.Fatal("a critter inside a hungry fish's reach survived the tick")
	}
	if f.Satiety != 1 || w.Care != care+contract.CareFeedScore {
		t.Fatal("the meal did not register (satiety/care)")
	}
	found := false
	for _, e := range w.Log[logs:] {
		if strings.Contains(e.Text, "takes the struggling crab") {
			found = true
		}
	}
	if !found {
		t.Fatal("the meal never reached the tank log")
	}
	// hard shell: a walking critter in the same spot is NOT edible
	c2 := &Creature{Kind: "crab", Pos: w.fishes[1].Pos, Life: 60, Age: 0}
	w.fishes[1].Satiety = 0.5
	w.creatures = append(w.creatures, c2)
	w.tickCreatures(0.02)
	if len(w.creatures) != 1 {
		t.Fatal("the hard-shell walk phase was edible")
	}
}

// Persistence: critters are never snapshot; a v2 save carrying the retired
// Creatures field still loads, and restores to fresh emergence only.
func TestCrittersNeverPersistAndOldSavesLoad(t *testing.T) {
	w := critterWorld(t, 4)
	w.creatures = append(w.creatures, &Creature{Kind: "crab", Pos: v2(100, 500), Life: 60})
	s := w.Snapshot()
	if len(s.Creatures) != 0 {
		t.Fatal("critters leaked into the snapshot")
	}
	w2 := critterWorld(t, 2)
	old := contract.Save{SchemaVersion: 2, Creatures: []contract.SavedCreature{
		{Kind: "shrimp", Seed: 5, Pos: v2(90, 480), ShellT: 3},
	}}
	if err := w2.Restore(old); err != nil {
		t.Fatalf("old snapshot with creatures: %v", err)
	}
	if len(w2.creatures) != 0 {
		t.Fatal("a restore resurrected critters — emergence must stay fresh")
	}
}
