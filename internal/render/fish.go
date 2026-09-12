// G1.4: spine-based fish renderer — tapered gradient body, animated fins,
// five on-body pattern types, glowing eye and night bioluminescence.
package render

import (
	"image/color"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/hajimehoshi/ebiten/v2"
)

// FishAnim carries per-frame animation inputs.
type FishAnim struct {
	Time       float64 // global seconds
	Speed01    float64 // 0..1 normalized speed, drives tail beat
	ElderP     float64 // F7: 0..1 elder fade (desaturation + alpha + dim glow)
	Attached   bool    // F18: suctioned to the glass (flat sucker pose)
	AttachSide int     // F18: -1 left wall, +1 right wall (0 when free)
	Hide01     float64 // F25: binary (v0.3.8): 0 visible .. 1 inside a crag
}

// stageAlpha tunes body opacity per life stage (fry are translucent).
func stageAlpha(stage string) uint8 {
	switch stage {
	case "fry":
		return 190
	case "juvenile":
		return 228
	default:
		return 255
	}
}

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
	peakW := bodyLen * 0.16 * contract.Clamp(spec.Width, 0.5, 1.6)

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
	aMul = uint8(float64(aMul) * (1 - hide)) // F25: transit fade

	// ---- fins (behind the body, translucent, gently swaying) ----
	var fins mesh
	finC := withA(scaleRGBA(bodyC, 1.05), uint8(120*(1-hide)))
	d0 := spinePoint(spine, norms, widths, 0.28, 1)
	d1 := spinePoint(spine, norms, widths, 0.52, 1)
	peak := spinePoint(spine, norms, widths, 0.40, 1)
	dSway := sin(anim.Time*3.1+1) * bodyLen * 0.012
	fins.triT(add(d0, v2(0, -bodyLen*0.09*spec.Fin+dSway)), d1, add(peak, v2(0, -bodyLen*0.16*spec.Fin-dSway)), finC)
	p0 := spinePoint(spine, norms, widths, 0.20, -1)
	flap := sin(anim.Time*5.2) * bodyLen * 0.02
	pDir := segs[min(1, n-2)]
	fins.triT(p0, add(p0, v2(pDir.X*bodyLen*0.10, bodyLen*0.055*spec.Fin+flap)),
		add(p0, v2(-pDir.Y*bodyLen*0.05, pDir.X*bodyLen*0.08*spec.Fin+flap)), finC)
	tip := spine[n-1]
	td := segs[n-2]
	swish := sin(anim.Time*(4+7*clampF(anim.Speed01, 0, 1))) * (0.28 + 0.22*anim.Speed01)
	tailLen := bodyLen * 0.24 * contract.Clamp(spec.Tail, 0.5, 1.6)
	tailC := withA(scaleRGBA(accentC, 1.0), uint8(150*(1-hide)))
	rot := func(v contract.Vec2, a float64) contract.Vec2 {
		return v2(v.X*cos(a)-v.Y*sin(a), v.X*sin(a)+v.Y*cos(a))
	}
	back := v2(-td.X, -td.Y)
	l1 := add(tip, v2(rot(back, swish-0.38).X*tailLen, rot(back, swish-0.38).Y*tailLen))
	l2 := add(tip, v2(rot(back, swish).X*tailLen*1.12, rot(back, swish).Y*tailLen*1.12))
	l3 := add(tip, v2(rot(back, swish+0.38).X*tailLen, rot(back, swish+0.38).Y*tailLen))
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

	// eye on the head, slightly above the midline
	eyePos := add(spine[0], add(mul(segs[0], bodyLen*0.045), mul(norms[0], widths[0]*0.18)))
	var eye mesh
	eye.fan(eyePos, maxF(peakW*0.24, 1.6), color.RGBA{R: 235, G: 250, B: 255, A: aMul}, 10)
	eye.fan(add(eyePos, mul(segs[0], peakW*0.06)), maxF(peakW*0.12, 0.9), color.RGBA{R: 8, G: 10, B: 22, A: aMul}, 8)
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
