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

// titanGiant returns the pod leader (sizeMul ~ 1) -- the only hunter.
func (w *World) titanGiant() *Fish {
	for _, f := range w.fishes {
		if f.Sp.Role == contract.RoleTitan && !f.Dying && f.sizeMul >= 0.99 {
			return f
		}
	}
	return nil
}

// spawnPod builds the resident pod: 1 leader + 4 escorts, already inside
// the tank in a loose diagonal, all sweeping the same direction.
func (w *World) spawnPod() {
	sp := w.titanSpecies()
	if sp == nil {
		return
	}
	dir := 1.0
	if w.rng.Float64() < 0.5 {
		dir = -1.0
	}
	leadX := w.W * (0.30 + w.rng.Float64()*0.40)
	y := w.H * (0.30 + w.rng.Float64()*0.10)
	for i := 0; i < contract.TitanPodMax; i++ {
		mul := 1.0
		if i > 0 {
			mul = 0.31 + w.rng.Float64()*0.25 // escorts: ~130-230 px bodies
		}
		f := newFish(sp, w.rng.Int63(), v2(leadX+float64(i)*46, y+float64(i%3)*26), 6, w.nextID())
		f.sizeMul = mul
		f.cruise = dir
		f.headingA = 0
		if dir < 0 {
			f.headingA = 3.14159
		}
		// staggered parallel slots behind the leader (G59) -- a loose
		// procession whose bodies overlap a little, never a stack
		f.slotBack = float64(i) * 45
		f.slotY = float64(i%3-1) * 34
		f.bodyLen = f.targetLen()
		f.segLen = f.bodyLen / (contract.SpineSegments - 1)
		for j := range f.Spine {
			f.Spine[j] = v2(f.Pos.X-float64(j)*f.segLen*dir, f.Pos.Y)
		}
		// layout hygiene: the laid chain stays inside the canvas (G66) --
		// test tanks are small, and a clamp-folded spawn reads as a glitch
		for j := range f.Spine {
			f.Spine[j].X = clampF(f.Spine[j].X, 10, w.W-10)
			f.Spine[j].Y = clampF(f.Spine[j].Y, 10, w.H-10)
		}
		w.fishes = append(w.fishes, f)
	}
	w.logf("nature", "the silver elders glide in -- five shadows, one drift")
}

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
	// the curl: at the glass rotate the heading half a turn across the
	// window; the spine relaxed bend draws the curl
	if f.turnT > 0 {
		f.turnT -= dt
	}
	if f.turning > 0 {
		f.turning -= dt
	}
	// G66: the curl needs room for its whole arc — the trigger scales with
	// the body, so the sweeping tail never runs past the view edge
	edge := f.bodyLen*0.5 + 24
	if (f.cruise > 0 && f.Pos.X > w.W-edge) || (f.cruise < 0 && f.Pos.X < edge) {
		if f.turnT <= 0 {
			f.turnT = contract.TitanTurnWindow + 60
			f.turning = contract.TitanTurnWindow
			f.cruise = -f.cruise
		}
	}
	// G59: the body faces where it swims -- the heading eases toward the
	// velocity direction, so a scalare can never glide tail-first. The
	// 180 degree curl at the glass emerges from the forces: the formation
	// pulls the pod around and the heading follows the arc.
	if v := hyp2(f.Vel); v > 8 {
		va := math.Atan2(f.Vel.Y, f.Vel.X)
		da := math.Mod(va-f.headingA+3.14159, 6.28318) - 3.14159
		f.headingA += clampF(da, -2.5*dt, 2.5*dt)
	}
	// the parallel sweep: a steady cruise toward the sweep side -- the
	// heading follows the velocity, so the fish always faces forward
	des := v2(f.cruise*0.8*maxSp, sin(f.wanderA)*0.06*maxSp)
	addForce(des, 0.7)
	// level swimming: vertical drift is damped hard -- the body tracks the
	// sand line and steep climbs or dives stay rare
	addForce(v2(0, -f.Vel.Y*1.2), 1.2)
	// the pod favors the upper 80% -- the sand line is not their water
	if f.Pos.Y > w.H*contract.TitanUpperBand {
		addForce(v2(0, -maxSp), 1.1)
	}
	// G66 ceiling twin: the view top is not their water either -- the
	// ceiling scales with the body so the trailing chain fits during dives
	if f.Pos.Y < f.bodyLen*0.42+20 {
		addForce(v2(0, maxSp), 1.1)
	}
	// a gentle altitude pull at the scalare cruise height
	if off := w.H*0.36 - f.Pos.Y; absF(off) > 200 {
		addForce(v2(0, off*0.25), 0.10)
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
	return acc
}

// carveCurlTurn floors the speed mid-curl (G59): the 180 degree glass turn
// is carved forward — the scalare never stalls inside it or backs out.
func (f *Fish) carveCurlTurn(maxSp float64) {
	if sp := hyp2(f.Vel); sp < maxSp*0.5 {
		dir := f.Vel
		if sp < 0.01 {
			dir = v2(f.cruise, 0)
		}
		f.Vel = mulS(norm2(dir), maxSp*0.5)
	}
}
