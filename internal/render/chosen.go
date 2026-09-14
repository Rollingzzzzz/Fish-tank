// N3: the Chosen One's aurora shimmer — an animated iridescent overlay only
// the eternal lilac fish wears. Separate mesh (additive) per the fill rule.
package render

import (
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/hajimehoshi/ebiten/v2"
)

// drawChosenShimmer paints a slow aurora sweep along the body: two lilac
// tones mixing with a traveling sine, additive, budget-dimmed.
func drawChosenShimmer(dst, glowDst *ebiten.Image, spine []contract.Vec2,
	norms []contract.Vec2, widths []float64, time float64) {
	if dst == nil || len(spine) < 3 {
		return
	}
	if glowDst == nil {
		glowDst = dst
	}
	n := len(spine)
	cool := hexRGBA("#e6d6ff")
	warm := hexRGBA("#9b5de5")
	var m mesh
	var prevT, prevB contract.Vec2
	for i := 0; i < n; i++ {
		u := float64(i) / float64(n-1)
		mixK := 0.5 + 0.5*sin(time*1.8+u*7.0) // traveling aurora wave
		c := lerpRGBA(withA(cool, 46), withA(warm, 46), mixK)
		t := add(spine[i], mul(norms[i], widths[i]*0.9))
		b := add(spine[i], mul(norms[i], -widths[i]*0.9))
		if i > 0 {
			m.quad(prevT, t, b, prevB, c, c, c, c)
		}
		prevT, prevB = t, b
	}
	m.draw(dst, true) // additive → GlowBudget applies
}
