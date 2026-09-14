// G1.1 demo: animated water shader with a slow day/night sweep.
// `go run ./cmd/demo-shader` or `go run ./cmd/demo-shader --png frame.png`
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

type demo struct {
	bg    *render.Background
	frame int
	out   string
}

func (d *demo) Update() error {
	if d.frame == 1 {
		println("SHADER BUILD MARKER v7-red-final")
	}
	d.frame++
	return nil
}

func (d *demo) Draw(screen *ebiten.Image) {
	t := float64(d.frame) / 60
	day := 0.5 + 0.5*math.Sin(t*0.12)
	hx := func(s string) [3]float64 {
		r, g, b := contract.HexToRGB(s)
		return [3]float64{float64(r) / 255, float64(g) / 255, float64(b) / 255}
	}
	d.bg.Draw(screen, render.WaterState{
		Time: t, DayFactor: day,
		Top: hx("#17346e"), Bottom: hx("#04081c"), Accent: hx("#00ffe1"),
		Rays: 0.75, Caustics: 0.85,
	})
	if d.out != "" && d.frame >= 90 {
		if err := render.SavePNG(screen, d.out); err != nil {
			panic(err)
		}
		if os.Getenv("KEEP_OPEN") == "" {
			os.Exit(0)
		}
	}
}

func (d *demo) Layout(ow, oh int) (int, int) { return w, h }

func main() {
	out := flag.String("png", "", "save a frame to this path and exit")
	flag.Parse()
	d := &demo{bg: render.NewBackground(), out: *out}
	ebiten.SetWindowTitle("NEON TANK — shader demo")
	ebiten.SetWindowSize(w, h)
	if err := ebiten.RunGame(d); err != nil && err != ebiten.Termination {
		panic(err)
	}
}
