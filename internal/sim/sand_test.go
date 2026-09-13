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
