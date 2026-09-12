// v0.3.7 (F30): two new plant silhouettes — FEATHER (a sea-pen like plume:
// one central stalk with dense silky barbs) and SILK (fine hair threads that
// sway like soft hair in a current). Both matte (F17): the seas grow strange
// soft flora, but neon still belongs to the Chosen alone.
package render

import (
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/hajimehoshi/ebiten/v2"
)

// drawFeatherPlant paints one central plume per frond: a tapered stalk with
// a curved tip, and dense barbs feathering out both sides along its length.
func drawFeatherPlant(dst *ebiten.Image, p *contract.PlantDesign, x, y, heightPx, t, shade float64) {
	sway := contract.Clamp(p.Sway, 0, 1)
	curve := contract.Clamp(p.Curve, 0, 1)
	swayAmp := heightPx * (0.05 + 0.08*sway)
	swaySpeed := 0.5 + 0.9*sway
	spread := 16 * contract.Clamp(p.Width, 0.5, 1.5)

	for f := 0; f < p.Fronds; f++ {
		hv := plantHash(f)
		side := 1.0
		if f%2 == 1 {
			side = -1.0
		}
		h := heightPx * (0.70 + 0.55*hv)
		baseX := x + float64(f-(p.Fronds-1)/2)*spread + (hv-0.5)*8
		ph := hv * 6.283
		swayOff := sin(t*swaySpeed+ph) * swayAmp

		// the stalk: a slim tapered ribbon along one bezier
		p0 := v2(baseX, y)
		p1 := v2(baseX+curve*h*0.10*side+swayOff*0.4, y-h*0.55)
		p2 := v2(baseX+curve*h*0.30*side+swayOff, y-h)
		stalkW := 2.6 * contract.Clamp(p.Width, 0.5, 1.5)

		// barbs: dense silky strokes rooted along the stalk, sweeping upward
		// — short, near-vertical and slightly bowed so they read as soft
		// plume texture, never as thorny twigs
		const bn = 18
		var barbs mesh
		for b := 1; b <= bn; b++ {
			u := float64(b) / float64(bn+1)
			bx := bez(p0.X, p1.X, p2.X, u)
			by := bez(p0.Y, p1.Y, p2.Y, u)
			bend := swayOff * u * 0.8
			len_ := h * (0.26 - 0.13*u) * (0.85 + 0.3*plantHash(b+7*f))
			col := dimCoolRGBA(multiStop(p.Colors, u, uint8(110+90*(1-u))), shade)
			for _, sgn := range [2]float64{-1, 1} {
				tipX := bx + sgn*len_*(0.30+0.22*curve) + bend
				tipY := by - len_*(0.95-0.18*curve)
				mx2, my2 := (bx+tipX)/2+sgn*len_*0.10, (by+tipY)/2-len_*0.08
				strokeQuads(&barbs, []contract.Vec2{v2(bx, by), v2(mx2, my2), v2(tipX, tipY)}, 1.0, col)
			}
		}
		barbs.draw(dst, false)

		// the stalk rides OVER its barbs
		var stem mesh
		prevL, prevR := p0, p0
		var prevC colorRGBA
		const segs = 20
		for s := 1; s <= segs; s++ {
			u := float64(s) / float64(segs)
			cx2, cy2 := bez(p0.X, p1.X, p2.X, u), bez(p0.Y, p1.Y, p2.Y, u)
			col := dimCoolRGBA(multiStop(p.Colors, u, 225), shade)
			w := stalkW * (1 - 0.7*u)
			l := v2(cx2+w, cy2)
			r := v2(cx2-w, cy2)
			stem.quad(prevL, l, r, prevR, prevC, col, col, prevC)
			prevL, prevR, prevC = l, r, col
		}
		stem.draw(dst, false)
	}
}

// drawSilkPlant paints hair-fine threads clumping like soft hair — many
// thin bowed strokes, strong sway, layered alpha so they read silky, not wiry.
func drawSilkPlant(dst *ebiten.Image, p *contract.PlantDesign, x, y, heightPx, t, shade float64) {
	sway := contract.Clamp(p.Sway, 0, 1)
	curve := contract.Clamp(p.Curve, 0, 1)
	swayAmp := heightPx * (0.10 + 0.14*sway)
	swaySpeed := 0.7 + 1.3*sway
	spread := 8 * contract.Clamp(p.Width, 0.5, 1.5)
	threads := p.Fronds * 4

	for f := 0; f < threads; f++ {
		hv := plantHash(f)
		side := 1.0
		if f%2 == 1 {
			side = -1.0
		}
		h := heightPx * (0.55 + 0.70*hv)
		baseX := x + float64(f-(threads-1)/2)*spread*0.35 + (hv-0.5)*5
		ph := hv * 6.283
		swayOff := sin(t*swaySpeed+ph) * swayAmp
		h0, h1 := y-h*0.5, y-h
		p1 := v2(baseX+curve*h*0.34*side+swayOff*0.35, h0)
		p2 := v2(baseX+curve*h*0.85*side+swayOff, h1)

		// two passes per thread: a soft halo pass and a brighter core
		pts := make([]contract.Vec2, 0, 9)
		const segs = 8
		for s := 0; s <= segs; s++ {
			u := float64(s) / float64(segs)
			pts = append(pts, v2(bez(baseX, p1.X, p2.X, u), bez(y, h0, h1, u)))
		}
		var soft, core mesh
		mid := dimCoolRGBA(multiStop(p.Colors, 0.5, 60), shade)
		tip := dimCoolRGBA(multiStop(p.Colors, 1, 40), shade)
		strokeQuads(&soft, pts, 2.4, mid)
		soft.draw(dst, false)
		root := dimCoolRGBA(multiStop(p.Colors, 0.15, 170), shade)
		strokeQuads(&core, pts[:5], 1.1, root)
		core.draw(dst, false)
		var tipM mesh
		strokeQuads(&tipM, pts[4:], 1.0, tip)
		tipM.draw(dst, false)
	}
}
