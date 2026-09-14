// F16/N11/F22: Lane B acceptance tests — the Chosen-only glow gate (pure),
// plus pixel assertions run inside a brief ebiten game loop (ReadPixels
// refuses to run before a game starts on v2.10, so treats/creatures/mites
// are verified through a tiny harness that skips on headless machines).
package render

import (
	"fmt"
	"image/color"
	"testing"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/hajimehoshi/ebiten/v2"
)

// --- F16: the pure gate helper (no GPU needed) ---

func TestGlowVisible(t *testing.T) {
	chosen := &contract.Species{ID: "t-chosen", Role: contract.RoleChosen}
	normal := &contract.Species{ID: "t-normal", Role: contract.RoleNormal}
	if !GlowVisible(chosen) {
		t.Errorf("GlowVisible(chosen) = false, want true")
	}
	if GlowVisible(normal) {
		t.Errorf("GlowVisible(normal) = true, want false")
	}
	if GlowVisible(nil) {
		t.Errorf("GlowVisible(nil) = true, want false")
	}
	// empty role (legacy packs) must not glow either
	if GlowVisible(&contract.Species{ID: "t-bare"}) {
		t.Errorf("GlowVisible(empty role) = true, want false")
	}
}

// --- pixel harness ---

var testBG = color.RGBA{R: 10, G: 12, B: 30, A: 255}

// pixelHarness runs every GPU-needing assertion once, inside a real frame.
type pixelHarness struct {
	fails *[]string
	frame int
}

func (g *pixelHarness) Update() error {
	g.frame++
	if g.frame >= 5 {
		return ebiten.Termination
	}
	return nil
}

func (g *pixelHarness) Draw(screen *ebiten.Image) {
	if g.frame == 3 {
		g.runAssertions()
	}
}

func (g *pixelHarness) Layout(ow, oh int) (int, int) { return 320, 240 }

func (g *pixelHarness) fail(format string, args ...any) {
	*g.fails = append(*g.fails, fmt.Sprintf(format, args...))
}

// TestPixelAcceptance drives the harness; on machines where no window can
// open the pixel proofs skip (demo PNGs carry the visual evidence instead).
func TestPixelAcceptance(t *testing.T) {
	var fails []string
	err := ebiten.RunGame(&pixelHarness{fails: &fails})
	if err != nil && err != ebiten.Termination {
		t.Skipf("pixel harness could not open a game context: %v", err)
	}
	for _, f := range fails {
		t.Error(f)
	}
}

// countNonBG counts pixels that differ from the background fill.
func countNonBG(img *ebiten.Image, bg color.RGBA) int {
	return len(pixelsDifferent(img, bg))
}

// pixelsDifferent returns indices of pixels differing from bg.
func pixelsDifferent(img *ebiten.Image, bg color.RGBA) []int {
	b := img.Bounds()
	buf := make([]byte, 4*b.Dx()*b.Dy())
	img.ReadPixels(buf)
	var out []int
	for i := 0; i < len(buf); i += 4 {
		if buf[i] != bg.R || buf[i+1] != bg.G || buf[i+2] != bg.B {
			out = append(out, i)
		}
	}
	return out
}

// framesDiffer compares two same-size images pixel by pixel.
func framesDiffer(a, b *ebiten.Image) bool {
	x, y := pixelsOf(a), pixelsOf(b)
	if len(x) != len(y) {
		return true
	}
	for i := range x {
		if x[i] != y[i] {
			return true
		}
	}
	return false
}

func pixelsOf(img *ebiten.Image) []byte {
	b := img.Bounds()
	buf := make([]byte, 4*b.Dx()*b.Dy())
	img.ReadPixels(buf)
	return buf
}

func newBG(w, h int, bg color.RGBA) *ebiten.Image {
	img := ebiten.NewImage(w, h)
	img.Fill(bg)
	return img
}

func straightSpine(x, y, seg float64) []contract.Vec2 {
	n := contract.SpineSegments
	sp := make([]contract.Vec2, n)
	for i := 0; i < n; i++ {
		sp[i] = v2(x+float64(i)*seg, y)
	}
	return sp
}

