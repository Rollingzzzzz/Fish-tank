// v1.1 G64: the corner near-glass fronds are baked rasters breathed with a
// small per-frame rotation. Under the default nearest filter that rotation
// sampled texel by texel — pixelated, freeze-step sway next to the smooth
// live vector plants. The composite must use linear filtering.
package render

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestForegroundBreatheSamplesSmoothly(t *testing.T) {
	sp := &fgSpot{bx: 100, by: 200}
	op := breatheOptions(sp, 0.012, 150, 390)
	if op.Filter != ebiten.FilterLinear {
		t.Fatalf("breathe composite uses filter %v — corner fronds step texel by texel again", op.Filter)
	}
	// the transform still breathes the plant around its base: the base sits
	// near (50,190) but the rotation swings the offset by ~2.4 px
	px, py := op.GeoM.Apply(0, 0)
	d := ((px-50)*(px-50) + (py-190)*(py-190))
	if d < 1 {
		t.Fatalf("breathe rotation is not applied to the base offset: %.2f,%.2f", px, py)
	}
	if d > 36 {
		t.Fatalf("breathe deflection exploded: %.2f,%.2f", px, py)
	}
}
