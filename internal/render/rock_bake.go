// F25: baked crag geometry helpers — skyline interpolation and the mouth
// arches (interior, rim light, jaws, sill, lip) plus the ember fissures.
// Split from rock.go (line ceiling).
package render

import (
	"math"
	"math/rand"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// envAt interpolates the skyline height at x (inside the crag base).
func envAt(cg *crag, x float64) skyPt {
	e := cg.env
	if x <= e[0].x {
		return e[0]
	}
	if x >= e[len(e)-1].x {
		return e[len(e)-1]
	}
	for i := 1; i < len(e); i++ {
		if e[i].x >= x {
			u := (x - e[i-1].x) / maxF(e[i].x-e[i-1].x, 1e-6)
			return skyPt{x: x, y: e[i-1].y + (e[i].y-e[i-1].y)*u}
		}
	}
	return e[len(e)-1]
}

// Holes returns the enterable door mouths (sim transit + tests).
func (l *RockLayout) Holes() []contract.Hole {
	out := make([]contract.Hole, len(l.holes))
	copy(out, l.holes)
	return out
}

// BaseWidth reports the total floor-line width the rock field occupies
// (v0.3.6 composition budget: two crags ≈ 25% each = 50% of the floor).
func (l *RockLayout) BaseWidth() float64 {
	return float64(l.W) * contract.FloorShareRocks
}

// halfDome bakes a filled upper half-ellipse with a base→crown gradient; a
// single non-overlapping fan, safe inside one mesh (v2.10 fill rule).
func halfDome(m *mesh, cx, baseY, rx, ry float64, top, bot colorRGBA, n int) {
	ci := m.vert(v2(cx, baseY), bot)
	for i := 0; i <= n; i++ {
		a := math.Pi + math.Pi*float64(i)/float64(n)
		elev := -sin(a) // 0 at the base .. 1 at the crown
		m.vert(v2(cx+rx*cos(a), baseY+ry*sin(a)), lerpRGBA(bot, top, elev))
	}
	for i := 1; i <= n; i++ {
		m.tri(ci, uint16(i), uint16(i+1))
	}
}

// bakeSill bakes the dark bottom chord under a floating mouth so the arch
// reads as a hole in the rock face, not a bump.
func bakeSill(m *mesh, cx, baseY, rx float64) {
	hw := rx * 1.16
	c := withA(cFrontLo, 255)
	m.quad(v2(cx-hw, baseY-2), v2(cx+hw, baseY-2), v2(cx+hw, baseY+3), v2(cx-hw, baseY+3), c, c, c, c)
}

// bakeRimArc bakes a thin soft cyan-violet arc hugging the top inside of a
// cave mouth (F18): light spilling over the lip makes the hole read as depth
// instead of a black rectangle. Additive at draw time; alpha peaks ~64 and
// fades to zero at both ends of the arc.
func bakeRimArc(m *mesh, cx, baseY, rx, ry float64) {
	const steps = 12
	a0, a1 := math.Pi+0.45, 2*math.Pi-0.45 // the top span of the interior arch
	cyan, violet := hexRGBA("#37d8ff"), hexRGBA("#8f7bff")
	for i := 0; i < steps; i++ {
		b0 := a0 + (a1-a0)*float64(i)/steps
		b1 := a0 + (a1-a0)*float64(i+1)/steps
		u := (float64(i) + 0.5) / steps
		env := sin(3.14159 * u) // 0 at both ends, 1 mid-arc
		c := withA(lerpRGBA(cyan, violet, u), uint8(110*env))
		o0 := v2(cx+rx*0.97*cos(b0), baseY+ry*0.97*sin(b0))
		o1 := v2(cx+rx*0.97*cos(b1), baseY+ry*0.97*sin(b1))
		i0 := v2(cx+rx*0.90*cos(b0), baseY+ry*0.90*sin(b0))
		i1 := v2(cx+rx*0.90*cos(b1), baseY+ry*0.90*sin(b1))
		m.quad(o0, o1, i1, i0, c, c, c, c) // perimeter order
	}
}

// bakeJaws bakes the lower arch outline (two jaw strips hugging the mouth
// sides), the front-face rocks that visually swallow a sheltering fish.
func bakeJaws(m *mesh, cx, baseY, rx, ry float64) {
	const lipFrac = 0.38
	ay := math.Asin(1 - lipFrac) // arc angle where the lip line meets the arch
	span := [2][2]float64{{math.Pi, math.Pi + ay}, {2*math.Pi - ay, 2 * math.Pi}}
	for _, s := range span {
		const steps = 6
		for i := 0; i < steps; i++ {
			a0 := s[0] + (s[1]-s[0])*float64(i)/steps
			a1 := s[0] + (s[1]-s[0])*float64(i+1)/steps
			in0 := v2(cx+rx*cos(a0), baseY+ry*sin(a0))
			in1 := v2(cx+rx*cos(a1), baseY+ry*sin(a1))
			out0 := v2(cx+rx*1.38*cos(a0), baseY+ry*1.30*sin(a0))
			out1 := v2(cx+rx*1.38*cos(a1), baseY+ry*1.30*sin(a1))
			c := withA(lerpRGBA(cFrontLo, cFrontHi, float64(i)/steps), 255)
			m.quad(in0, in1, out1, out0, c, c, c, c)
		}
	}
}

// bakeLip bakes the rock slab covering the mouth floor so fish inside the
// cave read as hidden on their lower half.
func bakeLip(m *mesh, cx, baseY, rx, lipH float64) {
	const steps = 8
	hw := rx * 1.12
	for i := 0; i < steps; i++ {
		u0 := float64(i) / steps
		u1 := float64(i+1) / steps
		x0, x1 := cx-hw+2*hw*u0, cx-hw+2*hw*u1
		t0 := v2(x0, baseY-lipH-5*sin(math.Pi*u0))
		t1 := v2(x1, baseY-lipH-5*sin(math.Pi*u1))
		c := withA(lerpRGBA(cFrontLo, cFrontHi, 0.5+0.5*sin(math.Pi*u0)), 255)
		c2 := withA(lerpRGBA(cFrontLo, cFrontHi, 0.5+0.5*sin(math.Pi*u1)), 255)
		m.quad(t0, t1, v2(x1, baseY+1), v2(x0, baseY+1), c, c2, cFrontLo, cFrontLo)
	}
}

// bakeCracks bakes n jagged ember polylines as tapering additive ribbons
// (alpha ≤ 35 baked; the per-frame pulse scales it via drawA).
func bakeCracks(m *mesh, cx, baseY, rx, ry float64, rng *rand.Rand, n int) {
	for j := 0; j < n; j++ {
		x := cx + ((rng.Float64()-0.5)*0.6+0.3*float64(j%3))*rx
		y := baseY - ry*(0.55+0.2*rng.Float64())
		steps := 6 + rng.Intn(3)
		type pt struct{ x, y float64 }
		pts := make([]pt, 1, steps+2)
		pts[0] = pt{x, y}
		for i := 0; i < steps; i++ {
			x += (rng.Float64() - 0.5) * rx * 0.10 // steep fissures, not twigs
			y += (baseY - y) * (0.55 + 0.3*rng.Float64()) / float64(steps-i)
			pts = append(pts, pt{x, math.Min(y, baseY-2)})
		}
		for i := 1; i < len(pts); i++ {
			u := float64(i) / float64(len(pts)-1)
			dx, dy := pts[i].x-pts[i-1].x, pts[i].y-pts[i-1].y
			inv := 1 / hyp(dx, dy)
			nx, ny := -dy*inv, dx*inv
			w := 2.4*(1-u) + 0.6
			a := uint8(35 * (1 - 0.6*u))
			c := withA(hexRGBA(cEmber), a)
			p0, p1 := pts[i-1], pts[i]
			// cracks walk DOWNWARD, so the tangent's perpendicular mirrors the
			// ribbons': the front-facing perimeter here is (p0+, p0-, p1-, p1+)
			// (a bowtie cycle cancels under the v2.10 fill rule)
			m.quad(v2(p0.x+nx*w, p0.y+ny*w), v2(p0.x-nx*w, p0.y-ny*w),
				v2(p1.x-nx*w, p1.y-ny*w), v2(p1.x+nx*w, p1.y+ny*w), c, c, c, c)
		}
	}
}
