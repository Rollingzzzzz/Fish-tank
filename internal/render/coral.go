// N1: coral rendering — fan/branch/brain kinds with phase-shifted sine-rib
// membrane flutter (rive-loop feel), through the F2 glow budget. Every
// translucent volume is its own mesh draw (v2.10 fill rule: overlapping
// triangles inside one mesh cancel); ribbons sample 20+ points so joints stay
// smooth (F1 acceptance, same as plants).
package render

import (
	"math"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/hajimehoshi/ebiten/v2"
)

// CoralSpot is one placed coral instance (mirrors the plant placement data).
type CoralSpot struct {
	Def *contract.CoralDesign
	X   float64
	Y   float64
	Sc  float64
}

// DrawCorals paints every placed coral for this frame. h derives from the
// destination height (full-screen canvas) times the design fraction times the
// spot scale; night lifts the additive budget by up to +60%.
func DrawCorals(dst *ebiten.Image, spots []CoralSpot, time, night float64) {
	if dst == nil {
		return
	}
	for i := range spots {
		s := &spots[i]
		if s.Def == nil {
			continue
		}
		h := float64(dst.Bounds().Dy()) * contract.Clamp(s.Def.Height, 0.08, 0.35) * s.Sc
		w := contract.Clamp(s.Def.Width, 0.5, 1.6)
		glowK := (0.35 + 0.65*contract.Clamp(s.Def.Glow, 0, 1)) * (1 + 0.6*night)
		switch s.Def.Kind {
		case "brain":
			drawBrain(dst, s.Def, s.X, s.Y, h, w, time, glowK)
		case "branch":
			drawBranch(dst, s.Def, s.X, s.Y, h, w, time, glowK)
		default:
			drawFan(dst, s.Def, s.X, s.Y, h, w, time, glowK)
		}
	}
}

// ribMod is the transverse membrane-flutter factor shared by fan and brain:
// brightness *= 0.85+0.15*sin(u*ribs*2pi + speed*t + phase).
func ribMod(u, ribs, speed, t, phase float64) float64 {
	return 0.85 + 0.15*sin(u*ribs*2*math.Pi+speed*t+phase)
}

// tipOrb stamps the glowing ribbon end: a small orb plus a tiny additive halo
// (additive alpha capped at 40, v0.2 restraint).
func tipOrb(dst *ebiten.Image, p contract.Vec2, r float64, colors []string, glowK float64) {
	var orb mesh
	orb.fan(p, r, multiStop(colors, 1, 190), 10)
	orb.draw(dst, false)
	orb.reset()
	orb.fan(p, r*1.7, multiStop(colors, 1, uint8(math.Min(40, 40*glowK))), 10)
	orb.draw(dst, true)
	DrawGlow(dst, p.X, p.Y, r*3.2, colors[len(colors)-1], math.Min(0.12, 0.12*glowK))
}

// ribbonQuad appends one tapered gradient segment of a bezier ribbon with the
// rib brightness factor applied on top of the multi-stop gradient. Corner
// order is the repo perimeter order (prevL, l, r, prevR) — the reverse cycle
// is back-facing and cancels under the v2.10 fill rule.
func ribbonQuad(m *mesh, prevL, prevR contract.Vec2, prevC colorRGBA,
	l, r contract.Vec2, col colorRGBA) {
	m.quad(prevL, l, r, prevR, prevC, col, col, prevC)
}

