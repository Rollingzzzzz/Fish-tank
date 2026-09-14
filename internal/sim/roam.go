// v0.3.8: roaming + shelter steering (split from fish_steering.go, line
// ceiling) — where free-swimming fish GO. The tour sweeps idle cruisers
// across the whole tank (the school used to huddle at the rock towers);
// the shelter pull carries scared or weary fish to the caves.
package sim

import (
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// tourSteer steers an idle cruiser toward its current tour waypoint,
// refreshing the waypoint when reached or expired. Returns the desired
// velocity delta and whether the tour is active at all — it pauses while
// the fish is busy (courtship, chase, zoomies, glass-attach, lounge,
// transit, rest) or scared (shelter comes first) or, for the Chosen, while
// she is home in her nest: her own sanctuary must never push her outward.
func (f *Fish) tourSteer(w *World, dt, maxSp float64) (contract.Vec2, bool) {
	if f.CourtID != "" || f.chaseT > 0 || f.zoomT > 0 || f.attachT > 0 ||
		f.loungeT > 0 || f.transiting || f.Resting || f.scareT > 0 ||
		hyp2(f.fleeImp) > 40 {
		return v2(0, 0), false
	}
	if f.Sp.Role == contract.RoleChosen && f.inNest(w) {
		return v2(0, 0), false
	}
	f.tourT -= dt
	if f.tourT <= 0 || hyp2(sub(f.tourC, f.Pos)) < 60 {
		yLo, yHi := 0.20, 0.80 // v1.1 G66: leave the trailing body headroom
		xLo, xMul := 0.08, 0.84
		persist := 1.0
		if f.Sp.Role == contract.RoleShark {
			yLo, yHi = 0.15, 0.62 // v1.1: the hunter roams the upper water
			// G79: the pair OWNS the stage. A 7 s waypoint at ~55 px/s is
			// ~300 px of travel — the hunters looped locally and never once
			// crossed into the right quarter (measured x-span [0.00, 0.78]
			// over 3 min). They now hold a waypoint long enough to reach the
			// far side, and most draws pull to the OPPOSITE half of the tank,
			// so a crossing reads as a patrol, not a stroll.
			persist = 2.5
			if w.rng.Float64() < 0.6 {
				if f.Pos.X < w.W*0.5 {
					xLo, xMul = 0.50, 0.42
				} else {
					xLo, xMul = 0.08, 0.42
				}
			}
		}
		f.tourC = v2(w.W*(xLo+w.rng.Float64()*xMul), w.H*(yLo+w.rng.Float64()*(yHi-yLo)))
		f.tourT = contract.RoamMeanSec * persist * (0.6 + w.rng.Float64()*0.8)
	}
	d := sub(f.tourC, f.Pos)
	l := hyp2(d)
	if l <= 60 {
		return v2(0, 0), false
	}
	return mulS(d, maxSp*0.6/maxF(l, 1)), true
}

// inNest reports whether the fish stands inside the chosen-aura
// zone (only the Chosen ever does — N3 hard-excludes everyone else).
func (f *Fish) inNest(w *World) bool {
	for i := range w.zones {
		z := &w.zones[i]
		if z.Owner == "chosen" && hyp2(sub(z.Center, f.Pos)) < z.Radius+10 {
			return true
		}
	}
	return false
}

// shelterSteer pulls a scared or weary fish toward the nearest cave. The
// pull outlives the startle impulse (scareT) so a frightened fish actually
// REACHES cover, and the pure energy bar sits at 0.15 — caves are for the
// truly weary, not a pit stop at the first yawn.
func (f *Fish) shelterSteer(w *World, maxSp float64) (contract.Vec2, bool) {
	if f.Sp.Role == contract.RoleShark {
		return v2(0, 0), false // v1.1: the hunter does not hide
	}
	if f.CourtID != "" || (hyp2(f.fleeImp) <= 40 && f.Energy >= 0.15 && f.scareT <= 0) {
		return v2(0, 0), false
	}
	var cave *contract.Zone
	bestD := 1e18
	for i := range w.zones {
		z := &w.zones[i]
		if z.Owner != "cave" {
			continue
		}
		if d := hyp2(sub(z.Center, f.Pos)); d < bestD {
			bestD, cave = d, z
		}
	}
	if cave == nil || bestD <= 26 {
		return v2(0, 0), false
	}
	d := sub(cave.Center, f.Pos)
	return mulS(d, maxSp*0.7/maxF(hyp2(d), 1)), true
}
