// v1.1: the scalare pod -- five big angelfish-shaped residents (G39-G43,
// G58-G59). The pod is a permanent fixture of the tank: it is spawned with
// the world, sweeps the sand line left-to-right and right-to-left forever,
// curls 180 degrees at the glass, darts only rarely when hungry, and --
// after a long starvation, with a tank-wide cooldown and a population
// floor -- may take one small fish. The pod is ambient: it never counts
// toward popCap, breeding, aging or the behavior event picks, and it is
// never persisted.
package sim

import (
	"math"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// titanSpecies finds the core-seeded scalare species (nil when absent).
func (w *World) titanSpecies() *contract.Species {
	for _, s := range w.species {
		if s.Role == contract.RoleTitan {
			return s
		}
	}
	return nil
}

// titanCount reports live pod members (debug overlay evidence line).
func (w *World) titanCount() int {
	n := 0
	for _, f := range w.fishes {
		if f.Sp.Role == contract.RoleTitan && !f.Dying {
			n++
		}
	}
	return n
}

// TitanCount is the exported overlay/evidence accessor.
func (w *World) TitanCount() int { return w.titanCount() }

// ensurePod keeps the pod resident (G39) -- the mirror of ensureChosen and
// ensureSharks: whatever wipes the five, the next tick brings them back.
// Nothing in the sim can remove a pod member (no aging, culling, fade or
// predation on titans), so the only realistic gap is a fresh or restored
// session -- exactly the paths the visit-era refactor left unseeded.
func (w *World) ensurePod() {
	if w.titanCount() == 0 {
		w.spawnPod()
	}
}

// tickTitans advances the pod rare-hunt gate. The pod itself is a
// permanent resident: no visit machine, no departures -- it sweeps forever.
func (w *World) tickTitans(dt float64) {
	w.tickPredation(dt)
}

// steerTitan replaces the whole steering mix for pod members -- the
// parallel sweep along the sand line, the curl at the glass, the hunger
// lunge and the rare hunt. Titans opt out of caves, doors, tours,
// courtship and cursor games entirely.
func (f *Fish) steerTitan(dt, maxSp float64, w *World) contract.Vec2 {
	acc := v2(0, 0)
	f.seekBonus = 1.0
	addForce := func(desired contract.Vec2, weight float64) {
		acc.X += (desired.X - f.Vel.X) * weight
		acc.Y += (desired.Y - f.Vel.Y) * weight
	}
	// pod cohesion + gentle separation -- tall bodies may overlap a
	// little, that overlap is what sells the depth
	var coh, sep contract.Vec2
	cnt, sepCnt := 0, 0
	for _, o := range w.fishes {
		if o == f || o.Sp.Role != contract.RoleTitan || o.Dying {
			continue
		}
		d := sub(o.Pos, f.Pos)
		dist := hyp2(d)
		if dist < 1e-3 {
			continue
		}
		if dist < 260 {
			coh.X += d.X
			coh.Y += d.Y
			cnt++
		}
		if dist < f.bodyLen*0.3 {
			sep.X -= d.X / dist
			sep.Y -= d.Y / dist
			sepCnt++
		}
	}
	if cnt > 0 {
		toC := mulS(coh, 1/float64(cnt))
		toC.Y *= contract.TitanHeadFlat // the pack holds a horizontal band
		if l := hyp2(toC); l > 1 {
			addForce(mulS(toC, maxSp*0.7/l), 0.9)
		}
	}
	if sepCnt > 0 {
		addForce(mulS(sep, maxSp), 1.2)
	}
	// the convoy U-turn: near the glass the leader calls it, the whole pod
	// turns as one caravan across the window; the relaxed spine bend draws
	// the arc
	if f.turnT > 0 {
		f.turnT -= dt
	}
	if f.turning > 0 {
		f.turning -= dt
	}
	// G67: mid-turn the body rides a KINEMATIC arc — the heading sweeps a
	// true half circle at constant pace while the head keeps travelling
	// forward. A force-fought turn stalls the nose around a pivot and reads
	// like clock hands; the path cannot.
	if f.turning > 0 {
		p := 1 - f.turning/contract.TitanTurnWindow
		ang := f.turnH0 + f.turnS*math.Pi*p
		// the arc is a true wide U (G67): radius scaled to the body, pace
		// quickening through the turn — v = πR/window ≈ 2× cruise, so the
		// head TRAVELS a quarter tank while it turns, never spins on a spot
		r := f.bodyLen * contract.TitanTurnRadiusFrac
		// the arc fits the room it actually has — an up-curl shrinks to the
		// top margin, a down-curl to the upper band
		if f.turnS*f.cruise > 0 {
			if room := f.Pos.Y - (f.bodyLen*0.30 + 6) - 15; room < r {
				r = maxF(f.bodyLen*0.12, room)
			}
		} else if room := w.H*contract.TitanUpperBand - 40 - f.Pos.Y; room < r {
			r = maxF(f.bodyLen*0.12, room)
		}
		// the pace never sinks below cruise — a shrunken arc must not
		// become a slow-motion spin (G67 carve)
		v := maxF(math.Pi*r/contract.TitanTurnWindow, 0.9*maxSp)
		f.Vel = mulS(v2(math.Cos(ang), math.Sin(ang)), v)
		// the arc still bows to her circle (G55): the rim bends the sweep
		// outward while the heading keeps the pure arc — the chain rides
		// the heading, so nothing tears
		for _, z := range w.zones {
			if z.Owner != "chosen" {
				continue
			}
			d := sub(f.Pos, z.Center)
			r := z.Radius + 40 + f.bodyLen*0.5
			if l := hyp2(d); l < r && l > 1 {
				f.Vel = add(f.Vel, mulS(d, (r-l)/r*maxSp*4.5/l))
			}
		}
		f.headingA = ang
		return v2(0, 0) // forces stand down for the arc
	}
	// G66: the turn needs room for its whole arc — the trigger scales with
	// the body, so the sweeping tail never runs past the view edge. The
	// LEADER owns the clock: one call flips the whole caravan together.
	edge := f.bodyLen*(contract.TitanTurnRadiusFrac+0.85) + 30
	if (f.cruise > 0 && f.Pos.X > w.W-edge) || (f.cruise < 0 && f.Pos.X < edge) {
		if f.turnT <= 0 {
			w.titanStartConvoyTurn()
		}
	}
	// the parallel sweep: a steady cruise toward the sweep side -- the
	// heading follows the velocity, so the fish always faces forward
	des := v2(f.cruise*0.8*maxSp, sin(f.wanderA)*0.06*maxSp)
	addForce(des, 0.7)
	// G69: the sweep is a slow wandering glide, not a rail -- each elder
	// draws a personal altitude every ~20-40 s, anywhere from the surface
	// light down to just above the nest level, and eases toward it. The
	// members draw independently (small phase drift), so the pod breathes
	// instead of marching in lockstep.
	f.altT -= dt
	if f.altT <= 0 {
		lo := (f.bodyLen*0.30+6)/w.H + 0.03
		// biased to the ends: a third of the draws ride high under the
		// surface light, the rest glide the broad lower water — the sweep
		// visits both worlds instead of parking mid-tank
		if f.rng.Float64() < 0.34 {
			f.altY = lo + f.rng.Float64()*(0.30-lo)
		} else {
			f.altY = 0.30 + f.rng.Float64()*(contract.TitanAltMax-0.30)
		}
		f.altT = 18 + f.rng.Float64()*22
	}
	// level swimming stays gentle: vertical drift is softly damped -- the
	// body may glide up or down along its sweep, but not ballistically
	addForce(v2(0, -f.Vel.Y*0.75), 0.75)
	// the pod favors the upper 80% -- the sand line is not their water
	if f.Pos.Y > w.H*contract.TitanUpperBand {
		addForce(v2(0, -maxSp), 1.1)
	}
	// G66 ceiling twin: the view top is not their water either -- the
	// ceiling scales with the body so the trailing chain fits during dives
	if f.Pos.Y < f.bodyLen*0.42+20 {
		addForce(v2(0, maxSp), 1.1)
	}
	// the altitude pull always runs (G69): toward the personal drawn
	// altitude, both up and down, at a soft weight -- the dive toward the
	// nest level and the climb to the surface light are the same gentle law
	if off := w.H*f.altY - f.Pos.Y; absF(off) > 8 {
		addForce(v2(0, clampF(off*0.22, -maxSp*0.4, maxSp*0.4)), 1.0)
	}
	// overshoot brake: drifting well past the drawn altitude turns the
	// glide around before it can settle into the bottom band
	if f.Pos.Y > w.H*f.altY+80 {
		addForce(v2(0, -maxSp*0.5), 0.9)
	}
	f.formationSteer(w, maxSp, addForce)
	f.lungeSteer(w, dt, maxSp, addForce)
	// an active hunt overrides everything
	if w.predTgtID == f.ID {
		if tgt := w.fishByID(w.predTgtID); tgt != nil {
			d := sub(tgt.Pos, f.Pos)
			addForce(mulS(d, maxSp*f.burstMul()/maxF(hyp2(d), 1)), 3.0)
			f.seekBonus = f.burstMul()
		}
	}
	// N10 for the deep ones (G49): a right-click scare bolts the whole pod
	// away from the point -- the burst decays as the startle wears off. Off
	// the frame the scare stands down: edge recovery owns the giant there,
	// or the two forces cancel and the giant hovers out of view forever.
	if f.scareT > 0 && f.Pos.X >= 30 && f.Pos.X <= w.W-30 {
		d := sub(f.Pos, f.scarePt)
		addForce(mulS(d, maxSp*f.burstMul()/maxF(hyp2(d), 1)), 2.5)
		burst := 1 + (f.burstMul()-1)*clampF(f.scareT/contract.ScareShelterSec, 0, 1)
		f.seekBonus = maxF(f.seekBonus, burst)
	}
	// G52: edge recovery -- a scare bolt can throw a giant past the frame;
	// anything beyond the margin steers back into view
	if f.Pos.X < 30 || f.Pos.X > w.W-30 {
		tgt := v2(w.W*0.5, w.H*0.45)
		d := sub(tgt, f.Pos)
		addForce(mulS(d, maxSp*f.burstMul()/maxF(hyp2(d), 1)), 2.8)
		f.seekBonus = maxF(f.seekBonus, f.burstMul())
	}
	// N3: her circle holds -- even the deep respects the Chosen aura, and
	// a 640 px body turns away half a body-length early. The urgency grows
	// as the line nears: the sweep arcs around the nest, never across it.
	for _, z := range w.zones {
		if z.Owner != "chosen" {
			continue
		}
		d := sub(f.Pos, z.Center)
		r := z.Radius + 40 + f.bodyLen*0.5
		if l := hyp2(d); l < r && l > 1 {
			urgency := 1 + 2*(1-l/r)
			addForce(mulS(d, maxSp*urgency/l), 2.4)
		}
	}
	if l := hyp2(acc); l > contract.MaxForce*f.seekBonus {
		acc = mulS(acc, contract.MaxForce*f.seekBonus/l)
	}
	// G59/G67 LAST: the body faces where it swims -- the heading eases
	// toward the velocity (at burst pace through a strike), and whatever
	// the mixed forces did, the velocity is shaved off the face's blind
	// side. A scalare may be slowed sideways, never pushed tail-first.
	follow := 2.5
	if f.seekBonus > 1.01 {
		follow = 6.5
	}
	if v := hyp2(f.Vel); v > 1 { // even a slow drift is already a heading
		va := math.Atan2(f.Vel.Y, f.Vel.X)
		da := math.Mod(va-f.headingA+3.14159, 6.28318) - 3.14159
		f.headingA += clampF(da, -follow*dt, follow*dt)
		ax, ay := math.Cos(f.headingA), math.Sin(f.headingA)
		if back := f.Vel.X*ax + f.Vel.Y*ay; back < 0 {
			f.Vel.X -= ax * back
			f.Vel.Y -= ay * back
		}
		// the mixed acceleration may not push the nose backwards either —
		// integration happens after this pass
		if back := acc.X*ax + acc.Y*ay; back < 0 {
			acc.X -= ax * back
			acc.Y -= ay * back
		}
	}
	return acc
}