// drawFan paints N curved membrane ribbons spreading from one anchor.
func drawFan(dst *ebiten.Image, def *contract.CoralDesign, x, y, h, w, t, glowK float64) {
	n := int(contract.Clamp(float64(def.Fronds), 3, 9))
	spread := 0.5 + 0.4*contract.Clamp(def.Curve, 0, 1)
	swayAmp := 4 + 10*contract.Clamp(def.Sway, 0, 1)
	swayOff := sin(t*(0.5+0.9*def.Sway)) * swayAmp // whole-coral sway
	const samples = 22
	for f := 0; f < n; f++ {
		hv := plantHash(f)
		ang := -math.Pi/2 + (2*float64(f)/float64(n-1)-1)*spread
		length := h * (0.82 + 0.36*hv)
		phase := hv * 6.283
		flick := sin(t*swayAmp*0.12+phase) * 1.5
		p0 := v2(x, y)
		dirX, dirY := cos(ang), sin(ang)
		p1 := v2(x+dirX*length*0.5+(hv-0.5)*h*0.22+(swayOff+flick)*0.45,
			y+dirY*length*0.5)
		p2 := v2(x+dirX*length+swayOff+flick, y+dirY*length)
		wBase := (3.5 + 3*hv) * w
		var body, aura mesh
		var prevL, prevR contract.Vec2
		var prevC colorRGBA
		for s := 0; s <= samples; s++ {
			u := float64(s) / samples
			cx, cy := bez(p0.X, p1.X, p2.X, u), bez(p0.Y, p1.Y, p2.Y, u)
			tx, ty := dbez(p0.X, p1.X, p2.X, u), dbez(p0.Y, p1.Y, p2.Y, u)
			inv := 1 / hyp(tx, ty)
			nx, ny := -ty*inv, tx*inv
			wid := wBase*(1-u) + 1.1*u
			col := scaleRGBA(multiStop(def.Colors, u, 225),
				ribMod(u, 4, 1.2+def.Sway, t, phase))
			l, r := v2(cx+nx*wid, cy+ny*wid), v2(cx-nx*wid, cy-ny*wid)
			if s > 0 {
				ribbonQuad(&body, prevL, prevR, prevC, l, r, col)
				dim, dim2 := withA(prevC, uint8(8*glowK)), withA(col, uint8(8*glowK))
				aura.quad(prevL, l, r, prevR, dim, dim2, dim2, dim)
			}
			prevL, prevR, prevC = l, r, col
		}
		body.draw(dst, false)
		aura.draw(dst, true)
		tipOrb(dst, p2, wBase*0.55+1.2, def.Colors, glowK)
	}
}

// drawBranch paints a main stem with 1-2 recursive sub-branches (depth 2),
// thicker base and rounded tips.
func drawBranch(dst *ebiten.Image, def *contract.CoralDesign, x, y, h, w, t, glowK float64) {
	wBase := 5 * w
	swayOff := sin(t*(0.5+0.9*def.Sway)) * (4 + 10*def.Sway)
	p0 := v2(x, y)
	p1 := v2(x+def.Curve*h*0.10+swayOff*0.5, y-h*0.5)
	p2 := v2(x+def.Curve*h*0.22+swayOff, y-h)
	coralRibbon(dst, def, p0, p1, p2, wBase, wBase*0.45, t, glowK, 2,
		1+int(contract.Clamp(float64(def.Fronds-4), 0, 1)), 0)
	DrawGlow(dst, p2.X, p2.Y, wBase*3.2, def.Colors[len(def.Colors)-1], math.Min(0.14, 0.14*glowK))
}

// coralRibbon draws one tapered bezier ribbon then recursively sprouts
// smaller side ribbons from its mid until depth runs out.
func coralRibbon(dst *ebiten.Image, def *contract.CoralDesign, p0, p1, p2 contract.Vec2,
	w0, w1, t, glowK float64, depth, spread int, phase float64) {
	const samples = 20
	var body mesh
	var prevL, prevR contract.Vec2
	var prevC colorRGBA
	for s := 0; s <= samples; s++ {
		u := float64(s) / samples
		cx, cy := bez(p0.X, p1.X, p2.X, u), bez(p0.Y, p1.Y, p2.Y, u)
		tx, ty := dbez(p0.X, p1.X, p2.X, u), dbez(p0.Y, p1.Y, p2.Y, u)
		inv := 1 / hyp(tx, ty)
		nx, ny := -ty*inv, tx*inv
		wid := w0*(1-u) + w1*u
		col := scaleRGBA(multiStop(def.Colors, u, 230),
			ribMod(u, 3, 1.0+def.Sway, t, phase))
		l, r := v2(cx+nx*wid, cy+ny*wid), v2(cx-nx*wid, cy-ny*wid)
		if s > 0 {
			ribbonQuad(&body, prevL, prevR, prevC, l, r, col)
		}
		prevL, prevR, prevC = l, r, col
	}
	body.draw(dst, false)
	body.reset()
	body.fan(p2, w1*1.4+0.8, multiStop(def.Colors, 1, 235), 10) // rounded tip
	body.draw(dst, false)
	if depth <= 0 {
		tipOrb(dst, p2, w1*1.2, def.Colors, glowK)
		return
	}
	for j := 0; j < spread; j++ {
		side := 1.0
		if j%2 == 1 {
			side = -1
		}
		ang := math.Atan2(p2.Y-p1.Y, p2.X-p1.X) + side*(0.45+0.35*def.Curve)
		mid := v2(bez(p0.X, p1.X, p2.X, 0.62), bez(p0.Y, p1.Y, p2.Y, 0.62))
		length := hyp(p2.X-p0.X, p2.Y-p0.Y) * (0.42 + 0.1*float64(j))
		nP1 := v2(mid.X+cos(ang)*length*0.5, mid.Y+sin(ang)*length*0.5)
		nP2 := v2(mid.X+cos(ang)*length, mid.Y+sin(ang)*length)
		coralRibbon(dst, def, mid, nP1, nP2, w0*0.55, w1*0.55, t, glowK, depth-1, 1,
			phase+1.3+float64(j)*2.1)
	}
}

