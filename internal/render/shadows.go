// v1.1 (G74): floor shadows — every swimmer hangs a soft patch on the dunes
// directly under itself. The shadow obeys the depth lanes (a near-lane fish
// casts the darker, wider patch), thins as the fish climbs (penumbra), and
// dims with the sun at night. Matte normal-alpha on the bed: no additive
// blend, so the neon stays the Chosen's alone (F16).
package render

import (
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/hajimehoshi/ebiten/v2"
)

// ShadowCaster is one fish's projection data — the game layer fills it from
// the sorted draw order; the renderer never imports the sim.
type ShadowCaster struct {
	X, Hx   float64 // spine-forward x component (-1..1)
	BodyLen float64 // live spine length (px)
	Z       float64 // depth lane 0 far .. 1 near
	Lift    float64 // px the fish floats above the dune surface
	Fade    float64 // 0 solid .. 1 gone (the Chosen mid-portal)
}

var cShadow = hexRGBA("#6f6752") // warm grain shade, in family with the bed

// ShadowAlpha is the occlusion law: near lane > far lane, high fish thin out,
// night keeps ShadowNightMul of the day bite.
func ShadowAlpha(z, lift, h, night float64) float64 {
	a := contract.ShadowAlphaFar +
		(contract.ShadowAlphaNear-contract.ShadowAlphaFar)*clampF(z, 0, 1)
	a *= 0.55 + 0.45*clampF(1-lift/(0.72*h), 0, 1)
	return a * (1 - (1-contract.ShadowNightMul)*night)
}

// ShadowRadii is the footprint law: a flat swimmer stretches long and thin,
// a diver rounds out; the near lane buys a touch of width.
func ShadowRadii(bodyLen, hx, z float64) (rx, ry float64) {
	ax := clampF(absF(hx), 0, 1) // 1 = flat swimmer, 0 = nose-down diver
	rx = bodyLen * (0.20 + 0.14*ax) * (0.85 + 0.30*clampF(z, 0, 1))
	ry = bodyLen*(0.045+0.028*(1-ax)) + 2
	return rx, ry
}

// buildShadowMesh projects every caster onto the shared relief curve — one
// mesh, one draw call; overlapping ovals compound into a cluster shade.
func buildShadowMesh(casters []ShadowCaster, h, night float64) *mesh {
	m := &mesh{}
	for _, c := range casters {
		a := ShadowAlpha(c.Z, c.Lift, h, night) * (1 - clampF(c.Fade, 0, 1))
		if a <= 0.004 {
			continue
		}
		rx, ry := ShadowRadii(c.BodyLen, c.Hx, c.Z)
		sy := contract.SandSurfaceY(h, c.X) + 3
		oval(m, v2(c.X, sy), rx, ry, withA(cShadow, uint8(a*255)), 10)
	}
	return m
}

// DrawShadows paints the floor shadows — call after the sand bed, before the
// floor dwellers, so the shade rides on the grains the viewer sees.
func DrawShadows(dst *ebiten.Image, casters []ShadowCaster, w, h, night float64) {
	if dst == nil || len(casters) == 0 {
		return
	}
	buildShadowMesh(casters, h, night).draw(dst, false)
}
