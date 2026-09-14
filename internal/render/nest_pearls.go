// F24: the oyster's crown — exactly three bobbing pearls that blaze when
// the Chosen visits, and the lilac motes her presence raises from the mouth.
// Split from nest.go (line ceiling).
package render

import (
	"math"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/hajimehoshi/ebiten/v2"
)

// drawNestPearls paints exactly three pearls inside the gape — bobbing,
// blazing when she visits (contact) — plus her rising lilac motes.
func drawNestPearls(dst *ebiten.Image, cx, mouthY, hw, oh, time, contact float64) {
	var pearlsMesh mesh
	pearlC, pearlHi := hexRGBA("#efe6ff"), hexRGBA("#ffffff")
	for i := 0; i < 3; i++ {
		u := float64(i) / 2.0
		bob := sin(time*1.2+float64(i)*2.1) * 4
		px := cx - hw*0.26 + hw*0.26*2*u // well inside the cup's mouth line
		py := mouthY - oh*0.03 + bob*0.6
		pr := 7.5 - float64(i%2)*1.5
		glowA := 0.20 + 0.10*sin(time*1.1+float64(i)*1.3) + 0.45*contact
		DrawGlow(dst, px, py, 18+14*contact, "#c9a8ff", glowA)
		pearlsMesh.fan(v2(px, py), pr, pearlC, 14)
		pearlsMesh.fan(v2(px-pr*0.3, py-pr*0.3), pr*0.35, pearlHi, 8)
		if contact > 0.05 { // cross-flare sparkle on her approach
			fl := (0.5 + 0.5*sin(time*3+float64(i)*2.4)) * contact
			a := time*0.8 + float64(i)*1.9
			dx, dy := cos(a)*(9+7*fl), sin(a)*(9+7*fl)*0.4
			strokeQuads(&pearlsMesh, []contract.Vec2{v2(px-dx, py-dy*0.4), v2(px+dx, py+dy*0.4)}, 1.1, withA(pearlHi, uint8(200*fl)))
			strokeQuads(&pearlsMesh, []contract.Vec2{v2(px-dy, py-dx*0.4), v2(px+dy, py+dx*0.4)}, 1.1, withA(pearlHi, uint8(160*fl)))
		}
	}
	pearlsMesh.draw(dst, false)

	// her moment: lilac motes rising from the open mouth
	if contact > 0.03 {
		var motes mesh
		moteC := hexRGBA("#d9baff")
		for i := 0; i < 10; i++ {
			ph := (time*0.30 + float64(i)*0.37)
			prog := ph - math.Floor(ph) // 0..1 rise
			mx := cx + sin(float64(i)*2.4+time*1.5)*hw*0.5
			my := mouthY + oh*0.25 - prog*oh*0.75
			a := clampF(sin(prog*math.Pi), 0, 1) * contact
			motes.fan(v2(mx, my), 2.2+1.5*sin(time*2+float64(i)), withA(moteC, uint8(180*a)), 6)
		}
		motes.draw(dst, true)
	}
}