// drawBrain paints a dome silhouette with sinuous brightness-modulated ridge
// lines: no sway, only ridge drift plus a slow 2% breathing scale.
func drawBrain(dst *ebiten.Image, def *contract.CoralDesign, x, y, h, w, t, glowK float64) {
	breath := 1 + 0.02*sin(t*0.9)
	rx, ry := h*(0.85+0.35*w)*breath, h*0.92*breath
	var dome mesh
	const segs = 26
	ci := dome.vert(v2(x, y), scaleRGBA(multiStop(def.Colors, 0, 255), 0.55))
	for i := 0; i <= segs; i++ {
		a := math.Pi + math.Pi*float64(i)/segs
		elev := maxF(0, -sin(a)) // clamp: sin(pi) is ~1e-16, not 0
		// curved ramp: dim base, bright crown only near the top
		col := scaleRGBA(multiStop(def.Colors, math.Pow(elev, 1.6), 255), 0.42+0.34*elev) // dim ember crown (F2 spirit)
		dome.vert(v2(x+rx*cos(a), y+ry*sin(a)), col)
	}
	for i := 1; i <= segs; i++ {
		dome.tri(ci, uint16(i), uint16(i+1))
	}
	dome.draw(dst, false)
	ridges := int(contract.Clamp(float64(def.Fronds), 3, 5))
	tip := hexRGBA(def.Colors[len(def.Colors)-1])
	for i := 0; i < ridges; i++ {
		v := float64(i+1) / float64(ridges+1)
		phase := float64(i)*1.9 + sin(t*0.35)*0.8
		chord := sqrt(maxF(1-v*v, 0.02)) * 0.94
		const samples = 36
		var gro, rib mesh
		var prevL, prevR contract.Vec2
		var prevC colorRGBA
		for s := 0; s <= samples; s++ {
			u := float64(s) / samples
			cx := x + (u*2-1)*rx*chord
			cy := y - v*ry + (2+3*def.Sway)*sin(u*5+t*(0.6+0.5*def.Glow)+phase)
			base := lerpRGBA(multiStop(def.Colors, v, 235), tip, 0.3)
			col := scaleRGBA(base, 0.75+0.45*ribMod(u, 3, 1.0+0.5*def.Glow, t, phase))
			l, r := v2(cx, cy-1.4), v2(cx, cy+1.4)
			gl, gr := v2(cx, cy+0.6), v2(cx, cy+3.4) // shadow groove below
			gc := withA(scaleRGBA(base, 0.35), 200)
			if s > 0 {
				ribbonQuad(&rib, prevL, prevR, prevC, l, r, col)
				ribbonQuad(&gro, prevL, prevR, prevC, gl, gr, gc)
			}
			prevL, prevR, prevC = l, r, col
		}
		gro.draw(dst, false)
		rib.draw(dst, false)
	}
	DrawGlow(dst, x, y-ry*0.55, rx*0.9, def.Colors[len(def.Colors)-1], math.Min(0.12, 0.12*glowK))
}
