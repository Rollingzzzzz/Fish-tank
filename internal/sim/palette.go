// G2.3/G2.4: palette recipe math and water crossfade.
package sim

import (
	"math"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// expF is 1 - e^-x shaped helper name kept for readability at call sites.
func expF(x float64) float64 { return math.Exp(x) }

// lerpWater crossfades two water states by k (0 = cur, 1 = tgt).
func lerpWater(cur, tgt WaterLive, k float64) WaterLive {
	mix3 := func(a, b [3]float64) [3]float64 {
		return [3]float64{a[0] + (b[0]-a[0])*k, a[1] + (b[1]-a[1])*k, a[2] + (b[2]-a[2])*k}
	}
	return WaterLive{
		Top: mix3(cur.Top, tgt.Top), Bottom: mix3(cur.Bottom, tgt.Bottom),
		Accent:   mix3(cur.Accent, tgt.Accent),
		Rays:     cur.Rays + (tgt.Rays-cur.Rays)*k,
		Caustics: cur.Caustics + (tgt.Caustics-cur.Caustics)*k,
		Bubbles:  cur.Bubbles + (tgt.Bubbles-cur.Bubbles)*k,
	}
}

// applyRecipePal re-tints a palette by the recipe: saturation × satMul and a
// gentle brightness trim, keeping hex form (alphaMul/glowMul are consumed by
// the renderer's stage/night passes — documented in NOTES G2.3).
func applyRecipePal(pal contract.Palette, r contract.PatternRecipe) (out contract.Palette) {
	out.Body = retint(pal.Body, r.SatMul)
	out.Belly = retint(pal.Belly, r.SatMul)
	out.Accent = retint(pal.Accent, r.SatMul)
	out.Glow = retint(pal.Glow, r.SatMul)
	return out
}

// retint scales the HSL saturation of a hex color by mul.
func retint(hex string, mul float64) string {
	r, g, b := hexF(hex)
	h, s, l := rgbToHSL(r, g, b)
	s = clampF(s*mul, 0, 1)
	l = clampF(l*(0.92+0.16*mul), 0, 1)
	r2, g2, b2 := hslToRGB(h, s, l)
	return "#" + hexByte(r2) + hexByte(g2) + hexByte(b2)
}

func hexF(hex string) (r, g, b float64) {
	v := func(c byte) float64 {
		switch {
		case c >= '0' && c <= '9':
			return float64(c - '0')
		case c >= 'a' && c <= 'f':
			return float64(c-'a') + 10
		case c >= 'A' && c <= 'F':
			return float64(c-'A') + 10
		}
		return 0
	}
	if len(hex) == 7 && hex[0] == '#' {
		return (v(hex[1])*16 + v(hex[2])) / 255, (v(hex[3])*16 + v(hex[4])) / 255, (v(hex[5])*16 + v(hex[6])) / 255
	}
	return 0, 0, 0
}

func hexByte(v float64) string {
	const digits = "0123456789abcdef"
	n := int(clampF(v, 0, 1)*255 + 0.5)
	return string([]byte{digits[n/16], digits[n%16]})
}

// rgbToHSL converts 0..1 rgb to h (0..1), s, l.
func rgbToHSL(r, g, b float64) (h, s, l float64) {
	mx, mn := maxF(r, maxF(g, b)), minF(r, minF(g, b))
	l = (mx + mn) / 2
	if mx == mn {
		return 0, 0, l
	}
	d := mx - mn
	if l > 0.5 {
		s = d / (2 - mx - mn)
	} else {
		s = d / (mx + mn)
	}
	switch mx {
	case r:
		h = (g - b) / d
		if g < b {
			h += 6
		}
	case g:
		h = (b-r)/d + 2
	default:
		h = (r-g)/d + 4
	}
	return h / 6, s, l
}

// hslToRGB converts h (0..1), s, l back to 0..1 rgb.
func hslToRGB(h, s, l float64) (r, g, b float64) {
	if s == 0 {
		return l, l, l
	}
	var q float64
	if l < 0.5 {
		q = l * (1 + s)
	} else {
		q = l + s - l*s
	}
	p := 2*l - q
	return hue2rgb(p, q, h+1.0/3), hue2rgb(p, q, h), hue2rgb(p, q, h-1.0/3)
}

func hue2rgb(p, q, t float64) float64 {
	if t < 0 {
		t++
	}
	if t > 1 {
		t--
	}
	switch {
	case t < 1.0/6:
		return p + (q-p)*6*t
	case t < 0.5:
		return q
	case t < 2.0/3:
		return p + (q-p)*(2.0/3-t)*6
	}
	return p
}
