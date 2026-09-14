// G1.4: preview helpers — species thumbnails for the UI catalog, and a PNG
// dump used by demo binaries for developer self-review (`--png out.png`).
package render

import (
	"errors"
	"image/png"
	"os"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/hajimehoshi/ebiten/v2"
)

// DrawFishPreview draws a straight, calmly-phased fish fit into the box.
func DrawFishPreview(dst *ebiten.Image, spec *contract.Species, w, h int) {
	if dst == nil || spec == nil || w < 8 || h < 8 {
		return
	}
	n := contract.SpineSegments
	pad := float64(w) * 0.30 // room for tail fin behind the body
	usable := float64(w) - pad*2
	if usable < float64(n) {
		usable = float64(n)
	}
	segLen := usable / float64(n-1)
	y := float64(h) / 2
	spine := make([]contract.Vec2, n)
	for i := 0; i < n; i++ {
		spine[i] = v2(pad+usable-float64(i)*segLen, y+sin(float64(i)*0.55+1.2)*float64(h)*0.02)
	}
	DrawFish(dst, dst, spine, spec, &spec.Palette, "adult", 0.25, FishAnim{Time: 1.35, Speed01: 0.35})
}

// PixelAt reads one RGBA pixel (debug helper).
func PixelAt(img *ebiten.Image, x, y int) [4]byte {
	sub := img.SubImage(imageRect(x, y, x+1, y+1)).(*ebiten.Image)
	var px [4]byte
	sub.ReadPixels(px[:])
	return px
}

// SavePNG writes the current pixels of an ebiten image to path (PNG).
func SavePNG(img *ebiten.Image, path string) error {
	b := img.Bounds()
	out := imageNewRGBA(b.Dx(), b.Dy())
	img.ReadPixels(out.Pix)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, out)
}

// SavePNGBytes writes a raw RGBA pixel buffer (w*h*4, top-left origin) to
// path — the -shot harness freezes pixels at the capture frame (Update) and
// writes the file later (Draw), so a lagging Draw can never save the wrong
// game state.
func SavePNGBytes(pix []byte, w, h int, path string) error {
	if len(pix) < 4*w*h {
		return errors.New("SavePNGBytes: short pixel buffer")
	}
	out := imageNewRGBA(w, h)
	copy(out.Pix, pix[:4*w*h])
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, out)
}
