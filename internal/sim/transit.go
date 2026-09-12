// F25: door-to-door transit — a fish that drifts near one of the volcanic
// crag's mouths sometimes swims in, glides through the hollow rock and darts
// out of a DIFFERENT mouth of the same crag. v0.3.8: the hide flip is
// BINARY — the fish stays fully opaque on the crag face until it crosses
// the mouth plane, then vanishes (the near-black hole sells the swallow),
// and re-appears fully opaque just outside the far mouth. Partial alpha
// used to render translucent ghosts over open water. The Chosen never
// transits: she is always on camera.
package sim

import (
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// SetHoles installs the enterable crag mouths (game builds them from the
// rock layout; holes survive independently of zones).
func (w *World) SetHoles(hs []contract.Hole) { w.holes = hs }

// Holes returns the installed door mouths.
func (w *World) Holes() []contract.Hole { return w.holes }

// tickTransit advances the transit state machine for one fish and schedules
// new transits for idle cruisers near a mouth.
func (w *World) tickTransit(f *Fish, dt float64) {
	if len(w.holes) == 0 {
		return
	}
	if !f.transiting {
		w.maybeStartTransit(f, dt)
		return
	}
	f.transitT += dt
	switch f.transitPh {
	case 0: // approach — steering aims at the mouth; watch for arrival
		d := hyp2(sub(f.Pos, f.inHole.Center))
		if d < maxF(12, f.inHole.Radius*0.45) {
			f.transitPh = 1
			f.transitT = 0
			f.transitDur = contract.TransitMinSec + w.rng.Float64()*(contract.TransitMaxSec-contract.TransitMinSec)
			f.Vel = v2(0, 0)
			f.Hide01 = 1 // binary swallow: opaque up to the mouth plane, gone past it
			w.logf("behavior", f.Sp.Name+" slips into the volcanic door")
		} else if f.transitT > 6 {
			f.endTransit() // scared off or lost interest
		}
	case 1: // inside — hidden, gliding along the interior curve
		u := clampF(f.transitT/maxF(f.transitDur, 0.1), 0, 1)
		f.Pos = transitPath(f.inHole.Center, f.outHole.Center, w.H, u)
		if u >= 1 {
			// emerge from the far mouth with a burst + a puff of bubbles
			f.transitPh = 2
			f.transitT = 0
			dir := sub(f.outHole.Center, f.Pos)
			if l := hyp2(dir); l > 1 {
				dir = mulS(dir, 1/l)
			} else {
				dir = v2(0, -1)
			}
			burst := f.maxSpeed(f.curNight) * contract.TransitExitBoost
			f.Vel = mulS(dir, burst)
			for i := 0; i < 4; i++ {
				w.bubbles = append(w.bubbles, Bubble{
					Pos:    v2(f.Pos.X+float64(i)*3, f.Pos.Y),
					R:      1 + w.rng.Float64()*2.2,
					Speed:  30 + w.rng.Float64()*30,
					Wobble: w.rng.Float64() * 6.283,
					Seed:   w.rng.Float64(),
				})
			}
			w.logf("behavior", f.Sp.Name+" bursts out of another door")
		}
	case 2: // leaving — still hidden while the burst clears the dark mouth
		if d := hyp2(sub(f.Pos, f.outHole.Center)); d > maxF(26, f.outHole.Radius*0.9) {
			f.Hide01 = 0 // pops out fully opaque at the mouth's edge
			f.endTransit()
		} else if f.transitT > 3 {
			f.endTransit() // failsafe: never stay a ghost forever
		}
	}
}

// maybeStartTransit rolls the dice for an idle fish near a mouth.
func (w *World) maybeStartTransit(f *Fish, dt float64) {
	if f.Sp.Role == contract.RoleChosen || f.Dying || f.Resting || f.Satiety < 0.15 ||
		f.loungeT > 0 || f.CourtID != "" || f.chaseT > 0 || f.zoomT > 0 || f.attachT > 0 {
		return
	}
	best, bestD := -1, contract.TransitNearPx
	for i, h := range w.holes {
		if d := hyp2(sub(f.Pos, h.Center)); d < bestD {
			best, bestD = i, d
		}
	}
	if best < 0 || w.rng.Float64() >= contract.TransitChanceSec*dt {
		return
	}
	in := w.holes[best]
	var outs []contract.Hole
	for i, h := range w.holes {
		if i != best && h.Rock == in.Rock {
			outs = append(outs, h)
		}
	}
	if len(outs) == 0 {
		return
	}
	f.transiting = true
	f.inHole = in
	f.outHole = outs[w.rng.Intn(len(outs))]
	f.transitPh = 0
	f.transitT = 0
	f.transitDur = contract.TransitMinSec + w.rng.Float64()*(contract.TransitMaxSec-contract.TransitMinSec)
	f.Resting = false
	f.restTarget = nil
}

// endTransit clears the state (leaving only after the mouth is cleared).
func (f *Fish) endTransit() {
	f.transiting = false
	f.transitPh = 0
	f.transitT = 0
	f.Hide01 = 0
}

// abortTransit is called from flee(): a scared fish abandons the approach,
// but nothing can reach it once it is inside the rock.
func (f *Fish) abortTransit() {
	if f.transiting && f.transitPh == 0 {
		f.endTransit()
	}
}

// Transiting reports whether the fish is mid door-to-door trip.
func (f *Fish) Transiting() bool { return f.transiting }

// steerTransit replaces the steering mix while swimming into the mouth.
func (f *Fish) steerTransit(maxSp float64) contract.Vec2 {
	d := sub(f.inHole.Center, f.Pos)
	if l := hyp2(d); l > 1 {
		d = mulS(d, 1/l)
	}
	desired := mulS(d, maxSp*0.85)
	return v2((desired.X-f.Vel.X)*2.2, (desired.Y-f.Vel.Y)*2.2)
}

// transitPath walks the fish through the rock along a quiet arc between the
// two mouths (invisible — Hide01 — but coherent if a frame ever shows it).
func transitPath(a, b contract.Vec2, h float64, u float64) contract.Vec2 {
	// control point: the crag's core, above the midpoint
	p1 := v2((a.X+b.X)/2, (a.Y+b.Y)/2-h*0.06)
	iu := 1 - u
	return v2(
		iu*iu*a.X+2*iu*u*p1.X+u*u*b.X,
		iu*iu*a.Y+2*iu*u*p1.Y+u*u*b.Y)
}

// DebugParkByDoor teleports the nearest normal fish to just outside a door —
// evidence harness only (deterministic -shot sequence).
func (w *World) DebugParkByDoor(i int) {
	if i < 0 || i >= len(w.holes) || len(w.fishes) == 0 {
		return
	}
	h := w.holes[i]
	best, bestD := -1, 1e9
	for idx, f := range w.fishes {
		if f.Dying || f.Sp.Role == contract.RoleChosen || f.transiting {
			continue
		}
		if d := hyp2(sub(f.Pos, h.Center)); d < bestD {
			best, bestD = idx, d
		}
	}
	if best < 0 {
		return
	}
	f := w.fishes[best]
	f.Pos = add(h.Center, v2(-h.Radius-18, 6))
	f.Vel = v2(0, 0)
	f.followSpine(0)
}

// DebugTransit forces the nearest normal fish into a transit through the
// crag (evidence harness only — deterministic proof for -shot).
func (w *World) DebugTransit() bool {
	if len(w.holes) < 2 {
		return false
	}
	best, bestD := -1, 1e9
	for i, f := range w.fishes {
		if f.Dying || f.Sp.Role == contract.RoleChosen {
			continue
		}
		if d := hyp2(sub(f.Pos, w.holes[0].Center)); d < bestD {
			best, bestD = i, d
		}
	}
	if best < 0 {
		return false
	}
	f := w.fishes[best]
	f.transiting = true
	f.inHole = w.holes[0]
	var outs []contract.Hole
	for _, h := range w.holes[1:] {
		if h.Rock == f.inHole.Rock {
			outs = append(outs, h)
		}
	}
	if len(outs) == 0 {
		f.endTransit()
		return false
	}
	f.outHole = outs[w.rng.Intn(len(outs))]
	f.transitPh = 0
	f.transitT = 0
	f.transitDur = contract.TransitMinSec
	return true
}
