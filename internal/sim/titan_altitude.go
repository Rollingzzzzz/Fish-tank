// v1.1 split (line ceiling): the pod's vertical life — the altitude
// wander, the level damper and the water's floor all live here.
package sim

import (
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// titanAltitudeWander is the G86 glide law: the sweep is a slow wandering
// glide, not a rail — the LEADER draws the pod's altitude every ~20-40 s
// (high under the light, the broad water, or the rare dip) and each
// member draws a small PERSONAL offset around her, so the convoy reads as
// one loose body that breathes — never lockstep, never a scattered fan
// (the fully independent draws spread the pod across ~170 px of height).
// The damper bleeds vertical drift (a giant does not bob; a strike owns
// the column), and the water's FLOOR under the red line escalates the
// climb back with depth — the cruise never lives below it.
func (f *Fish) titanAltitudeWander(w *World, dt, maxSp float64, addForce func(contract.Vec2, float64)) {
	// G69: the sweep is a slow wandering glide, not a rail -- each elder
	// draws a personal altitude every ~20-40 s, anywhere from the surface
	// light down to just above the nest level, and eases toward it. The
	// members draw independently (small phase drift), so the pod breathes
	// instead of marching in lockstep.
	f.altT -= dt
	if f.altT <= 0 {
		lo := (f.bodyLen*0.30+6)/w.H + 0.03
		// G86: the elders live WELL ABOVE the red line — the LEADER draws
		// the pod's water (high under the light, the broad water, or the
		// rare dip) and each member draws a small PERSONAL offset around
		// her, so the convoy reads as one loose body that breathes — the
		// fully independent draws spread the pod across 170 px of height
		// and read as a broken convoy.
		if lead := w.titanGiant(); lead != nil && lead != f {
			f.altY = clampF(lead.altY+(f.rng.Float64()-0.5)*0.10, lo, contract.TitanAltDip)
		} else {
			r := f.rng.Float64()
			switch {
			case r < 0.40:
				f.altY = lo + f.rng.Float64()*(0.30-lo)
			case r < 0.88:
				f.altY = 0.30 + f.rng.Float64()*(contract.TitanAltMax-0.30)
			default:
				f.altY = contract.TitanAltMax + f.rng.Float64()*(contract.TitanAltDip-contract.TitanAltMax)
			}
		}
		f.altT = 18 + f.rng.Float64()*22
	}
	// level swimming stays gentle: vertical drift is softly damped -- the
	// body may glide up or down along its sweep, but not ballistically.
	// Through a strike the damper stands aside: the lunge keeps its full
	// 6x+ burst (G41) on both axes
	// G82: a giant does not bob — the vertical drift bleeds at TitanVyDamp
	// (probe: |vy| p95 was 20.7 px/s, the same size as the whole cruise —
	// the "weird Y-axis turns" read). A strike still owns the water column.
	dampW := contract.TitanVyDamp
	if f.seekBonus > 1.01 {
		dampW = 0.12
	}
	// far from the drawn altitude the climb/dive is deliberate — the damp
	// relaxes so the glide keeps its momentum (full damp near the band is
	// what levels the body out; crush it everywhere and the sweep becomes
	// a rail on one height)
	if off := w.H*f.altY - f.Pos.Y; absF(off) > 60 {
		dampW *= 0.35
	}
	addForce(v2(0, -f.Vel.Y*dampW), dampW)
}
