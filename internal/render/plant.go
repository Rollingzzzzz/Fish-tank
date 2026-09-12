// G1.3: plant renderer — alien ribbon fronds with layered sway, multi-stop
// gradients and bubble tips (art direction §3). F17: all additive glow was
// removed — plants stay lush but matte; neon belongs to the Chosen alone.
package render

import (
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/hajimehoshi/ebiten/v2"
)

// plantHash is a tiny deterministic per-frond variation source.
func plantHash(i int) float64 {
	x := float64(i+1) * 0.6180339887
	return x - float64(int(x))
}

// multiStop interpolates a color along the design's color list (u 0..1).
func multiStop(colors []string, u float64, alpha uint8) (colorOut colorRGBA) {
	n := len(colors)
	switch {
	case n == 0:
		return colorRGBA{R: 255, G: 255, B: 255, A: alpha}
	case n == 1 || u <= 0:
		return withA(hexRGBA(colors[0]), alpha)
	case u >= 1:
		return withA(hexRGBA(colors[n-1]), alpha)
	}
	seg := u * float64(n-1)
	i := int(seg)
	if i >= n-1 {
		i = n - 2
	}
	return withA(lerpRGBA(hexRGBA(colors[i]), hexRGBA(colors[i+1]), seg-float64(i)), alpha)
}

// DrawPlant draws one plant whose base sits at (x, y). heightPx is the full
// frond height in px; t is global time; night is 0 day .. 1 night. glowDst
// and night are kept for the frozen v0.2 signature but unused since F17 —
// plants render matte, all additive glow removed.
// v0.3.7 (F30): the Kind field picks the silhouette — the classic ribbon,
// or the new feathery/silky archetypes (plant_feather.go).
func DrawPlant(dst, glowDst *ebiten.Image, p *contract.PlantDesign, x, y, heightPx, t, night float64) {
	drawPlantKind(dst, p, x, y, heightPx, t, 1)
}

// DrawPlantShaded is DrawPlant with a cool dim: shade 1 = unchanged,
// ~0.4 = a dark near-glass silhouette (the v0.3.8 foreground depth plane).
func DrawPlantShaded(dst, glowDst *ebiten.Image, p *contract.PlantDesign, x, y, heightPx, t, night, shade float64) {
	drawPlantKind(dst, p, x, y, heightPx, t, shade)
}

func drawPlantKind(dst *ebiten.Image, p *contract.PlantDesign, x, y, heightPx, t, shade float64) {
	if dst == nil || p == nil || len(p.Colors) < 1 || p.Fronds < 1 {
		return
	}
	switch p.Kind {
	case "feather":
		drawFeatherPlant(dst, p, x, y, heightPx, t, shade)
	case "silk":
		drawSilkPlant(dst, p, x, y, heightPx, t, shade)
	default:
		drawRibbonPlant(dst, p, x, y, heightPx, t, shade)
	}
}

