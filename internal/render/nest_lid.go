// v0.3.7 (F27): the oyster's upper half — a layered gape cavity (never a
// flat wedge), a taller hinge heel with a soft ligament shadow instead of
// the old hard slab, and a lid whose scalloped lower lip interlocks with
// the cup rim so the two valves read as ONE shell.
package render

import (
	"math"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/hajimehoshi/ebiten/v2"
)

// gapeSegs is dense enough that the cavity never shows a polygon facet.
const gapeSegs = 32

// gapePt is one point on a mouth-cavity ellipse. spread scales the
// half-width (outer ring → inner core), lift raises the layer's center.
func gapePt(cx, mouthY, hw, oh float64, i int, spread, lift float64) contract.Vec2 {
	a := math.Pi - math.Pi*float64(i)/float64(gapeSegs)
	return v2(cx-hw*spread*cos(a), mouthY+oh*(lift+0.12*sin(a)))
}

// drawNestGape paints the mouth as three nested ellipse fans — deep shadow
// core, mid tone, warm rim where nacre light catches the cavity edge.
func drawNestGape(dst *ebiten.Image, cx, mouthY, hw, oh float64) {
	type layer struct {
		spread, lift float64
		deep, warm   string
	}
	for _, l := range []layer{
		{0.80, 0.010, "#1d1330", "#33234e"},
		{0.62, 0.022, "#161028", "#261a3c"},
		{0.42, 0.034, "#110b20", "#1c1430"},
	} {
		var m mesh
		dCol, wCol := hexRGBA(l.deep), hexRGBA(l.warm)
		ci := m.vert(v2(cx, mouthY+oh*l.lift), dCol)
		for i := 0; i <= gapeSegs; i++ {
			a := sin(math.Pi - math.Pi*float64(i)/float64(gapeSegs))
			m.vert(gapePt(cx, mouthY, hw, oh, i, l.spread, l.lift), lerpRGBA(dCol, wCol, a*0.5))
		}
		for i := 1; i <= gapeSegs; i++ {
			m.tri(ci, uint16(i), uint16(i+1))
		}
		m.draw(dst, false)
	}
}

// drawNestHinge paints the heel where the valves lock, plus a soft gradient
// shadow in the crevice under the lid — shading, not a cutting slab.
func drawNestHinge(dst *ebiten.Image, cx, baseY, mouthY, hw, oh float64) {
	var hinge mesh
	hDeep, hHi := hexRGBA("#1d1032"), hexRGBA("#4a3268")
	hn := 14
	hci := hinge.vert(v2(cx, baseY), hDeep)
	for i := 0; i <= hn; i++ {
		a := math.Pi - math.Pi*float64(i)/float64(hn)
		x := cx - hw*0.34*cos(a)
		y := baseY - oh*0.105*sin(a)
		hinge.vert(v2(x, y), lerpRGBA(hDeep, hHi, sin(a)))
	}
	for i := 1; i <= hn; i++ {
		hinge.tri(hci, uint16(i), uint16(i+1))
	}
	// chevron growth bands across the heel
	for k := 1; k <= 3; k++ {
		f := 0.30 + 0.22*float64(k)
		var chev mesh
		strokeQuads(&chev, []contract.Vec2{
			v2(cx-hw*0.34*0.9*f, baseY-oh*0.09*f),
			v2(cx, baseY-oh*0.105*clampF(f+0.12, 0, 1)),
			v2(cx+hw*0.34*0.9*f, baseY-oh*0.09*f),
		}, 1.1, withA(hHi, 80))
		chev.draw(dst, false)
	}
	hinge.draw(dst, false)
	// ligament crevice: a narrow soft shadow tight against the hinge — wide
	// enough to shade the joint, too small to read as a cutting bar
	var lig mesh
	dark := withA(hexRGBA("#0e081c"), 110)
	fade := withA(hexRGBA("#0e081c"), 0)
	lig.quad(
		v2(cx-hw*0.22, baseY-oh*0.075), v2(cx+hw*0.22, baseY-oh*0.075),
		v2(cx+hw*0.20, mouthY-oh*0.005), v2(cx-hw*0.20, mouthY-oh*0.005),
		fade, fade, dark, dark)
	lig.draw(dst, false)
}

