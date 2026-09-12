// N11: live-treat rendering — bug / worm / shrimp / chicken, palette-true to
// the tray glyphs, animated purely by the phase param (the sim advances it
// at per-kind rates). Shapes run ~12-18 px; normal alpha only — matte.
package render

import (
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/hajimehoshi/ebiten/v2"
)

// treat palettes — palette-true to the tray glyphs (ui drawTreatGlyph).
var (
	tBugBody  = hexRGBA("#ff4fd8") // magenta
	tBugHead  = hexRGBA("#a9208f")
	tBugWing  = hexRGBA("#ffe0fa")
	tWormLo   = hexRGBA("#4f8f22")
	tWormHi   = hexRGBA("#8dff3d") // lime green
	tShrBody  = hexRGBA("#7fd4ff") // cyan
	tShrHi    = hexRGBA("#d8f6ff")
	tChickLo  = hexRGBA("#e8c88f")
	tChickHi  = hexRGBA("#ffe9b0") // warm cream
	tChickShe = hexRGBA("#fff6d8") // sheen highlight
)

// DrawTreat paints one live treat at pos; every motion derives from phase.
func DrawTreat(dst *ebiten.Image, kind string, pos contract.Vec2, phase float64) {
	if dst == nil {
		return
	}
	switch kind {
	case contract.TreatBug:
		drawTreatBug(dst, pos, phase)
	case contract.TreatWorm:
		drawTreatWorm(dst, pos, phase)
	case contract.TreatShrimp:
		drawTreatShrimp(dst, pos, phase)
	case contract.TreatChicken:
		drawTreatChicken(dst, pos, phase)
	}
}

// drawTreatWorm — a segmented sine ribbon that thrashes: ~4 px amplitude and
// the whole body curls with phase. Quads run in perimeter order (prevL, l,
// r, prevR) — the (prevL, prevR, l, r) order would bowtie and cancel under
// the v2.10 fill rule.
func drawTreatWorm(dst *ebiten.Image, pos contract.Vec2, phase float64) {
	const segs = 8
	curl := sin(phase*1.6) * 2.4 // slow whole-body curl
	var m mesh
	var prevL, prevR contract.Vec2
	var prevC colorRGBA
	for s := 0; s <= segs; s++ {
		u := float64(s) / segs
		x := pos.X - 9 + u*18
		y := pos.Y + sin(phase*3.2+u*6.2)*4 + curl*u*u
		w := 2.3*sin(3.14159*(0.08+0.84*u)) + 0.5 // fat middle, rounded ends
		col := withA(lerpRGBA(tWormLo, tWormHi, 0.25+0.75*u), 235)
		l, r := v2(x, y-w), v2(x, y+w)
		if s > 0 {
			m.quad(prevL, l, r, prevR, prevC, col, col, prevC)
		}
		prevL, prevR, prevC = l, r, col
	}
	m.draw(dst, false)
	// darker seams between segments sell the segmentation
	var seams mesh
	for s := 1; s < segs; s++ {
		u := float64(s) / segs
		x := pos.X - 9 + u*18
		y := pos.Y + sin(phase*3.2+u*6.2)*4 + curl*u*u
		w := 2.3*sin(3.14159*(0.08+0.84*u)) + 0.5
		c := withA(tWormLo, 95)
		seams.quad(v2(x-0.7, y-w+0.4), v2(x+0.7, y-w+0.4),
			v2(x+0.7, y+w-0.4), v2(x-0.7, y+w-0.4), c, c, c, c)
	}
	seams.draw(dst, false)
}

// drawTreatBug — a small oval body, six skittering legs and a faint wing
// shimmer; the hover bob and every jitter come from phase.
func drawTreatBug(dst *ebiten.Image, pos contract.Vec2, phase float64) {
	c := v2(pos.X, pos.Y+sin(phase*7)*1.2) // nervous hover bob
	// legs under the body: 3 per side, skittering at phase*12
	var legs mesh
	legC := withA(tBugHead, 220)
	for s := -1; s <= 1; s += 2 {
		for i := 0; i < 3; i++ {
			lx := c.X - 3.2 + float64(i)*3.2
			sk := sin(phase*12+float64(i)*2.1+float64(s)*0.7) * 1.8
			root := v2(lx, c.Y+float64(s)*2.0)
			tip := v2(lx-1.6+sk*0.5, c.Y+float64(s)*(4.9+sk*0.6))
			strokeQuads(&legs, []contract.Vec2{root, tip}, 0.55, legC)
		}
	}
	legs.draw(dst, false)
	// faint wing shimmer — two buzzing translucent ovals above the body
	buzz := uint8(34 + 30*(0.5+0.5*sin(phase*24)))
	var wings mesh
	oval(&wings, v2(c.X-2.4, c.Y-3.6), 3.4, 1.7, withA(tBugWing, buzz), 10)
	oval(&wings, v2(c.X+2.4, c.Y-3.6), 3.4, 1.7, withA(tBugWing, buzz), 10)
	wings.draw(dst, false)
	// body + head + two tiny antennae (separate meshes — they overlap)
	var body mesh
	oval(&body, c, 4.6, 3.1, withA(tBugBody, 235), 12)
	body.draw(dst, false)
	var head mesh
	oval(&head, v2(c.X-4.6, c.Y-0.4), 1.9, 1.9, withA(tBugHead, 235), 10)
	head.draw(dst, false)
	var ant mesh
	sway := sin(phase*9) * 1.0
	strokeQuads(&ant, []contract.Vec2{v2(c.X-5.6, c.Y-1.2), v2(c.X-8.2, c.Y-3.0+sway)}, 0.5, legC)
	strokeQuads(&ant, []contract.Vec2{v2(c.X-5.6, c.Y-1.2), v2(c.X-8.6, c.Y-1.6-sway)}, 0.5, legC)
	ant.draw(dst, false)
}

