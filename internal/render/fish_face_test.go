// v1.2 G93: the face-on pose renders through the shared batch — full
// blend replaces the side mesh, mid-blend crossfades both.
package render

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

func faceSpine() []contract.Vec2 {
	spine := make([]contract.Vec2, contract.SpineSegments)
	for i := range spine {
		spine[i] = contract.Vec2{X: 400 - float64(i)*4, Y: 300}
	}
	return spine
}

func TestFaceOnPoseRenders(t *testing.T) {
	dst := ebiten.NewImage(800, 600)
	sp := testPaletteSpecies()
	pal := &sp.Palette

	draw := func(gaze float64) int {
		anim := FishAnim{Time: 12.3, Z: 0.9, Gaze01: gaze}
		b := &FishBatch{}
		b.Draw(dst, nil, faceSpine(), sp, pal, "adult", 0, anim)
		n := b.n // read before Flush — Flush resets the batch
		b.Flush(dst, nil)
		return n
	}
	if n := draw(1.0); n != 1 {
		t.Fatalf("face-on pose did not draw (n=%d)", n)
	}
	if n := draw(0.0); n != 1 {
		t.Fatalf("side view did not draw (n=%d)", n)
	}
	if n := draw(0.5); n != 1 {
		t.Fatalf("mid-blend did not draw (n=%d)", n)
	}
}

func testPaletteSpecies() *contract.Species {
	return &contract.Species{
		ID:   "face-test",
		Role: contract.RoleNormal,
		Size: 1.0,
		Palette: contract.Palette{
			Body:   "#7fd4e8",
			Belly:  "#e8f6fa",
			Accent: "#2f6f8f",
			Glow:   "#9fe4f2",
		},
	}
}
