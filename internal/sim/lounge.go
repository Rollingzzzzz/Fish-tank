// F15: cave lounging — idle, well-fed fish occasionally drift into a volcanic
// cave and hover there. State fields
// live on Fish; this file owns the scheduler and the calm steering hold.
package sim

import (
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// tickLounge runs the per-fish lounge scheduler. A staggered countdown
// (mean ≈ LoungeMeanSec per fish) picks eligible idlers — free of courtship,
// chases, zoomies, glass-attach and rest, with a full-enough belly — and
// claims the nearest cave for LoungeDwellMin..Max seconds. Fear (flee) and
// courtship cut the stay short. v0.3.8: a cave hosts ONE normal fish at a
// time and at most LoungeSchoolShare of the school lounges at once — the
// towers must not vacuum the tank. The Chosen is exempt: she may always
// join and never counts against either cap.
func (w *World) tickLounge(f *Fish, dt float64) {
	if f.Dying {
		return
	}
	if f.loungeT > 0 {
		f.loungeT -= dt
		// exits: the dwell ran out, love called (courtship), or the belly
		// emptied back below the entry bar — hunger beats comfort
		if f.loungeT <= 0 || f.CourtID != "" || f.Satiety <= 0.3 {
			f.loungeT, f.loungeC = 0, v2(0, 0)
		}
		return
	}
	f.loungeNext -= dt
	if f.loungeNext > 0 {
		return
	}
	idle := f.CourtID == "" && f.chaseT <= 0 && f.zoomT <= 0 &&
		f.attachT <= 0 && !f.Resting && f.Satiety > 0.3
	if !idle {
		f.loungeNext = 5 + w.rng.Float64()*5 // busy right now — re-check soon
		return
	}
	chosen := f.Sp.Role == contract.RoleChosen
	if !chosen { // v0.3.8 caps — she is the exception, everyone else waits
		lounging := 0
		for _, o := range w.fishes {
			if o != f && o.loungeT > 0 && o.Sp.Role != contract.RoleChosen {
				lounging++
			}
		}
		if float64(lounging) >= contract.LoungeSchoolShare*float64(len(w.fishes)) {
			f.loungeNext = 12 + w.rng.Float64()*8 // school cap full — roam on
			return
		}
	}
	var cave *contract.Zone
	bestD := 1e18
	for i := range w.zones {
		z := &w.zones[i]
		if z.Owner != "cave" {
			continue
		}
		if !chosen && w.caveOccupied(z.Center, f) {
			continue // one fish per cave
		}
		if d := hyp2(sub(z.Center, f.Pos)); d < bestD {
			bestD, cave = d, z
		}
	}
	if cave == nil {
		f.loungeNext = 12 + w.rng.Float64()*8 // every cave taken — try later
		return
	}
	f.loungeC = cave.Center
	f.loungeT = contract.LoungeDwellMin +
		w.rng.Float64()*(contract.LoungeDwellMax-contract.LoungeDwellMin)
	f.loungeNext = contract.LoungeMeanSec * (0.5 + w.rng.Float64())
	w.lounges++
	w.logf("behavior", f.Sp.Name+" lounges in the cave")
}

// caveOccupied reports whether another NORMAL fish currently lounges at
// the given cave center (v0.3.8: one fish per cave).
func (w *World) caveOccupied(c contract.Vec2, self *Fish) bool {
	for _, o := range w.fishes {
		if o != self && o.loungeT > 0 && o.loungeC == c &&
			o.Sp.Role != contract.RoleChosen {
			return true
		}
	}
	return false
}

// steerLounge replaces the normal steering mix while a fish lounges: a
// gentle hold inside the cave near its center, a soft bob when settled —
// wander, boids and food seeking are all suppressed (pure cave calm).
// Wall margins are skipped: the hold anchors the fish well inside the tank
// and advance() still hard-clamps the tank bounds.
func (f *Fish) steerLounge(maxSp float64, w *World) contract.Vec2 {
	acc := v2(0, 0)
	toC := sub(f.loungeC, f.Pos)
	if d := hyp2(toC); d > 26 {
		desired := mulS(toC, maxSp*0.4/maxF(d, 1))
		acc.X += (desired.X - f.Vel.X) * 1.2
		acc.Y += (desired.Y - f.Vel.Y) * 1.2
	} else {
		bob := sin(w.time*1.1+float64(f.Seed%9)) * 5
		acc.Y += (bob - f.Vel.Y*2) * 0.5
	}
	if l := hyp2(acc); l > contract.MaxForce {
		acc = mulS(acc, contract.MaxForce/l)
	}
	return acc
}

// Lounging reports whether the fish is currently enjoying a cave (F15) —
// read by the renderer for the lazy tail curl.
func (f *Fish) Lounging() bool { return f.loungeT > 0 && !f.Dying }
