// v1.2 split (line ceiling): the frame boundary passes — the impenetrable
// tank bounds, and the whole-body + rope-slide passes that keep every
// drawn joint inside the view (G66) without pins or hops (G90/G91).
package sim

import (
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// applyFrameBounds is the impenetrable tank bounds (v1.0) plus the G66
// body margin: a fish can never leave the water, and the big residents
// keep their tall frames below the top edge — the drawn body must fit.
// G81: the inbound velocity component now decays (10/s) instead of being
// zeroed in one frame — the instant kill read as collision-response
// physics; the wall itself still stops the position dead, the easing only
// governs how the speed reads after contact.
func (f *Fish) applyFrameBounds(w *World, dt float64) {
	soft := maxF(0, 1-10*dt)
	if f.Pos.X < 8 {
		f.Pos.X = 8
		if f.Vel.X < 0 {
			f.Vel.X *= soft
		}
	}
	if f.Pos.X > w.W-8 {
		f.Pos.X = w.W - 8
		if f.Vel.X > 0 {
			f.Vel.X *= soft
		}
	}
	if f.Pos.Y < 8 {
		f.Pos.Y = 8
		if f.Vel.Y < 0 {
			f.Vel.Y *= soft
		}
	}
	if f.Pos.Y > w.H-8 {
		f.Pos.Y = w.H - 8
		if f.Vel.Y > 0 {
			f.Vel.Y *= soft
		}
	}
	if f.Sp.Role != contract.RoleTitan && f.Sp.Role != contract.RoleShark {
		return // short chains fit once the head is inside
	}
	// the tall frames ride below the top edge; the trailing cone's worst
	// dive excursion is caught by the final canvas clamp, so the hard
	// margin keeps only the body proper inside (G66/G67 — leaving room
	// for the convoy's up-curl to exist at cruise height)
	myTop := f.bodyLen*0.30 + 6
	if f.Pos.Y < myTop {
		f.Pos.Y = myTop
		if f.Vel.Y < 0 {
			f.Vel.Y *= soft
		}
	}
}

// clampBodyInFrame is the G66 guarantee pass: every spine point of a big
// body stays inside the canvas, every frame. G87: when the trailing cone
// pokes past an edge the WHOLE FISH shifts rigidly (a few px per frame)
// instead of the tail being pinned while the head keeps cruising — the
// pin-and-slide read as a snag with the body oscillating around the still
// eye. The fish glides level under the ceiling; the clamp itself remains
// the last invisible guarantee.
func (f *Fish) clampBodyInFrame(w *World) {
	shiftX, shiftY := 0.0, 0.0
	for i := range f.Spine {
		if f.Spine[i].X < 3 {
			shiftX = maxF(shiftX, 3-f.Spine[i].X)
		}
		if f.Spine[i].X > w.W-3 {
			shiftX = minF(shiftX, w.W-3-f.Spine[i].X)
		}
		if f.Spine[i].Y < 3 {
			shiftY = maxF(shiftY, 3-f.Spine[i].Y)
		}
		if f.Spine[i].Y > w.H-3 {
			shiftY = minF(shiftY, w.H-3-f.Spine[i].Y)
		}
	}
	// the rigid shift itself is bounded per frame — an unbounded jump is
	// just another teleport wearing a fix's clothes. G90: 5 px/frame was
	// still a visible hop at school scale; the cap now sits at
	// BodyShiftCapPx and the edge deficit closes over a few gliding frames.
	shiftX = clampF(shiftX, -contract.BodyShiftCapPx, contract.BodyShiftCapPx)
	shiftY = clampF(shiftY, -contract.BodyShiftCapPx, contract.BodyShiftCapPx)
	if shiftX != 0 || shiftY != 0 {
		for i := range f.Spine {
			f.Spine[i].X += shiftX
			f.Spine[i].Y += shiftY
		}
		f.Pos.X += shiftX
		f.Spine[0] = f.Pos
	}
	// G90: the final pass is a ROPE SLIDE, not a pin. A point pressed past
	// the pane used to be clamped inward outright — a feeding frenzy at the
	// glass folded chains and the next re-lay snapped them back out (the
	// 9-18 px neck hops). Sliding along the pane at segment length reads as
	// a rope dragged over glass; the last op is always the frame clamp, so
	// the G66 in-view guarantee stands.
	for i := 1; i < len(f.Spine); i++ {
		p := f.Spine[i]
		p.X = clampF(p.X, 3, w.W-3)
		p.Y = clampF(p.Y, 3, w.H-3)
		for iter := 0; iter < 3; iter++ {
			d := sub(p, f.Spine[i-1])
			if l := hyp2(d); l > 1e-6 {
				p = add(f.Spine[i-1], mulS(d, f.segLen/l))
			}
			p.X = clampF(p.X, 3, w.W-3)
			p.Y = clampF(p.Y, 3, w.H-3)
		}
		f.Spine[i] = p
	}
}
