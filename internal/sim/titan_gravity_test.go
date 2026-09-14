// v1.1 G82/G83: the titan gravity pass — the elders swim like the alien
// flora grows. Ponderous level gliding (the vertical drift that matched
// the whole cruise is gone), strikes reserved for living food alone (the
// random no-food dart and the dead-flake dive are dead), the formation
// slot is the one cruise reflex — and the first strike shocks the school:
// the small fish part before it and settle back to normal.
package sim

import (
	"math"
	"testing"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// G82: no strike without living food. A hungry giant next to a dead flake
// holds the sweep; the same giant next to a wriggling treat commits.
func TestTitansStrikeOnlyLivingFood(t *testing.T) {
	w := titanWorld(t, 3)
	w.spawnPod()
	g := w.titanGiant()
	g.Satiety = 0.1
	g.lungeCD = 0
	cruise := g.maxSpeed(1)
	w.foods = append(w.foods, Food{Pos: add(g.Pos, v2(150, 0)), Age: 1})
	const dt = 1 / 60.0
	for i := 0; i < 60*60; i++ {
		g.Satiety = 0.1
		g.advance(dt, 0, w)
		if g.lungeT > 0 {
			t.Fatal("the giant struck at a dead flake — strikes are for living food")
		}
		if g.turning <= 0 && g.lungeT <= 0 { // the convoy arc owns its own pace
			if s := hyp2(g.Vel); s > 2*cruise {
				t.Fatalf("flake-side speed %.1f px/s — a dead flake deserves no burst (cruise %.1f)", s, cruise)
			}
		}
	}
	// the same hunger, a live treat: the strike comes
	w2 := titanWorld(t, 3)
	w2.spawnPod()
	g2 := w2.titanGiant()
	g2.Satiety = 0.1
	g2.lungeCD = 0
	w2.treats = append(w2.treats, &Treat{Kind: contract.TreatWorm, Pos: add(g2.Pos, v2(150, 0)), BitesLeft: 3})
	lunged := false
	for i := 0; i < 60*15 && !lunged; i++ {
		g2.Satiety = 0.1
		g2.advance(dt, 0, w2)
		lunged = g2.lungeT > 0
	}
	if !lunged {
		t.Fatal("a starving giant ignored a live treat — the reflex is dead")
	}
}

// G82: ponderous level gliding — with nothing armed (fed, calm) the
// vertical drift stays a fraction of the cruise and the nose never jumps.
func TestTitanGlidesLevelAndSteady(t *testing.T) {
	w := titanWorld(t, 4)
	w.spawnPod()
	const dt = 1 / 60.0
	maxVy, maxJump := 0.0, 0.0
	prev := map[*Fish]float64{}
	grace := map[*Fish]float64{}
	for i := 0; i < 60*120; i++ {
		w.Update(dt, Input{})
		for _, f := range w.fishes {
			if f.Sp.Role != contract.RoleTitan || f.Dying {
				continue
			}
			f.Satiety = 1 // fed: strikes are a separate, welcome behavior
			armed := f.seekBonus > 1.01 || f.turning > 0 || f.scareT > 0 || f.lungeT > 0
			if armed {
				grace[f] = 1.2 // bursts and arcs leave tails — judge settled water
			}
			grace[f] = maxF(0, grace[f]-dt)
			if !armed && grace[f] <= 0 {
				if vy := math.Abs(f.Vel.Y); vy > maxVy {
					maxVy = vy
				}
				if p, ok := prev[f]; ok {
					if j := math.Abs(f.headingA - p); j > maxJump {
						maxJump = j
					}
				}
			}
			prev[f] = f.headingA
		}
	}
	if maxVy > 12 {
		t.Fatalf("cruise |vy| %.1f px/s — a giant bobs, it does not glide (want ≤ 12)", maxVy)
	}
	if maxJump > 0.35 {
		t.Fatalf("a single-frame heading jump of %.2f rad at cruise", maxJump)
	}
}

// G83: the first strike shocks the school — small fish near the strike
// part away at once and settle back within seconds; the pressure ring is
// born with the dive and gone inside a second.
func TestTheStrikeShocksTheSchool(t *testing.T) {
	w := titanWorld(t, 6)
	w.spawnPod()
	g := w.titanGiant()
	// cluster the small fish around the strike point
	pt := add(g.Pos, v2(150, 0))
	k := 0
	for _, f := range w.fishes {
		if f.Sp.Role != contract.RoleNormal || k >= 5 {
			continue
		}
		f.Pos = add(pt, v2(float64(k%3)*60+150, float64(k/3)*60-60))
		f.Vel = v2(0, 0)
		f.Satiety = 1 // the bait is for the giant — the school just watches
		k++
	}
	if k < 3 {
		t.Fatal("not enough small fish staged around the strike")
	}
	g.Satiety = 0.1
	g.lungeCD = 0
	w.treats = append(w.treats, &Treat{Kind: contract.TreatWorm, Pos: pt, BitesLeft: 3})
	const dt = 1 / 60.0
	shocked, ringSeen := 0, false
	var shockedFish []*Fish
	d0 := map[*Fish]float64{}
	for _, f := range w.fishes { // pre-strike distances
		if f.Sp.Role == contract.RoleNormal {
			d0[f] = hyp2(sub(f.Pos, pt))
		}
	}
	for i := 0; i < 60*12; i++ {
		g.Satiety = 0.1
		w.Update(dt, Input{})
		if len(w.Shocks()) > 0 {
			ringSeen = true
		}
		if i == 36 { // 0.6 s after the strike: the water has PARTED
			for _, f := range w.fishes {
				if f.Sp.Role != contract.RoleNormal || f.Dying {
					continue
				}
				if f.scareT > 0 && d0[f] < contract.StrikeScareR {
					if hyp2(sub(f.Pos, pt))-d0[f] > 30 { // genuinely pushed outward
						shocked++
						shockedFish = append(shockedFish, f)
					}
				}
			}
		}
	}
	if !ringSeen {
		t.Fatal("the strike rang no pressure ring")
	}
	if shocked < 3 {
		t.Fatalf("only %d small fish shocked by the strike — the water must part", shocked)
	}
	// ...and the panic passes: eight seconds later everyone is back to cruise
	for i := 0; i < 60*8; i++ {
		w.Update(dt, Input{})
	}
	for _, f := range shockedFish {
		if f.Dying {
			continue
		}
		if s := hyp2(f.Vel); s > 1.3*f.maxSpeed(0) {
			t.Fatalf("a shocked fish never settled: %.1f px/s after 8 s", s)
		}
	}
	// the ring itself dies inside its lifetime
	for i := 0; i < 60*2; i++ {
		w.Update(dt, Input{})
	}
	if len(w.Shocks()) != 0 {
		t.Fatal("a pressure ring outlived its moment")
	}
}
