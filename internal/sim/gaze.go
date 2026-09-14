// v1.2 G93: the glass gaze — a school fish now and then swims to a spot
// near the front pane, turns front-on and STARES at the viewer. The
// aquarist's favourite: the tank looks back. School fish only (the big
// residents keep their own gravity); two concurrent gazers, tops — a tank
// that always stares is a horror, a tank that never does is furniture.
package sim

import (
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// gazeCount counts the fish currently running the gaze behavior.
func (w *World) gazeCount() int {
	n := 0
	for _, f := range w.fishes {
		if f.gazePhase > 0 {
			n++
		}
	}
	return n
}

// gazeEligible: who may pick the glass, and when the pick survives.
func (f *Fish) gazeEligible() bool {
	return f.Sp.Role == contract.RoleNormal && !f.Dying && !f.transiting &&
		f.Hide01 == 0 && f.attachT <= 0 && f.loungeT <= 0 && !f.Resting &&
		f.CourtID == "" && f.chaseT <= 0 && f.zoomT <= 0 && f.scareT <= 0
}

// gazeInterrupted: an active gaze yields to real life instantly.
func (f *Fish) gazeInterrupted() bool {
	return f.Dying || f.transiting || f.attachT > 0 || f.loungeT > 0 ||
		f.CourtID != "" || f.chaseT > 0 || f.zoomT > 0 || f.scareT > 0
}

// endGaze folds the pose away fast and schedules the next urge.
func (f *Fish) endGaze(w *World) {
	if f.gazePhase == 0 {
		return
	}
	f.gazePhase = 3
	f.rollGazeCD()
}

// rollGazeCD schedules the next urge, curiosity-weighted.
func (f *Fish) rollGazeCD() {
	cur := 0.6 + 0.8*contract.Clamp(f.Sp.Behavior.Curiosity, 0, 1)
	f.gazeCD = (contract.GazeCDMin + f.rng.Float64()*(contract.GazeCDMax-contract.GazeCDMin)) / cur
}

// beginGaze picks the spot: mid-water, clear of the walls, the floor and
// her circle — a place a fish can actually hover and stare from.
func (f *Fish) beginGaze(w *World) bool {
	for attempt := 0; attempt < 8; attempt++ {
		x := w.W * (0.25 + f.rng.Float64()*0.5)
		y := w.H * (0.30 + f.rng.Float64()*0.40)
		pt := v2(x, y)
		if hyp2(sub(pt, f.Pos)) < 120 {
			continue // a visible swim, not a shuffle in place
		}
		clear := true
		for _, z := range w.zones {
			if hyp2(sub(pt, z.Center)) < z.Radius+60+f.bodyLen*0.5 {
				clear = false
				break
			}
		}
		if !clear {
			continue
		}
		f.gazePt = pt
		f.gazePhase = 1
		return true
	}
	return false
}

// tickGaze runs the state machine; called from advance before steering.
func (f *Fish) tickGaze(dt float64, w *World) {
	switch f.gazePhase {
	case 0:
		if f.Sp.Role != contract.RoleNormal {
			return
		}
		f.gazeCD -= dt
		if f.gazeCD <= 0 {
			if f.gazeEligible() && w.gazeCount() < contract.GazeMaxGazers && f.beginGaze(w) {
				f.gazeCD = 0 // consumed; the CD re-rolls on release
			} else {
				f.gazeCD = 8 // retry soon
			}
		}
	case 1:
		if f.gazeInterrupted() {
			f.endGaze(w)
			return
		}
		f.gazeBlend = maxF(0, f.gazeBlend-contract.GazeBlendOut*dt)
		if hyp2(sub(f.gazePt, f.Pos)) < contract.GazeArrivePx {
			f.gazePhase = 2
			f.gazeT = contract.GazeStareMin + f.rng.Float64()*(contract.GazeStareMax-contract.GazeStareMin)
		}
	case 2:
		if f.gazeInterrupted() {
			f.endGaze(w)
			return
		}
		f.gazeT -= dt
		f.gazeBlend = minF(1, f.gazeBlend+contract.GazeBlendInRate*dt)
		// the stare is a rest: it recharges half as fast as a cave (F15)
		f.Energy = minF(1, f.Energy+dt*0.15)
		if f.gazeT <= 0 {
			f.gazePhase = 3
		}
	case 3:
		f.gazeBlend = maxF(0, f.gazeBlend-contract.GazeBlendOut*dt)
		if f.gazeBlend <= 0 {
			f.gazePhase = 0
			f.rollGazeCD()
		}
	}
}

// gazeSteer replaces the steering mix while the gaze runs: a committed
// glide to the spot, then a hover with a slow, curious sway.
func (f *Fish) gazeSteer(dt, maxSp float64, w *World) contract.Vec2 {
	_ = dt
	if f.gazePhase == 1 {
		d := sub(f.gazePt, f.Pos)
		desired := mulS(norm2(d), maxSp*0.55)
		return mulS(v2(desired.X-f.Vel.X, desired.Y-f.Vel.Y), 0.8)
	}
	// stare (and release): brake to a hover; the sway is gentle enough to
	// read as water, not as swimming
	sway := sin(w.time*1.9+f.zPhase) * 6
	brake := mulS(f.Vel, -3.0)
	return v2(brake.X+sway*0.02, brake.Y+sway)
}

// gazeHover enforces the stare's stillness: while blended in, excess speed
// GLIDES down to the hover ceiling (the G88 pattern — an instant clamp
// would be the mid-water brake the speed-envelope law forbids).
func (f *Fish) gazeHover(dt float64) {
	if f.gazePhase == 2 && f.gazeBlend > 0.5 {
		if sp := hyp2(f.Vel); sp > contract.GazeSpeedMax {
			f.Vel = mulS(f.Vel, glideCap(sp, contract.GazeSpeedMax))
		}
	}
}

// Gaze01 reports the face-on blend for the renderer.
func (f *Fish) Gaze01() float64 { return clampF(f.gazeBlend, 0, 1) }

// DebugForceGaze sends a fish to the glass now — evidence harness only.
func (w *World) DebugForceGaze(f *Fish) {
	f.gazePt = v2(w.W*0.5, w.H*0.5)
	f.gazePhase = 1
}
