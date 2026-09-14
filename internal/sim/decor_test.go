// v1.1: mite spawn laws — teeming cadence, her circle untouched.
package sim

import (
	"testing"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// Owner's rule: the mites teem (spawn cadence tripled) and never appear
// inside the lilac one's circle — her home stays undisturbed.
func TestMitesTeemButNotAtHerDoor(t *testing.T) {
	w := testWorld(t, testSpecies(0), 3)
	nest := v2(w.W*0.6, w.H*0.8)
	w.SetZones([]contract.Zone{{Center: nest, Radius: contract.ZoneRadius, Owner: "chosen"}})
	w.miteT = 0.01
	w.mites = w.mites[:0]
	spawns, inCircle := 0, 0
	const dt = 0.05
	for i := 0; i < int(180/dt); i++ {
		before := len(w.mites)
		w.tickMites(dt)
		if len(w.mites) > before {
			spawns++
			m := w.mites[len(w.mites)-1]
			// the spawn keeps a +70 px buffer, the mite's own wiggle eats a
			// few px of it — the law is HER CIRCLE itself
			if hyp2(sub(m.Pos, nest)) < contract.ZoneRadius {
				inCircle++
			}
		}
	}
	if spawns < 8 {
		t.Fatalf("only %d mite spawns in 180 s — the tank must teem (tripled cadence)", spawns)
	}
	if inCircle > 0 {
		t.Fatalf("%d mites appeared inside her circle — her home stays undisturbed", inCircle)
	}
}

// G89: mites are RELEASED at the surface — the same lane the auto-feeder
// drops flakes into — and settle from there; none are conjured mid-water.
func TestMitesDropFromTheSurfaceLikeFeed(t *testing.T) {
	w := testWorld(t, testSpecies(0), 3)
	w.miteT = 0.01
	w.mites = w.mites[:0]
	const dt = 0.05
	spawns, atSurface, midWater := 0, 0, 0
	var deepest float64
	for i := 0; i < int(60/dt); i++ {
		before := len(w.mites)
		w.tickMites(dt)
		if len(w.mites) > before {
			spawns++
			m := w.mites[len(w.mites)-1]
			if absF(m.Pos.Y-contract.MiteDropY) < 5 { // the wiggle already sways the fresh release
				atSurface++
			}
		}
		for _, m := range w.mites {
			deepest = maxF(deepest, m.Pos.Y)
		}
	}
	if spawns == 0 {
		t.Fatal("no mite released in 60 s")
	}
	if atSurface != spawns {
		t.Fatalf("%d of %d releases were not at the surface line — mites drop like feed", atSurface, spawns)
	}
	_ = midWater
	// the drift settles: mites sink away from the release line and never
	// carpet the floor band
	if deepest <= contract.MiteDropY+8 {
		t.Fatalf("mites never sank (deepest %.0f px) — the drop must settle", deepest)
	}
	if deepest > w.H*contract.MiteSinkMaxFrac+14 {
		t.Fatalf("a mite reached %.0f px — the drift stops at %.0f px", deepest, w.H*contract.MiteSinkMaxFrac)
	}
}
