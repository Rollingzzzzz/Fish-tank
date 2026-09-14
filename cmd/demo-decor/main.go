// N11/F22/F18 demo: decor gallery — four live treats x two phases,
// crustaceans shell-open + shell-closed, a water mite; -attached saves the
// sucker pose at two mouth phases (F18 motion proof).
package main

import (
	"flag"
	"fmt"
	"image/color"
	"os"
	"strconv"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/Rollingzzzzz/Fish-tank/internal/render"
	"github.com/Rollingzzzzz/Fish-tank/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	w = 1280
	h = 720
)

// pleco is the F18 demo species for the attached sucker pose.
var pleco = contract.Species{
	ID: "sucker-demo", Name: "Sucker Demo", Size: 1, Width: 1.1, Fin: 0.8, Tail: 0.9,
	Palette: contract.Palette{Body: "#4a3b2a", Belly: "#e8dcc0", Accent: "#8f7a4a", Glow: "#c9b98a"},
	Pattern: contract.Pattern{Type: "spot", Density: 0.5, Size: 0.5},
}

type demo struct {
	frame int
	out   string
	att   string
}

func (d *demo) Update() error { d.frame++; return nil }

// cell paints one labelled dark plate and hands the box to fn.
func cell(screen *ebiten.Image, x, y, cw, ch int, label string, fn func(*ebiten.Image)) {
	box := ebiten.NewImage(cw-8, ch-8)
	box.Fill(color.RGBA{R: 10, G: 12, B: 30, A: 255})
	fn(box)
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(x+4), float64(y+4))
	screen.DrawImage(box, opts)
	ui.DrawText(screen, label, x+12, y+10, 1, "#8fb8e8", 0.9)
}

// drawGallery lays out the evidence grid.
func drawGallery(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 6, G: 8, B: 24, A: 255})
	cw, ch := w/5, h/3
	mid := func() contract.Vec2 { return v2f(float64(cw)/2, float64(ch)/2+6) }

	// rows 0-1: the four treats, each at two phases (all motion = phase)
	kinds := []string{contract.TreatBug, contract.TreatWorm, contract.TreatShrimp, contract.TreatChicken}
	names := map[string]string{contract.TreatBug: "BUG", contract.TreatWorm: "WORM",
		contract.TreatShrimp: "SHRIMP", contract.TreatChicken: "CHICKEN"}
	phases := [4][2]float64{{1.2, 3.9}, {0.6, 2.9}, {0.9, 3.4}, {0.4, 2.6}}
	c := 0
	for k, kind := range kinds {
		for p := 0; p < 2; p++ {
			kind, ph := kind, phases[k][p]
			col, row := c%4, c/4
			cell(screen, col*cw, row*ch, cw, ch, names[kind], func(box *ebiten.Image) {
				render.DrawTreat(box, kind, mid(), ph)
			})
			c++
		}
	}

	// mite cell (top right)
	cell(screen, 4*cw, 0, cw, ch, "MITE", func(box *ebiten.Image) {
		render.DrawMite(box, mid(), 1.1)
	})

	// row 2: mite phases across the floor band (v0.3.8: the crustacean row
	// was retired with the N4 crustaceans themselves)
	for i, ph := range []float64{0.0, 1.6, 3.2, 4.8} {
		i, ph := i, ph
		cell(screen, i*cw, 2*ch, cw, ch, fmt.Sprintf("MITE p%d", i), func(box *ebiten.Image) {
			render.DrawMite(box, mid(), ph)
		})
	}
	cell(screen, 4*cw, 2*ch, cw, ch, "MITE p2", func(box *ebiten.Image) {
		render.DrawMite(box, mid(), 3.6)
	})
}

// drawAttached paints the sucker pose on both walls at mouth time t
// (0.75 = sealed, 1.25 = wide open — half a pump cycle apart).
func drawAttached(screen *ebiten.Image, t float64) {
	screen.Fill(color.RGBA{R: 6, G: 8, B: 24, A: 255})
	// faint glass edges for context
	ui.FillRect(screen, 12, 0, 2, h, "#3d5a8a", 0.5)
	ui.FillRect(screen, w-14, 0, 2, h, "#3d5a8a", 0.5)
	n := contract.SpineSegments
	seg := 7.0
	spine := make([]contract.Vec2, n)
	anim := render.FishAnim{Time: t, Speed01: 0, Attached: true}
	for _, side := range []struct {
		wall int
		x    float64
		y    float64
	}{{-1, 40, h * 0.35}, {1, w - 40, h * 0.65}} {
		for i := 0; i < n; i++ {
			spine[i] = v2f(side.x-float64(side.wall)*seg*float64(i), side.y)
		}
		anim.AttachSide = side.wall
		render.DrawFish(screen, screen, spine, &pleco, &pleco.Palette, "adult", 0.3, anim)
	}
	ui.DrawText(screen, "F18 SUCKER  mouth phase t="+strconv.FormatFloat(t, 'f', 2, 64), 24, 20, 1, "#8fb8e8", 0.9)
}

func (d *demo) Draw(screen *ebiten.Image) {
	switch {
	case d.att != "": // two mouth phases ~half a pump cycle apart
		if d.frame == 8 {
			drawAttached(screen, 0.75)
			if err := render.SavePNG(screen, d.att+"-1.png"); err != nil {
				panic(err)
			}
		}
		if d.frame >= 16 {
			drawAttached(screen, 1.25)
			if err := render.SavePNG(screen, d.att+"-2.png"); err != nil {
				panic(err)
			}
			os.Exit(0)
		}
	case d.out != "":
		if d.frame >= 5 {
			drawGallery(screen)
			if err := render.SavePNG(screen, d.out); err != nil {
				panic(err)
			}
			os.Exit(0)
		}
	}
}

func (d *demo) Layout(ow, oh int) (int, int) { return w, h }

// v2f builds a contract.Vec2 without importing contract at call sites twice.
func v2f(x, y float64) contract.Vec2 { return contract.Vec2{X: x, Y: y} }

func main() {
	out := flag.String("png", "", "save the gallery frame to this path and exit")
	att := flag.String("attached", "", "save two sucker-pose frames (<p>-1.png/<p>-2.png) and exit")
	flag.Parse()
	d := &demo{out: *out, att: *att}
	ebiten.SetWindowTitle("NEON TANK — decor gallery")
	ebiten.SetWindowSize(w, h)
	if err := ebiten.RunGame(d); err != nil && err != ebiten.Termination {
		panic(err)
	}
}
