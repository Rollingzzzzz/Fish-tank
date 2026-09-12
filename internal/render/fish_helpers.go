// G1.4: vector and pattern helpers for the fish renderer.
package render

import (
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// GlowVisible reports whether this species may paint ANY additive glow
// (F16): neon is the Chosen's alone — normal fish render matte body colors
// and patterns with no halos, rim lights, phosphor echoes or glowing veins.
func GlowVisible(spec *contract.Species) bool {
	return spec != nil && spec.Role == contract.RoleChosen
}

// add returns p + v.
func add(p, v contract.Vec2) contract.Vec2 { return v2(p.X+v.X, p.Y+v.Y) }

// mul returns p scaled by s.
func mul(p contract.Vec2, s float64) contract.Vec2 { return v2(p.X*s, p.Y*s) }

// lerp3 blends a→b→c across u 0..1 (two-segment gradient).
func lerp3(a, b, c colorRGBA, u float64) colorRGBA {
	if u < 0.5 {
		return lerpRGBA(a, b, u*2)
	}
	return lerpRGBA(b, c, (u-0.5)*2)
}

// widthAtU interpolates the half-width at spine parameter u.
func widthAtU(widths []float64, u float64) float64 {
	n := len(widths)
	x := clampF(u, 0, 1) * float64(n-1)
	i := int(x)
	if i >= n-1 {
		return widths[n-1]
	}
	f := x - float64(i)
	return widths[i] + (widths[i+1]-widths[i])*f
}

// widthsAt is the half-width at spine parameter t (named for pattern code).
func widthsAt(_ []contract.Vec2, _ []contract.Vec2, widths []float64, t float64) float64 {
	return widthAtU(widths, t)
}

// spinePoint returns the world point at spine parameter t (0..1) offset
// laterally by lat × local half-width.
func spinePoint(spine, norms []contract.Vec2, widths []float64, t, lat float64) contract.Vec2 {
	n := len(spine)
	x := clampF(t, 0, 1) * float64(n-1)
	i := int(x)
	if i >= n-1 {
		i = n - 2
	}
	f := x - float64(i)
	cx := spine[i].X + (spine[i+1].X-spine[i].X)*f
	cy := spine[i].Y + (spine[i+1].Y-spine[i].Y)*f
	nx := norms[i].X + (norms[i+1].X-norms[i].X)*f
	ny := norms[i].Y + (norms[i+1].Y-norms[i].Y)*f
	w := widthAtU(widths, t) * lat
	return v2(cx+nx*w, cy+ny*w)
}

// stripeQuad draws a slightly slanted bar across the body at parameter t.
func stripeQuad(m *mesh, spine, norms []contract.Vec2, widths []float64, t, halfW float64,
	c colorRGBA, slant float64) {
	a := spinePoint(spine, norms, widths, t-0.02, 1)
	b := spinePoint(spine, norms, widths, t+0.02, 1)
	a2 := add(a, v2(slant, 0))
	b2 := add(b, v2(slant, 0))
	// shrink toward the local width so the bar hugs the body edge
	m.quad(a, a2, b2, b, c, c, c, c)
	_ = halfW
}

// strokeQuads strokes a polyline with round-ish joints as quad strips.
func strokeQuads(m *mesh, pts []contract.Vec2, w float64, c colorRGBA) {
	if len(pts) < 2 {
		return
	}
	var prevL, prevR contract.Vec2
	for i, p := range pts {
		var d contract.Vec2
		if i == 0 {
			d = v2(pts[1].X-p.X, pts[1].Y-p.Y)
		} else if i == len(pts)-1 {
			d = v2(p.X-pts[i-1].X, p.Y-pts[i-1].Y)
		} else {
			d = v2(pts[i+1].X-pts[i-1].X, pts[i+1].Y-pts[i-1].Y)
		}
		dl := sqrt(maxF(d.X*d.X+d.Y*d.Y, 1e-6))
		nx, ny := -d.Y/dl, d.X/dl
		l, r := v2(p.X+nx*w, p.Y+ny*w), v2(p.X-nx*w, p.Y-ny*w)
		if i > 0 {
			// perimeter order — bowties cancel under the v2.10 fill rule
			m.quad(prevL, l, r, prevR, c, c, c, c)
		}
		prevL, prevR = l, r
	}
}

// strokeSpine strokes the top and bottom body edges (rim light).
func strokeSpine(m *mesh, spine, norms []contract.Vec2, widths []float64, w float64, c colorRGBA) {
	n := len(spine)
	for _, side := range []float64{1, -1} {
		var pts []contract.Vec2
		for i := 0; i < n; i++ {
			pts = append(pts, add(spine[i], mul(norms[i], widths[i]*side)))
		}
		strokeQuads(m, pts, w, c)
	}
}

// desat pulls a color toward its luminance by (1-k) — the F7 elder fade
// (k = 1 keeps the color, k = 0.4 is the end-of-life saturation).
func desat(c colorRGBA, k float64) colorRGBA {
	if k >= 1 {
		return c
	}
	lum := 0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B)
	mix := func(v uint8) uint8 {
		return uint8(float64(v)*k + lum*(1-k) + 0.5)
	}
	return colorRGBA{R: mix(c.R), G: mix(c.G), B: mix(c.B), A: c.A}
}
