// G1.4: spine-based fish renderer — tapered gradient body, animated fins,
// five on-body pattern types, glowing eye and night bioluminescence.
package render

import (
	"image/color"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/hajimehoshi/ebiten/v2"
)

// FishAnim carries per-frame animation inputs.
// stageAlpha tunes body opacity per life stage (fry are translucent).

// patternSeed derives a stable per-species RNG seed from the species ID so
// every fish of a species shares deterministic pattern placement.
func patternSeed(id string) int64 {
	var h int64 = 1469598103934665603
	for _, r := range id {
		h = (h ^ int64(r)) * 1099511628211
	}
	return h
}

// FishBatch accumulates many fish into two shared meshes — the opaque body
// parts onto the scene, the additive glow bits onto the trail image. The
// whole school then costs ~2 draw calls instead of ~5 per fish (F29 perf
// guard: the CPU profile put 86% of frame time into per-call driver
// submission, not triangles). Only normals are batched; the Chosen keeps her
// immediate path so her glow extras stay untouched.
type FishBatch struct {
	opaque mesh
	add    mesh
	n      int
}

// Reset empties the batch (called implicitly by Flush).
func (b *FishBatch) Reset() {
	b.opaque.reset()
	b.add.reset()
	b.n = 0
}

// Draw appends one fish to the batch. Nothing is rasterized until Flush —
// the arguments and the rendered result match DrawFish.
func (b *FishBatch) Draw(dst, glowDst *ebiten.Image, spine []contract.Vec2, spec *contract.Species,
	pal *contract.Palette, stage string, night float64, anim FishAnim) {
	if dst == nil || len(spine) < 3 || spec == nil || pal == nil {
		return
	}
	if anim.Attached { // F18: the flat pleco pose draws immediately (rare)
		drawSuckerFish(dst, glowDst, spine, spec, pal, stage, night, anim)
		return
	}
	hide := clampF(anim.Hide01, 0, 1)
	if hide >= 1 { // F25: fully inside the crag — nothing to rasterize
		return
	}
	// G93: the glass gaze — at full blend the face-on pose REPLACES the
	// side mesh; mid-blend both draw, alpha-weighted (a crossfade inside
	// the shared batch, so the lane sort stays honest).
	gaze := clampF(anim.Gaze01, 0, 1)
	if gaze >= 0.85 {
		drawFaceOn(b, spine, spec, pal, stage, night, anim, gaze)
		b.n++
		return
	}
	n := len(spine)

	// cumulative body length + per-point direction/normals
	segs := make([]contract.Vec2, n-1)
	norms := make([]contract.Vec2, n)
	lens := make([]float64, n-1)
	bodyLen := 0.0
	for i := 0; i < n-1; i++ {
		dx := spine[i+1].X - spine[i].X
		dy := spine[i+1].Y - spine[i].Y
		l := sqrt(maxF(dx*dx+dy*dy, 1e-6))
		segs[i] = v2(dx/l, dy/l)
		lens[i] = l
		bodyLen += l
	}
	for i := 0; i < n; i++ {
		s := segs[min(i, n-2)]
		norms[i] = v2(-s.Y, s.X)
	}
	// v1.1 G58: the scalare read — a TALL, laterally flat diamond whose
	// height rivals its length; the dorsal and anal fins tower above it
	peakW := bodyLen * 0.16 * contract.Clamp(spec.Width, 0.5, 1.6)
	finK := 1.0
	if spec.Role == contract.RoleTitan {
		peakW = bodyLen * contract.TitanTallPeak
		finK = contract.TitanFinScale
	}

	// v1.1: depth lanes — far fish sit slightly smaller and dimmer, near
	// fish full size and alpha (the 3D read; Z 0 counts as mid lane)
	z := anim.Z
	if z == 0 {
		z = 0.5
	}
	peakW *= contract.DepthFarScale + (1-contract.DepthFarScale)*z
	aDepth := contract.DepthNearAlpha + (1-contract.DepthNearAlpha)*z

	// width profile: rounded head, max at ~30%, needle tail
	widths := make([]float64, n)
	for i := 0; i < n; i++ {
		u := float64(i) / float64(n-1)
		widths[i] = peakW*sin(3.141592653589793*contract.Clamp(u, 0.02, 0.999)) + 0.6
	}

	bodyC := hexRGBA(pal.Body)
	bellyC := hexRGBA(pal.Belly)
	accentC := hexRGBA(pal.Accent)
	aMul := stageAlpha(stage)

	// F7: the slow fade of old age — desaturated colors, lower alpha
	if p := clampF(anim.ElderP, 0, 1); p > 0 {
		k := 1 - 0.6*p
		bodyC = desat(bodyC, k)
		bellyC = desat(bellyC, k)
		accentC = desat(accentC, k)
		aMul = uint8(float64(aMul) * (1 - 0.31*p))
	}
	aMul = uint8(float64(aMul) * (1 - hide))            // F25: transit fade
	aMul = uint8(float64(aMul) * (1 - anim.PortalFade)) // G73: wormhole pass
	aMul = uint8(float64(aMul) * aDepth)                // v1.1: depth lane dimming
	aMul = uint8(float64(aMul) * (1 - gaze))            // G93: the side view yields to the face-on pose

	// ---- fins (behind the body, translucent, gently swaying) ----
	var fins mesh
	finC := withA(scaleRGBA(bodyC, 1.05), uint8(120*(1-hide)))
	d0 := spinePoint(spine, norms, widths, 0.28, 1)
	d1 := spinePoint(spine, norms, widths, 0.52, 1)
	peak := spinePoint(spine, norms, widths, 0.40, 1)
	dSway := sin(anim.Time*3.1+1) * bodyLen * 0.012
	fins.triT(add(d0, v2(0, -bodyLen*0.09*spec.Fin*finK+dSway)), d1, add(peak, v2(0, -bodyLen*0.16*spec.Fin*finK-dSway)), finC)
	p0 := spinePoint(spine, norms, widths, 0.20, -1)
	flap := sin(anim.Time*5.2) * bodyLen * 0.02
	pDir := segs[min(1, n-2)]
	fins.triT(p0, add(p0, v2(pDir.X*bodyLen*0.10, bodyLen*0.055*spec.Fin*finK+flap)),
		add(p0, v2(-pDir.Y*bodyLen*0.05, pDir.X*bodyLen*0.08*spec.Fin*finK+flap)), finC)
	// G58: the anal fin mirrors the dorsal below the belly
	a0 := spinePoint(spine, norms, widths, 0.42, -1)
	a1 := spinePoint(spine, norms, widths, 0.60, -1)
	ap := spinePoint(spine, norms, widths, 0.50, -1)
	fins.triT(add(a0, v2(0, bodyLen*0.05*spec.Fin*finK+dSway)), a1,
		add(ap, v2(0, bodyLen*0.11*spec.Fin*finK-dSway)), finC)
	// G58: the scalare's trailing ventral streamers — two long filaments
	// sweep down and back off the belly, swaying slower than the fins
	if spec.Role == contract.RoleTitan {
		strC := withA(scaleRGBA(bellyC, 0.98), uint8(165*(1-hide)))
		for k, u := range [2]float64{0.24, 0.34} {
			root := spinePoint(spine, norms, widths, u, -1)
			swayS := sin(anim.Time*2.1+float64(k)*1.3) * bodyLen * 0.045
			end := add(root, add(mul(pDir, bodyLen*0.13), v2(swayS, bodyLen*0.30)))
			strokeQuads(&fins, []contract.Vec2{root, end}, 1.6, strC)
		}
	}
	tip := spine[n-1]
	td := segs[n-2]
	// G72: the caudal fin is bone-mounted — the sweep stays a narrow wag
	// around the last vertebra, never a circular propeller swing
	swish := sin(anim.Time*(4+7*clampF(anim.Speed01, 0, 1))) * (0.20 + 0.14*anim.Speed01)
	tailLen := bodyLen * 0.24 * contract.Clamp(spec.Tail, 0.5, 1.6)
	tailC := withA(scaleRGBA(accentC, 1.0), uint8(150*(1-hide)))
	rot := func(v contract.Vec2, a float64) contract.Vec2 {
		return v2(v.X*cos(a)-v.Y*sin(a), v.X*sin(a)+v.Y*cos(a))
	}
	back := v2(-td.X, -td.Y)
	l1 := add(tip, v2(rot(back, swish-0.30).X*tailLen, rot(back, swish-0.30).Y*tailLen))
	l2 := add(tip, v2(rot(back, swish).X*tailLen*1.12, rot(back, swish).Y*tailLen*1.12))
	l3 := add(tip, v2(rot(back, swish+0.30).X*tailLen, rot(back, swish+0.30).Y*tailLen))
	fins.triT(tip, l1, l2, tailC)
	fins.triT(tip, l2, l3, tailC)
	b.opaque.merge(&fins)

	// ---- body strip with head→tail + belly gradients ----
	var body mesh
	headC := scaleRGBA(bodyC, 1.18)
	midC := bodyC
	tailBodyC := scaleRGBA(bodyC, 0.62)
	var prevTop, prevBot contract.Vec2
	var prevTopC, prevBotC colorRGBA
	for i := 0; i < n; i++ {
		u := float64(i) / float64(n-1)
		along := lerp3(headC, midC, tailBodyC, u)
		alongTop := along
		alongBot := lerpRGBA(along, bellyC, 0.55)
		if i > 0 {
			// perimeter order (prevTop→top→bot→prevBot) — a bowtie here would
			// cancel under the v2.10 fill rule (same bug the plants had)
			body.quad(prevTop,
				add(spine[i], mul(norms[i], widths[i])),
				add(spine[i], mul(norms[i], -widths[i])),
				prevBot,
				prevTopC, withA(alongTop, aMul), withA(alongBot, aMul), prevBotC)
		}
		prevTop = add(spine[i], mul(norms[i], widths[i]))
		prevBot = add(spine[i], mul(norms[i], -widths[i]))
		prevTopC, prevBotC = withA(alongTop, aMul), withA(alongBot, aMul)
	}
	b.opaque.merge(&body)

	// ---- pattern overlay (built along the spine, stays inside the body) ----
	drawPattern(b, spine, segs, norms, widths, bodyLen, spec, pal, stage, night, anim, aMul)

	// v1.1: the hammerhead — a crossbar rostrum ahead of the head with an
	// eye at each tip, plus the wild dress: predator dorsal, breathing
	// gill slits, hunter's eyes (G71)
	if spec.Role == contract.RoleShark {
		drawSharkWild(b, spine, segs, norms, widths, peakW, bodyLen, aMul, bodyC, anim)
		return
	}

	// eye on the head, slightly above the midline — a tall scalare caps the
	// eye at a proportional size (G58)
	eyePos := add(spine[0], add(mul(segs[0], bodyLen*0.045), mul(norms[0], widths[0]*0.18)))
	eyeR := maxF(peakW*0.24, 1.6)
	if spec.Role == contract.RoleTitan {
		eyeR = min(eyeR, bodyLen*0.05)
	}
	var eye mesh
	eye.fan(eyePos, eyeR, color.RGBA{R: 235, G: 250, B: 255, A: aMul}, 10)
	eye.fan(add(eyePos, mul(segs[0], peakW*0.06)), maxF(eyeR*0.5, 0.9), color.RGBA{R: 8, G: 10, B: 22, A: aMul}, 8)
	b.opaque.merge(&eye)
	b.n++
}

