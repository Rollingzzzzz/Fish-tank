// v1.1 (G83): the strike shock — the moment a giant commits to an attack,
// the water around the strike point opens up. Every small fish inside the
// radius bolts from the point (a startle, with its natural decay back to
// calm), and a soft pressure ring rides the render for two thirds of a
// second. The viewer reads one thing: something big just moved.
package sim

import (
	"math"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// Shock is one expanding pressure ring (runtime-only, never persisted).
type Shock struct {
	Pos contract.Vec2
	T   float64 // seconds since the strike
}

// strikeShock scatters the small fish around pt and stamps a shock ring.
// The radius is the reach of the event: StrikeScareR for a dive,
// StrikeScareR2 for the swallow itself.
func (w *World) strikeShock(pt contract.Vec2, radius float64, log string) {
	for _, f := range w.fishes {
		if f.Sp.Role != contract.RoleNormal || f.Dying {
			continue
		}
		d := sub(f.Pos, pt)
		if l := hyp2(d); l < radius && l > 1 {
			// the flinch: the body snaps a half-turn toward open water and
			// takes a real kick — the parting must read AT ONCE, then the
			// decaying startle impulse carries it home
			away := math.Atan2(d.Y, d.X)
			da := math.Mod(away-f.headingA+3.14159, 6.28318) - 3.14159
			f.headingA += clampF(da, -1.1, 1.1)
			f.Vel = add(f.Vel, mulS(d, 55/l))
			flee := contract.MaxForce * (0.55 + 0.45*contract.Clamp(f.Sp.Behavior.Skittish, 0, 1))
			f.flee(pt.X, pt.Y, flee)
		}
	}
	w.shocks = append(w.shocks, Shock{Pos: pt, T: 0})
	if log != "" {
		w.logf("nature", log)
	}
}

// tickShocks ages the pressure rings out.
func (w *World) tickShocks(dt float64) {
	kept := w.shocks[:0]
	for i := range w.shocks {
		w.shocks[i].T += dt
		if w.shocks[i].T < contract.ShockRingSec {
			kept = append(kept, w.shocks[i])
		}
	}
	w.shocks = kept
}

// Shocks exposes the live pressure rings for the renderer.
func (w *World) Shocks() []Shock { return w.shocks }
