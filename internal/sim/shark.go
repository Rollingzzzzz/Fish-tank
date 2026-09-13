// v1.1: the hammerhead pair (G50) — a resident hunter that roams the whole
// tank, snapped to the same rules as the rest of the ambient life: outside
// popCap, breeding and aging, immune to the titan's hunt, capped at
// SharkMax, and never faster than the Chosen at any hour (supremacy test).
// The sharks persist in snapshots — they are residents, not visits.
package sim

import (
	"math"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// sharkSpecies finds the core-seeded hammerhead (nil when absent).
func (w *World) sharkSpecies() *contract.Species {
	for _, s := range w.species {
		if s.Role == contract.RoleShark {
			return s
		}
	}
	return nil
}

// sharkCount reports live hammerheads.
func (w *World) sharkCount() int {
	n := 0
	for _, f := range w.fishes {
		if f.Sp.Role == contract.RoleShark && !f.Dying {
			n++
		}
	}
	return n
}

// ensureSharks keeps the resident pair whole (G50): whatever sweeps them
// away, the hunter returns — the mirror of ensureChosen, capped at SharkMax.
func (w *World) ensureSharks() {
	sp := w.sharkSpecies()
	if sp == nil {
		return
	}
	for w.sharkCount() < contract.SharkMax {
		p := v2(w.W*(0.25+w.rng.Float64()*0.5), w.H*(0.30+w.rng.Float64()*0.35))
		f := newFish(sp, w.rng.Int63(), p, 6, w.nextID()) // frozen adult
		f.headingA = math.Atan2(w.H/2-p.Y, w.W/2-p.X)     // born facing the open water
		f.Vel = mulS(v2(cos(f.headingA), sin(f.headingA)), 20)
		// the chain is stretched to face where she swims — never tail-first
		for j := range f.Spine {
			f.Spine[j] = v2(p.X-float64(j)*f.segLen*cos(f.headingA),
				p.Y-float64(j)*f.segLen*sin(f.headingA))
		}
		w.fishes = append(w.fishes, f)
		w.logf("nature", "a hammerhead glides out of the blue")
	}
}

// constrainForward pins the hammerhead to its own nose (G62): the velocity
// rides the body axis and the axis itself swings at most SharkTurnRate —
// the steering mix can steer the head around, but never slide the body
// tail-first. A target behind the back turns into a carve, not a reverse.
// The hunter is also always under way (G50): the pin may shave speed while
// forces fight the turn, so it is floored at a dignified cruise.
func (f *Fish) constrainForward(dt, maxSp float64) {
	sp := hyp2(f.Vel)
	if sp > 6 {
		va := math.Atan2(f.Vel.Y, f.Vel.X)
		da := math.Mod(va-f.headingA+3.14159, 6.28318) - 3.14159
		f.headingA += clampF(da, -contract.SharkTurnRate*dt, contract.SharkTurnRate*dt)
	}
	if sp < maxSp*0.65 {
		sp = maxSp * 0.65
	}
	f.Vel = mulS(v2(cos(f.headingA), sin(f.headingA)), sp)
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

// titanStartConvoyTurn flips the caravan as ONE (G67): the leader's glass
// call arms every member's arc at the same instant — each fish sweeps its
// own parallel half circle from where it swims, so the five arrive on the
// opposite sweep still clustered, never strung out.
func (w *World) titanStartConvoyTurn() {
	for _, f := range w.fishes {
		if f.Sp.Role != contract.RoleTitan || f.Dying {
			continue
		}
		f.turnH0 = f.headingA
		f.cruise = -f.cruise
		// the pod turns UP by default (against the surface light, G67): the
		// radius shrinks to fit the top margin, and only a truly closed top
		// falls back to a down-curl — repeated down-curls walked the pod
		// into the bottom band and its tail into the corner
		f.turnS = f.cruise
		room := f.Pos.Y - (f.bodyLen*0.30 + 6) - 15
		if room < f.bodyLen*0.12 {
			f.turnS = -f.cruise
		}
		f.turnT = contract.TitanTurnWindow + 60
		f.turning = contract.TitanTurnWindow
	}
	w.logf("nature", "the silver elders wheel as one -- five shadows, one turn")
}
