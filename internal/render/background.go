// G1.1: water background — CPU-composited deep alien ocean.
//
// NOTE (G7.2): the Kage fragment-shader version was shelved — ebiten v2.10.1
// fails to deliver more than one named uniform per draw on this setup (a
// single vec4 works, two+ arrive as zero). The same art direction is
// implemented on the stable DrawTriangles/sprite path: vertex-gradient water,
// drifting caustic light patches, god-ray shafts, bioluminescent plankton,
// vignette. Revisit the shader when the uniform bug is fixed upstream.
package render

import (
	"image/color"
	"math"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/hajimehoshi/ebiten/v2"
)

// OK reports whether the legacy GPU path is active (always false since the
// CPU water replaced the Kage shader — kept for API compatibility).
func (b *Background) OK() bool { return false }

// Background draws the fullscreen water every frame.
type Background struct {
	w, h     int
	caustics []causticPatch
	rays     []godRay
	plankton []planktonMote
	inited   bool
}

type causticPatch struct {
	x, y, r  float64 // normalized space
	phase    float64
	speed    float64
	strength float64
}

type godRay struct {
	ang   float64
	sway  float64
	width float64
	x0    float64
	alpha float64
}

type planktonMote struct {
	x, y  float64
	r     float64
	speed float64
	phase float64
	drift float64
	wrapX float64
}

// NewBackground prepares the water layers.
func NewBackground() *Background { return &Background{} }

// initSeeded lays out the caustic patches, rays and plankton once.
func (b *Background) initSeeded() {
	if b.inited {
		return
	}
	b.inited = true
	rng := contract.RandSeed(20260911)
	for i := 0; i < 16; i++ {
		b.caustics = append(b.caustics, causticPatch{
			x: rng.Float64(), y: 0.05 + rng.Float64()*0.55,
			r: 0.06 + rng.Float64()*0.13, phase: rng.Float64() * 6.283,
			speed: 0.10 + rng.Float64()*0.16, strength: 0.35 + rng.Float64()*0.65,
		})
	}
	for i := 0; i < 6; i++ {
		b.rays = append(b.rays, godRay{
			ang:  -0.5 + float64(i)*0.2 + rng.Float64()*0.08,
			sway: rng.Float64() * 6.283, width: 40 + rng.Float64()*90,
			x0: 0.15 + rng.Float64()*0.7, alpha: 0.028 + rng.Float64()*0.030,
		})
	}
	for i := 0; i < 64; i++ {
		b.plankton = append(b.plankton, planktonMote{
			x: rng.Float64(), y: rng.Float64(),
			r: 1 + rng.Float64()*2.2, speed: 0.008 + rng.Float64()*0.02,
			phase: rng.Float64() * 6.283, drift: (rng.Float64() - 0.5) * 0.01,
		})
	}
}

