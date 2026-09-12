// G1.2: shared render utilities — white fill source, color conversion, mesh
// builder for triangle-based shapes with per-vertex gradients.
package render

import (
	"image"
	"image/color"
	"math"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/hajimehoshi/ebiten/v2"
)

// imageRect builds an image.Rectangle.
func imageRect(x0, y0, x1, y1 int) image.Rectangle { return image.Rect(x0, y0, x1, y1) }

// imageNewRGBA allocates an RGBA image of the given size.
func imageNewRGBA(w, h int) *image.RGBA { return image.NewRGBA(image.Rect(0, 0, w, h)) }

var whiteColor = color.RGBA{R: 255, G: 255, B: 255, A: 255}

// white is the shared 3x3 white source image for DrawTriangles fills;
// vertex colors carry all tinting and gradients.
var white = ebiten.NewImage(3, 3)

func init() { white.Fill(whiteColor) }

// WaterState carries everything the background water needs per frame.
type WaterState struct {
	Time      float64    // seconds, drives all animation
	DayFactor float64    // 0 night .. 1 day
	TOD       float64    // F9: time of day 0..1 (0 dawn, 0.25 noon, 0.5 dusk, 0.75 deep night)
	Top       [3]float64 // 0..1 RGB, surface water color
	Bottom    [3]float64 // 0..1 RGB, floor water color
	Accent    [3]float64 // 0..1 RGB, glow accent
	Rays      float64    // 0..1 god-ray intensity
	Caustics  float64    // 0..1 caustics intensity
}

// v2 builds a contract.Vec2.
func v2(x, y float64) contract.Vec2 { return contract.Vec2{X: x, Y: y} }

// clampF is the package-local float clamp.
func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// colorRGBA is a local alias keeping renderer signatures short.
type colorRGBA = color.RGBA

// withA returns c with the alpha replaced.
func withA(c colorRGBA, a uint8) colorRGBA { c.A = a; return c }

// sqrt / maxF micro-helpers.
func sqrt(x float64) float64 { return math.Sqrt(x) }
func maxF(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// shorthand trig (radians)
func cos(a float64) float64 { return math.Cos(a) }
func sin(a float64) float64 { return math.Sin(a) }

// hexRGBA converts "#rrggbb" to a color; invalid hex yields black (D4).
func hexRGBA(hex string) color.RGBA {
	r, g, b := contract.HexToRGB(hex)
	return color.RGBA{R: r, G: g, B: b, A: 255}
}

// scaleRGBA multiplies an RGBA color by f (component-wise, clamped).
func scaleRGBA(c color.RGBA, f float64) color.RGBA {
	cl := func(v uint8) uint8 {
		x := float64(v) * f
		if x > 255 {
			x = 255
		}
		return uint8(x)
	}
	return color.RGBA{R: cl(c.R), G: cl(c.G), B: cl(c.B), A: c.A}
}

// dimCoolRGBA dims a color toward a cool near-glass silhouette: k=1 keeps
// the color untouched, smaller k darkens it with a blue bias — the v0.3.8
// foreground depth plane reads as dark glass-adjacent flora, not flat black.
func dimCoolRGBA(c color.RGBA, k float64) color.RGBA {
	if k >= 1 {
		return c
	}
	cl := func(v float64) uint8 {
		if v > 255 {
			v = 255
		}
		return uint8(v)
	}
	return color.RGBA{
		R: cl(float64(c.R) * k),
		G: cl(float64(c.G) * (0.15 + 0.85*k)),
		B: cl(float64(c.B) * (0.30 + 0.70*k)),
		A: c.A,
	}
}

// lerpRGBA blends two colors by t (0 = a, 1 = b).
func lerpRGBA(a, b color.RGBA, t float64) color.RGBA {
	mix := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t) }
	return color.RGBA{R: mix(a.R, b.R), G: mix(a.G, b.G), B: mix(a.B, b.B), A: mix(a.A, b.A)}
}

// mesh accumulates triangles with per-vertex colors for one additive or
// normal draw call. Reused across frames (C3: zero hot-path allocation after
// warm-up — capacity grows once and is retained).
type mesh struct {
	verts   []ebiten.Vertex
	indices []uint16
}

func (m *mesh) reset() {
	m.verts = m.verts[:0]
	m.indices = m.indices[:0]
}

// merge appends another mesh's triangles (index base rebased). Only
// like-wound parts may share a mesh: opposite-winding overlap cancels under
// the v2.10 fill rule. Reused capacity is retained on both meshes.
func (m *mesh) merge(o *mesh) {
	base := uint16(len(m.verts))
	m.verts = append(m.verts, o.verts...)
	for _, ix := range o.indices {
		m.indices = append(m.indices, ix+base)
	}
}

func (m *mesh) vert(p contract.Vec2, c color.RGBA) uint16 {
	m.verts = append(m.verts, ebiten.Vertex{
		DstX: float32(p.X), DstY: float32(p.Y),
		SrcX: 1, SrcY: 1, // center of the white 3x3
		ColorR: float32(c.R) / 255, ColorG: float32(c.G) / 255,
		ColorB: float32(c.B) / 255, ColorA: float32(c.A) / 255,
	})
	return uint16(len(m.verts) - 1)
}

func (m *mesh) tri(a, b, c uint16) {
	m.indices = append(m.indices, a, b, c)
}

// triT adds a filled triangle from three points sharing one color.
func (m *mesh) triT(a, b, c contract.Vec2, col color.RGBA) {
	ia, ib, ic := m.vert(a, col), m.vert(b, col), m.vert(c, col)
	m.tri(ia, ib, ic)
}

