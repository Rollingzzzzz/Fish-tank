// G95: headless REC capture tests — pure crop geometry, byte crops, and
// the full 60-frame state machine against a temp dir (no ebiten involved).
package game

import (
	"bytes"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rollingzzzzz/Fish-tank/internal/ui"
)

func TestRecCropRectFramesTheFocus(t *testing.T) {
	r := recCropRect(1280, 720, 72, 640)
	if r.Dx() != 518 || r.Dy() != 648 {
		t.Fatalf("got %dx%d, want 518x648 (4:5 under the inset)", r.Dx(), r.Dy())
	}
	if r.Min.Y != 72 {
		t.Fatalf("crop starts at y=%d — the chrome row must stay out", r.Min.Y)
	}
	if c := r.Min.X + r.Dx()/2; c != 640 {
		t.Fatalf("crop centered on %d, want 640", c)
	}
	if l := recCropRect(1280, 720, 72, 10); l.Min.X != 0 {
		t.Fatalf("left clamp: x=%d, want 0", l.Min.X)
	}
	if rr := recCropRect(1280, 720, 72, 1270); rr.Max.X != 1280 {
		t.Fatalf("right clamp: x2=%d, want 1280", rr.Max.X)
	}
}

func TestRecCropBytesExtractsRegion(t *testing.T) {
	// pixel (x,y) = (x, y, 0, 255) on an 8x6 canvas
	pix := make([]byte, 4*8*6)
	for y := 0; y < 6; y++ {
		for x := 0; x < 8; x++ {
			o := (y*8 + x) * 4
			pix[o], pix[o+1], pix[o+2], pix[o+3] = byte(x), byte(y), 0, 255
		}
	}
	out := recCropBytes(pix, 8, image.Rect(2, 1, 6, 5))
	if len(out) != 4*4*4 {
		t.Fatalf("crop is %d bytes, want 64", len(out))
	}
	check := func(idx, wantR, wantG byte) {
		if out[idx*4] != wantR || out[idx*4+1] != wantG {
			t.Fatalf("pixel %d = (%d,%d), want (%d,%d)", idx, out[idx*4], out[idx*4+1], wantR, wantG)
		}
	}
	check(0, 2, 1)  // top-left of the crop
	check(3, 5, 1)  // top-right
	check(12, 2, 4) // bottom-left
	check(15, 5, 4) // bottom-right
}

func TestRecFocusLerpGlide(t *testing.T) {
	if v := focusLerp(0, 100, 0.5); v != 50 {
		t.Fatalf("lerp %.1f, want 50", v)
	}
	if v := focusLerp(0, 100, 2); v != 100 {
		t.Fatalf("k>1 must clamp: %.1f", v)
	}
	if v := focusLerp(0, 100, -1); v != 0 {
		t.Fatalf("k<0 must clamp: %.1f", v)
	}
}

func TestRecDirBesideTheExe(t *testing.T) {
	if got, want := recDir(filepath.Join("D:", "apps", "tank.exe")),
		filepath.Join("D:", "apps", "screenshots"); got != want {
		t.Fatalf("recDir = %q, want %q", got, want)
	}
	for _, dev := range []string{"", filepath.Join("tmp", "go-build", "tank.exe")} {
		if got := recDir(dev); got != "screenshots" {
			t.Fatalf("dev recDir = %q, want the working directory", got)
		}
	}
}

func TestRecWritesSixtyFrames(t *testing.T) {
	dir := t.TempDir()
	r := &recorder{}
	r.begin(dir, 100, 100)
	pix := make([]byte, 4*100*100)
	for i := range pix {
		pix[i] = byte(i % 251)
	}
	dt := 1.0 / 60.0
	focus := 20.0
	for steps := 0; r.active && steps < 60*61; steps++ {
		if steps%60 == 0 {
			focus += 2 // the pan glides across the session, never jumps
		}
		r.tick(dt, pix, 100, 100, 10, focus)
	}
	if r.active {
		t.Fatal("the session never finished")
	}
	r.wait()
	if r.n != RecSeconds {
		t.Fatalf("%d frames captured, want %d", r.n, RecSeconds)
	}
	w, h := 72, 90 // (100-10) tall, 4:5 wide
	for i := 1; i <= RecSeconds; i++ {
		b, err := os.ReadFile(filepath.Join(r.dir, fmt.Sprintf("frame-%02d.jpg", i)))
		if err != nil {
			t.Fatalf("frame %d: %v", i, err)
		}
		if len(b) < 2 || b[0] != 0xFF || b[1] != 0xD8 {
			t.Fatalf("frame %d is not a JPEG", i)
		}
		cfg, _, err := image.DecodeConfig(bytes.NewReader(b))
		if err != nil {
			t.Fatalf("frame %d decode: %v", i, err)
		}
		if cfg.Width != w || cfg.Height != h {
			t.Fatalf("frame %d is %dx%d, want %dx%d", i, cfg.Width, cfg.Height, w, h)
		}
		if len(b) > 150*1024 {
			t.Fatalf("frame %d is %d KB — over the size budget", i, len(b)/1024)
		}
	}
}

func TestRecCancelKeepsPartialFrames(t *testing.T) {
	dir := t.TempDir()
	r := &recorder{}
	r.begin(dir, 100, 100)
	pix := make([]byte, 4*100*100)
	dt := 1.0 / 60.0
	// frames land at t=0 then every whole second — three seconds of ticks
	// spans four frames (0s, 1s, 2s, 3s)
	for i := 0; i < 60*3; i++ {
		r.tick(dt, pix, 100, 100, 10, 32)
	}
	if !r.active {
		t.Fatal("the session ended early")
	}
	r.cancel()
	r.wait()
	entries, err := os.ReadDir(r.dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 4 {
		t.Fatalf("%d frames survived the cancel, want 4", len(entries))
	}
	if label, accent := r.view(); label != "SAVED" || accent != ui.ColOk {
		t.Fatalf("after cancel the button reads %q/%q, want the SAVED flash", label, accent)
	}
	for i := 0; i < 60*4; i++ { // the flash fades, the button re-arms
		r.tick(dt, pix, 64, 100, 10, 32)
	}
	if label, _ := r.view(); label != "REC 60s" {
		t.Fatalf("idle button reads %q, want REC 60s", label)
	}
}
