// G95: the REC 60s one-button capture — one JPEG frame per second of game
// time, cropped to LinkedIn's 4:5 portrait under the HUD chrome, written
// beside the exe for the owner's GIF pipeline. The crop pans with the
// school: the sim's weighted FocusX glides the window, so the vertical
// slice frames the action instead of blind center. Encoding runs on a
// single writer goroutine (buffered hand-off) — the draw thread only reads
// pixels; disk I/O never bends the G88 speed law.
package game

import (
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Rollingzzzzz/Fish-tank/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

// REC tuning (G95) — the owner retunes these after a live look.
const (
	RecSeconds     = 60   // frames = seconds at 1 fps
	RecAspect      = 0.80 // crop width/height — LinkedIn 4:5 portrait
	RecJPEGQuality = 90
	RecFocusLerp   = 0.5 // pan glide per capture step (share of the gap)
)

// recJob is one frame handed to the writer goroutine.
type recJob struct {
	path string
	pix  []byte
	w, h int
}

// recTopInset is the crop's top edge: below every chrome row (clock chip,
// X, REC, tray button) so no UI ever leaks into a frame.
func recTopInset() int { return ui.TrayBtnY + ui.TrayBtnH + 2 }

// recCropRect frames a RecAspect-wide window of the sw×sh canvas under the
// inset, centered on focusX and clamped to the canvas. Pure.
func recCropRect(sw, sh, inset int, focusX float64) image.Rectangle {
	if inset > sh-40 {
		inset = sh - 40
	}
	if inset < 0 {
		inset = 0
	}
	h := sh - inset
	w := int(float64(h)*RecAspect + 0.5)
	if w > sw {
		w = sw
	}
	x := int(focusX+0.5) - w/2
	if x < 0 {
		x = 0
	}
	if x+w > sw {
		x = sw - w
	}
	return image.Rect(x, inset, x+w, inset+h)
}

// recCropBytes copies the rect out of a 4*sw*sh RGBA buffer (top-left
// origin, the ebiten ReadPixels layout). Pure.
func recCropBytes(pix []byte, sw int, r image.Rectangle) []byte {
	cw, ch := r.Dx(), r.Dy()
	out := make([]byte, 4*cw*ch)
	for row := 0; row < ch; row++ {
		src := ((r.Min.Y + row)*sw + r.Min.X) * 4
		copy(out[row*cw*4:(row+1)*cw*4], pix[src:src+cw*4])
	}
	return out
}

// focusLerp glides cur toward tgt by share k of the gap (k clamped 0..1).
func focusLerp(cur, tgt, k float64) float64 {
	if k < 0 {
		k = 0
	}
	if k > 1 {
		k = 1
	}
	return cur + (tgt-cur)*k
}

// recDir resolves the output root: beside the exe for a real install, the
// working directory under `go run` (its binary lives in the go-build cache)
// so dev frames never vanish into a temp folder.
func recDir(exePath string) string {
	if exePath != "" && !strings.Contains(exePath, "go-build") {
		return filepath.Join(filepath.Dir(exePath), "screenshots")
	}
	return "screenshots"
}

// recorder is the REC state machine: begin → one tick per drawn frame →
// 60 JPEGs land in screenshots/rec-<stamp>/. Zero value ready and idle.
type recorder struct {
	active bool
	dir    string
	left   int     // frames still to capture
	n      int     // frames enqueued this session
	clock  float64 // seconds until the next frame
	focusX float64 // smoothed crop center
	pix    []byte  // reused full-canvas read buffer
	jobs   chan recJob
	wg     sync.WaitGroup
	savedT float64 // > 0: "SAVED" flash remaining
}

// begin opens a session folder and starts the writer goroutine; the first
// frame lands on the next tick.
func (r *recorder) begin(root string, sw, sh int) {
	if r.active {
		return
	}
	r.dir = filepath.Join(root, time.Now().Format("rec-20060102-150405"))
	if err := os.MkdirAll(r.dir, 0o755); err != nil {
		fmt.Println("rec:", err)
		return
	}
	r.active = true
	r.left = RecSeconds
	r.n = 0
	r.clock = 0
	r.focusX = float64(sw) / 2
	r.savedT = 0
	r.jobs = make(chan recJob, 4)
	r.wg.Add(1)
	go func() {
		for j := range r.jobs {
			r.write(j)
		}
		r.wg.Done()
	}()
}

// write encodes one frame as JPEG (the tank is opaque, so the
// premultiplied-alpha readout encodes losslessly).
func (r *recorder) write(j recJob) {
	f, err := os.Create(j.path)
	if err != nil {
		fmt.Println("rec:", err)
		return
	}
	defer f.Close()
	if err := jpeg.Encode(f, &image.RGBA{
		Pix: j.pix, Stride: 4 * j.w, Rect: image.Rect(0, 0, j.w, j.h),
	}, &jpeg.Options{Quality: RecJPEGQuality}); err != nil {
		fmt.Println("rec:", err)
	}
}

// tick advances the capture clock by dt of game time and, on each whole
// second, crops and queues one frame. pix is the full 4*sw*sh canvas
// snapshot the caller read off the composed screen this frame.
func (r *recorder) tick(dt float64, pix []byte, sw, sh, inset int, focusX float64) {
	if r.savedT > 0 {
		r.savedT -= dt
	}
	if !r.active {
		return
	}
	r.focusX = focusLerp(r.focusX, focusX, RecFocusLerp)
	r.clock -= dt
	if r.clock > 0 {
		return
	}
	r.clock += 1
	r.left--
	r.n++
	path := filepath.Join(r.dir, fmt.Sprintf("frame-%02d.jpg", r.n))
	rect := recCropRect(sw, sh, inset, r.focusX)
	r.jobs <- recJob{path: path, pix: recCropBytes(pix, sw, rect), w: rect.Dx(), h: rect.Dy()}
	if r.left <= 0 {
		r.finish()
	}
}

// finish closes a finished or cancelled session: the writer drains, the
// button flashes SAVED and the frame count reports.
func (r *recorder) finish() {
	r.active = false
	r.savedT = 3
	close(r.jobs)
	fmt.Printf("rec: %d frames in %s\n", r.n, r.dir)
}

// cancel stops a running session early — the frames so far are kept.
func (r *recorder) cancel() {
	if r.active {
		r.finish()
	}
}

// wait blocks until every queued frame is on disk (the shutdown flush).
func (r *recorder) wait() { r.wg.Wait() }

// view is the button face: red countdown while recording, green flash when
// the frames are saved, red armed otherwise.
func (r *recorder) view() (label, accent string) {
	switch {
	case r.active:
		return fmt.Sprintf("REC %ds", r.left), ui.ColErr
	case r.savedT > 0:
		return "SAVED", ui.ColOk
	default:
		return fmt.Sprintf("REC %ds", RecSeconds), ui.ColErr
	}
}

// recToggle arms or cancels a session (the REC button's click action).
func (g *Game) recToggle() {
	if g.rec.active {
		g.rec.cancel()
		return
	}
	g.rec.begin(recDir(exePath()), ScreenW, ScreenH)
}

// exePath wraps os.Executable for recDir ("" when unresolved).
func exePath() string {
	p, err := os.Executable()
	if err != nil {
		return ""
	}
	return p
}

// recCapture is the Draw-side hook, called after everything is rendered:
// while recording it reads the composed frame once per whole second.
func (g *Game) recCapture(screen *ebiten.Image) {
	if !g.rec.active {
		return
	}
	if len(g.rec.pix) != 4*ScreenW*ScreenH {
		g.rec.pix = make([]byte, 4*ScreenW*ScreenH)
	}
	screen.ReadPixels(g.rec.pix)
	g.rec.tick(1.0/float64(ebiten.TPS()), g.rec.pix, ScreenW, ScreenH, recTopInset(), g.world.FocusX())
}
