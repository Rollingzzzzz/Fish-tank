// v1.1 (split from titan.go, line ceiling): the pod's heading alignment —
// the follow that eases the nose toward the velocity and the blind-side
// reflection that keeps a scalare nose-first without ever deleting speed.
package sim

import (
	"math"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// titanAlign is the G59/G67 pose pass: the heading eases toward the
// velocity (at burst pace through a strike), and whatever the mixed forces
// did, the velocity's blind-side component is REFLECTED forward (G81: the
// old deletion threw away up to 145 px/s in one frame — the collision
// read; the reflection preserves speed, lands the nose law instantly and
// leaves the lateral drift the staggered formation rides on).
func (f *Fish) titanAlign(dt, maxSp float64, acc *contract.Vec2) {
	follow := 2.5 // G84: a heavier nose — the ease takes its time (2.5 scattered the post-arc re-align)
	if f.seekBonus > 1.01 {
		follow = 6.5
	}
	if v := hyp2(f.Vel); v > 1 { // even a slow drift is already a heading
		// G80: the ease rides the fin flow and the convoy arc's own
		// curvature (a quarter body-length radius) — a near-stalled giant
		// swings its nose gently, it does not pirouette. A burst needs its
		// nose NOW: strikes and scare bolts are exempt from the cap.
		if f.seekBonus <= 1.01 {
			follow = minF(follow, v/maxF(f.bodyLen*0.08, 8))
		}
		va := math.Atan2(f.Vel.Y, f.Vel.X)
		da := math.Mod(va-f.headingA+3.14159, 6.28318) - 3.14159
		f.headingA += clampF(da, -follow*dt, follow*dt)
		ax, ay := math.Cos(f.headingA), math.Sin(f.headingA)
		if back := f.Vel.X*ax + f.Vel.Y*ay; back < 0 {
			f.Vel.X -= 2 * ax * back
			f.Vel.Y -= 2 * ay * back
		}
		// the mixed acceleration may not push the nose backwards either —
		// integration happens after this pass
		if back := acc.X*ax + acc.Y*ay; back < 0 {
			acc.X -= ax * back
			acc.Y -= ay * back
		}
	}
	// G82/G84: the level-flight envelope — a scalare is a disc; it slips
	// vertical SLOWLY. Outside a strike the vertical speed eases back under
	// 55 % of cruise at 6/s (an instant clamp was itself a hard cut — the
	// last stiff edge), so the body reads as a gliding plate, never a
	// bobbing tube and never a snapped one.
	if f.seekBonus <= 1.01 {
		if cap := maxSp * 0.55; f.Vel.Y > cap {
			f.Vel.Y -= minF(f.Vel.Y-cap, (f.Vel.Y-cap)*6*dt+6*dt)
		} else if f.Vel.Y < -cap {
			f.Vel.Y += minF(-cap-f.Vel.Y, (-cap-f.Vel.Y)*6*dt+6*dt)
		}
	}
}
