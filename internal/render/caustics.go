// v1.1 (G75): the sun's fingers on the bed — two drifting wave trains
// interfere into a light dapple that rides the shared relief curve and
// settles into the grain with depth. The dapple is normal-alpha warm light
// on the matte bed (F16: no additive glow), fades to a moon whisper at deep
// night, and never leaves the sand strip.
package render

import (
	"math"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/hajimehoshi/ebiten/v2"
)

var cCaustic = hexRGBA("#fff3c9") // warm sunlight on grain

// Caustic01 is the interference field: two angled wave trains, squared for
// sharp caustic nodes, phase-breathing so the pattern never loops visibly.
// Output is strictly 0..1.
func Caustic01(x, y, t float64) float64 {
	a := 0.5 + 0.5*sin(x*0.020+y*0.052+t*0.55)
	b := 0.5 + 0.5*sin(x*0.033-y*0.037+t*0.83+sin(t*0.23)*0.8)
	return a * a * b * b
}

// causticGain is the day law: full sun by day, CausticNightK of it at deep
// night — the moon keeps a whisper of the pattern.
func causticGain(night float64) float64 {
	return 1 - (1-contract.CausticNightK)*clampF(night, 0, 1)
}

// causticRows are the dapple depth bands below the local dune surface: each
// row rides the relief curve and fades as it settles into the grain.
var causticRows = [4]struct{ d, f float64 }{
	{3, 1.0}, {11, 0.66}, {20, 0.42}, {30, 0.24},
}

// DrawSandCaustics paints the dapple onto the bed — one mesh, one draw call.
// Columns blend their edge samples so the 20 px grid never reads as stripes.
func DrawSandCaustics(dst *ebiten.Image, w, h, t, night float64) {
	if dst == nil || w <= 0 || h <= 0 {
		return
	}
	gain := contract.CausticAmp * causticGain(night)
	if gain <= 0.004 {
		return
	}
	m := &mesh{}
	for x := 0.0; x < w; x += contract.CausticStepPx {
		x2 := math.Min(x+contract.CausticStepPx, w)
		for _, r := range causticRows {
			aL := gain * Caustic01(x, r.d, t) * r.f
			aR := gain * Caustic01(x2, r.d, t) * r.f
			if aL < 0.006 && aR < 0.006 {
				continue
			}
			sL := contract.SandSurfaceY(h, x) + r.d
			sR := contract.SandSurfaceY(h, x2) + r.d
			cLT := withA(cCaustic, uint8(clampF(aL, 0, 1)*255))
			cRT := withA(cCaustic, uint8(clampF(aR, 0, 1)*255))
			cLB := withA(cCaustic, uint8(clampF(aL*0.45, 0, 1)*255))
			cRB := withA(cCaustic, uint8(clampF(aR*0.45, 0, 1)*255))
			m.quad(v2(x, sL), v2(x2, sR), v2(x2, sR+9), v2(x, sL+9), cLT, cRT, cRB, cLB)
		}
	}
	m.draw(dst, false)
}
