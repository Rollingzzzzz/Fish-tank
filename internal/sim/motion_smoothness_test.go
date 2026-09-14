// v1.1 G80/G81: motion smoothness laws — the analysis round measured the
// tank frame-by-frame and found (1) near-stationary fish rotating their
// noses at up to 5 rad/s for seconds (the "eyes stay put while the body
// turns every direction" read) and (2) single-frame speed changes of
// 100+ px/s from the instant cap cut, the glass velocity kill and the
// blind-side shave (the "collision detection" read). The fixes: turn
// authority rides fin flow and curvature (a U-turn/startle still flexes),
// the spent-burst allowance decays, the glass component eases off, the
// shave bleeds. These tests pin the envelopes.
package sim

import (
	"math"
	"testing"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

func smoothWorld(t *testing.T) *World {
	t.Helper()
	cfg := contract.Config{MaxFish: 30, DaySeconds: 60}
	w := NewWorld(1720, 720, cfg, []*contract.Species{
		chosenTestSpecies(), sharkTestSpecies(), titanTestSpecies(),
		cappedNormalSpecies("ms-neon", false), cappedNormalSpecies("ms-dart", false),
	}, nil, nil)
	w.SeedRng(5)
	w.spawnPod()
	return w
}

// TestNoStationaryNoseWhip: a fish slower than half its body length per
// second may reorient and may even circle a flake coherently — what it
// may never do is FLAIL: a long heading path built from frequent turn
// reversals (the "eyes fixed while the body turns every direction" read).
// Whip = path > 7.5 rad with > 8 reversals inside one rolling 2 s window,
// measured per fish.
func TestNoStationaryNoseWhip(t *testing.T) {
	w := smoothWorld(t)
	const dt = 1 / 60.0
	const win = 120 // 2 s
	type sample struct {
		sp, hd, bl float64
	}
	rings := map[*Fish][]sample{}
	for i := 0; i < 60*150; i++ {
		if i%(20*60) == 100 {
			w.foods = append(w.foods, Food{Pos: v2(w.W*0.35, 240), Seed: 1, Age: 1},
				Food{Pos: v2(w.W*0.65, 460), Seed: 2, Age: 1})
		}
		w.Update(dt, Input{})
		for _, f := range w.fishes {
			if f.Dying || f.transiting || (f.Sp.Role == contract.RoleChosen && f.portalPh != 0) {
				continue
			}
			if f.Sp.Role == contract.RoleTitan || f.Sp.Role == contract.RoleShark {
				continue // the pod's rotation is arc + zone machinery and the hunter rides its own body-axis pin — each owned by its own tests
			}
			s := sample{hyp2(f.Vel), f.headingA, f.bodyLen}
			r := append(rings[f], s)
			if len(r) > win {
				r = r[1:]
			}
			rings[f] = r
			if len(r) < win {
				continue
			}
			slow := true
			for _, e := range r {
				if e.sp >= e.bl*0.5 || e.bl < 1 {
					slow = false
					break
				}
			}
			if !slow {
				continue
			}
			path, rev := 0.0, 0
			prevSign := 0.0
			for j := 1; j < len(r); j++ {
				d := r[j].hd - r[j-1].hd
				path += math.Abs(d)
				if math.Abs(d) > 0.02 {
					sign := math.Copysign(1, d)
					if prevSign != 0 && sign != prevSign {
						rev++
					}
					prevSign = sign
				}
			}
			if path > 7.5 && rev > 8 {
				t.Fatalf("stationary whip: %s flailed %.1f rad with %d reversals in 2 s under half body-length speed",
					f.Sp.Role, path, rev)
			}
		}
	}
}

// TestSpeedChangesStayPhysical: the per-frame speed change stays inside
// the thrust envelope — hard ceiling for every fish in every state, and a
// cruise-tight budget when nothing is armed (no scare, no strike, no arc,
// no startle residue). The pre-fix tank changed speed by 122 px/s in a
// single frame with nothing armed.
func TestSpeedChangesStayPhysical(t *testing.T) {
	w := smoothWorld(t)
	const dt = 1 / 60.0
	type snapS struct {
		sp, flee          float64
		scare, seek, turn float64
		lunge, attach     float64
		dying, trans      bool
		role              string
	}
	for i := 0; i < 60*150; i++ {
		if i%(20*60) == 100 {
			w.foods = append(w.foods, Food{Pos: v2(w.W*0.4, 300), Seed: 1, Age: 1})
		}
		before := map[*Fish]snapS{}
		for _, f := range w.fishes {
			before[f] = snapS{hyp2(f.Vel), hyp2(f.fleeImp), f.scareT, f.seekBonus,
				f.turning, f.lungeT, f.attachT, f.Dying, f.transiting, f.Sp.Role}
		}
		w.Update(dt, Input{})
		for _, f := range w.fishes {
			b, ok := before[f]
			if !ok {
				continue // a fish born this frame — no before-state to judge
			}
			if b.dying || b.trans || b.attach > 0 || f.attachT > 0 ||
				(f.Sp.Role == contract.RoleChosen && f.portalPh != 0) {
				continue // scripted states own their velocity
			}
			// a fresh startle (the strike shock's flinch kick) is a reflex —
			// the envelopes judge open-water mechanics, not the C-start
			startled := f.scareT > b.scare+0.5
			if startled {
				continue
			}
			dSp := math.Abs(hyp2(f.Vel)-b.sp) / dt
			// wall contact is physical: the inbound component eases off at
			// the glass by design (10/s of the component) — both budgets
			// judge open water
			nearWall := f.Pos.X < 48 || f.Pos.X > w.W-48 || f.Pos.Y < 48 || f.Pos.Y > w.H-48
			if !nearWall && dSp > 1600 {
				t.Fatalf("%s changed speed by %.0f px/s in one frame (armed: scare %.1f seek %.2f)",
					b.role, dSp, f.scareT, f.seekBonus)
			}
			armedNow := b.scare > 0 || b.seek > 1.01 || b.turn > 0 || b.flee >= 30 ||
				f.scareT > 0 || f.seekBonus > 1.01 || f.turning > 0 || b.lunge > 0 || f.lungeT > 0
			if !armedNow && !nearWall {
				if dSp > 380 {
					t.Fatalf("%s changed speed by %.0f px/s in one frame with NOTHING armed — cruise must stay smooth",
						b.role, dSp)
				}
			}
		}
	}
}
