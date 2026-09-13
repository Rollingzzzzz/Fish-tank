// v1.1: the sand bed renderer (G54/G57) — a SOLID yellowish-white layer
// from the floor line down to the bottom edge, full width: the background
// never shows through and the nest pedestal sits on the sand. Speckle
// clusters ride on top, lifted and pushed sideways by the disturbance
// offsets. The bed is the FRONT-most terrain (v1.1): crags, flora and the
// school render behind it, so at night it dims by color toward the night
// tone while staying fully opaque (the v0.3.8 dark-clutter lesson, inverted:
// here the bed must stay readable). Matte, normal-alpha only: the neon
// stays the Chosen's alone (F16).
package render

import (
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/hajimehoshi/ebiten/v2"
)

var (
	cSandPale  = hexRGBA("#f6f0dc") // whitest grain
	cSandGold  = hexRGBA("#ecd9a0") // yellowish grain
	cSandShade = hexRGBA("#d4c492") // shaded grain
	cSandBase  = hexRGBA("#e9d9ab") // the solid layer, warm sand
	cSandDeep  = hexRGBA("#c9b483") // toward the bottom edge
	cSandNight = hexRGBA("#56524a") // night tone — the bed dims by COLOR,
	// never by alpha: it is the front-most layer, the scene behind it must
	// stay fully occluded even at night
)

func sandFract(v float64) float64 { return v - float64(int(v)) }

// DrawSandBed paints the floor sand: a solid layer from the floor line down
// to the bottom edge (h), full width, with one speckle cluster per cell on
// top. night (0..1) shifts the palette toward the night tone (opaque).
func DrawSandBed(dst *ebiten.Image, cells []float64, w, h, floorY, night float64) {
	if dst == nil || len(cells) == 0 {
		return
	}
	cellW := w / float64(len(cells))
	baseC := lerpRGBA(cSandBase, cSandNight, night)
	deepC := lerpRGBA(cSandDeep, cSandNight, night)
	var base mesh
	// the solid layer follows the shared relief curve (G68): gentle swells
	// and a shallow channel instead of a ruler line — the same surface the
	// simulation rests flakes on
	const step = 24.0
	surf := func(x float64) float64 { return contract.SandSurfaceY(h, x) }
	prevX, prevY := 0.0, surf(0)
	for x := step; x <= w+step; x += step {
		y := surf(x)
		base.quad(
			v2(prevX, prevY), v2(x, y), v2(x, h), v2(prevX, h),
			withA(baseC, 245), withA(baseC, 245),
			withA(deepC, 245), withA(deepC, 245))
		prevX, prevY = x, y
	}
	base.draw(dst, false)
	var m mesh
	for i, off := range cells {
		x := (float64(i) + 0.5) * cellW
		lift := off
		if lift < 0 {
			lift = 0 // a dip only dims and flattens the speckles
		}
		for k := 0; k < 5; k++ {
			f1 := sandFract(sin(float64(i*7+k*13)) * 43758.5453)
			f2 := sandFract(sin(float64(i*11+k*29)) * 24634.6345)
			gx := x + (f1-0.5)*cellW*1.7 + off*1.4
			gy := surf(gx) + 2 + f2*20 - lift*0.7
			size := 1.1 + f1*1.5
			col := cSandPale
			if k == 1 || k == 3 {
				col = cSandGold
			}
			if k == 2 {
				col = cSandShade
			}
			gra := 200.0 + 45*f2
			if liftBon := lift * 25; liftBon < 60 {
				gra += liftBon
			} else {
				gra += 60
			}
			alpha := uint8(clampF(gra, 0, 245))
			oval(&m, v2(gx, gy), size, size*0.75, withA(lerpRGBA(col, cSandNight, night), alpha), 6)
		}
	}
	m.draw(dst, false)
}
