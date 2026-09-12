// N9 demo: 3x3 gallery of the nine core seed species, drawn at true relative
// scale (spine length proportional to Size) so the Chosen Lilastar is visibly
// the largest and most ornate fish in the tank — no other species comes close.
package main

import (
	"flag"
	"image/color"
	"math"
	"os"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/Rollingzzzzz/Fish-tank/internal/render"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	w = 1280
	h = 720
)

// gallery mirrors the nine seed species of internal/content/seed/species
// (values kept in sync there). The Chosen sits center stage (cell 5).
var gallery = []contract.Species{
	{ID: "glass-sucker", Name: "Glass Sucker", Size: 0.9, Width: 1.15, Fin: 0.7, Tail: 0.8,
		Palette: contract.Palette{Body: "#5f7d8c", Belly: "#8fb3b6", Accent: "#9fe8d8", Glow: "#6fd8c8"},
		Pattern: contract.Pattern{Type: "spot", Density: 0.25, Size: 0.3}},
	{ID: "cyan-veilglow", Name: "Cyan Veilglow", Size: 0.85, Width: 0.9, Fin: 1.3, Tail: 1.2,
		Palette: contract.Palette{Body: "#18c5ff", Belly: "#b8f4ff", Accent: "#7b4dff", Glow: "#4df2ff"},
		Pattern: contract.Pattern{Type: "wave", Density: 0.45, Size: 0.55}},
	{ID: "neon-emberfin", Name: "Neon Emberfin", Size: 1.0, Width: 1.0, Fin: 1.1, Tail: 1.0,
		Palette: contract.Palette{Body: "#ff5a2a", Belly: "#ffb36b", Accent: "#ff2a8d", Glow: "#ff7b4d"},
		Pattern: contract.Pattern{Type: "stripe", Density: 0.55, Size: 0.4}},
	{ID: "ribbon-streamer", Name: "Ribbon Streamer", Size: 0.95, Width: 0.7, Fin: 1.2, Tail: 1.3,
		Palette: contract.Palette{Body: "#5d84b8", Belly: "#a9c4e4", Accent: "#d98ba3", Glow: "#7fa9dc"},
		Pattern: contract.Pattern{Type: "wave", Density: 0.35, Size: 0.7}},
	{ID: "chosen-lilastar", Name: "Chosen Lilastar", Role: contract.RoleChosen, Size: 1.4, Width: 1.1, Fin: 1.5, Tail: 1.4,
		Palette: contract.Palette{Body: "#b57edc", Belly: "#d9c2f0", Accent: "#e6d6ff", Glow: "#c77dff"},
		Pattern: contract.Pattern{Type: "wave", Density: 0.7, Size: 0.6}},
	{ID: "puff-orbit", Name: "Puff Orbit", Size: 0.8, Width: 1.25, Fin: 0.9, Tail: 0.7,
		Palette: contract.Palette{Body: "#d9a53d", Belly: "#f0d9a3", Accent: "#e0785a", Glow: "#e8c46b"},
		Pattern: contract.Pattern{Type: "spot", Density: 0.5, Size: 0.55}},
	{ID: "violet-koidrift", Name: "Violet Koidrift", Size: 1.15, Width: 1.15, Fin: 0.9, Tail: 1.1,
		Palette: contract.Palette{Body: "#8a3dff", Belly: "#e8d5ff", Accent: "#ffc36b", Glow: "#b06dff"},
		Pattern: contract.Pattern{Type: "koi", Density: 0.5, Size: 0.65}},
	{ID: "lime-spotlure", Name: "Lime Spotlure", Size: 0.7, Width: 0.8, Fin: 1.35, Tail: 0.8,
		Palette: contract.Palette{Body: "#39ff88", Belly: "#d0ffd8", Accent: "#c8ff3d", Glow: "#7bffa8"},
		Pattern: contract.Pattern{Type: "spot", Density: 0.7, Size: 0.3}},
	{ID: "dart-spindle", Name: "Dart Spindle", Size: 0.62, Width: 0.75, Fin: 1.0, Tail: 1.1,
		Palette: contract.Palette{Body: "#2f8f6b", Belly: "#b5e3cf", Accent: "#6db5e8", Glow: "#4fc99b"},
		Pattern: contract.Pattern{Type: "stripe", Density: 0.65, Size: 0.45}},
}