// lidTopPt sweeps the lid's outer silhouette (scalloped, leaning crown).
func lidTopPt(cx, mouthY, hw, lidH, lidSkew float64, i, n int) contract.Vec2 {
	a := math.Pi - math.Pi*float64(i)/float64(n)
	side := 0.8
	if i > n/2 {
		side = 1.12
	}
	fl := fluteShape(float64(i)+float64(n), side) * 0.9
	return v2(cx+lidSkew*sin(a)-hw*0.80*cos(a), mouthY-lidH*0.35-(lidH*0.65+fl)*sin(a))
}

// lidBotPt sweeps the lid's LOWER LIP: a scalloped arc of its own phase that
// arches over the mouth — well above the mouth plane at the center, dropping
// ONTO the cup's rim shoulders at the ends (below the rim line) so the two
// valves visibly close into one shell with no see-through gap (F27).
func lidBotPt(cx, mouthY, hw, oh, lidSkew float64, i, n int) contract.Vec2 {
	a := math.Pi - math.Pi*float64(i)/float64(n)
	side := 0.95
	if i > n/2 {
		side = 1.05
	}
	sa := sin(a)
	fl := fluteShape(float64(i)+13.7, side) * 0.55
	y := mouthY + oh*(0.13-0.28*sa) + fl*0.5*sa
	return v2(cx+lidSkew*0.5*sa-hw*0.75*cos(a), y)
}

// drawNestLid paints the upper valve as a closed strip between the two arcs
// — per-vertex colors along BOTH edges, so no bright center wedge — then its
// fluting ribs and the wet nacre streak along the lip.
func drawNestLid(dst *ebiten.Image, cx, mouthY, hw, oh, time, contact float64) {
	n := 26
	lidTilt := 0.55 + 0.06*sin(time*0.5) + 0.10*contact // opens wider for her
	lidH := oh * 0.52 * lidTilt
	lidSkew := hw * 0.05 // the crown leans — no mirror symmetry up here either
	lidO, lidLip := hexRGBA("#231539"), hexRGBA("#d9caf2")
	var lid mesh
	for i := 0; i < n; i++ {
		t0, t1 := lidTopPt(cx, mouthY, hw, lidH, lidSkew, i, n), lidTopPt(cx, mouthY, hw, lidH, lidSkew, i+1, n)
		b0, b1 := lidBotPt(cx, mouthY, hw, oh, lidSkew, i, n), lidBotPt(cx, mouthY, hw, oh, lidSkew, i+1, n)
		ct0 := lerpRGBA(lidO, hexRGBA("#413061"), sin(math.Pi-math.Pi*float64(i)/float64(n)))
		ct1 := lerpRGBA(lidO, hexRGBA("#413061"), sin(math.Pi-math.Pi*float64(i+1)/float64(n)))
		cb0 := lerpRGBA(lidO, lidLip, sin(math.Pi-math.Pi*float64(i)/float64(n))*0.85)
		cb1 := lerpRGBA(lidO, lidLip, sin(math.Pi-math.Pi*float64(i+1)/float64(n))*0.85)
		lid.quad(t0, t1, b1, b0, ct0, ct1, cb1, cb0)
	}
	lid.draw(dst, false)
	// fluting ribs from the lip up the outer face
	var lidRibs mesh
	for k := 1; k < 8; k++ {
		i := n * k / 8
		strokeQuads(&lidRibs, []contract.Vec2{
			lidBotPt(cx, mouthY, hw, oh, lidSkew, i, n),
			lidTopPt(cx, mouthY, hw, lidH, lidSkew, i, n),
		}, 1.4, withA(hexRGBA("#221440"), 80))
	}
	lidRibs.draw(dst, false)
	// wet nacre highlight riding the lip
	var shine mesh
	var lipPts []contract.Vec2
	for i := 0; i <= n; i += 2 {
		p := lidBotPt(cx, mouthY, hw, oh, lidSkew, i, n)
		lipPts = append(lipPts, v2(p.X, p.Y+2))
	}
	strokeQuads(&shine, lipPts, 1.6, withA(hexRGBA("#efe6ff"), 90))
	shine.draw(dst, false)
}
