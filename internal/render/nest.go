// v0.3.6 (F24): the Chosen's home — a camera-facing open OYSTER, dead
// center. v0.3.7 (F27): ONE shell — the lid's scalloped lip interlocks with
// the cup rim, the gape is a layered cavity (no flat wedge), silky byssus
// threads bind the valves and trail onto the floor. v0.3.8: the shell shrank
// to scenery scale, moved up to the shared floor line and behind the school,
// and its shadow pocket lightened — it is a planted reef throne now, not a
// wall in front of the tank. Hers alone in the sim.
package render

import (
	"math"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/hajimehoshi/ebiten/v2"
)

// oysterSize picks the visual span: ~4-5 body-lengths of the Chosen so the
// shell reads as her palace without walling off the view (v0.3.8: was 470).
func (r *RockLayout) oysterSize() (ow, oh float64) {
	ow = clampF(360, 260, float64(r.W)*0.30)
	oh = ow * 0.78
	return ow, oh
}

// NestBaseY is the y the shell sits on: the shared floor line with the crags
// (v0.3.8 — one ground plane across the stage; the old window-bottom anchor
// read as a flat cutout pasted on the glass).
func (r *RockLayout) NestBaseY() float64 { return float64(r.H) * 0.925 }

// fluteShape perturbs a rim so the valves read as scalloped oyster shell:
// a fast ripple layered on a slow irregular wave, anchored to die out at the
// hinge ends (multiplied by sin(a) by the caller). side scales the amplitude
// per half so the shell is NOT mirror-symmetric (real oysters are lumpy).
// The envelope smooths the wave tips so no scallop point tears.
func fluteShape(k, side float64) float64 {
	env := 0.6 + 0.4*math.Sin(k*3.1+0.7) // breathe between shallow and deep flutes
	return (math.Sin(k*7.0)*6.8 + math.Sin(k*2.6+1.2)*4.6) * side * env
}

