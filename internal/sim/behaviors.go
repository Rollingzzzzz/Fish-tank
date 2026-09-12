// F4: ambient behavior events — startles, bully chases, zoomies and nudges.
// All rate-limited by world timers, seeded by the world RNG (D5), and logged
// so the viewer can follow the drama in the tank log.
package sim

import (
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// tickBehaviors advances the random event schedulers and per-fish states.
func (w *World) tickBehaviors(dt float64) {
	w.evStartle -= dt
	if w.evStartle <= 0 {
		w.evStartle = contract.StartleMeanSec * (0.5 + w.rng.Float64())
		w.doStartle()
	}
	w.evChase -= dt
	if w.evChase <= 0 {
		// on reject, retry soon — the 70 s mean only counts real chases
		w.evChase = 70 * (0.5 + w.rng.Float64())
		if !w.doChase() {
			w.evChase = 12
		}
	}
	w.evZoom -= dt
	if w.evZoom <= 0 {
		w.evZoom = 55 * (0.5 + w.rng.Float64())
		w.doZoomies()
	}
	w.evNudge -= dt
	if w.evNudge <= 0 {
		w.evNudge = 30 * (0.5 + w.rng.Float64())
		w.doNudge()
	}
	for _, f := range w.fishes {
		if f.chaseT > 0 {
			f.chaseT -= dt
			if f.chaseT <= 0 {
				f.chaseID = ""
			}
		}
		if f.zoomT > 0 {
			f.zoomT -= dt
		}
		w.tickLounge(f, dt)  // F15: cave-lounge scheduler
		w.tickTransit(f, dt) // F25: door-to-door transit scheduler
	}
}

// doStartle panics part of the school around a random origin; skittish fish
// bolt hardest (the Skittish knob finally does its job).
func (w *World) doStartle() {
	live := w.aliveFishes()
	if len(live) < 3 {
		return
	}
	src := live[w.rng.Intn(len(live))]
	srcPos := src.Pos
	count := 0
	for _, f := range live {
		if f == src {
			continue
		}
		if d := hyp2(sub(f.Pos, srcPos)); d < 150 &&
			w.rng.Float64() < 0.3+0.7*contract.Clamp(f.Sp.Behavior.Skittish, 0, 1) {
			f.flee(srcPos.X, srcPos.Y, contract.MaxForce*(0.5+0.5*f.Sp.Behavior.Skittish))
			count++
		}
	}
	if count >= 2 {
		w.startles++
		w.logf("behavior", src.Sp.Name+" startles the school")
	}
}

// doChase picks a bully and a victim; the chase breaks off before contact.
// Returns false when no valid pair was found this attempt.
func (w *World) doChase() bool {
	live := w.aliveFishes()
	if len(live) < 3 {
		return false
	}
	a := live[w.rng.Intn(len(live))]
	b := live[w.rng.Intn(len(live))]
	if a == b || b == nil || a.Sp.Role == contract.RoleChosen ||
		b.Sp.Role == contract.RoleChosen { // nobody bullies the eternal one
		return false
	}
	d := hyp2(sub(a.Pos, b.Pos))
	if d < 40 || d > 600 || a.chaseT > 0 || b.chaseT > 0 || a.CourtID != "" ||
		a.loungeT > 0 { // a lounging bully won't chase (F15); a scared victim
		return false // is welcome — flee() snaps it out of the cave anyway
	}
	a.chaseID, a.chaseT = b.ID, 1.0+w.rng.Float64()*1.5
	b.flee(a.Pos.X, a.Pos.Y, contract.MaxForce*0.4)
	w.chases++
	w.logf("behavior", a.Sp.Name+" chases "+b.Sp.Name+" around the tank")
	return true
}

// doZoomies sends one fish on a fast celebratory loop.
func (w *World) doZoomies() {
	live := w.aliveFishes()
	if len(live) == 0 {
		return
	}
	f := live[w.rng.Intn(len(live))]
	if f.zoomT > 0 || f.CourtID != "" || f.chaseT > 0 || f.loungeT > 0 {
		return // a lounging fish is too comfortable for zoomies (F15)
	}
	f.zoomT = 2.0 + w.rng.Float64()*1.5
	f.zoomC = f.Pos
	f.zoomAng = f.rng.Float64() * 6.283
	w.zoomies++
	w.logf("behavior", f.Sp.Name+" zooms around the tank")
}

// doNudge bumps two nearby schoolmates into a comedic veer.
func (w *World) doNudge() {
	live := w.aliveFishes()
	if len(live) < 2 {
		return
	}
	a := live[w.rng.Intn(len(live))]
	for _, b := range live {
		if b == a || b.Sp.ID != a.Sp.ID {
			continue
		}
		if d := hyp2(sub(a.Pos, b.Pos)); d < 60 {
			away := mulS(norm2(sub(b.Pos, a.Pos)), contract.MaxForce*0.35)
			a.fleeImp.X -= away.X
			a.fleeImp.Y -= away.Y
			b.fleeImp.X += away.X
			b.fleeImp.Y += away.Y
			a.Resting, b.Resting = false, false
			w.nudges++
			w.logf("behavior", a.Sp.Name+" bumps into "+b.Sp.Name)
			return
		}
	}
}