// quad adds a two-triangle quad (cycle p0→p1→p2→p3); colors are per-corner
// for gradients. Both triangles share the same winding — v2.10 culls
// back-facing triangles, so inconsistent winding loses half the quad.
func (m *mesh) quad(p0, p1, p2, p3 contract.Vec2, c0, c1, c2, c3 color.RGBA) {
	i0 := m.vert(p0, c0)
	i1 := m.vert(p1, c1)
	i2 := m.vert(p2, c2)
	i3 := m.vert(p3, c3)
	m.tri(i0, i1, i2)
	m.tri(i0, i2, i3)
}

// fan adds a filled circle-ish polygon (n segments).
func (m *mesh) fan(center contract.Vec2, radius float64, c color.RGBA, n int) {
	ci := m.vert(center, c)
	base := uint16(len(m.verts))
	for i := 0; i < n; i++ {
		a0 := float64(i) / float64(n) * 2 * 3.141592653589793
		a1 := float64(i+1) / float64(n) * 2 * 3.141592653589793
		m.verts = append(m.verts,
			ebiten.Vertex{
				DstX: float32(center.X + radius*cos(a0)), DstY: float32(center.Y + radius*sin(a0)),
				SrcX: 1, SrcY: 1,
				ColorR: float32(c.R) / 255, ColorG: float32(c.G) / 255,
				ColorB: float32(c.B) / 255, ColorA: float32(c.A) / 255,
			},
			ebiten.Vertex{
				DstX: float32(center.X + radius*cos(a1)), DstY: float32(center.Y + radius*sin(a1)),
				SrcX: 1, SrcY: 1,
				ColorR: float32(c.R) / 255, ColorG: float32(c.G) / 255,
				ColorB: float32(c.B) / 255, ColorA: float32(c.A) / 255,
			})
	}
	for i := 0; i < n; i++ {
		m.tri(ci, base+uint16(i*2), base+uint16(i*2+1))
	}
}

// oval adds a filled ellipse fan (rx/ry radii) — same winding discipline as
// fan; each overlapping shape goes in its own mesh (v2.10 fill rule).
func oval(m *mesh, c contract.Vec2, rx, ry float64, col colorRGBA, n int) {
	ci := m.vert(c, col)
	base := uint16(len(m.verts))
	for i := 0; i < n; i++ {
		a0 := float64(i) / float64(n) * 2 * 3.141592653589793
		a1 := float64(i+1) / float64(n) * 2 * 3.141592653589793
		m.verts = append(m.verts,
			ebiten.Vertex{
				DstX: float32(c.X + rx*cos(a0)), DstY: float32(c.Y + ry*sin(a0)),
				SrcX: 1, SrcY: 1,
				ColorR: float32(col.R) / 255, ColorG: float32(col.G) / 255,
				ColorB: float32(col.B) / 255, ColorA: float32(col.A) / 255,
			},
			ebiten.Vertex{
				DstX: float32(c.X + rx*cos(a1)), DstY: float32(c.Y + ry*sin(a1)),
				SrcX: 1, SrcY: 1,
				ColorR: float32(col.R) / 255, ColorG: float32(col.G) / 255,
				ColorB: float32(col.B) / 255, ColorA: float32(col.A) / 255,
			})
	}
	for i := 0; i < n; i++ {
		m.tri(ci, base+uint16(i*2), base+uint16(i*2+1))
	}
}

// drawA flushes an alpha-scaled additive copy (for trail echoes).
func (m *mesh) drawA(dst *ebiten.Image, k float64) {
	if len(m.indices) == 0 || dst == nil || k <= 0 {
		return
	}
	opts := &ebiten.DrawTrianglesOptions{AntiAlias: true, Filter: ebiten.FilterLinear, Blend: ebiten.BlendLighter}
	saved := m.verts
	for i := range m.verts {
		m.verts[i].ColorA = float32(float64(m.verts[i].ColorA) * k)
	}
	dst.DrawTriangles(m.verts, m.indices, white, opts)
	m.verts = saved
}

// draw flushes the mesh; lighter=true uses additive blending (glow).
// DrawTrianglesOptions carries no ColorScale in v2.10, so the F2 glow budget
// is applied by temporarily scaling vertex RGB in place.
func (m *mesh) draw(dst *ebiten.Image, lighter bool) {
	if len(m.indices) == 0 || dst == nil {
		return
	}
	opts := &ebiten.DrawTrianglesOptions{AntiAlias: true, Filter: ebiten.FilterLinear}
	if lighter {
		opts.Blend = ebiten.BlendLighter
		saved := m.verts
		for i := range m.verts {
			m.verts[i].ColorR *= GlowBudget
			m.verts[i].ColorG *= GlowBudget
			m.verts[i].ColorB *= GlowBudget
		}
		dst.DrawTriangles(m.verts, m.indices, white, opts)
		m.verts = saved
		return
	}
	dst.DrawTriangles(m.verts, m.indices, white, opts)
}

// drawAliased is draw without edge anti-aliasing — the batched school path
// (F29): one call rasterizes hundreds of small triangles, where per-pixel
// AA dominates. Edges are 1px harder at fish scale; the rate doubles.
func (m *mesh) drawAliased(dst *ebiten.Image) {
	if len(m.indices) == 0 || dst == nil {
		return
	}
	dst.DrawTriangles(m.verts, m.indices, white,
		&ebiten.DrawTrianglesOptions{Filter: ebiten.FilterLinear})
}
