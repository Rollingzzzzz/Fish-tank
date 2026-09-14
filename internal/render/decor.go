// Wild water mites (N7) — paddle-legged food specks. Matte, normal-alpha
// only — neon stays the Chosen's alone (F16). v0.3.8: the crustaceans (N4)
// were retired; their dark floor silhouettes read as clutter against the
// night-dimmed floor.
package render

import (
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/hajimehoshi/ebiten/v2"
)

var cMiteBody = hexRGBA("#b9cf8e") // greenish food speck

// DrawMite paints one wild water mite (N7): a ~4 px oval with paddle legs
// wiggling by phase and a faint bright food dot (glow alpha stays ≤ 0.15).
func DrawMite(dst *ebiten.Image, pos contract.Vec2, phase float64) {
	if dst == nil {
		return
	}
	var legs mesh
	for s := -1; s <= 1; s += 2 {
		for i := 0; i < 2; i++ {
			wig := sin(phase*6+float64(i)*1.9+float64(s)) * 1.6
			root := v2(pos.X-float64(s)*1.4, pos.Y-1.1+float64(i)*2.2)
			tip := v2(pos.X-float64(s)*4.4+wig*0.4, pos.Y-1.6+float64(i)*2.2+wig)
			strokeQuads(&legs, []contract.Vec2{root, tip}, 0.6, withA(cMiteBody, 205))
		}
	}
	legs.draw(dst, false)
	var body mesh
	oval(&body, pos, 2.9, 2.1, withA(cMiteBody, 235), 10)
	body.draw(dst, false)
	DrawGlow(dst, pos.X, pos.Y, 5, "#eaffd0", 0.14) // faint food glint
	var core mesh
	core.fan(v2(pos.X+0.7, pos.Y-0.6), 0.8, withA(hexRGBA("#f4ffd6"), 235), 6)
	core.draw(dst, false)
}
