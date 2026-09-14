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
// G79 glass glide: a force-only wall push could never turn a nose-first
// hunter at the pane — the push is anti-parallel to the body axis, it
// changes SPEED not direction, and the cruise floor erased the speed
// change every frame (measured: a pair pinned at x=0.00, heading frozen at
// π for minutes). When the axis points at a pane inside the lookahead, the
// pin itself yaws the heading toward the wall tangent — along the glass,
// the way a real shark slides past the pane — so steering forces regain a
// perpendicular component and the tour takes over again.
func (f *Fish) constrainForward(dt, maxSp float64, w *World) {
	sp := hyp2(f.Vel)
	if sp > 6 {
		va := math.Atan2(f.Vel.Y, f.Vel.X)
		da := math.Mod(va-f.headingA+3.14159, 6.28318) - 3.14159
		f.headingA += clampF(da, -contract.SharkTurnRate*dt, contract.SharkTurnRate*dt)
	}
	// the glass glide: project the nose one glide-length ahead and yaw to
	// the tangent before the pane can pin the body
	const edge = 52.0
	look := maxF(sp*0.8, 30)
	nx, ny := cos(f.headingA), sin(f.headingA)
	xAhead, yAhead := f.Pos.X+nx*look, f.Pos.Y+ny*look
	var tgt float64
	glide := false
	switch {
	case xAhead < edge && nx < 0: // left pane
		glide = true
		tgt = -0.35
		if f.Pos.Y > w.H*0.55 {
			tgt = 0.35
		}
	case xAhead > w.W-edge && nx > 0: // right pane
		glide = true
		tgt = math.Pi + 0.35
		if f.Pos.Y > w.H*0.55 {
			tgt = math.Pi - 0.35
		}
	case yAhead < edge*0.7 && ny < 0: // surface
		glide = true
		tgt = 1.25
		if f.Pos.X > w.W*0.5 {
			tgt = math.Pi - 1.25
		}
	case yAhead > w.H-edge*0.6 && ny > 0: // floor side
		glide = true
		tgt = -1.25
		if f.Pos.X > w.W*0.5 {
			tgt = -(math.Pi - 1.25)
		}
	}
	if glide {
		da := math.Mod(tgt-f.headingA+3.14159, 6.28318) - 3.14159
		f.headingA += clampF(da, -contract.SharkTurnRate*1.6*dt, contract.SharkTurnRate*1.6*dt)
	}
	if sp < maxSp*0.65 {
		// G81: the cruise floor is approached (3/s), not snapped — an
		// instant lift read as a collision kick when the pin shaved speed
		sp += minF(maxSp*0.65-sp, maxSp*0.65*3*dt)
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

// tickSharkExhale breathes the pair (G76): every few gill beats the hunter
// releases a puff of micro-bubbles where the gills sit — the water reads as
// water, and the rendered gill slits get a physical echo.
func (w *World) tickSharkExhale(dt float64) {
	for _, f := range w.fishes {
		if f.Sp.Role != contract.RoleShark || f.Dying {
			continue
		}
		if f.exhaleT == 0 {
			f.exhaleT = contract.SharkExhaleMin + w.rng.Float64()*(contract.SharkExhaleMax-contract.SharkExhaleMin)
		}
		f.exhaleT -= dt
		if f.exhaleT > 0 || len(w.bubbles) >= maxBubbles {
			continue
		}
		f.exhaleT = contract.SharkExhaleMin + w.rng.Float64()*(contract.SharkExhaleMax-contract.SharkExhaleMin)
		gx := f.Pos.X + cos(f.headingA)*f.bodyLen*0.14
		gy := f.Pos.Y + sin(f.headingA)*f.bodyLen*0.14
		n := 3 + int(w.rng.Float64()*3)
		for k := 0; k < n && len(w.bubbles) < maxBubbles; k++ {
			w.bubbles = append(w.bubbles, Bubble{
				Pos:    v2(gx+(w.rng.Float64()-0.5)*f.bodyLen*0.10, gy+(w.rng.Float64()-0.5)*f.bodyLen*0.08),
				R:      1.1 + w.rng.Float64()*0.8, // G78: floor 1.1 — gill breaths stay readable
				Speed:  34 + w.rng.Float64()*30,
				Wobble: w.rng.Float64() * 6.283,
				Seed:   w.rng.Float64(),
			})
		}
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
