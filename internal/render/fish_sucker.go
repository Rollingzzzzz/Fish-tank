// F18: the sucker pose — a suctioned fish pressed flat against the glass:
// head into the wall, body widened ~1.25x, a strong pale belly band on the
// flank away from the glass, and a mouth disc pumping at ~1 Hz.
package render

import (
	"image/color"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/hajimehoshi/ebiten/v2"
)

// drawSuckerFish renders the attached "pleco on the glass" pose. The incoming
// spine only supplies the head anchor and body length; the pose is rebuilt
// straight into the wall so the fish always reads as clamped, whatever the
// swim history left in the spine.
func drawSuckerFish(dst, glowDst *ebiten.Image, spine []contract.Vec2, spec *contract.Species,
	pal *contract.Palette, stage string, night float64, anim FishAnim) {
	n := len(spine)
	// unit direction the head presses INTO (-1: left wall, +1: right wall)
	wd := v2(1, 0)
	if anim.AttachSide < 0 {
		wd = v2(-1, 0)
	}
	head := spine[0]
	bodyLen := hyp(spine[n-1].X-head.X, spine[n-1].Y-head.Y)
	segLen := bodyLen / float64(n-1)

	// straight spine away from the glass with a gentle suction ripple
	sp := make([]contract.Vec2, n)
	for i := 0; i < n; i++ {
		u := float64(i) / float64(n-1)
		off := sin(u*4.2+anim.Time*1.7) * (0.3 + 0.7*u)
		sp[i] = v2(head.X-wd.X*segLen*float64(i), head.Y+off)
	}
	// directions + normals of the synthetic spine (constant, it is straight)
	segs := make([]contract.Vec2, n-1)
	norms := make([]contract.Vec2, n)
	for i := range segs {
		segs[i] = v2(-wd.X, -wd.Y)
	}
	for i := range norms {
		norms[i] = v2(-segs[min(i, n-2)].Y, segs[min(i, n-2)].X)
	}

	// F18: the pressed pose runs a hand-width wider than swimming
	peakW := bodyLen * 0.16 * contract.Clamp(spec.Width, 0.5, 1.6) * 1.25
	widths := make([]float64, n)
	for i := 0; i < n; i++ {
		u := float64(i) / float64(n-1)
		widths[i] = peakW*sin(3.141592653589793*contract.Clamp(u, 0.02, 0.999)) + 0.6
	}

	bodyC := hexRGBA(pal.Body)
	bellyC := hexRGBA(pal.Belly)
	glowC := hexRGBA(pal.Glow)
	aMul := stageAlpha(stage)
	if p := clampF(anim.ElderP, 0, 1); p > 0 { // F7 elder fade, same as free fish
		k := 1 - 0.6*p
		bodyC, bellyC, glowC = desat(bodyC, k), desat(bellyC, k), desat(glowC, k)
		aMul = uint8(float64(aMul) * (1 - 0.31*p))
	}

	// the pale band rides the flank away from the glass — the pressed
	// underside (screen-down edge: +norm at the left wall, -norm at right)
	bellySide := 1.0
	if wd.X > 0 {
		bellySide = -1.0
	}

	// clamped fins — one small pectoral fluttering near the head only
	var fins mesh
	finC := withA(scaleRGBA(bodyC, 1.05), 100)
	flutter := sin(anim.Time*4.6) * bodyLen * 0.010
	p0 := spinePoint(sp, norms, widths, 0.20, bellySide)
	fins.triT(p0,
		add(p0, v2(wd.X*bodyLen*0.06, bellySide*(bodyLen*0.05+flutter))),
		add(p0, v2(wd.X*bodyLen*0.02, bellySide*(bodyLen*0.075-flutter))), finC)
	fins.draw(dst, false)

	// body strip — head→tail gradient with the strong pale belly band
	var body mesh
	headC, midC, tailBodyC := scaleRGBA(bodyC, 1.18), bodyC, scaleRGBA(bodyC, 0.62)
	bandC := scaleRGBA(bellyC, 1.35) // F18: clearly visible pale band
	var prevT, prevB contract.Vec2
	var prevTC, prevBC colorRGBA
	for i := 0; i < n; i++ {
		u := float64(i) / float64(n-1)
		along := lerp3(headC, midC, tailBodyC, u)
		alongT, alongB := along, along
		if bellySide > 0 { // +norm edge is the belly edge
			alongT = lerpRGBA(along, bandC, 0.80)
		} else {
			alongB = lerpRGBA(along, bandC, 0.80)
		}
		if i > 0 {
			// perimeter order — a bowtie here would cancel (v2.10 fill rule)
			body.quad(prevT, add(sp[i], mul(norms[i], widths[i])),
				add(sp[i], mul(norms[i], -widths[i])), prevB,
				prevTC, withA(alongT, aMul), withA(alongB, aMul), prevBC)
		}
		prevT = add(sp[i], mul(norms[i], widths[i]))
		prevB = add(sp[i], mul(norms[i], -widths[i]))
		prevTC, prevBC = withA(alongT, aMul), withA(alongB, aMul)
	}
	body.draw(dst, false)

	// the species pattern still rides the pressed flank
	var pb FishBatch
	drawPattern(&pb, sp, segs, norms, widths, bodyLen, spec, pal, stage, night, anim, aMul)
	pb.Flush(dst, glowDst)

	// eye above the mouth end
	eyePos := add(sp[0], v2(wd.X*bodyLen*0.05, -widths[0]*0.30))
	var eye mesh
	eye.fan(eyePos, maxF(peakW*0.20, 1.4), color.RGBA{R: 235, G: 250, B: 255, A: aMul}, 10)
	eye.fan(eyePos, maxF(peakW*0.10, 0.8), color.RGBA{R: 8, G: 10, B: 22, A: aMul}, 8)
	eye.draw(dst, false)

	// F18: mouth disc pressing into the glass — the aperture pumps through
	// one full open/close cycle per second (~1 Hz), subtle and even.
	pump := 0.5 + 0.5*sin(anim.Time*6.2831853)
	mouthR := maxF(peakW*0.34, 1.8) * (0.62 + 0.38*pump)
	mouthPos := add(sp[0], v2(wd.X*maxF(peakW*0.10, 1.0), 0))
	var lip, disc, hole mesh // separate meshes: they overlap (fill rule)
	lip.fan(mouthPos, mouthR+1.4, withA(scaleRGBA(bodyC, 0.72), aMul), 12)
	lip.draw(dst, false)
	disc.fan(mouthPos, mouthR, withA(scaleRGBA(bellyC, 1.6), aMul), 12)
	disc.draw(dst, false)
	hole.fan(mouthPos, mouthR*(0.55-0.35*pump), withA(color.RGBA{R: 20, G: 14, B: 30, A: aMul}, 90), 10)
	hole.draw(dst, false)

	// F16: neon stays the Chosen's alone, also in the pressed pose
	if GlowVisible(spec) {
		DrawGlow(glowDst, sp[0].X, sp[0].Y, bodyLen*0.5, pal.Glow, 0.20*(1+night))
	}
}