// drawTreatShrimp — a semi-transparent curled body on an inward spiral; the
// curl oscillates (tail flick) and two antennae sway from the head.
func drawTreatShrimp(dst *ebiten.Image, pos contract.Vec2, phase float64) {
	flick := sin(phase * 2.6)
	span := 1.15 + 0.38*flick // curvature oscillates → the tail flicks
	const segs = 6
	ctr := v2(pos.X, pos.Y-4.5)
	pt := func(u float64) (contract.Vec2, float64) { // point + radial normal
		ang := 0.42*3.14159 + u*span
		rad := 9 - 3.2*u // the spiral tightens toward the tail
		return v2(ctr.X+rad*cos(ang), ctr.Y+rad*sin(ang)), ang
	}
	var m mesh
	var prevL, prevR contract.Vec2
	var prevC colorRGBA
	for s := 0; s <= segs; s++ {
		u := float64(s) / segs
		p, ang := pt(u)
		w := 2.5 - 1.3*u // body tapers to the tail
		nx, ny := cos(ang), sin(ang)
		l, r := v2(p.X+nx*w, p.Y+ny*w), v2(p.X-nx*w, p.Y-ny*w)
		col := withA(lerpRGBA(tShrBody, tShrHi, 0.3+0.5*u), 165) // translucent
		if s > 0 {
			m.quad(prevL, l, r, prevR, prevC, col, col, prevC)
		}
		prevL, prevR, prevC = l, r, col
	}
	m.draw(dst, false)
	// tail fin — a small triangle flicking off the tail end
	tp, tang := pt(1)
	td := v2(cos(tang+1.57), sin(tang+1.57)) // outward from the spiral
	var tail mesh
	tailC := withA(tShrHi, 150)
	tail.triT(tp, add(tp, mul(td, 3.4)), add(tp, v2(td.X*2.4-td.Y*1.8, td.Y*2.4+td.X*1.8)), tailC)
	tail.draw(dst, false)
	// two antennae from the head
	hp, _ := pt(0)
	sway := sin(phase*3) * 1.2
	var ant mesh
	strokeQuads(&ant, []contract.Vec2{hp, add(hp, v2(5.2, 1.8+sway))}, 0.55, withA(tShrHi, 170))
	strokeQuads(&ant, []contract.Vec2{hp, add(hp, v2(4.2, 3.6-sway))}, 0.55, withA(tShrHi, 170))
	ant.draw(dst, false)
}

// drawTreatChicken — a pale irregular chunk with a jiggle wobble and a soft
// sheen highlight riding its upper-left shoulder.
func drawTreatChicken(dst *ebiten.Image, pos contract.Vec2, phase float64) {
	c := v2(pos.X, pos.Y+sin(phase*5)*0.8)
	var m mesh
	const n = 9
	// deterministic lumpy radii — an irregular chunk, never a ball
	lump := [9]float64{0.85, 1.15, 0.9, 1.2, 0.8, 1.12, 0.95, 1.18, 0.88}
	ci := m.vert(c, tChickHi)
	base := uint16(len(m.verts))
	for i := 0; i <= n; i++ {
		j := i % n
		a := float64(j) / n * 2 * 3.141592653589793
		r := 7.2 * lump[j] * (1 + 0.07*sin(phase*5+float64(j)*1.3)) // jiggle
		shade := 0.75 + 0.25*sin(a+2.1)                             // light from upper-left
		col := withA(lerpRGBA(tChickLo, tChickHi, shade), 235)
		m.verts = append(m.verts, ebiten.Vertex{
			DstX: float32(c.X + r*cos(a)), DstY: float32(c.Y + r*sin(a)),
			SrcX: 1, SrcY: 1,
			ColorR: float32(col.R) / 255, ColorG: float32(col.G) / 255,
			ColorB: float32(col.B) / 255, ColorA: float32(col.A) / 255,
		})
		if i > 0 {
			m.tri(ci, base+uint16(i-1), base+uint16(i))
		}
	}
	m.draw(dst, false)
	// soft sheen — a short pale stroke on the shoulder
	var sh mesh
	strokeQuads(&sh, []contract.Vec2{v2(c.X-5.5, c.Y-2.5), v2(c.X-2.5, c.Y-4.4), v2(c.X+0.5, c.Y-4.6)},
		1.3, withA(tChickShe, 95))
	sh.draw(dst, false)
}
