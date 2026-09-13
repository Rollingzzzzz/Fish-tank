// v1.1: floor critters — crab + shrimp (G44). Vivid coral-orange, matte,
// normal-alpha only: the neon stays the Chosen's alone (F16) and nothing
// here repeats the v0.3.8 dark-clutter failure — the palettes clear the
// measured floor-luminance gate day and night (docs/art evidence).
package render

import (
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/hajimehoshi/ebiten/v2"
)

var (
	cCritterShell  = hexRGBA("#ff9f43") // vivid coral-orange shell
	cCritterShell2 = hexRGBA("#ffd166") // warm-gold highlight
	cCritterLeg    = hexRGBA("#e0631f") // bright ember legs
	cCritterPale   = hexRGBA("#ffedd0") // cream belly / struggle pulse
)

// DrawCreature paints one floor critter. Burrow sinks it into the sand with
// a fade; vulnerable adds a helpless flail and a pale struggle pulse.
func DrawCreature(dst *ebiten.Image, kind string, pos contract.Vec2, phase, dir, burrow float64, vulnerable bool) {
	if dst == nil {
		return
	}
	sink := burrow * 10
	squash := 1 - burrow*0.8
	a := uint8(235 * (1 - burrow))
	p := v2(pos.X, pos.Y+sink)
	var m mesh
	switch kind {
	case "crab":
		drawCrab(&m, p, phase, dir, squash, a, vulnerable)
	default:
		drawShrimp(&m, p, phase, dir, squash, a, vulnerable)
	}
	m.draw(dst, false)
	if vulnerable && burrow <= 0 {
		DrawGlow(dst, p.X, p.Y-3, 14, "#ffd9a0", 0.10+0.06*sin(phase*3))
	}
}

// drawCrab: a wide oval shell, two claws on stalks and six strutting legs.
func drawCrab(m *mesh, p contract.Vec2, phase, dir, squash float64, a uint8, vulnerable bool) {
	flail := 1.0
	if vulnerable {
		flail = 2.2 // the helpless struggle
	}
	// struggle pulse: a pale halo that breathes
	if vulnerable {
		halo := 10.5 + 1.2*sin(phase*3)
		oval(m, p, halo, (halo*0.62)*squash, withA(cCritterPale, uint8(70+50*sin(phase*3))), 12)
	}
	// legs: three per side, wiggling
	for s := -1; s <= 1; s += 2 {
		for i := 0; i < 3; i++ {
			wig := sin(phase*3.4*flail+float64(i)*1.7+float64(s)) * 2.4 * flail
			root := v2(p.X+float64(i)*2.6-2.6, p.Y+float64(s)*3.4*squash)
			tip := v2(root.X+(float64(i)-1)*2.4+wig*0.3, root.Y+float64(s)*(6.4+wig*wig*0.12)*squash)
			strokeQuads(m, []contract.Vec2{root, tip}, 1.3, withA(cCritterLeg, a))
		}
	}
	// claws on short stalks, pointing the walking direction
	cl := v2(p.X+dir*9.5, p.Y-2.5*squash+sin(phase*2.6*flail)*1.1)
	strokeQuads(m, []contract.Vec2{v2(p.X+dir*4.5, p.Y-1), cl}, 1.5, withA(cCritterLeg, a))
	oval(m, cl, 3.1, 2.3, withA(cCritterShell2, a), 8)
	// shell
	oval(m, p, 9.0, 5.6*squash, withA(cCritterShell, a), 14)
	oval(m, v2(p.X, p.Y-1.2*squash), 7.2, 3.4*squash, withA(cCritterShell2, uint8(uint16(a)*8/10)), 12)
	// eyes
	for s := -1; s <= 1; s += 2 {
		oval(m, v2(p.X+dir*5.5, p.Y-4.2*squash+float64(s)*1.6), 0.9, 0.9, withA(cCritterPale, a), 6)
	}
}

// drawShrimp: an arced translucent body, a fan tail and long antennae.
func drawShrimp(m *mesh, p contract.Vec2, phase, dir, squash float64, a uint8, vulnerable bool) {
	flail := 1.0
	if vulnerable {
		flail = 2.4
	}
	if vulnerable {
		halo := 9.5 + 1.2*sin(phase*3)
		oval(m, p, halo, (halo*0.55)*squash, withA(cCritterPale, uint8(70+50*sin(phase*3))), 12)
	}
	// antennae sweep forward
	for s := -1; s <= 1; s += 2 {
		tip := v2(p.X+dir*(15.0+2.0*sin(phase*2.2*flail+float64(s))), p.Y-4*squash+float64(s)*3.0+2.0*sin(phase*3.1*flail+float64(s)))
		strokeQuads(m, []contract.Vec2{p, tip}, 0.8, withA(cCritterPale, uint8(uint16(a)*7/10)))
	}
	// swimmerets under the belly
	for i := 0; i < 3; i++ {
		wig := sin(phase*4*flail+float64(i)) * 1.4 * flail
		root := v2(p.X-dir*(1.5+float64(i)*2.6), p.Y+2.6*squash)
		strokeQuads(m, []contract.Vec2{root, v2(root.X+wig*0.4, root.Y+3.2*squash)}, 0.8, withA(cCritterLeg, a))
	}
	// arced body: head segment + belly + fan tail
	oval(m, p, 8.4, 3.6*squash, withA(cCritterShell, a), 12)
	oval(m, v2(p.X-dir*2.4, p.Y+1.0*squash), 5.6, 2.6*squash, withA(cCritterShell2, uint8(uint16(a)*8/10)), 10)
	tail := v2(p.X-dir*9.6, p.Y+0.6*squash+sin(phase*2.4*flail)*1.2)
	for s := -1; s <= 1; s += 2 {
		m.triT(tail, v2(tail.X-dir*4.4, tail.Y+float64(s)*2.8*squash+0.6),
			v2(tail.X-dir*3.4, tail.Y+float64(s)*0.6), withA(cCritterShell2, a))
	}
}
