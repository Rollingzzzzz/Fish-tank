// v1.1 (G76): the bubble batch — every bubble in the tank renders as one
// thin ring with an off-center glint, in ONE mesh and ONE draw call (the
// per-bubble orb path cost a call each; the seep columns would have made
// that the loudest layer in the profile).
package render

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// BubbleView is the render-side bubble (the game layer converts; the
// renderer never imports the sim).
type BubbleView struct {
	X, Y, R, Seed float64
}

var (
	cBubbleRing  = hexRGBA("#bfe9ff") // pale glass-water cyan
	cBubbleGlint = hexRGBA("#eafaff") // the catch-light dot
)

// DrawBubbles paints the batch: an 8-segment ring (thickness grows with the
// radius, small bubbles stay glassy-thin) plus a highlight riding the upper
// right of the rim.
func DrawBubbles(dst *ebiten.Image, bs []BubbleView) {
	if dst == nil || len(bs) == 0 {
		return
	}
	m := &mesh{}
	ring := withA(cBubbleRing, 105)
	glint := withA(cBubbleGlint, 165)
	const n = 8
	for _, b := range bs {
		r := maxF(b.R, 0.6)
		th := maxF(r*0.30, 0.5)
		for i := 0; i < n; i++ {
			a0 := float64(i)/n*6.28318 + b.Seed
			a1 := float64(i+1)/n*6.28318 + b.Seed
			ri, ro := r-th*0.5, r+th*0.5
			m.quad(
				v2(b.X+cos(a0)*ri, b.Y+sin(a0)*ri),
				v2(b.X+cos(a1)*ri, b.Y+sin(a1)*ri),
				v2(b.X+cos(a1)*ro, b.Y+sin(a1)*ro),
				v2(b.X+cos(a0)*ro, b.Y+sin(a0)*ro),
				ring, ring, ring, ring)
		}
		m.fan(v2(b.X+r*0.38, b.Y-r*0.38), maxF(r*0.26, 0.5), glint, 5)
	}
	m.draw(dst, false)
}
