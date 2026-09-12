// G1.4/F16: species pattern overlay — stripe / spot / koi / vein / wave,
// built along the spine so no clipping is needed. The vein glows only for
// the Chosen; normal fish render it as a plain alpha stroke (F16).
package render

import (
	"image/color"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// drawPattern paints the species pattern along the spine; every shape is
// constructed between the top/bottom body edges so no clipping is needed.
// Pieces merge into the fish batch (F29): one draw for the whole school.
func drawPattern(b *FishBatch, spine []contract.Vec2, segs, norms []contract.Vec2,
	widths []float64, bodyLen float64, spec *contract.Species, pal *contract.Palette,
	stage string, night float64, anim FishAnim, aMul uint8) {
	rng := contract.RandSeed(patternSeed(spec.ID))
	accentC := hexRGBA(pal.Accent)
	glowC := hexRGBA(pal.Glow)
	whiteC := color.RGBA{R: 245, G: 250, B: 255, A: aMul}
	dens := contract.Clamp(spec.Pattern.Density, 0, 1)
	psz := contract.Clamp(spec.Pattern.Size, 0, 1)

	// v2.10 fill rule: overlapping pattern pieces draw separately.
	switch spec.Pattern.Type {
	case "stripe":
		count := 3 + int(dens*5)
		for k := 0; k < count; k++ {
			t := 0.16 + (0.70 * float64(k) / float64(maxF(float64(count-1), 1)))
			t += (rng.Float64() - 0.5) * 0.02
			wd := widthsAt(spine, norms, widths, t) * (0.28 + 0.5*psz)
			c := accentC
			if k%3 == 2 {
				c = whiteC
			}
			var m mesh
			stripeQuad(&m, spine, norms, widths, t, wd, withA(c, uint8(200*float64(aMul)/255)), bodyLen*0.02)
			b.opaque.merge(&m)
		}
	case "spot":
		count := 3 + int(dens*6)
		for k := 0; k < count; k++ {
			t := 0.18 + rng.Float64()*0.66
			lat := (rng.Float64()*2 - 1) * 0.62
			r := widthsAt(spine, norms, widths, t) * (0.16 + 0.42*psz)
			p := spinePoint(spine, norms, widths, t, lat)
			var m mesh
			m.fan(p, r, withA(accentC, uint8(185*float64(aMul)/255)), 10)
			b.opaque.merge(&m)
		}
	case "koi":
		for k := 0; k < 3; k++ {
			t := 0.22 + 0.26*float64(k) + (rng.Float64()-0.5)*0.06
			p := spinePoint(spine, norms, widths, t, (rng.Float64()*2-1)*0.35)
			c := accentC
			if k == 1 {
				c = whiteC
			}
			var m mesh
			m.fan(p, widthsAt(spine, norms, widths, t)*(0.75+0.45*psz), withA(c, uint8(190*float64(aMul)/255)), 14)
			b.opaque.merge(&m)
		}
	case "vein":
		// lateral vein + two branches. F16: the Chosen paints it additively
		// into the glow layer; normal fish keep the very same geometry as a
		// plain alpha stroke on the body — visible, just not neon.
		steps := 10
		var pts []contract.Vec2
		for s := 0; s <= steps; s++ {
			t := 0.08 + 0.84*float64(s)/float64(steps)
			lat := sin(t*7+float64(len(spec.ID))) * 0.45
			pts = append(pts, spinePoint(spine, norms, widths, t, lat))
		}
		branches := make([][]contract.Vec2, 2)
		for b := 0; b < 2; b++ {
			t0 := 0.25 + 0.35*float64(b)
			branches[b] = []contract.Vec2{
				spinePoint(spine, norms, widths, t0, 0),
				spinePoint(spine, norms, widths, t0+0.07, 0.55),
				spinePoint(spine, norms, widths, t0+0.12, 0.85),
			}
		}
		if GlowVisible(spec) {
			var gm mesh
			strokeQuads(&gm, pts, 1.6, withA(glowC, uint8(140*0.3*float64(aMul)/255)))
			for b := 0; b < 2; b++ {
				strokeQuads(&gm, branches[b], 1.1, withA(glowC, uint8(105*0.3*float64(aMul)/255)))
			}
			b.add.merge(&gm)
		} else {
			var vm mesh
			strokeQuads(&vm, pts, 1.7, withA(glowC, uint8(175*float64(aMul)/255)))
			for b := 0; b < 2; b++ {
				strokeQuads(&vm, branches[b], 1.2, withA(accentC, uint8(150*float64(aMul)/255)))
			}
			b.opaque.merge(&vm)
		}
	case "wave":
		count := 3 + int(dens*3)
		for k := 0; k < count; k++ {
			var pts []contract.Vec2
			for s := 0; s <= 8; s++ {
				t := 0.12 + 0.76*float64(s)/float64(8)
				lat := sin(t*9+float64(k)*1.8+anim.Time*0.4) * 0.55
				pts = append(pts, spinePoint(spine, norms, widths, t, lat))
			}
			var m mesh
			strokeQuads(&m, pts, 2.0+2.5*psz, withA(accentC, uint8(120*float64(aMul)/255)))
			b.opaque.merge(&m)
		}
	}
	_ = glowC
}
