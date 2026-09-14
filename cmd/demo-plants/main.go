// G1.3 demo: five alien plant designs swaying over the shader water.
package main

import (
	"flag"
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

var designs = []contract.PlantDesign{
	{ID: "spiral-kelp", Name: "Spiral Kelp", Fronds: 6, Height: 0.42, Width: 1.0, Curve: 0.85, Sway: 0.7,
		Colors: []string{"#0d3b4f", "#00ffd0", "#b8ffef"}, Glow: 0.8},
	{ID: "bubble-tendril", Name: "Bubble Tendril", Fronds: 4, Height: 0.5, Width: 0.8, Curve: 0.4, Sway: 0.9,
		Colors: []string{"#3a1052", "#c13dff", "#ffd9fa"}, Glow: 1.0},
	{ID: "lime-frond", Name: "Lime Frond", Fronds: 5, Height: 0.3, Width: 1.3, Curve: 0.15, Sway: 0.4,
		Colors: []string{"#123a12", "#8dff3d", "#e9ffd1"}, Glow: 0.55},
	{ID: "magenta-whip", Name: "Magenta Whip", Fronds: 3, Height: 0.55, Width: 0.6, Curve: 0.95, Sway: 1.0,
		Colors: []string{"#4f0d2a", "#ff3d8f", "#ffe0ef"}, Glow: 0.95},
	{ID: "cyan-fan", Name: "Cyan Fan", Fronds: 8, Height: 0.26, Width: 1.1, Curve: 0.55, Sway: 0.55,
		Colors: []string{"#0a2c4f", "#3dc9ff", "#d9f6ff"}, Glow: 0.7},
}

type demo struct {
	bg    *render.Background
	trail *render.Trail
	frame int
	out   string
}

func (d *demo) Update() error { d.frame++; return nil }

func (d *demo) Draw(screen *ebiten.Image) {
	t := float64(d.frame) / 60
	day := 0.35 + 0.3*math.Sin(t*0.1)
	hx := func(s string) [3]float64 {
		r, g, b := contract.HexToRGB(s)
		return [3]float64{float64(r) / 255, float64(g) / 255, float64(b) / 255}
	}
	night := 1 - day
	d.bg.Draw(screen, render.WaterState{TOD: 0.25, // noon — the demo pins a neutral phaseTime: t, DayFactor: day,
		Top: hx("#17346e"), Bottom: hx("#04081c"), Accent: hx("#00ffe1"), Rays: 0.7, Caustics: 0.8})
	d.trail.Fade(0.30)
	for i := range designs {
		pd := designs[i]
		x := w * (0.12 + 0.76*float64(i)/float64(len(designs)-1))
		render.DrawPlant(screen, d.trail.Image(), &pd, x, h-8, h*pd.Height, t, night)
	}
	d.trail.Blit(screen)
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
	d := &demo{bg: render.NewBackground(), trail: render.NewTrail(w, h), out: *out}
	ebiten.SetWindowTitle("NEON TANK — plants demo")
	ebiten.SetWindowSize(w, h)
	if err := ebiten.RunGame(d); err != nil && err != ebiten.Termination {
		panic(err)
	}
}
