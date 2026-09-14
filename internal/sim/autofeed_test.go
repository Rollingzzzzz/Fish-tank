// v0.3: auto-feed population-stability acceptance. With the maintenance feeder
// on, the school
// must neither starve away nor explode: alive count never drops below the
// starting population, and flakes never carpet the floor.
package sim

import (
	"testing"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

func TestAutoFeedPreservesPopulation(t *testing.T) {
	sp := testSpecies(0)
	cfg := contract.Config{MaxFish: 30, DaySeconds: 60, AutoFeed: true}
	w := NewWorld(800, 600, cfg, []*contract.Species{sp}, nil, nil)
	w.SeedRng(4242)
	w.fishes = w.fishes[:0]
	for i := 0; i < 8; i++ {
		p := v2(150+w.rng.Float64()*500, 150+w.rng.Float64()*300)
		w.fishes = append(w.fishes, newFish(sp, w.rng.Int63(), p, 6, w.nextID()))
	}
	// 5 virtual days (DaySeconds=60 → 300 sim seconds), no player care:
	// maintenance feeding alone must hold the school together.
	start := len(w.aliveFishes())
	minAlive, maxFoods := start, 0
	for i := 0; i < int(5*60*30); i++ {
		w.Update(1/30.0, Input{})
		if a := len(w.aliveFishes()); a < minAlive {
			minAlive = a
		}
		if len(w.foods) > maxFoods {
			maxFoods = len(w.foods)
		}
	}
	if minAlive < start {
		t.Fatalf("auto-feed let the school shrink: %d -> %d alive", start, minAlive)
	}
	if maxFoods > 10 {
		t.Fatalf("maintenance feeder carpeted the floor: %d flakes at once", maxFoods)
	}
}
