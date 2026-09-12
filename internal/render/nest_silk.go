// v0.3.7 (F27): the oyster's surroundings and its silk — an ambient shadow
// pocket (backdrop dome + ground contact shadow) that seats the shell INTO
// the scene instead of glowing on top of it, and the byssus: silky threads
// the oyster spins to hold itself, flowing from the hinge over the shell and
// trailing onto the floor — silky hair-like strands that read as part of the
// oyster; they also bind the two valves visually.
package render

import (
	"math"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/hajimehoshi/ebiten/v2"
)

// drawNestAmbient paints the shadow pocket BEFORE the shell: a dark backdrop
// dome in the crag palette family hugging the shell's silhouette, and a flat
// ground-contact shadow on the floor line. No glow — belonging, not floating.
// v0.3.8: much lighter and tighter — the shell is scenery now, the old
// near-black pocket suffocated the view.
func drawNestAmbient(dst *ebiten.Image, cx, baseY, hw, oh float64) {
	var ground mesh
	oval(&ground, v2(cx, baseY-3), hw*1.08, oh*0.075, withA(hexRGBA("#060310"), 110), 28)
	ground.draw(dst, false)
	// backdrop dome: center dark like the cave grades, fading to nothing
	var back mesh
	bn := 26
	bci := back.vert(v2(cx, baseY), withA(hexRGBA("#0f0a20"), 80))
	for i := 0; i <= bn; i++ {
		a := math.Pi - math.Pi*float64(i)/float64(bn)
		back.vert(v2(cx-hw*1.06*cos(a), baseY-oh*0.55*sin(a)), withA(hexRGBA("#0f0a20"), 0))
	}
	for i := 1; i <= bn; i++ {
		back.tri(bci, uint16(i), uint16(i+1))
	}
	back.draw(dst, false)
}

// byssusStrands is the pure geometry of the silk bundle — testable: at least
// six strands, every one rooted at the hinge and trailing onto the floor.
func byssusStrands(cx, baseY, hw, oh float64) [][]contract.Vec2 {
	const num, seg = 9, 6
	out := make([][]contract.Vec2, 0, num)
	for s := 0; s < num; s++ {
		u := float64(s)/float64(num-1)*2 - 1 // -1..1 across the shell
		root := v2(cx+u*hw*0.09, baseY-oh*0.075+cos(u*1.8)*oh*0.012)
		endX := cx + u*hw*0.62 + cos(u*2.4)*hw*0.05
		pts := make([]contract.Vec2, 0, seg+1)
		for i := 0; i <= seg; i++ {
			t := float64(i) / float64(seg)
			x := root.X + (endX-root.X)*t
			y := root.Y + (baseY-root.Y)*t + sin(t*math.Pi)*u*10 // gentle bow
			pts = append(pts, v2(x, y))
		}
		out = append(out, pts)
	}
	return out
}

// drawNestSilk strokes the byssus: a soft under-pass and a bright core pass
// per strand, breathing with a slow sway. They cross the hinge junction —
// the "seam" between the valves — tying the shell into one piece.
func drawNestSilk(dst *ebiten.Image, cx, baseY, hw, oh, time float64) {
	soft, core := withA(hexRGBA("#8a6fc4"), 80), withA(hexRGBA("#d8c6f4"), 165)
	for si, pts := range byssusStrands(cx, baseY, hw, oh) {
		sw := make([]contract.Vec2, len(pts))
		for i, p := range pts {
			t := float64(i) / float64(len(pts)-1)
			sw[i] = v2(p.X+sin(time*0.7+float64(si)*1.3+t*3)*2.2*t, p.Y)
		}
		var under, line mesh
		strokeQuads(&under, sw, 3.4, soft)
		under.draw(dst, false)
		strokeQuads(&line, sw, 1.5, core)
		line.draw(dst, false)
	}
}