type demo struct {
	frame int
	out   string
	fade  string
	boxes []*ebiten.Image
}

// drawScaled renders one species at true relative scale inside its box: the
// spine length is proportional to spec.Size, so supremacy reads at a glance.
func drawScaled(box *ebiten.Image, sp *contract.Species, cw, ch int) {
	n := contract.SpineSegments
	bodyLen := float64(cw) * 0.70 * sp.Size / 1.4 // Chosen spans 70% of the cell
	cx, cy := float64(cw)/2, float64(ch)/2
	segLen := bodyLen / float64(n-1)
	spine := make([]contract.Vec2, n)
	for i := 0; i < n; i++ {
		spine[i] = contract.Vec2{
			X: cx + bodyLen/2 - float64(i)*segLen,
			Y: cy + math.Sin(float64(i)*0.55+1.2)*float64(ch)*0.025,
		}
	}
	anim := render.FishAnim{Time: 1.35, Speed01: 0.35}
	render.DrawFish(box, box, spine, sp, &sp.Palette, "adult", 0.25, anim)
}

func (d *demo) ensureBoxes() {
	if d.boxes != nil {
		return
	}
	cellW, cellH := w/3, h/3
	for i := range gallery {
		box := ebiten.NewImage(cellW, cellH)
		box.Fill(color.RGBA{R: 6, G: 8, B: 24, A: 255})
		drawScaled(box, &gallery[i], cellW, cellH)
		d.boxes = append(d.boxes, box)
	}
}

func (d *demo) Update() error { d.frame++; return nil }

// drawFadePair renders the Chosen twice (ElderP 0 vs 0.9) for the F7
// elder-fade evidence frame.
func drawFadePair(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 6, G: 8, B: 24, A: 255})
	sp := &gallery[4] // the Chosen
	drawAt := func(x float64, elderP float64) {
		n := contract.SpineSegments
		spine := make([]contract.Vec2, n)
		for i := 0; i < n; i++ {
			spine[i] = contract.Vec2{X: x - float64(i)*6, Y: 360}
		}
		anim := render.FishAnim{Time: 1.35, Speed01: 0.35, ElderP: elderP}
		render.DrawFish(screen, screen, spine, sp, &sp.Palette, "elder", 0.25, anim)
	}
	drawAt(380, 0)
	drawAt(900, 0.9)
}

func (d *demo) Draw(screen *ebiten.Image) {
	if d.fade != "" {
		if d.frame >= 5 {
			drawFadePair(screen)
			if err := render.SavePNG(screen, d.fade); err != nil {
				panic(err)
			}
			os.Exit(0)
		}
		return
	}
	screen.Fill(color.RGBA{R: 6, G: 8, B: 24, A: 255})
	d.ensureBoxes()
	cellW, cellH := w/3, h/3
	for i, box := range d.boxes {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(float64((i%3)*cellW), float64((i/3)*cellH))
		screen.DrawImage(box, opts)
	}
	if d.out != "" && d.frame >= 30 {
		if err := render.SavePNG(screen, d.out); err != nil {
			panic(err)
		}
		os.Exit(0)
	}
}

func (d *demo) Layout(ow, oh int) (int, int) { return w, h }

func main() {
	out := flag.String("png", "", "save a frame to this path and exit")
	fade := flag.String("fade", "", "save an elder-fade pair frame and exit")
	flag.Parse()
	d := &demo{out: *out, fade: *fade}
	ebiten.SetWindowTitle("NEON TANK — species gallery")
	ebiten.SetWindowSize(w, h)
	if err := ebiten.RunGame(d); err != nil && err != ebiten.Termination {
		panic(err)
	}
}