// DrawNest paints the giant oyster. contact = 0..1 proximity of the Chosen
// to the mouth — she lights the pearls, the tongue swells toward her, and
// soft lilac motes rise from the shell.
func (r *RockLayout) DrawNest(dst *ebiten.Image, time, contact float64) {
	if dst == nil {
		return
	}
	contact = clampF(contact, 0, 1)
	cx, baseY := r.Nest.X, r.NestBaseY()
	ow, oh := r.oysterSize()
	mouthY := r.Nest.Y // the mouth plane (cup lip level)
	hw := ow * 0.5
	breathe := 1 + 0.02*sin(time*0.9) // the mantle never stops breathing
	n := 26

	// ambient shadow pocket + ground contact shadow — the shell sits IN the
	// scene (F27), the old halo never made it belong to the background
	drawNestAmbient(dst, cx, baseY, hw, oh)
	// halo — v0.3.8: halved again; a whisper of sanctuary light, not a bloom
	DrawGlow(dst, cx, mouthY, ow*0.62, "#c9a8ff", 0.02+0.055*contact+0.012*sin(time*1.7))

	// ---- lower valve: a deep nacred cup facing the camera, fluted rim ----
	cupO, cupI := hexRGBA("#2b1d42"), hexRGBA("#6b4fa0")
	var cup mesh
	rimPt := func(i int) contract.Vec2 {
		a := math.Pi - math.Pi*float64(i)/float64(n) // π..0 sweeps left→right
		side := 1.15
		if i > n/2 {
			side = 0.82 // lopsided on purpose — no mirror shells in nature
		}
		fl := fluteShape(float64(i), side) * breathe // scalloped shell edge
		return v2(cx-hw*cos(a), baseY-(oh*0.30*breathe+fl)*sin(a))
	}
	ci := cup.vert(v2(cx, baseY), cupO)
	for i := 0; i <= n; i++ {
		cup.vert(rimPt(i), lerpRGBA(cupO, cupI, sin(math.Pi-float64(i)*math.Pi/float64(n))))
	}
	for i := 1; i <= n; i++ {
		cup.tri(ci, uint16(i), uint16(i+1))
	}
	cup.draw(dst, false)

	// radial fluting — raised ribs running from the hinge out to the rim
	var ribs mesh
	ribC := withA(hexRGBA("#150b26"), 110)
	for k := 1; k < 12; k++ {
		i := n * k / 12
		a := math.Pi - math.Pi*float64(i)/float64(n)
		inner := v2(cx-hw*0.10*cos(a), baseY-oh*0.04*sin(a))
		outer := rimPt(i)
		mid := v2((inner.X+outer.X)/2, (inner.Y+outer.Y)/2-4)
		strokeQuads(&ribs, []contract.Vec2{inner, mid, outer}, 2.2, ribC)
	}
	ribs.draw(dst, false)

	// growth arcs — quiet shell rings (an oyster grows one ring at a time)
	var rings mesh
	ringC := withA(hexRGBA("#9a7cc9"), 46)
	for k := 1; k <= 3; k++ {
		f := 0.35 + 0.2*float64(k)
		for i := 0; i < n; i++ {
			a0 := math.Pi - math.Pi*float64(i)/float64(n)
			a1 := math.Pi - math.Pi*float64(i+1)/float64(n)
			p0 := v2(cx-hw*cos(a0)*f, baseY-oh*0.30*sin(a0)*f)
			p1 := v2(cx-hw*cos(a1)*f, baseY-oh*0.30*sin(a1)*f)
			strokeQuads(&rings, []contract.Vec2{p0, p1}, 1.4, ringC)
		}
	}
	rings.draw(dst, false)

	// iridescent nacre sweep inside the cup — pink/cyan sheen over the purple
	var irid mesh
	for b := 0; b < 2; b++ {
		bandC := hexRGBA("#7fe9ff")
		if b == 1 {
			bandC = hexRGBA("#ff9fe0")
		}
		for i := 0; i < n; i++ {
			a0 := math.Pi - math.Pi*float64(i)/float64(n)
			a1 := math.Pi - math.Pi*float64(i+1)/float64(n)
			u := float64(i) / float64(n)
			ampl := clampF(sin(u*math.Pi*2+float64(b)*2.1)+0.6, 0, 1.2)
			f0 := 0.42 + 0.14*float64(b)
			irid.quad(
				v2(cx-hw*cos(a0)*f0, baseY-oh*0.30*sin(a0)*f0),
				v2(cx-hw*cos(a1)*f0, baseY-oh*0.30*sin(a1)*f0),
				v2(cx-hw*cos(a1)*(f0+0.18), baseY-oh*0.30*sin(a1)*(f0+0.18)),
				v2(cx-hw*cos(a0)*(f0+0.18), baseY-oh*0.30*sin(a0)*(f0+0.18)),
				withA(bandC, uint8(20*ampl)), withA(bandC, uint8(20*ampl)),
				withA(bandC, uint8(14*ampl)), withA(bandC, uint8(14*ampl)))
		}
	}
	irid.draw(dst, true)

	// nacre ridge along the fluted lip
	var lip mesh
	ridgeC := withA(hexRGBA("#c3a8e8"), 235)
	for i := 0; i < n; i++ {
		p0, p1 := rimPt(i), rimPt(i+1)
		in0 := v2(cx+(p0.X-cx)*0.965, p0.Y-3)
		in1 := v2(cx+(p1.X-cx)*0.965, p1.Y-3)
		lip.quad(v2(p0.X, p0.Y+2), v2(p1.X, p1.Y+2), in1, in0,
			ridgeC, ridgeC, ridgeC, ridgeC)
	}
	lip.draw(dst, false)

	// ---- the gape: a layered dark cavity between the valves (nest_lid.go)
	// — dense segments + nested tones so it reads as depth, never a wedge ----
	drawNestGape(dst, cx, mouthY, hw, oh)

	// ---- hinge heel + soft ligament shadow (nest_lid.go) ----
	drawNestHinge(dst, cx, baseY, mouthY, hw, oh)

	// ---- upper valve: tilted-open lid whose scalloped lip interlocks with
	// the cup rim (nest_lid.go) ----
	drawNestLid(dst, cx, mouthY, hw, oh, time, contact)

	// ---- byssus silk: the threads the oyster spins to hold itself — they
	// bind the two valves and trail onto the floor (nest_silk.go) ----
	drawNestSilk(dst, cx, baseY, hw, oh, time)

	// ---- mantle: a thin fleshy crescent tucked under the lid's lip — a
	// strip between two arcs (never a fan: wedges read as a scallop fan,
	// the old straight edge read as a plank), scalloped and breathing ----
	var mantle mesh
	mIn, mOut := hexRGBA("#452f66"), hexRGBA("#8a68c4")
	pts := 22
	innerPt := func(i int) contract.Vec2 {
		a := math.Pi - math.Pi*float64(i)/float64(pts)
		wave := sin(float64(i)*1.7+time*1.3)*2.5 + sin(float64(i)*0.9-time*0.8)*1.5
		return v2(cx-hw*0.80*cos(a), mouthY-oh*(0.045+0.085*sin(a))+wave)
	}
	outerPt := func(i int) contract.Vec2 {
		a := math.Pi - math.Pi*float64(i)/float64(pts)
		wave := sin(float64(i)*1.7+time*1.3)*3.2 + sin(float64(i)*0.9-time*0.8)*2.0
		sc := 3.5 * sin(float64(i)*2.6) // scalloped fringe tips
		return v2(cx-hw*0.85*cos(a), mouthY-oh*(0.075+0.115*sin(a))+wave-sc)
	}
	for i := 0; i < pts; i++ {
		a := math.Pi - math.Pi*float64(i)/float64(pts)
		c := lerpRGBA(mIn, mOut, sin(a))
		mantle.quad(innerPt(i), innerPt(i+1), outerPt(i+1), outerPt(i), c, c, c, c)
	}
	mantle.draw(dst, false)

	// ---- the tongue: satin, alive — a soft curl lolling out of the gape,
	// reaching toward her when she visits. F27: the bend is capped so the
	// spine never folds over itself — folded segments read as a shapeless
	// pink fan (the old shapeless triangle). ----
	var tongue mesh
	tBase, tTip := hexRGBA("#e8a9d8"), hexRGBA("#b57bff")
	tn := 16
	tLen := oh * 0.24 * (1 + 0.40*contact)
	tW := hw * 0.26 * (1 + 0.15*contact)
	lean := 0.45 + 0.20*contact // how far the curl leans toward her (radians off vertical)
	curl := 0.55 + 0.20*contact // tip curvature — gentle, never a fold
	// spine of the tongue: starts inside the gape, arcs out
	var tPts []contract.Vec2
	for i := 0; i <= tn; i++ {
		u := float64(i) / float64(tn)
		bend := lean*u + curl*easeOut(u)*0.35
		px := cx + sin(bend)*tLen*u
		py := (mouthY + oh*0.05) - cos(bend)*tLen*u
		tPts = append(tPts, v2(px, py))
	}
	// watertight strip: shared edge vertices between segments — shrinking
	// each quad's leading edge (the old 0.92 trick) left wedge-shaped gaps
	// that read as comb teeth against the dark cavity
	var lft, rgt []contract.Vec2
	for i := 0; i <= tn; i++ {
		u := float64(i) / float64(tn)
		w := tW * (1 - 0.70*u)
		lo, hi := i-1, i+1
		if lo < 0 {
			lo = 0
		}
		if hi > tn {
			hi = tn
		}
		a, b := tPts[lo], tPts[i]
		c := tPts[hi]
		dx, dy := c.X-a.X, c.Y-a.Y
		l := sqrt(maxF(dx*dx+dy*dy, 1e-6))
		nx, ny := -dy/l, dx/l
		lft = append(lft, v2(b.X+nx*w, b.Y+ny*w))
		rgt = append(rgt, v2(b.X-nx*w, b.Y-ny*w))
	}
	for i := 0; i < tn; i++ {
		u := float64(i) / float64(tn)
		c0 := lerpRGBA(tBase, tTip, u)
		c1 := lerpRGBA(tBase, tTip, u+1/float64(tn))
		tongue.quad(lft[i], rgt[i], rgt[i+1], lft[i+1], c0, c0, c1, c1)
	}
	// rounded tip
	tipW := tW * 0.30
	tongue.fan(tPts[tn], tipW, lerpRGBA(tTip, tBase, 0.3), 16)
	tongue.draw(dst, false)
	// satin sheen: a soft center highlight tracing the spine
	var gloss mesh
	gl := make([]contract.Vec2, 0, tn+1)
	for i := 0; i <= tn; i += 2 {
		u := float64(i) / float64(tn)
		gl = append(gl, v2(tPts[i].X-3, tPts[i].Y-2-u*2))
	}
	strokeQuads(&gloss, gl, 1.8, withA(hexRGBA("#ffffff"), 90))
	gloss.draw(dst, false)

	// ---- exactly three pearls + her rising motes (nest_pearls.go) ----
	drawNestPearls(dst, cx, mouthY, hw, oh, time, contact)
}

func easeOut(u float64) float64 { return u * u }
