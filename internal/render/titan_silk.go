// v1.2 G94: the silver elders wear WHITE SILK. Where school fish carry
// hard triangular fins, the giants trail mustache-like silk filaments —
// two long barbels off the snout and four soft veils off the crest and
// belly — that undulate with a slow traveling wave. The wave rides the
// sim's own swim-beat phase (FishAnim.Beat), so a cruising elder reads as
// a heavy, mesmerizing crawl and a striking one snaps into a fast flick
// without any render-side state.
package render

import (
	"image/color"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// drawTitanSilk appends the silk filaments to the fins mesh (drawn behind
// the body). silkWhite is palette-independent — the elders' silk is white.
func drawTitanSilk(fins *mesh, spine []contract.Vec2, norms []contract.Vec2,
	widths []float64, segs []contract.Vec2, bodyLen float64, anim FishAnim, hide float64) {
	if len(spine) < 3 {
		return
	}
	pDir := segs[min(1, len(segs)-1)] // points head → tail
	perp := v2(-pDir.Y, pDir.X)
	ampBase := bodyLen * 0.055 * (0.55 + 0.75*clampF(anim.Speed01, 0, 1))
	rate := 1.3 + 1.4*clampF(anim.Speed01, 0, 1)
	white := color.RGBA{R: 232, G: 240, B: 252, A: uint8(115 * (1 - hide))}
	whiteSoft := color.RGBA{R: 238, G: 244, B: 253, A: uint8(88 * (1 - hide))}

	// one silk strand: a tapered ribbon laid along dir from root, waving
	// with a traveling sine whose amplitude grows toward the tip
	strand := func(root, dir, lift contract.Vec2, length float64, phase float64, ampK float64, c colorRGBA) {
		n := contract.TitanSilkSegs
		pts := make([]contract.Vec2, n+1)
		for k := 0; k <= n; k++ {
			t := float64(k) / float64(n)
			travel := length * t
			wave := sin(anim.Beat*rate-t*2.6+phase) * ampBase * ampK * (0.2 + 0.8*t)
			droop := lift.Y * t * bodyLen * 0.01
			pts[k] = v2(
				root.X+dir.X*travel+perp.X*wave,
				root.Y+dir.Y*travel+perp.Y*wave+droop,
			)
		}
		strokeQuads(fins, pts, 2.6, c)
		strokeQuads(fins, pts[n/2:], 1.3, c) // the taper
	}

	// the MUSTACHE — two long barbels off the snout, sweeping back past
	// the cheeks; the elder's signature
	for _, side := range [2]float64{-1, 1} {
		root := spinePoint(spine, norms, widths, 0.05, side)
		strand(root, pDir, v2(side*0.0, side*0.5), bodyLen*0.30,
			anim.Beat*0.0+side*0.7, 1.15, white)
	}
	// the veils — two off the crest flowing up-back, two off the belly
	// flowing down-back; softer, slower, layered
	type veil struct {
		u, side, liftY, lenK, phase float64
	}
	for _, v := range []veil{
		{0.30, 1, -0.9, 0.24, 0.0},
		{0.46, 1, -0.7, 0.20, 1.4},
		{0.34, -1, 1.1, 0.26, 2.1},
		{0.50, -1, 0.9, 0.22, 3.0},
	} {
		root := spinePoint(spine, norms, widths, v.u, v.side)
		strand(root, pDir, v2(0, v.liftY), bodyLen*v.lenK, v.phase, 0.85, whiteSoft)
	}
}
