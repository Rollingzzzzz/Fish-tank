// G5.1/G5.2/G5.3: demo-ui — acceptance binary for the Lane C UI package.
// Shows the full widget kit in a gallery panel, streams fake agent events
// through the slide-in menu, and auto-opens it. Flags:
//
//	--png <out>   render ~90 frames, save the LAST frame as PNG, exit
//	--closed      keep the menu closed (PNG proves the pixel-pure tank:
//	              only the 8 px handle is drawn)
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/Rollingzzzzz/Fish-tank/internal/content"
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/Rollingzzzzz/Fish-tank/internal/render"
	"github.com/Rollingzzzzz/Fish-tank/internal/ui"
)

const (
	screenW, screenH = 1280, 720
	totalFrames      = 90
)

func main() {
	pngOut := flag.String("png", "", "dump the final frame as PNG to this path and exit")
	closed := flag.Bool("closed", false, "keep the menu closed (handle-only frame)")
	flag.Parse()
	d := &demo{pngOut: *pngOut, keepClosed: *closed}
	d.bg = render.NewBackground()
	d.menu = ui.NewMenu(&contract.Config{
		Endpoint: "https://api.z.ai/api/coding/paas/v4/chat/completions",
		Model:    "glm-5.3-flash", AutoFeed: true, AutoCare: true, MusicOn: true,
		AgentFreq: 1, DaySeconds: 60, MaxFish: 24,
	})
	if s, err := content.Load("."); err == nil { // demo folder has no content/
		d.menu.SetContentStore(s) // -> demo-sample fallbacks are shown
	}
	d.g = newGallery()
	ebiten.SetWindowTitle("NEON TANK - UI demo")
	ebiten.SetWindowSize(screenW, screenH)
	if err := ebiten.RunGame(d); err != nil && err != ebiten.Termination {
		fmt.Fprintln(os.Stderr, "demo-ui:", err)
		os.Exit(1)
	}
}

type demo struct {
	pngOut     string
	keepClosed bool
	frame      int
	wt         float64
	bg         *render.Background
	menu       *ui.Menu
	g          *gallery
	frameImg   *ebiten.Image
	saved      bool
}

func (d *demo) Update() error {
	d.frame++
	d.wt += 1.0 / 60.0
	mx, my := ebiten.CursorPosition()
	pressed := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	released := inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft)
	wheelX, wheelY := ebiten.Wheel()
	_ = wheelX // horizontal wheel unused in the demo

	// Open the menu early so the capture shows the slid-in panel.
	if d.frame == 8 && !d.keepClosed {
		d.menu.SetOpen(true)
	}
	d.script()
	d.menu.Update(mx, my, pressed, released)
	d.menu.Wheel(int(wheelY), mx, my)
	d.g.update(mx, my, pressed, released, wheelY, 1.0/60.0)
	d.g.feed(d.frame)

	if d.frame >= totalFrames && !d.saved {
		d.saved = true
		if d.pngOut != "" {
			if err := d.dumpPNG(d.pngOut); err != nil {
				fmt.Fprintln(os.Stderr, "demo-ui: png dump failed:", err)
			} else {
				fmt.Println("demo-ui: wrote", d.pngOut)
			}
		}
		return ebiten.Termination
	}
	return nil
}

// script feeds a small fake agent run so the capture shows streaming states.
func (d *demo) script() {
	step := func(f int, ev contract.Event) {
		if d.frame == f {
			d.menu.PushEvent(ev)
		}
	}
	step(14, contract.Event{Kind: contract.EventStatus, Agent: "species", Status: contract.StatusThinking})
	step(16, contract.Event{Kind: contract.EventThought, Agent: "species", Text: "sketching a bioluminescent tetra, distinct hue walk..."})
	step(20, contract.Event{Kind: contract.EventStatus, Agent: "species", Status: contract.StatusWriting})
	step(22, contract.Event{Kind: contract.EventChunk, Agent: "species", Text: "Aqua Wisp drifts "})
	step(24, contract.Event{Kind: contract.EventChunk, Agent: "species", Text: "in loose schools near the surface."})
	step(28, contract.Event{Kind: contract.EventStatus, Agent: "species", Status: contract.StatusDone})
	step(30, contract.Event{Kind: contract.EventArtifact, Agent: "species", ArtifactID: "aqua-wisp", Text: "new species registered"})
	step(34, contract.Event{Kind: contract.EventStatus, Agent: "water", Status: contract.StatusSimulated})
	step(36, contract.Event{Kind: contract.EventArtifact, Agent: "water", ArtifactID: "midnight-lagoon", Text: "water mood applied"})
	step(40, contract.Event{Kind: contract.EventStatus, Agent: "pattern", Status: contract.StatusError})
	step(41, contract.Event{Kind: contract.EventLog, Text: "water preset crossfaded: midnight-lagoon"})
	if d.frame%24 == 0 {
		d.menu.PushEvent(contract.Event{Kind: contract.EventLog,
			Text: fmt.Sprintf("tank log tick %d (fish 24, care 12)", d.frame)})
	}
}

// dumpPNG reads the composed frame back once and encodes it.
func (d *demo) dumpPNG(path string) error {
	if d.frameImg == nil {
		return fmt.Errorf("no frame rendered")
	}
	b := d.frameImg.Bounds()
	img := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	pix := make([]byte, b.Dx()*b.Dy()*4)
	d.frameImg.ReadPixels(pix) // single bulk GPU readback
	img.Pix = pix
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func (d *demo) Draw(screen *ebiten.Image) {
	if d.frameImg == nil {
		d.frameImg = ebiten.NewImage(screenW, screenH)
	}
	day := 0.7 + 0.3*math.Sin(d.wt/8)
	d.bg.Draw(d.frameImg, render.WaterState{
		Time: d.wt, DayFactor: day,
		Top: [3]float64{0.024, 0.16, 0.29}, Bottom: [3]float64{0.008, 0.05, 0.11},
		Accent: [3]float64{0, 1, 0.88}, Rays: 0.5, Caustics: 0.6,
	})
	render.DrawGlow(d.frameImg, 200+120*math.Sin(d.wt/2), 300+60*math.Cos(d.wt/1.4), 40, "#00ffe1", 0.25)
	d.g.draw(d.frameImg)
	d.menu.Draw(d.frameImg)
	screen.DrawImage(d.frameImg, nil)
}

func (d *demo) Layout(int, int) (int, int) { return screenW, screenH }
