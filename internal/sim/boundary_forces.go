// v1.1 (split from fish_steering.go, line ceiling): the boundary family —
// the aura repulsion that keeps everyone but the Chosen out of her circle,
// and the soft wall margins of the tank itself.
package sim

import (
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// auraRepulsion is N3: nobody but the Chosen enters the nest aura; a long
// fish turns away early enough that no part of it crosses the line.
func (f *Fish) auraRepulsion(w *World, maxSp float64, addForce func(contract.Vec2, float64)) {
	for _, z := range w.zones {
		if z.Owner != "chosen" || f.Sp.Role == contract.RoleChosen {
			continue
		}
		d := sub(f.Pos, z.Center)
		r := z.Radius + 40 + f.bodyLen*0.45
		if l := hyp2(d); l < r && l > 1 {
			urgency := 1 + 2*(1-l/r)
			addForce(mulS(d, maxSp*urgency/l), 1.8)
		}
	}
}

// zoneLookahead is G92: the nose probes one glide ahead (shark-wall
// pattern); if the probe lands inside her band the fish steers to SLIDE
// AROUND the rim — a tangential glide plus a mild outward push — instead of
// driving into the circle and leaving the exit to the positional drain.
// Pure repulsion (auraRepulsion) is a position field: a committed fish only
// feels a sideways shove once inside the band. The probe turns the
// encounter into a deflection before contact.
func (f *Fish) zoneLookahead(w *World, maxSp float64, addForce func(contract.Vec2, float64)) {
	if f.Sp.Role == contract.RoleChosen {
		return
	}
	v := hyp2(f.Vel)
	n := f.Vel
	if v < 4 {
		n = v2(cos(f.headingA), sin(f.headingA)) // slow: probe off the nose axis
		v = maxSp * 0.5
	} else {
		n = mulS(n, 1/v)
	}
	look := maxF(v*contract.ZoneLookFrac, contract.ZoneLookMinPx)
	px, py := f.Pos.X+n.X*look, f.Pos.Y+n.Y*look
	for _, z := range w.zones {
		if z.Owner != "chosen" {
			continue
		}
		d := sub(v2(px, py), z.Center)
		r := z.Radius + f.bodyLen*0.5 + 24
		if l := hyp2(d); l < r && l > 1 {
			out := mulS(d, 1/l)
			// slide around the rim keeping the current approach side: the
			// tangent sign follows which way the nose currently leans
			side := 1.0
			if n.X*out.Y-n.Y*out.X < 0 {
				side = -1.0
			}
			tang := v2(-out.Y*side, out.X*side)
			urgency := 0.6 + 1.4*(1-l/r)
			desired := add(mulS(out, maxSp*urgency), mulS(tang, maxSp))
			addForce(desired, 2.0)
		}
	}
}

// wallMargins pushes cruisers softly off the tank edges.
func (f *Fish) wallMargins(w *World, maxSp float64, addForce func(contract.Vec2, float64)) {
	const mgn = 60.0
	if f.Pos.X < mgn {
		addForce(v2(maxSp, 0), 1.2)
	}
	if f.Pos.X > w.W-mgn {
		addForce(v2(-maxSp, 0), 1.2)
	}
	if f.Pos.Y < mgn*0.7 {
		addForce(v2(0, maxSp), 1.2)
	}
	if f.Pos.Y > w.H-mgn*0.6 {
		addForce(v2(0, -maxSp), 1.2)
	}
}
