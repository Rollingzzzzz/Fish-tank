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
	follow := 2.0 // G84: a heavier nose — the ease takes its time
	if f.seekBonus > 1.01 {
		follow = 6.5
	}
	if v := hyp2(f.Vel); v > 1 { // even a slow drift is already a heading
		// G80: the ease rides the fin flow and the convoy arc's own
		// curvature — a near-stalled giant swings its nose gently, it does
		// not pirouette. A burst needs its nose NOW.
		if f.seekBonus <= 1.01 {
			follow = minF(follow, v/maxF(f.bodyLen*0.08, 8))
		}
		va := math.Atan2(f.Vel.Y, f.Vel.X)
		da := math.Mod(va-f.headingA+3.14159, 6.28318) - 3.14159
		// G87: the head is the pivot of a very long lever — a chattering
		// correction of 0.015 rad/frame swung the tail 6-18 px/frame while
		// the eye stood still. The coherence gate (the school's own from
		// G80): a correction sign that flip-flops is noise and turns a
		// ponderous nose at less than a third of the rate.
		if da*f.lastDa < 0 {
			f.noiseT = 0.35
		}
		f.lastDa = da
		f.noiseT = maxF(0, f.noiseT-dt)
		if f.noiseT > 0 && f.seekBonus <= 1.01 {
			follow *= 0.3
		}
		f.headingA += clampF(da, -follow*dt, follow*dt)
		// G87: ONE master. The velocity is rebuilt along the heading, the
		// school's own proven pin (G63) — magnitude preserved exactly (no
		// speed cuts, no collision read), the nose law holds by
		// construction, and the lateral/pitch tug-of-war that rocked the
		// body around a still eye is structurally gone: the level law
		// below bounds the pitch, the rebuilt velocity inherits it, no
		// second authority fights back.
		f.Vel = mulS(v2(math.Cos(f.headingA), math.Sin(f.headingA)), v)
		// the mixed acceleration may not push the nose backwards either —
		// integration happens after this pass
		ax, ay := math.Cos(f.headingA), math.Sin(f.headingA)
		if back := acc.X*ax + acc.Y*ay; back < 0 {
			acc.X -= ax * back
			acc.Y -= ay * back
		}
	}
	// G87: the giants fly LEVEL. Outside strikes and arcs a steeply
	// pitched nose is rotated back toward the horizon at a bounded rate
	// (the wall-glide idea, vertical edition, with hysteresis so it never
	// chatters at the bound) — a pitched giant parks its 448 px trailing
	// cone through the ceiling and the frame clamp drags the tail along
	// the glass while the eye stands still. With the velocity rebuilt
	// along the heading this law is the ONE vertical authority at cruise.
	f.pitchT = maxF(0, f.pitchT-dt)
	if f.seekBonus <= 1.01 && f.turning <= 0 {
		ay := math.Sin(f.headingA)
		const lim, arm = 0.38, 0.46 // arm at ~27°, settle at ~22°
		if ay > arm || ay < -arm || (f.pitchT > 0 && (ay > lim || ay < -lim)) {
			f.pitchT = 0.5
			ax := math.Cos(f.headingA)
			base := math.Asin(lim)
			var tgt float64
			switch {
			case ax >= 0 && ay > 0:
				tgt = base
			case ax >= 0:
				tgt = -base
			case ay > 0:
				tgt = math.Pi - base
			default:
				tgt = -(math.Pi - base)
			}
			da := math.Mod(tgt-f.headingA+3.14159, 6.28318) - 3.14159
			// authoritative: applied after the follow and faster than it —
			// the level law outvotes the chase, or a pitched acc spiral
			// drags the pair steep anyway (measured vy 21.5)
			f.headingA += clampF(da, -2.5*dt, 2.5*dt)
		}
	}
}
