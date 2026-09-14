// v1.1 (G76): the bubble batch — every bubble in the tank renders as one
// thin ring with an off-center glint, in ONE mesh and ONE draw call (the
// per-bubble orb path cost a call each; the seep columns would have made
// that the loudest layer in the profile).
// G78: a bubble is AIR — the interior must read hollow. At screen scale a
// bare 8-segment ring collapses into a solid dot (live-observation
// finding), so each bubble now carries a faint interior window: the rim
// darkens the water behind it while the window lets it through, and the
// eye reads the bubble as a sphere of air even at 2 px.
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
	cBubbleRim   = hexRGBA("#bfe9ff") // pale glass-water cyan
	cBubbleWin   = hexRGBA("#d8f4ff") // the through-water window
	cBubbleGlint = hexRGBA("#f2fcff") // the catch-light dot
)

// bubbleParts is the anatomy law (G78): rim thickness, window and glint
// radii all scale with the bubble, with floors that survive a 0.75×
// downscale — a 2 px bubble still shows rim > window > glint contrast.
func bubbleParts(r float64) (rimTh, winR, glintR float64) {
	r = maxF(r, 1.1)
	rimTh = maxF(r*0.34, 0.7)
	winR = maxF(r-rimTh*0.45, 0.4)
	glintR = maxF(r*0.30, 0.6)
	return rimTh, winR, glintR
}

// buildBubbleMesh lays the whole batch: per bubble a faint filled window,
// an 8-segment ring around it and a highlight riding the upper right.
func buildBubbleMesh(bs []BubbleView) *mesh {
	m := &mesh{}
	window := withA(cBubbleWin, 34)
	ring := withA(cBubbleRim, 140)
	glint := withA(cBubbleGlint, 190)
	const n = 8
	for _, b := range bs {
		rimTh, winR, glintR := bubbleParts(b.R)
		m.fan(v2(b.X, b.Y), winR, window, 7)
		for i := 0; i < n; i++ {
			a0 := float64(i)/n*6.28318 + b.Seed
			a1 := float64(i+1)/n*6.28318 + b.Seed
			ri, ro := winR, winR+rimTh
			m.quad(
				v2(b.X+cos(a0)*ri, b.Y+sin(a0)*ri),
				v2(b.X+cos(a1)*ri, b.Y+sin(a1)*ri),
				v2(b.X+cos(a1)*ro, b.Y+sin(a1)*ro),
				v2(b.X+cos(a0)*ro, b.Y+sin(a0)*ro),
				ring, ring, ring, ring)
		}
		m.fan(v2(b.X+winR*0.55, b.Y-winR*0.55), glintR, glint, 5)
	}
	return m
}

// DrawBubbles paints the batch in one draw call.
func DrawBubbles(dst *ebiten.Image, bs []BubbleView) {
	if dst == nil || len(bs) == 0 {
		return
	}
	buildBubbleMesh(bs).draw(dst, false)
}
