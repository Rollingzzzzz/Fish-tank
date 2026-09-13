// v1.1 G73: the Chosen's rare superpower — a wormhole pass. Every few
// minutes she opens a small portal where she swims, slips through it, and
// steps out of a second portal far across the tank. The pass is fully
// staged (opening, entering, exiting) so it reads as magic, never as a
// teleport glitch. Runtime-only state: never persisted.
package sim

import (
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// PortalFrom reports where her entry portal currently is (nil when idle).
func (f *Fish) PortalFrom() *contract.Vec2 {
	if f.portalPh == 0 {
		return nil
	}
	return &f.scarePt // reused scratch: the entry point is frozen at open
}

// PortalFade01 reports her body fade for the renderer (0 solid .. 1 gone).
func (f *Fish) PortalFade01() float64 {
	switch f.portalPh {
	case 2:
		return clampF(f.portalT/0.6, 0, 1)
	case 3:
		return 1 - clampF(f.portalT/0.6, 0, 1)
	}
	return 0
}

// tickChosenPortal advances the wormhole state machine.
func (w *World) tickChosenPortal(f *Fish, dt float64) {
	if f.Dying {
		f.portalPh = 0
		return
	}
	f.portalT += dt
	switch f.portalPh {
	case 0:
		f.portalCD -= dt
		if f.portalCD <= 0 {
			// pick a far exit: at least half a tank away, inside the water
			for try := 0; try < 12; try++ {
				tgt := v2(w.W*(0.1+w.rng.Float64()*0.8), w.H*(0.15+w.rng.Float64()*0.6))
				if d := hyp2(sub(tgt, f.Pos)); d > w.W*0.35 {
					f.restTarget = &tgt
					f.scarePt = f.Pos // freeze the entry point
					f.portalPh = 1
					f.portalT = 0
					f.portalCD = contract.PortalGapMin + w.rng.Float64()*(contract.PortalGapMax-contract.PortalGapMin)
					w.logf("magic", "the eternal one parts the water -- a door opens")
					return
				}
			}
			f.portalCD = 30 // nowhere far enough; try again shortly
		}
	case 1: // opening (0.9 s)
		if f.portalT >= 0.9 {
			f.portalPh = 2
			f.portalT = 0
		}
	case 2: // entering (0.6 s): drawn toward the door, fading
		d := sub(f.scarePt, f.Pos)
		if l := hyp2(d); l > 4 {
			f.Vel = mulS(d, minF(90, l*3)/maxF(l, 1))
		}
		if f.portalT >= 0.6 {
			f.Pos = *f.restTarget
			f.Spine[0] = f.Pos
			for j := 1; j < len(f.Spine); j++ {
				f.Spine[j] = v2(f.Pos.X-float64(j)*f.segLen, f.Pos.Y)
			}
			f.Vel = v2(0, 0)
			f.portalPh = 3
			f.portalT = 0
		}
	case 3: // exiting (0.6 s): step out, fade back in
		if f.portalT >= 0.6 {
			f.portalPh = 0
			f.portalT = 0
			f.restTarget = nil
		}
	}
}

// PortalPhase exposes the wormhole phase for the renderer (G73).
func (f *Fish) PortalPhase() int { return f.portalPh }

// PortalClock exposes the phase clock for the renderer (G73).
func (f *Fish) PortalClock() float64 { return f.portalT }

// night01 reports the current night factor (0 day .. 1 deep night) for
// hour-aware test measurements.
func (w *World) night01() float64 { return 1 - w.dayFactor() }
