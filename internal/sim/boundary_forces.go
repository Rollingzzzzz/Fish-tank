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
