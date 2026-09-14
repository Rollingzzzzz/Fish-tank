// v1.2 G93: the face-on stare. When a fish takes the glass she turns
// front-on: the body foreshortens to a rounded silhouette, BOTH eyes lock
// onto the viewer, the pectorals spread and sway. Drawn through the shared
// batch so the depth-lane sort stays honest; the side mesh crossfades out
// underneath (alpha-weighted in FishBatch.Draw).
package render

import (
	"image/color"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// drawFaceOn appends the front-view pose to the batch. g is the blend
// (0..1) — mid-blend it draws translucent over the fading side mesh.
func drawFaceOn(b *FishBatch, spine []contract.Vec2, spec *contract.Species,
	pal *contract.Palette, stage string, night float64, anim FishAnim, g float64) {
	if len(spine) < 3 || spec == nil || pal == nil {
		return
	}
	bodyLen := 0.0
	for i := 0; i < len(spine)-1; i++ {
		dx := spine[i+1].X - spine[i].X
		dy := spine[i+1].Y - spine[i].Y
		bodyLen += sqrt(dx*dx + dy*dy)
	}
	c := spine[0]
	rx := bodyLen * 0.15 * contract.Clamp(spec.Width, 0.5, 1.6)
	ry := bodyLen * 0.22
	if rx < 3 {
		return
	}

	// presence: stage, elder, depth and blend — the same dimming family
	// the side mesh uses, so the pose settles into the lane honestly
	aF := float64(stageAlpha(stage)) / 255
	aF *= 1 - 0.31*clampF(anim.ElderP, 0, 1)
	z := anim.Z
	if z == 0 {
		z = 0.5
	}
	aF *= contract.DepthNearAlpha + (1-contract.DepthNearAlpha)*z
	aF *= clampF(g, 0, 1)

	sway := sin(anim.Time*1.7) * 0.05
	bob := sin(anim.Time*1.3) * bodyLen * 0.012
	pt := func(px, py float64) contract.Vec2 {
		dx, dy := px, py+bob
		return v2(c.X+dx*cos(sway)-dy*sin(sway), c.Y+dx*sin(sway)+dy*cos(sway))
	}

	bodyC := hexRGBA(pal.Body)
	bellyC := hexRGBA(pal.Belly)
	accentC := hexRGBA(pal.Accent)
	finC := withA(scaleRGBA(bodyC, 1.05), uint8(120*aF))
	dark := color.RGBA{R: 10, G: 12, B: 26, A: uint8(255 * aF)}
	glint := color.RGBA{R: 240, G: 250, B: 255, A: uint8(230 * aF)}

	// pectorals — two translucent fans spread and sway; the "hands on the
	// glass" read
	flap := sin(anim.Time*3.2) * ry * 0.16
	pec := func(side float64) {
		root := pt(side*rx*0.55, ry*0.10)
		mid := pt(side*rx*1.75, ry*0.55+flap)
		tip := pt(side*rx*0.95, ry*0.75+flap*0.6)
		b.opaque.triT(root, mid, tip, finC)
	}
	pec(-1)
	pec(1)

	// dorsal hint above the crown
	b.opaque.triT(pt(-rx*0.45, -ry*0.85), pt(rx*0.45, -ry*0.85),
		pt(0, -ry*1.28), finC)

	// body — a rounded front silhouette as a triangle fan
	const segsN = 16
	prevX, prevY := rx, 0.0
	for k := 1; k <= segsN; k++ {
		a := float64(k) / segsN * 6.283185307
		x, y := rx*cos(a), ry*sin(a)
		b.opaque.triT(pt(0, 0), pt(prevX, prevY), pt(x, y),
			withA(bodyC, uint8(255*aF)))
		prevX, prevY = x, y
	}
	// belly sheen — a smaller ellipse shifted down, the light from above
	prevX, prevY = rx*0.55, ry*0.10
	for k := 1; k <= segsN; k++ {
		a := float64(k) / segsN * 6.283185307
		x, y := rx*0.55*cos(a), ry*0.10+ry*0.55*sin(a)
		b.opaque.triT(pt(0, ry*0.10), pt(prevX, prevY), pt(x, y),
			withA(bellyC, uint8(200*aF)))
		prevX, prevY = x, y
	}
	// accent collar — a thin arc under the head, the pattern's echo
	b.opaque.triT(pt(-rx*0.8, ry*0.28), pt(rx*0.8, ry*0.28), pt(0, ry*0.52),
		withA(accentC, uint8(110*aF)))

	// the STARE — two forward-facing eyes with glints, symmetric about the
	// body axis; the whole point of the behavior
	ex, ey := rx*0.45, -ry*0.12
	eyeR := maxF(ry*0.17, 1.8)
	for _, side := range [2]float64{-1, 1} {
		e := pt(side*ex, ey)
		b.opaque.fan(e, eyeR, dark, 10)
		g := pt(side*ex-eyeR*0.3, ey-eyeR*0.35)
		b.opaque.fan(g, maxF(eyeR*0.4, 0.8), glint, 6)
	}
	// mouth — a small dark wedge low on the silhouette
	b.opaque.triT(pt(-rx*0.16, ry*0.52), pt(rx*0.16, ry*0.52), pt(0, ry*0.72), dark)
}
