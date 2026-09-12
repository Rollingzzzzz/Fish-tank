// v0.3.8: the near-glass plane — a few large, cool-dimmed fronds rooted at
// the very bottom edge of the tank, drawn AFTER the rock front. They close
// the depth sandwich that makes a 2D canvas read 3D: dim crags at the back →
// midground flora, school and the nest → dark foreground silhouettes hugging
// the glass. Render-only: no sim coupling, no plant-cap interaction, pinned
// to the FLANKS so the center view stays open (the very mistake that made
// the old oyster suffocate the tank).
//
// Perf contract: the per-frond plant renderers cost ~30-70 draw calls per
// silk plant, which profiled as a measurable TPS drop at the 100-fish cap —
// so each frond is BAKED once into its own offscreen image at first draw and
// composited with a single DrawImage that breathes the whole plant (±0.7°)
// around its base. Alive, at one call per frond.
package render

import (
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/hajimehoshi/ebiten/v2"
)

// fgSpot is one near-glass frond: a design, a flank position (x as a
// fraction of the canvas) and a height multiplier over the design's own.
type fgSpot struct {
	def *contract.PlantDesign
	xf  float64
	mul float64

	img *ebiten.Image // baked silhouette (lazy, first Draw)
	bx  float64       // plant base inside the baked image
	by  float64
	ph  float64 // sway phase
}

// ForegroundStage holds the picked near-glass flora.
type ForegroundStage struct {
	spots []fgSpot
}

// NewForegroundStage picks 4-5 fronds from the store's designs — silky or
// feathery kinds preferred — deterministic under the rock seed.
func NewForegroundStage(defs []*contract.PlantDesign, seed int64) *ForegroundStage {
	rng := contract.RandSeed(seed)
	var pool []*contract.PlantDesign
	for _, d := range defs {
		if d == nil || len(d.Colors) < 1 || d.Fronds < 1 {
			continue
		}
		if d.Kind == "silk" || d.Kind == "feather" {
			pool = append(pool, d) // silky fronds read best as near-glass hair
		}
	}
	f := &ForegroundStage{}
	if len(pool) == 0 {
		return f
	}
	spots := []fgSpot{
		{xf: 0.055, mul: 1.55, ph: 0.0}, {xf: 0.112, mul: 1.30, ph: 2.1},
		{xf: 0.945, mul: 1.58, ph: 4.2},
	}
	if len(pool) > 3 { // a fourth frond only when there is variety to stage
		spots = append(spots, fgSpot{xf: 0.912, mul: 1.24, ph: 5.1})
	}
	for i := range spots {
		spots[i].def = pool[rng.Intn(len(pool))]
	}
	f.spots = spots
	return f
}

// bake renders one frond ONCE into its offscreen silhouette (cool-dimmed,
// frozen at a gentle mid-sway pose so it never reads as a stiff stick).
func (f *ForegroundStage) bake(sp *fgSpot, h float64) {
	hgt := h * sp.def.Height * sp.mul
	iw, ih := int(hgt*0.95)+24, int(hgt)+14
	img := ebiten.NewImage(iw, ih)
	sp.bx, sp.by = float64(iw)/2, float64(ih)-8
	DrawPlantShaded(img, img, sp.def, sp.bx, sp.by, hgt, 1.7, 0, 0.38)
	sp.img = img
}

// Draw composites the near-glass fronds — one DrawImage each, the whole
// plant breathing around its base.
func (f *ForegroundStage) Draw(dst *ebiten.Image, t, w, h float64) {
	if f == nil {
		return
	}
	for i := range f.spots {
		sp := &f.spots[i]
		if sp.img == nil {
			f.bake(sp, h)
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-sp.bx, -sp.by)
		op.GeoM.Rotate(sin(t*0.55+sp.ph) * 0.012) // ±0.7° whole-plant breathing
		op.GeoM.Translate(w*sp.xf, h*0.998)
		dst.DrawImage(sp.img, op)
	}
}