// Flush rasterizes the batch: one normal draw for every fish body, one
// additive draw for every glow bit. Resets the batch.
func (b *FishBatch) Flush(dst, glowDst *ebiten.Image) {
	if len(b.opaque.indices) == 0 && len(b.add.indices) == 0 {
		return
	}
	b.opaque.drawAliased(dst)
	if glowDst != nil && len(b.add.indices) > 0 {
		b.add.draw(glowDst, true)
	}
	b.Reset()
}

// DrawFish draws one fish immediately (not through the shared batch). Used
// for the Chosen — her rim light, aurora shimmer and bioluminescence are
// her own glow paths (F16) and she is always exactly one fish.
func DrawFish(dst, glowDst *ebiten.Image, spine []contract.Vec2, spec *contract.Species,
	pal *contract.Palette, stage string, night float64, anim FishAnim) {
	var b FishBatch
	b.Draw(dst, glowDst, spine, spec, pal, stage, night, anim)
	b.Flush(dst, glowDst)
	if !GlowVisible(spec) || anim.Attached || len(spine) < 3 {
		return
	}
	// the eternal one's aurora shimmer rides on top of the pattern
	n := len(spine)
	segs := make([]contract.Vec2, n-1)
	norms := make([]contract.Vec2, n)
	bodyLen := 0.0
	for i := 0; i < n-1; i++ {
		dx := spine[i+1].X - spine[i].X
		dy := spine[i+1].Y - spine[i].Y
		l := sqrt(maxF(dx*dx+dy*dy, 1e-6))
		segs[i] = v2(dx/l, dy/l)
		bodyLen += l
	}
	for i := 0; i < n; i++ {
		s := segs[min(i, n-2)]
		norms[i] = v2(-s.Y, s.X)
	}
	peakW := bodyLen * 0.16 * contract.Clamp(spec.Width, 0.5, 1.6)
	widths := make([]float64, n)
	for i := 0; i < n; i++ {
		u := float64(i) / float64(n-1)
		widths[i] = peakW*sin(3.141592653589793*contract.Clamp(u, 0.02, 0.999)) + 0.6
	}
	drawChosenShimmer(dst, glowDst, spine, norms, widths, anim.Time)
	glowC := hexRGBA(pal.Glow)
	var rim mesh
	rimC := withA(glowC, uint8((70+110*clampF(night, 0, 1))*0.4))
	strokeSpine(&rim, spine, norms, widths, 1.2, rimC)
	rim.draw(glowDst, true)
	eyePos := add(spine[0], add(mul(segs[0], bodyLen*0.045), mul(norms[0], widths[0]*0.18)))
	DrawGlow(glowDst, eyePos.X, eyePos.Y, peakW*1.4, pal.Glow, (0.10+0.14*night)*0.4)
	tip := spine[n-1]
	gA := (0.22 + 0.55*clampF(night, 0, 1)) * 0.30 * (1 - 0.7*clampF(anim.ElderP, 0, 1))
	DrawGlow(glowDst, spine[0].X, spine[0].Y, bodyLen*0.55, pal.Glow, gA)
	DrawGlow(glowDst, tip.X, tip.Y, bodyLen*0.32, pal.Accent, gA*0.8)
}
