// v1.1 realism pass tests (G76/G77): the seep columns breathe from their
// dune anchors even on silent water (VentFloor) and obey the evidence
// kill-switch; the hammerhead pair exhales micro-bubbles at its gills; a
// bite taken off the dune throws its dust and the dust settles; the bubble
// budget never exceeds its cap.
package sim

import (
	"math"
	"testing"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

func TestVentsBreatheFromTheDunes(t *testing.T) {
	w := titanWorld(t, 6)
	// silent water: every bubble observed near a vent anchor is vent-born,
	// so the VentFloor baseline is what the test measures
	w.WaterCur.Bubbles = 0
	w.WaterTgt.Bubbles = 0
	const dt = 0.05
	hits := 0
	for i := 0; i < 60*90; i++ { // 90 s
		w.Update(dt, Input{})
		for _, b := range w.bubbles {
			for _, vf := range ventFrac {
				if math.Abs(b.Pos.X-w.W*vf) < 10 && b.Pos.Y > w.H*0.55 {
					hits++
				}
			}
		}
	}
	if hits < 30 {
		t.Fatalf("seep columns too quiet: %d risen-bubble hits over 90 s of silent water", hits)
	}
}

func TestVentsObeyTheKillSwitch(t *testing.T) {
	w := titanWorld(t, 4)
	w.WaterCur.Bubbles = 0
	w.WaterTgt.Bubbles = 0
	contract.ExtrasEnabled = false
	defer func() { contract.ExtrasEnabled = true }()
	for i := 0; i < 60*30; i++ {
		w.Update(0.05, Input{})
	}
	if len(w.bubbles) != 0 {
		t.Fatalf("%d bubbles with the water silent and extras off", len(w.bubbles))
	}
}

func TestSharksExhaleAtTheGills(t *testing.T) {
	w := sharkWorld(t)
	w.WaterCur.Bubbles = 0
	w.WaterTgt.Bubbles = 0
	const dt = 0.05
	hits := 0
	for i := 0; i < 60*45; i++ { // 45 s
		w.Update(dt, Input{})
		for _, f := range w.fishes {
			if f.Sp.Role != contract.RoleShark || f.Dying {
				continue
			}
			for _, b := range w.bubbles {
				if math.Abs(b.Pos.X-f.Pos.X) < f.bodyLen*0.35 &&
					math.Abs(b.Pos.Y-f.Pos.Y) < f.bodyLen*0.25 {
					hits++
				}
			}
		}
	}
	if hits < 10 {
		t.Fatalf("the pair never breathes: %d gill-proximate bubble frames", hits)
	}
}

func TestFloorFeedingKicksSand(t *testing.T) {
	w := titanWorld(t, 3)
	w.foods = w.foods[:0]
	x := 500.0
	w.foods = append(w.foods, Food{Pos: v2(x, contract.SandSurfaceY(w.H, x)-1), Seed: 9, Age: 1})
	f := w.fishes[0]
	f.Satiety = 0.2
	f.Pos = v2(x+3, contract.SandSurfaceY(w.H, x)-6)
	w.tryEat()
	if len(w.foods) != 0 {
		t.Fatal("the floor bite never landed")
	}
	grains := 0
	for _, p := range w.particles {
		if p.Color == sandKickColor {
			grains++
		}
	}
	if grains < 4 {
		t.Fatalf("a bite off the dune kicked only %d grains", grains)
	}
	// the dust settles under its own weight — the bed is clean again
	for i := 0; i < 60*2; i++ {
		w.tickParticles(1 / 60.0)
	}
	for _, p := range w.particles {
		if p.Color == sandKickColor {
			t.Fatal("a sand grain is still airborne two seconds after the bite")
		}
	}
}

func TestMidWaterBiteKicksNoSand(t *testing.T) {
	w := titanWorld(t, 3)
	w.foods = w.foods[:0]
	w.foods = append(w.foods, Food{Pos: v2(500, 300), Seed: 9, Age: 1})
	f := w.fishes[0]
	f.Satiety = 0.2
	f.Pos = v2(503, 297)
	w.tryEat()
	if len(w.foods) != 0 {
		t.Fatal("the mid-water bite never landed")
	}
	for _, p := range w.particles {
		if p.Color == sandKickColor {
			t.Fatal("a mid-water bite threw sand — the floor is nowhere near")
		}
	}
}

func TestBubbleBudgetHolds(t *testing.T) {
	w := sharkWorld(t)
	w.WaterCur.Bubbles = 1
	w.WaterTgt.Bubbles = 1
	for i := 0; i < 60*120; i++ { // 2 min at full bubble water
		w.Update(1/60.0, Input{})
		if len(w.bubbles) > maxBubbles {
			t.Fatalf("%d bubbles over the cap %d", len(w.bubbles), maxBubbles)
		}
	}
}
