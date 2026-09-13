// v1.1: the sand bed (G54) — yellowish-white grains along the floor line,
// reaching up to the Chosen's nest boundary and nowhere past it. Fish that
// swim close stir the grains: they scatter sideways and lift, then the
// field relaxes back to perfectly flat on its own. Purely visual state —
// never persisted; a fresh boot opens with a calm, level bed.
package sim

import (
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// Sand returns the floor disturbance field: one offset per cell, 0 = level.
// The renderer spreads one grain cluster per cell across the floor line.
func (w *World) Sand() []float64 { return w.sand }

// tickSand stirs the bed under nearby fish and then lets it settle: a
// diffusion pass flattens piles sideways and a slow damping eases every
// offset back to zero (the natural re-leveling the viewer reads as calm).
func (w *World) tickSand(dt float64) {
	if w.sand == nil {
		w.sand = make([]float64, int(w.W/contract.SandCellPx)+1)
	}
	floorY := w.H * contract.FloorLineFrac
	// stir: fish skimming the floor scatter grains from under their body
	for _, f := range w.fishes {
		if f.Dying || f.Pos.Y < floorY-55 {
			continue
		}
		speed := hyp2(f.Vel)
		if speed < 4 {
			continue // a parked fish barely rustles the bed
		}
		kick := clampF(speed/80, 0.15, 1.4) * dt * 14
		ci := clampI(int(f.Pos.X/w.W*float64(len(w.sand))), 1, len(w.sand)-2)
		w.sand[ci] -= kick
		w.sand[ci-1] += kick * 0.45
		w.sand[ci+1] += kick * 0.45
		if w.rng.Float64() < 0.10 {
			w.sand[ci+w.rng.Intn(3)-1] += kick * 0.3
			w.burst(v2(f.Pos.X, floorY+2), "#efe4c4", 2)
		}
	}
	// settle: sideways diffusion + slow return to the flat baseline
	k := minF(1, dt*contract.SandDiffuse)
	for i := range w.sand {
		l, r := 0.0, 0.0
		if i > 0 {
			l = w.sand[i-1]
		}
		if i < len(w.sand)-1 {
			r = w.sand[i+1]
		}
		w.sand[i] += ((l+r)*0.5 - w.sand[i]) * k
		w.sand[i] *= 1 - minF(1, dt*contract.SandSettleRate)
		w.sand[i] = clampF(w.sand[i], -4, 10)
	}
}