// drawRibbonPlant is the v0.2 archetype: alien ribbon fronds with bubble tips.
func drawRibbonPlant(dst *ebiten.Image, p *contract.PlantDesign, x, y, heightPx, t, shade float64) {
	sway := contract.Clamp(p.Sway, 0, 1)
	curve := contract.Clamp(p.Curve, 0, 1)
	swayAmp := heightPx * (0.06 + 0.10*sway)
	swaySpeed := 0.55 + 1.1*sway
	spread := 12 * contract.Clamp(p.Width, 0.5, 1.5)
	samples := 32 // arc-length resampled below — low counts show polygon kinks (F1)

	for f := 0; f < p.Fronds; f++ {
		hv := plantHash(f)
		side := 1.0
		if f%2 == 1 {
			side = -1.0
		}
		h := heightPx * (0.72 + 0.55*hv)
		baseX := x + float64(f-(p.Fronds-1)/2)*spread + (hv-0.5)*6
		ph := hv * 6.283
		swayOff := sin(t*swaySpeed+ph) * swayAmp

		// quadratic bezier: base, control, tip — alien curve + live sway
		p0 := v2(baseX, y)
		p1 := v2(baseX+curve*h*0.18*side+swayOff*0.45, y-h*0.52)
		p2 := v2(baseX+curve*h*0.48*side+swayOff, y-h)

		// F1: cumulative-chord table maps arc length → parameter u, so
		// segment lengths stay even wherever the curve is steep. Sway already
		// lives in the control points, so the animation is untouched.
		const lutN = 64
		var lut [lutN + 1]float64
		lx, ly := p0.X, p0.Y
		for i := 1; i <= lutN; i++ {
			u := float64(i) / lutN
			cx, cy := bez(p0.X, p1.X, p2.X, u), bez(p0.Y, p1.Y, p2.Y, u)
			lut[i] = lut[i-1] + hyp(cx-lx, cy-ly)
			lx, ly = cx, cy
		}
		total := lut[lutN]

		var body mesh
		wBase := 11 * contract.Clamp(p.Width, 0.5, 1.5) * (0.75 + 0.5*hv)
		var prevL, prevR contract.Vec2
		var prevC colorRGBA
		for s := 0; s <= samples; s++ {
			// arc-length fraction → parameter u (bisection over the table)
			// arc-length fraction → parameter u (bisection over the table)
			target := total * float64(s) / float64(samples)
			lo, hi := 0, lutN
			for lo < hi {
				mid := (lo + hi) / 2
				if lut[mid] < target {
					lo = mid + 1
				} else {
					hi = mid
				}
			}
			i := max1(lo)
			seg := lut[i] - lut[i-1]
			u := (float64(i-1) + (target-lut[i-1])/maxF(seg, 1e-9)) / lutN
			cx := bez(p0.X, p1.X, p2.X, u)
			cy := bez(p0.Y, p1.Y, p2.Y, u)
			// tangent for the perpendicular
			tx := dbez(p0.X, p1.X, p2.X, u)
			ty := dbez(p0.Y, p1.Y, p2.Y, u)
			inv := 1 / hyp(tx, ty)
			nx, ny := -ty*inv, tx*inv
			// smoothstep width ramp: no tangent kink where stem meets floor
			ease := u * u * (3 - 2*u)
			w := wBase*(1-ease) + 1.2*ease
			col := dimCoolRGBA(multiStop(p.Colors, u, 235), shade)
			l := v2(cx+nx*w, cy+ny*w)
			r := v2(cx-nx*w, cy-ny*w)
			if s > 0 {
				// perimeter order (prevL→l→r→prevR): the old (prevL, prevR,
				// l, r) order made every quad a bowtie — opposite-winding
				// triangles cancelled under the v2.10 fill rule → sawtooth
				body.quad(prevL, l, r, prevR, prevC, col, col, prevC)
			}
			prevL, prevR, prevC = l, r, col
		}
		body.draw(dst, false)

		// bubble tip — alien orb body at the frond end (F17: halo + additive
		// glow removed; the orb itself stays)
		tipC2 := dimCoolRGBA(multiStop(p.Colors, 1, uint8(150)), shade)
		var orb mesh
		orb.fan(p2, wBase*1.35+1.5, tipC2, 12)
		orb.draw(dst, false)
	}
}

// bez evaluates a quadratic bezier component.
func bez(a, c, b, u float64) float64 {
	v := 1 - u
	return v*v*a + 2*v*u*c + u*u*b
}

// max1 clamps an index to at least 1 (table lookups are 1-based).
func max1(v int) int {
	if v < 1 {
		return 1
	}
	return v
}

// dbez is the derivative of bez (tangent direction).
func dbez(a, c, b, u float64) float64 {
	return 2*(1-u)*(c-a) + 2*u*(b-c)
}

// hyp avoids importing math twice at call sites.
func hyp(x, y float64) float64 {
	return sqrt(maxF(x*x+y*y, 1e-9))
}