// Draw paints the water over the full dst rect.
func (b *Background) Draw(dst *ebiten.Image, st WaterState) {
	w, h := dst.Bounds().Dx(), dst.Bounds().Dy()
	b.initSeeded()
	day := clampF(st.DayFactor, 0, 1)
	night := 1 - day
	dim := 0.30 + 0.70*day

	// F9 day-cycle tint: dawn warms the water, noon brightens it, dusk turns
	// violet-gold, deep night cools to blue-black. A smooth 4-key LUT keeps
	// the shift C0-continuous — no pops between frames.
	tintR, tintG, tintB, tintK := dayTint(st.TOD)
	acc := tintAcc(st.Accent, tintR, tintG, tintB, tintK)

	// v2.10 fill rule: triangles that OVERLAP inside one mesh cancel each
	// other, so every logically separate layer draws on its own mesh.
	var grad mesh
	tCol := tintMix(rgbaScale(st.Top, dim), tintR, tintG, tintB, tintK)
	bCol := tintMix(rgbaScale(st.Bottom, dim), tintR*0.6, tintG*0.6, tintB*0.8, tintK)
	midC := lerpRGBA(tCol, bCol, 0.55)
	midY := float64(h) * 0.52
	grad.quad(v2(0, 0), v2(float64(w), 0), v2(float64(w), midY), v2(0, midY),
		tCol, tCol, midC, midC)
	grad.quad(v2(0, midY), v2(float64(w), midY), v2(float64(w), float64(h)), v2(0, float64(h)),
		midC, midC, bCol, bCol)
	grad.draw(dst, false)

	// slow accent glow pools — alien depth (drifting soft fans)
	for k := 0; k < 3; k++ {
		cx := float64(w) * (0.5 + 0.4*math.Sin(st.Time*0.03+float64(k)*2.1))
		cy := float64(h) * (0.3 + 0.3*math.Sin(st.Time*0.021+float64(k)*1.3))
		var pool mesh
		pool.fan(v2(cx, cy), float64(h)*0.7, withA(rgbaScale(acc, 1), 6), 24)
		pool.draw(dst, false)
	}

	// god rays — long translucent shafts from the surface, additive
	for i, r := range b.rays {
		ang := r.ang + 0.06*math.Sin(st.Time*0.25+r.sway)
		ox := float64(w) * r.x0
		dirx, diry := math.Sin(ang), math.Cos(ang)
		length := float64(w+h) * 0.9
		hw := r.width
		tipX := ox + dirx*length + math.Sin(st.Time*0.4+float64(i))*20
		tipY := -40 + diry*length
		c0 := withA(rgbaScale(acc, 1), uint8(255*r.alpha*(0.6+0.7*night)))
		c1 := withA(rgbaScale(acc, 1), 0)
		var ray mesh
		ray.quad(
			v2(ox-hw, -40), v2(ox+hw, -40), v2(tipX+hw*2.4, tipY), v2(tipX-hw*2.4, tipY),
			c0, c0, c1, c1)
		ray.draw(dst, true)
	}

	// caustic light patches — soft drifting glows, denser near the surface
	for _, cp := range b.caustics {
		cx := (cp.x + 0.04*math.Sin(st.Time*cp.speed+cp.phase)) * float64(w)
		cy := (cp.y + 0.02*math.Sin(st.Time*cp.speed*1.3+cp.phase*2.0)) * float64(h)
		tw := 0.6 + 0.4*math.Sin(st.Time*cp.speed*3.0+cp.phase)
		a := cp.strength * tw * (0.16 + 0.14*day)
		var caust mesh
		caust.fan(v2(cx, cy), cp.r*float64(h), withA(rgbaScale(acc, 1), uint8(255*clampF(a, 0, 1))), 16)
		caust.draw(dst, true)
	}

	// plankton — drifting bioluminescent motes, popping at night
	for i := range b.plankton {
		pm := &b.plankton[i]
		pm.y -= pm.speed / 60.0
		pm.x += pm.drift / 60.0
		if pm.y < -0.02 {
			pm.y = 1.02
			pm.wrapX++
		}
		if pm.x < -0.02 {
			pm.x = 1.02
		}
		if pm.x > 1.02 {
			pm.x = -0.02
		}
		tw := 0.35 + 0.65*(0.5+0.5*math.Sin(st.Time*(0.4+pm.speed*8)+pm.phase))
		a := tw * (0.07 + 0.38*night) // F2: dimmed plankton
		DrawGlow(dst, pm.x*float64(w), pm.y*float64(h), pm.r*4, "#bfe9ff", a)
	}

	// vignette — edge quads darkening toward the frame. F18 regrade:
	// bottom band 144→96, top band 96→64 — depth without flat blackness.
	var vig mesh
	e := uint8(64)
	vig.quad(v2(0, 0), v2(float64(w), 0), v2(float64(w), float64(h)*0.14), v2(0, float64(h)*0.14),
		color.RGBA{0, 0, 20, e}, color.RGBA{0, 0, 20, e}, color.RGBA{0, 0, 0, 0}, color.RGBA{0, 0, 0, 0})
	vig.quad(v2(0, float64(h)), v2(float64(w), float64(h)), v2(float64(w), float64(h)*0.86), v2(0, float64(h)*0.86),
		color.RGBA{0, 0, 16, uint8(e * 3 / 2)}, color.RGBA{0, 0, 16, uint8(e * 3 / 2)}, color.RGBA{0, 0, 0, 0}, color.RGBA{0, 0, 0, 0})
	vig.draw(dst, false)
}

// rgbaScale scales a normalized color by f.
func rgbaScale(c [3]float64, f float64) color.RGBA {
	return color.RGBA{
		R: uint8(contract.Clamp(c[0], 0, 1) * 255 * f),
		G: uint8(contract.Clamp(c[1], 0, 1) * 255 * f),
		B: uint8(contract.Clamp(c[2], 0, 1) * 255 * f),
		A: 255,
	}
}

// dayTint returns the F9 day-cycle tint color and its blend strength for a
// time-of-day t (0 dawn, 0.25 noon, 0.5 dusk, 0.75 night). Keys are blended
// with a smooth 4-point interpolation — continuous, never popping.
func dayTint(t float64) (r, g, b, k float64) {
	t = clampF(t, 0, 1)
	// keyframes: dawn / noon / dusk / night
	kw := [4][3]float64{
		{0.95, 0.55, 0.30}, // dawn — warm amber
		{0.75, 0.90, 0.95}, // noon — airy cyan-white
		{0.85, 0.45, 0.45}, // dusk — violet-gold
		{0.10, 0.15, 0.35}, // night — blue-black
	}
	ks := [4]float64{0.45, 0.18, 0.45, 0.55} // tint strength per phase
	seg := t * 4
	i := int(seg)
	fr := seg - float64(i)
	// smoothstep the blend so each transition eases in and out
	fr = fr * fr * (3 - 2*fr)
	a, b2 := kw[i%4], kw[(i+1)%4]
	r = a[0] + (b2[0]-a[0])*fr
	g = a[1] + (b2[1]-a[1])*fr
	b = a[2] + (b2[2]-a[2])*fr
	k = ks[i%4] + (ks[(i+1)%4]-ks[i%4])*fr
	return r, g, b, k
}

// tintMix blends a color toward the tint color by k.
func tintMix(c color.RGBA, tr, tg, tb, k float64) color.RGBA {
	mix := func(v uint8, t float64) uint8 {
		return uint8(clampF(float64(v)*(1-k)+t*255*k, 0, 255))
	}
	return color.RGBA{R: mix(c.R, tr), G: mix(c.G, tg), B: mix(c.B, tb), A: c.A}
}

// tintAcc applies the day tint to a normalized accent color (rays, caustics).
func tintAcc(c [3]float64, tr, tg, tb, k float64) [3]float64 {
	return [3]float64{
		clampF(c[0]*(1-k)+tr*k, 0, 1),
		clampF(c[1]*(1-k)+tg*k, 0, 1),
		clampF(c[2]*(1-k)+tb*k, 0, 1),
	}
}
