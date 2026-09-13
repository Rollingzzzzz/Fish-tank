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
