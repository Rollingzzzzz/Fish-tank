// v1.2 G90: the motion governor — the verifiable controller behind the
// "movement reality" pass. The user's law: a fish can never move faster
// than its own motion allows, and every joint of its body must obey. In
// test terms: a single frame may displace the head by its own travel
// (velocity carried INTO the frame) plus tight reflex slack — never a
// discontinuous hop — and every spine joint by the head's allowance plus a
// small swing budget. Speed ENVELOPES (seekBonus bursts, the G88 glide
// down from a spent ceiling) are already pinned by TestSpeedChangesStay-
// Physical; what this controller pins is CONTINUITY: the ±3 px edge hops
// of the school and the 20–30 px nest-rim teleports of the big bodies all
// violated it. Scripted, by-design states (transit hide, glass attach,
// the Chosen's wormhole, dying drift) own their position and are excluded.
package sim

import (
	"testing"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// governorWorld runs the REAL tank proportions with the full cast —
// school, Chosen, titan pod, hammerhead pair — and the nest circle placed
// where the big traffic crosses.
func governorWorld(t *testing.T) *World {
	t.Helper()
	cfg := contract.Config{MaxFish: 30, DaySeconds: 60}
	w := NewWorld(1720, 720, cfg, []*contract.Species{
		chosenTestSpecies(), sharkTestSpecies(), titanTestSpecies(),
		cappedNormalSpecies("gv-neon", false), cappedNormalSpecies("gv-dart", true),
	}, nil, nil)
	w.SeedRng(23)
	w.Update(1/60.0, Input{}) // residents spawn
	w.SetZones([]contract.Zone{{Owner: "chosen",
		Center: v2(w.W*0.30, w.H*0.78), Radius: contract.ZoneRadius}})
	return w
}

type gvSnap struct {
	pos    contract.Vec2
	spine  []contract.Vec2
	vel    float64
	attach float64
	trans  bool
	hide   float64
	portal int
	dying  bool
}

func gvExcluded(f *Fish, p gvSnap, prev bool) bool {
	if prev {
		return p.dying || p.trans || p.hide > 0 || p.attach > 0 ||
			(f.Sp.Role == contract.RoleChosen && p.portal != 0)
	}
	return f.Dying || f.transiting || f.Hide01 > 0 || f.attachT > 0 ||
		(f.Sp.Role == contract.RoleChosen && f.portalPh != 0)
}

// runGovernorSoak drives a mixed tank through startles at the nest, edge-
// hugging food rushes and forced titan visits, and returns the worst
// single-frame head and joint excesses over the governor's allowance.
func runGovernorSoak(t *testing.T, frames int) (worstHead, worstJoint float64, who string) {
	t.Helper()
	w := governorWorld(t)
	const dt = 1 / 60.0
	nest := w.zones[0].Center
	for i := 0; i < frames; i++ {
		in := Input{}
		switch {
		case i%420 == 300: // a startle bolt rammed into the aura
			in = Input{MouseActive: true, MouseSpeed: 1500,
				MouseX: nest.X + 60, MouseY: nest.Y - 40}
		case i%420 == 100: // a feeding rush toward the middle
			w.foods = append(w.foods, Food{Pos: v2(w.W*0.5, w.H*0.5), Seed: float64(i), Age: 1})
		case i%420 == 250: // a flake hugging the glass — edge clamps must fire
			w.foods = append(w.foods, Food{Pos: v2(w.W - 40, 200), Seed: float64(i), Age: 1})
		}
		if i%1500 == 700 {
			w.DebugForceTitanVisit()
		}
		prev := map[*Fish]gvSnap{}
		for _, f := range w.fishes {
			prev[f] = gvSnap{pos: f.Pos, spine: append([]contract.Vec2(nil), f.Spine...),
				vel: hyp2(f.Vel), attach: f.attachT, trans: f.transiting,
				hide: f.Hide01, portal: f.portalPh, dying: f.Dying}
		}
		w.Update(dt, in)
		for _, f := range w.fishes {
			p, ok := prev[f]
			if !ok || gvExcluded(f, p, true) || gvExcluded(f, p, false) {
				continue
			}
			headAllow := (p.vel+90)*dt + contract.MotionSlackPx
			if j := hyp2(sub(f.Pos, p.pos)); j > worstHead {
				worstHead, who = j, f.Sp.ID
				if j > headAllow {
					t.Logf("head excess: %s frame %d jumped %.2f px (allow %.2f, v=%.0f) at (%.0f,%.0f)",
						f.Sp.ID, i, j, headAllow, p.vel, f.Pos.X, f.Pos.Y)
				}
			}
			if j := hyp2(sub(f.Pos, p.pos)); j > headAllow {
				t.Errorf("G90 head law: %s frame %d moved %.2f px in one frame (allow %.2f, carried v=%.0f)",
					f.Sp.ID, i, j, headAllow, p.vel)
			}
			for k, q := range f.Spine {
				j := hyp2(sub(q, p.spine[k]))
				jointAllow := headAllow + f.segLen*contract.SpineBendSlew*dt + contract.JointSwingPx
				if j > worstJoint {
					worstJoint = j
				}
				if j > jointAllow {
					t.Errorf("G90 joint law: %s frame %d joint %d moved %.2f px (allow %.2f, segLen=%.1f)",
						f.Sp.ID, i, k, j, jointAllow, f.segLen)
					break // one report per fish per frame is enough
				}
			}
		}
	}
	return worstHead, worstJoint, who
}

// TestHeadObeysSpeedLaw: no fish ever jumps — its single-frame displacement
// stays inside the travel its carried velocity implies plus reflex slack.
func TestHeadObeysSpeedLaw(t *testing.T) {
	frames := 60 * 360 // 6 min of tank time, real 1/60 frames
	worstHead, worstJoint, who := runGovernorSoak(t, frames)
	t.Logf("governor soak: worst head step %.2f px (%s), worst joint step %.2f px",
		worstHead, who, worstJoint)
	if worstHead > 0 && t.Failed() {
		t.Fatalf("motion governor violated: worst head step %.2f px by %s", worstHead, who)
	}
}

// TestJointsObeySpeedLaw: the body follows — every spine joint stays inside
// the head's allowance plus its swing budget, so a yanked tail can never
// read as a teleport even while the head glides.
func TestJointsObeySpeedLaw(t *testing.T) {
	frames := 60 * 360
	_, worstJoint, _ := runGovernorSoak(t, frames)
	if t.Failed() {
		t.Fatalf("motion governor violated: worst joint step %.2f px", worstJoint)
	}
}
