// v1.1 G63 time-lapse: the user's acceptance for the whole tank — let time
// run over a mixed population (school, Chosen, titan pod, hammerhead pair)
// and watch: every eye must LEAD its motion (no fish ever glides tail-
// first), and every spine stays a rope arcing around the swim axis, never
// a coil that whirls its bone vectors like clock hands.
package sim

import (
	"math"
	"testing"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

func TestTimeLapseEveryEyeLeadsItsMotion(t *testing.T) {
	cfg := contract.Config{MaxFish: 16, DaySeconds: 120}
	w := NewWorld(800, 600, cfg, []*contract.Species{
		cappedNormalSpecies("tl-neon-a", false),
		cappedNormalSpecies("tl-neon-b", false),
		titanTestSpecies(), sharkTestSpecies(), chosenTestSpecies(),
	}, nil, nil)
	w.SeedRng(21)
	w.Update(0.05, Input{}) // residents spawn (pod, pair, Chosen)

	const dt = 0.05
	const window = 10 // frames per motion window (0.5 s)
	snap := map[*Fish]contract.Vec2{}
	face := map[*Fish]contract.Vec2{}
	worstBack := 1e18            // worst windowed net step along the face axis
	worstBend := 0.0             // worst angle between neighboring spine segments
	eyes := 0                    // windows evaluated (must be plentiful)
	for i := 0; i < 60*90; i++ { // 90 s of tank time
		w.Update(dt, Input{})
		if i < 60 {
			continue // spawn warm-up: chains settle onto their headings
		}
		if i%window == 0 {
			snap = map[*Fish]contract.Vec2{}
			face = map[*Fish]contract.Vec2{}
			for _, f := range w.fishes {
				if f.Dying || f.attachT > 0 || f.transiting || f.Hide01 > 0 {
					continue
				}
				snap[f] = f.Pos
				a := sub(f.Spine[0], f.Spine[1])
				if l := hyp2(a); l > 1e-3 {
					face[f] = mulS(a, 1/l)
				}
			}
			continue
		}
		// spine-rope check every frame: neighbors may bend, never hairpin
		for _, f := range w.fishes {
			if f.Dying || len(f.Spine) < 3 {
				continue
			}
			for j := 1; j < len(f.Spine)-1; j++ {
				a := sub(f.Spine[j], f.Spine[j-1])
				b := sub(f.Spine[j+1], f.Spine[j])
				la, lb := hyp2(a), hyp2(b)
				if la < 1e-6 || lb < 1e-6 {
					continue
				}
				if ang := math.Acos(clampF(dot2(a, b)/(la*lb), -1, 1)); ang > worstBend {
					worstBend = ang
				}
			}
			// the swim law every frame: the velocity never opposes the eye
			// (zone projections shove POSITION, not the swim — they are not
			// tail-first gliding and are judged by the loose window below)
			a := sub(f.Spine[0], f.Spine[1])
			la := hyp2(a)
			if v := hyp2(f.Vel); !f.Dying && v > 6 && la > 1e-3 {
				if fwd := dot2(f.Vel, a) / (v * la); fwd < -0.25 {
					t.Fatalf("t=%.1fs: a %s SWIMS %.0f%% backward of its eye @ pos(%.0f,%.0f) vel(%.1f,%.1f) headingA=%.2f face=(%.2f,%.2f) s1=(%.1f,%.1f) role=%s stage=%s",
						float64(i)*dt, f.Sp.ID, -100*fwd, f.Pos.X, f.Pos.Y,
						f.Vel.X, f.Vel.Y, f.headingA, a.X/la, a.Y/la,
						f.Spine[1].X, f.Spine[1].Y, f.Sp.Role, f.Stage)
				}
			}
		}
		if i%window != window-1 {
			continue
		}
		for _, f := range w.fishes {
			p0, ok := snap[f]
			if !ok || f.Dying || f.attachT > 0 || f.transiting || f.Hide01 > 0 {
				continue
			}
			d := sub(f.Pos, p0)
			hd, ok2 := face[f]
			if !ok2 || hyp2(d) <= 1 || hyp2(hd) < 0.5 {
				continue // idle drift or sub-pixel zone corrections
			}
			eyes++
			if proj := dot2(d, hd); proj < worstBack {
				worstBack = proj
			}
			// coarse smoke only: nest-circle body drains legitimately shove
			// POSITION backward up to 12 px per frame — the swim law above
			// (velocity vs eye, every frame) is the tail-first assertion
			if proj := dot2(d, hd); proj < -40 {
				t.Fatalf("t=%.1fs: a %s netted %.1f px BACKWARD of its eye over a window",
					float64(i)*dt, f.Sp.ID, proj)
			}
		}
	}
	if worstBack < -40 {
		t.Fatalf("worst backward windowed motion %.2f px", worstBack)
	}
	if worstBend > 0.85 {
		t.Fatalf("a spine coiled: neighbor segments bent %.2f rad (hairpin territory)", worstBend)
	}
	if eyes < 5000 {
		t.Fatalf("time-lapse too thin: only %d motion windows evaluated", eyes)
	}
	t.Logf("time-lapse ok: %d windows, worst backward %.2f px, worst spine bend %.2f rad",
		eyes, worstBack, worstBend)
}

// G65: no fish ever TELEPORTS. A shark and a normal fish launched straight
// at the Chosen's circle must skim its rim — per-frame displacement stays
// within natural travel plus the bounded correction — and the nest still
// ends up absolute (head never left resting inside).
func TestNoFishTeleportsAtTheNestRim(t *testing.T) {
	w := sharkWorld(t)
	w.Update(0.05, Input{}) // spawn the pair
	w.SetZones([]contract.Zone{{Owner: "chosen", Center: v2(400, 300), Radius: 80}})
	sh := w.fishes[0]
	for _, f := range w.fishes {
		if f.Sp.Role == contract.RoleShark {
			sh = f
		}
	}
	neon := newFish(cappedNormalSpecies("tl-neon-rim", false), 42, v2(430, 60), 6, w.nextID())
	w.fishes = append(w.fishes, neon)
	// both dive straight at the circle's heart
	sh.Pos = v2(400, 40)
	sh.headingA = 1.5708
	sh.Vel = v2(0, 30)
	neon.Pos = v2(430, 60)
	neon.headingA = 1.5708
	neon.Vel = v2(0, 30)

	const dt = 0.05
	worst := 0.0
	worstFish := ""
	for i := 0; i < 60*8; i++ { // 8 s of rim contact
		prev := map[*Fish]contract.Vec2{}
		for _, f := range w.fishes {
			prev[f] = f.Pos
		}
		w.Update(dt, Input{})
		for _, f := range w.fishes {
			if f.Dying {
				continue
			}
			// speed-aware bound: bursts scale with the fish's own pace, so
			// the Chosen's dart triples hers — but the old 73 px rim snap
			// exceeded every budget and stays caught
			if j := hyp2(sub(f.Pos, prev[f])); 3*f.maxSpeed(0)*dt+8 < j && j > worst {
				worst, worstFish = j, f.Sp.ID
			}
		}
		hd := hyp2(sub(sh.Pos, v2(400, 300)))
		if hd < 78 { // the rim projection must keep her head on/above the ring
			t.Fatalf("frame %d: shark head rode %.0f px into the circle", i, 80-hd)
		}
	}
	if worst > 0 {
		t.Fatalf("teleport: %s jumped %.1f px beyond its own speed budget in a single frame", worstFish, worst)
	}
}
