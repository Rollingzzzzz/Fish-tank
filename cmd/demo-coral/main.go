// N1 demo: the four seeded coral designs over a volcanic rock layout —
// the acceptance window for fan/branch/brain rendering and the two-pass
// cave/nest draw. -png <path> saves frame 90 and exits (headless checks).
package main

import (
	"flag"
	"image/color"
	"os"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/Rollingzzzzz/Fish-tank/internal/render"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	w        = 1280
	h        = 720
	rockSeed = 42
)

// designs mirror internal/content/seed/corals/*.json (deterministic demo).
var designs = []contract.CoralDesign{
	{ID: "violet-seafern", Name: "Violet Seafern", Kind: "fan", Fronds: 7, Height: 0.18,
		Width: 1.0, Curve: 0.55, Sway: 0.5, Colors: []string{"#5a2d9e", "#8a5cff", "#d9c2ff"}, Glow: 0.7},
	{ID: "cyan-candlebranch", Name: "Cyan Candlebranch", Kind: "branch", Fronds: 6, Height: 0.22,
		Width: 0.9, Curve: 0.4, Sway: 0.3, Colors: []string{"#0a3d5c", "#18c5ff", "#b8f4ff"}, Glow: 0.65},
	{ID: "magma-brain", Name: "Magma Brain", Kind: "brain", Fronds: 4, Height: 0.14,
		Width: 1.4, Curve: 0.3, Sway: 0.1, Colors: []string{"#47190b", "#a8481f", "#e08a45"}, Glow: 0.4},
	{ID: "rose-veilfan", Name: "Rose Veilfan", Kind: "fan", Fronds: 8, Height: 0.16,
		Width: 1.2, Curve: 0.7, Sway: 0.6, Colors: []string{"#8a1d4f", "#ff3d8f", "#ffd2e8"}, Glow: 0.8},
}

type demo struct {
	rock  *render.RockLayout
	frame int
	out   string
	spots []render.CoralSpot
}

func newDemo(out string) *demo {
	d := &demo{rock: render.NewRockLayout(rockSeed, w, h), out: out}
	floor := h * 0.925
	for i := range designs {
		d.spots = append(d.spots, render.CoralSpot{
			Def: &designs[i], X: 115 + float64(i)*225, Y: floor, Sc: 1})
	}
	return d
}

func (d *demo) Update() error { d.frame++; return nil }

func (d *demo) Draw(screen *ebiten.Image) {
	t := float64(d.frame) / 60
	night := 0.65 // dusk: glow visible but cracks still restrained
	screen.Fill(color.RGBA{R: 8, G: 6, B: 22, A: 255})
	d.rock.DrawBack(screen, night)
	render.DrawCorals(screen, d.spots, t, night)
	d.rock.DrawFront(screen, night)
	d.rock.DrawNest(screen, t, 0.9)
	if d.out != "" && d.frame >= 90 {
		if err := render.SavePNG(screen, d.out); err != nil {
			panic(err)
		}
		os.Exit(0)
	}
}

func (d *demo) Layout(ow, oh int) (int, int) { return w, h }

func main() {
	out := flag.String("png", "", "save a frame to this path and exit")
	flag.Parse()
	ebiten.SetWindowTitle("NEON TANK — corals demo")
	ebiten.SetWindowSize(w, h)
	if err := ebiten.RunGame(newDemo(*out)); err != nil && err != ebiten.Termination {
		panic(err)
	}
}
