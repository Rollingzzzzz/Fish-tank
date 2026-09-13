// v1.1: depth lanes + big-body avoidance + crosswise turns (G51) — small
// fish drift between the near and far water so they cross the big bodies in
// front or slip behind them; nobody swims through a big body; and the pod
// bends into wide crosswise turns instead of cruising straight forever.
package sim

import (
	"math"
	"testing"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// the lane stays inside the water band and actually drifts over a minute.
func TestDepthLanesDrift(t *testing.T) {
	w := titanWorld(t, 2)
	w.spawnPod()
	var normal *Fish
	for _, f := range w.fishes {
		if f.Sp.Role == contract.RoleNormal {
			normal = f
			break
		}
	}
	if normal == nil {
		t.Fatal("no normal fish in the world")
	}
	seen := map[int]bool{}
	for i := 0; i < 60*75; i++ {
		w.Update(1/60.0, Input{})
		if normal.z < 0.5-contract.DepthSwing-0.001 || normal.z > 0.5+contract.DepthSwing+0.001 {
			t.Fatalf("lane %.2f outside the swing band", normal.z)
		}
		seen[int(normal.z*10)] = true
	}
	if len(seen) < 3 {
		t.Fatal("the lane never drifted — no depth crossing would happen")
	}
	// the big fish drift the far lane (G59): mostly behind the school
	for _, f := range w.fishes {
		if f.Sp.Role == contract.RoleTitan && (f.z < 0.15 || f.z > 0.45) {
			t.Fatalf("titan lane %.2f outside the far band [0.15,0.45]", f.z)
		}
	}
}

// a small fish next to a big body feels a push AWAY from it, and feels
// nothing once clear of the keep-clear radius.
func TestSmallFishFlowsAroundBigBodies(t *testing.T) {
	w := titanWorld(t, 1)
	w.spawnPod()
	g := w.titanGiant()
	g.Pos = v2(400, 300)
	g.Vel = v2(20, 0)
	var f *Fish
	for _, o := range w.fishes {
		if o.Sp.Role == contract.RoleNormal {
			f = o
			break
		}
	}
	// inside the keep-clear radius: the steer pushes away from the body
	f.Pos = v2(400+g.bodyLen*0.3, 300)
	f.Vel = v2(0, 0)
	acc := f.steer(1/60.0, f.maxSpeed(0), 0, w)
	away := sub(v2(f.Pos.X+acc.X, f.Pos.Y+acc.Y), f.Pos)
	toFish := sub(f.Pos, g.Pos)
	if away.X*toFish.X+away.Y*toFish.Y <= 0 {
		t.Fatal("no outward push near a big body")
	}
	// far away: no such force biases the steer
	f.Pos = v2(400+g.bodyLen*3, 300)
	f.Vel = v2(0, 0)
	acc = f.steer(1/60.0, f.maxSpeed(0), 0, w)
	away = sub(v2(f.Pos.X+acc.X, f.Pos.Y+acc.Y), f.Pos)
	if away.X*toFish.X+away.Y*toFish.Y > 0 && hyp2(acc) > contract.MaxForce*0.9 {
		t.Fatal("a far fish still feels the big-body push")
	}
}

// G51: the pod bends into crosswise turns — over a couple of minutes the
// giant's heading swings wide (≥75°) several times, and the clamp keeps the
// body an arc through every one of them.
func TestTitanCrossTurns(t *testing.T) {
	w := titanWorld(t, 3)
	w.spawnPod()
	g := w.titanGiant()
	const dt = 0.05
	type sample struct {
		x, y float64
	}
	prev := sample{g.Pos.X, g.Pos.Y}
	var headings []float64
	for i := 0; i < 240*20; i++ { // 240 s: two full sweeps
		w.Update(dt, Input{})
		if i%20 == 0 { // once a second: displacement heading
			cur := sample{g.Pos.X, g.Pos.Y}
			dx, dy := cur.x-prev.x, cur.y-prev.y
			prev = cur
			if dx*dx+dy*dy > 4 { // ignore near-standstill samples
				headings = append(headings, math.Atan2(dy, dx))
			}
		}
	}
	turns := 0
	skip := 5 // compare headings ~5 s apart
	for i := skip; i < len(headings); i++ {
		d := math.Abs(headings[i] - headings[i-skip])
		if d > 3.14159 {
			d = 6.28318 - d
		}
		if d > 1.3 {
			turns++
			i += skip // one turn per window
		}
	}
	if turns < 2 {
		t.Fatalf("only %d wide turns in 150 s — the giants cruise straight forever", turns)
	}
}

// G52: a giant thrown off the frame by a scare bolt always steers back into
// view — the tank never loses a titan over the edge for long. Legit visit
// exits (phase 3, boosted) and the absent gap between visits don't count.
func TestTitanEdgeRecovery(t *testing.T) {
	w := titanWorld(t, 3)
	w.spawnPod()
	longest, streak := 0.0, 0.0
	const dt = 0.05
	for i := 0; i < 120*20; i++ { // 120 s of scares near the right edge
		w.Update(dt, Input{})
		if i%(8*20) == 0 {
			if g := w.titanGiant(); g != nil {
				g.flee(w.W-80, g.Pos.Y, contract.MaxForce)
			}
		}
		if g := w.titanGiant(); g != nil && w.titanPhase == 2 {
			// only the roaming phase owes the viewer a visible giant — the
			// visit exit (phase 3) is a deliberate departure
			if g.Pos.X < 0 || g.Pos.X > w.W {
				streak += dt
				longest = max(longest, streak)
			} else {
				streak = 0
			}
		} else {
			streak = 0
		}
	}
	if longest > 10 {
		t.Fatalf("a giant stayed off-screen %.1f s — the escape the viewer caught", longest)
	}
}

// G52: the pod favors the upper 80% and hugs the sand line — headings stay
// near horizontal, deep water is rare.
func TestTitanFavorsUpperWaterAndHorizon(t *testing.T) {
	w := titanWorld(t, 3)
	w.spawnPod()
	g := w.titanGiant()
	deepSamples, steep := 0, 0
	prevX, prevY := g.Pos.X, g.Pos.Y
	for i := 0; i < 120*10; i++ { // 120 s, sampled every 0.1 s
		w.Update(0.1, Input{})
		if g.Pos.Y > w.H*contract.TitanUpperBand {
			deepSamples++
		}
		dx, dy := math.Abs(g.Pos.X-prevX), math.Abs(g.Pos.Y-prevY)
		if dy > 12 && dy > 2*dx {
			// a STEEP pitch: fast and more vertical than horizontal —
			// slow diagonal drift is the natural glide, not a violation
			steep++
		}
		prevX, prevY = g.Pos.X, g.Pos.Y
	}
	if n := 120 * 10; deepSamples*100/n > 5 {
		t.Fatalf("the pod spent %d%% of its time in the bottom band", deepSamples*100/n)
	}
	if steep*100/(120*10) > 1 {
		t.Fatalf("steep pitches on %d%% of samples — the giant must hug the sand line", steep*100/(120*10))
	}
}

// G52: big bodies beat slower — the heavy, real read.
func TestBeatSlowsWithGrowth(t *testing.T) {
	w := titanWorld(t, 1)
	w.spawnPod()
	g := w.titanGiant()
	var small *Fish
	for _, f := range w.fishes {
		if f.Sp.Role == contract.RoleNormal {
			small = f
			break
		}
	}
	gPhase, sPhase := g.phase, small.phase
	for i := 0; i < 60; i++ {
		w.Update(1/60.0, Input{})
	}
	if dg, ds := g.phase-gPhase, small.phase-sPhase; dg >= ds {
		t.Fatalf("the giant's tail beat %.2f is not slower than a small fish's %.2f", dg, ds)
	}
}
