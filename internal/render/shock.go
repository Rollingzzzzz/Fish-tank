// v1.1 (G83): the strike pressure ring — one soft expanding circle per
// shock, pale water-glass white on normal alpha (the neon stays the
// Chosen's alone). Every live ring rides ONE mesh, one draw call.
package render

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// ShockView is the render-side ring (the game layer converts).
type ShockView struct {
	X, Y, P float64 // P = 0 at the strike .. 1 at fade-out
}

var cShockRing = hexRGBA("#e8f6ff")

// DrawShocks paints the batch: a thin ring whose radius eases out to
// ~150 px while the alpha falls from 120 to nothing — a pressure wave,
// not a glow.
func DrawShocks(dst *ebiten.Image, sv []ShockView) {
	if dst == nil || len(sv) == 0 {
		return
	}
	m := &mesh{}
	const n = 26
	for _, s := range sv {
		p := clampF(s.P, 0, 1)
		ease := 1 - (1-p)*(1-p) // fast out, slow settle
		r := 26 + ease*130
		th := maxF(14*(1-p)+3, 3)
		col := withA(cShockRing, uint8(135*(1-p)))
		for i := 0; i < n; i++ {
			a0 := float64(i)/n*6.28318 + p*0.6
			a1 := float64(i+1)/n*6.28318 + p*0.6
			ri, ro := r-th, r
			m.quad(
				v2(s.X+cos(a0)*ri, s.Y+sin(a0)*ri),
				v2(s.X+cos(a1)*ri, s.Y+sin(a1)*ri),
				v2(s.X+cos(a1)*ro, s.Y+sin(a1)*ro),
				v2(s.X+cos(a0)*ro, s.Y+sin(a0)*ro),
				col, col, col, col)
		}
	}
	m.draw(dst, false)
}
