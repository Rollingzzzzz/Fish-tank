// v1.1: sand bed tests (G54) — fish skimming the floor stir the grains
// (they scatter sideways and lift), the bed re-levels itself naturally
// afterwards, and the field stays a bounded, bounded-cost visual.
package sim

import (
	"math"
	"testing"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

func TestSandStirsAndSettles(t *testing.T) {
	w := titanWorld(t, 3)
	for _, f := range w.fishes {
		if f.Sp.Role == contract.RoleNormal {
			// skimming fast just above the sand line
			f.Pos = v2(w.W*0.4, w.H*contract.FloorLineFrac-20)
			f.Vel = v2(120, 0)
		}
	}
	w.Update(0.05, Input{})
	disturbed := 0.0
	for _, h := range w.Sand() {
		disturbed += math.Abs(h)
	}
	if disturbed == 0 {
		t.Fatal("a fast fish skimming the floor left the bed untouched")
	}
	// the bed re-levels: move the stirrer away from the floor and let pure
	// relaxation run — half a minute later the bed is calm again
	for _, f := range w.fishes {
		if f.Sp.Role == contract.RoleNormal {
			f.Pos = v2(w.W*0.4, w.H*0.3)
			f.Vel = v2(0, 0)
		}
	}
	for i := 0; i < 60*45; i++ {
		w.tickSand(1 / 60.0)
	}
	worst := 0.0
	for _, h := range w.Sand() {
		if math.Abs(h) > worst {
			worst = math.Abs(h)
		}
	}
	if worst > 0.05 {
		t.Fatalf("the bed never settled: worst offset %.2f", worst)
	}
}

// the field is bounded no matter what — no runaway piles.
func TestSandStaysBounded(t *testing.T) {
	w := titanWorld(t, 2)
	for i := 0; i < 60*60; i++ {
		f := w.fishes[i%len(w.fishes)]
		f.Pos = v2(w.W*0.5, w.H*contract.FloorLineFrac-10)
		f.Vel = v2(200, 0)
		w.Update(1/60.0, Input{})
	}
	for _, h := range w.Sand() {
		if h < -4.01 || h > 10.01 {
			t.Fatalf("sand offset %.2f outside the bounded band", h)
		}
	}
}

// G68: flakes come to rest ON the dune surface — never buried under the
// bed, and once landed they hold their spot (the school dives for them
// where they lie). The relief is real terrain, not a ruler line.
func TestFlakesRestOnSand(t *testing.T) {
	w := titanWorld(t, 0)
	w.foods = w.foods[:0]
	drop := func(x float64) contract.Vec2 {
		return v2(x, contract.SandSurfaceY(w.H, x)-60)
	}
	w.foods = append(w.foods,
		Food{Pos: drop(300), Seed: 1},
		Food{Pos: drop(900), Seed: 2},
		Food{Pos: drop(1500), Seed: 3})
	const dt = 0.05
	resting := 0
	for i := 0; i < 60*6; i++ { // 18 s of sinking (the flake TTL is 25)
		w.tickFood(dt)
		for k := range w.foods {
			fd := &w.foods[k]
			surf := contract.SandSurfaceY(w.H, fd.Pos.X)
			if fd.Pos.Y > surf {
				t.Fatalf("flake %d buried: y=%.1f below the dune surface %.1f", k, fd.Pos.Y, surf)
			}
			if hyp2(fd.Vel) < 0.01 && fd.Pos.Y > surf-3 {
				resting++
			}
		}
	}
	if resting == 0 {
		t.Fatal("no flake ever came to rest on the sand")
	}
	// at rest they hold: no drift, no further sink
	x0, y0 := w.foods[0].Pos.X, w.foods[0].Pos.Y
	for i := 0; i < 60; i++ {
		w.tickFood(dt)
	}
	if d := hyp2(sub(w.foods[0].Pos, v2(x0, y0))); d > 0.5 {
		t.Fatalf("a resting flake drifted %.2f px — it must sit still", d)
	}
}

// G68: the relief curve is terrain — swells and a channel, never flat.
func TestSandReliefIsNotFlat(t *testing.T) {
	lo, hi := 1e18, -1e18
	for x := 0.0; x <= 1720; x += 8 {
		r := contract.SandRelief(x)
		lo, hi = min(lo, r), max(hi, r)
	}
	if hi-lo < 10 {
		t.Fatalf("relief spans only %.1f px — reads as a ruler line", hi-lo)
	}
}