// runAssertions holds every pixel proof (executed at frame 3 of the harness).
func (g *pixelHarness) runAssertions() {
	black := color.RGBA{A: 255}

	// F16: a normal vein fish paints NO glow pixels but keeps its body and
	// the vein pattern as a plain alpha stroke.
	spec := &contract.Species{
		ID: "t-vein", Role: contract.RoleNormal, Size: 1, Width: 1, Fin: 1, Tail: 1,
		Palette: contract.Palette{Body: "#0d5c4f", Belly: "#032b25", Accent: "#3dfff0", Glow: "#b6fff8"},
		Pattern: contract.Pattern{Type: "vein", Density: 0.6, Size: 0.5},
	}
	dst := newBG(200, 100, testBG)
	glow := newBG(200, 100, black)
	DrawFish(dst, glow, straightSpine(20, 50, 10), spec, &spec.Palette, "adult", 0.8,
		FishAnim{Time: 1.0, Speed01: 0.4})
	if n := countNonBG(dst, testBG); n < 300 {
		g.fail("normal vein fish: %d body pixels, want >= 300 (fish + vein must stay visible)", n)
	}
	if n := countNonBG(glow, black); n != 0 {
		g.fail("normal fish wrote %d glow pixels, want 0 (F16)", n)
	}

	// F16: the Chosen still glows.
	chosen := &contract.Species{
		ID: "t-chosen2", Role: contract.RoleChosen, Size: 1, Width: 1, Fin: 1, Tail: 1,
		Palette: contract.Palette{Body: "#7a5cc8", Belly: "#2a1a4d", Accent: "#c9a8ff", Glow: "#e6d6ff"},
		Pattern: contract.Pattern{Type: "vein", Density: 0.6, Size: 0.5},
	}
	cDst := newBG(200, 100, testBG)
	cGlow := newBG(200, 100, black)
	DrawFish(cDst, cGlow, straightSpine(20, 50, 10), chosen, &chosen.Palette, "adult", 0.8,
		FishAnim{Time: 1.0, Speed01: 0.4})
	if countNonBG(cGlow, black) == 0 {
		g.fail("Chosen wrote 0 glow pixels, want > 0 (Chosen stays flashy)")
	}

	// N11: every treat paints >= 150 px and animates with phase.
	for _, kind := range []string{contract.TreatBug, contract.TreatWorm, contract.TreatShrimp, contract.TreatChicken} {
		a := newBG(64, 64, testBG)
		DrawTreat(a, kind, v2(32, 32), 0.0)
		if n := countNonBG(a, testBG); n < 150 {
			g.fail("treat %s: %d pixels at phase 0, want >= 150", kind, n)
		}
		b := newBG(64, 64, testBG)
		DrawTreat(b, kind, v2(32, 32), 2.7)
		if n := countNonBG(b, testBG); n < 150 {
			g.fail("treat %s: %d pixels at phase 2.7, want >= 150", kind, n)
		}
		if !framesDiffer(a, b) {
			g.fail("treat %s: phase 0 and 2.7 identical (no animation)", kind)
		}
	}

	// F22: the mite paints and wiggles.
	ma := newBG(48, 48, testBG)
	DrawMite(ma, v2(24, 24), 0.0)
	if n := countNonBG(ma, testBG); n < 20 {
		g.fail("mite: %d pixels, want >= 20", n)
	}
	mb := newBG(48, 48, testBG)
	DrawMite(mb, v2(24, 24), 2.1)
	if !framesDiffer(ma, mb) {
		g.fail("mite: phases 0 and 2.1 identical (legs do not wiggle)")
	}

	// F18: the sucker mouth pumps — half a ~1 Hz cycle changes the frame.
	sucker := &contract.Species{
		ID: "t-sucker", Role: contract.RoleNormal, Size: 1, Width: 1.1, Fin: 0.8, Tail: 0.9,
		Palette: contract.Palette{Body: "#4a3b2a", Belly: "#e8dcc0", Accent: "#8f7a4a", Glow: "#c9b98a"},
		Pattern: contract.Pattern{Type: "spot", Density: 0.5, Size: 0.5},
	}
	draw := func(t float64) *ebiten.Image {
		img := newBG(200, 100, testBG)
		anim := FishAnim{Time: t, Attached: true, AttachSide: -1}
		DrawFish(img, nil, straightSpine(30, 50, 8), sucker, &sucker.Palette, "adult", 0.3, anim)
		return img
	}
	if !framesDiffer(draw(0.75), draw(1.25)) {
		g.fail("sucker pose identical at mouth-closed and mouth-open times")
	}
	// F18: the attached pose itself must paint.
	if n := countNonBG(draw(1.0), testBG); n < 300 {
		g.fail("sucker pose: %d pixels, want >= 300", n)
	}

	// F24: the oyster throne touches the window bottom (nest_test.go).
	nestFloorPixelProof(g)

	// F25: every crag door renders dark depth (rock_test.go).
	doorMouthPixelProof(g)
}
