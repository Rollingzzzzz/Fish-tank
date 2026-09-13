// v1.1 G73: the Chosen's wormhole door — a small breathing ring of light
// where she steps through the water. Opening rings grow, exit rings fade.
// The neon stays hers alone (F16).
package render

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// DrawPortal paints one portal at pos. grow 0..1 is the opening (ring
// radius + inner light), t animates the shimmer.
func DrawPortal(dst *ebiten.Image, x, y, grow, t float64) {
	if dst == nil || grow <= 0.01 {
		return
	}
	r := 6 + 26*grow
	// soft core glow + a crisp ring, gently breathing
	DrawGlow(dst, x, y, r*2.2, "#c9a8ff", 0.16*grow*(0.8+0.2*math.Sin(t*3)))
	var ring mesh
	for i := 0; i < 26; i++ {
		a0 := float64(i) / 26 * 2 * math.Pi
		a1 := float64(i+1) / 26 * 2 * math.Pi
		wob := 1 + 0.08*math.Sin(t*2.4+float64(i)*0.5)
		p0 := v2(x+math.Cos(a0)*r*wob, y+math.Sin(a0)*r*wob*0.92)
		p1 := v2(x+math.Cos(a1)*r*wob, y+math.Sin(a1)*r*wob*0.92)
		p2 := v2(x+math.Cos(a1)*(r+3.2)*wob, y+math.Sin(a1)*(r+3.2)*wob*0.92)
		p3 := v2(x+math.Cos(a0)*(r+3.2)*wob, y+math.Sin(a0)*(r+3.2)*wob*0.92)
		ring.quad(p0, p1, p2, p3,
			withA(hexRGBA("#e6d5ff"), uint8(200*grow)),
			withA(hexRGBA("#e6d5ff"), uint8(200*grow)),
			withA(hexRGBA("#9a7fd4"), uint8(170*grow)),
			withA(hexRGBA("#9a7fd4"), uint8(170*grow)))
	}
	ring.draw(dst, false)
}
