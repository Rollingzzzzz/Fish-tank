// G2.x demo: the full living tank — shader water, plants, trails, fish,
// food, bubbles — driven by the real sim. Click to feed; swipe to scatter.
// `--png out.png` saves a frame for self-review.
package main

import (
	"flag"
	"fmt"
	"math"
	"os"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/Rollingzzzzz/Fish-tank/internal/render"
	"github.com/Rollingzzzzz/Fish-tank/internal/sim"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	w = 1280
	h = 720
)

var species = []contract.Species{
	{ID: "stripe-demo", Name: "Stripe Demo", Size: 1, Width: 1, Fin: 1, Tail: 1,
		Palette:  contract.Palette{Body: "#1b6bd6", Belly: "#0a1f3d", Accent: "#ffd23d", Glow: "#7fd4ff"},
		Pattern:  contract.Pattern{Type: "stripe", Density: 0.6, Size: 0.5},
		Behavior: contract.Behavior{Speed: 1, Schooling: 0.9, Curiosity: 0.6, Depth: 0.35}},
	{ID: "spot-demo", Name: "Spot Demo", Size: 1, Width: 1.15, Fin: 1.2, Tail: 1,
		Palette:  contract.Palette{Body: "#8a2be2", Belly: "#1c0a3d", Accent: "#3dffb8", Glow: "#e0b3ff"},
		Pattern:  contract.Pattern{Type: "spot", Density: 0.7, Size: 0.4},
		Behavior: contract.Behavior{Speed: 0.9, Schooling: 0.6, Curiosity: 0.8, Skittish: 0.6, Depth: 0.5}},
	{ID: "koi-demo", Name: "Koi Demo", Size: 1.2, Width: 1.1, Fin: 1.35, Tail: 1.25,
		Palette:  contract.Palette{Body: "#ff7a1f", Belly: "#fff1e0", Accent: "#ff1f5a", Glow: "#ffd9a0"},
		Pattern:  contract.Pattern{Type: "koi", Density: 0.5, Size: 0.7},
		Behavior: contract.Behavior{Speed: 0.7, Schooling: 0.1, Curiosity: 0.3, Depth: 0.75}},
	{ID: "vein-demo", Name: "Vein Demo", Size: 0.9, Width: 0.9, Fin: 1.5, Tail: 1.4,
		Palette:  contract.Palette{Body: "#0d5c4f", Belly: "#032b25", Accent: "#3dfff0", Glow: "#b6fff8"},
		Pattern:  contract.Pattern{Type: "vein", Density: 0.6, Size: 0.5},
		Behavior: contract.Behavior{Speed: 1.1, Schooling: 0.3, Curiosity: 0.5, Skittish: 0.8, Depth: 0.6, NightActive: true}},
	{ID: "wave-demo", Name: "Wave Demo", Size: 1.05, Width: 0.95, Fin: 0.9, Tail: 1.1,
		Palette:  contract.Palette{Body: "#d6439a", Belly: "#3d0a26", Accent: "#ffe13d", Glow: "#ffb3e2"},
		Pattern:  contract.Pattern{Type: "wave", Density: 0.5, Size: 0.6},
		Behavior: contract.Behavior{Speed: 1.2, Schooling: 0.5, Curiosity: 0.4, Depth: 0.3}},
}

var plants = []contract.PlantDesign{
	{ID: "kelp", Name: "Spiral Kelp", Fronds: 6, Height: 0.4, Width: 1.0, Curve: 0.85, Sway: 0.7,
		Colors: []string{"#0d3b4f", "#00ffd0", "#b8ffef"}, Glow: 0.8},
	{ID: "tendril", Name: "Bubble Tendril", Fronds: 4, Height: 0.48, Width: 0.8, Curve: 0.4, Sway: 0.9,
		Colors: []string{"#3a1052", "#c13dff", "#ffd9fa"}, Glow: 1.0},
	{ID: "frond", Name: "Lime Frond", Fronds: 5, Height: 0.28, Width: 1.3, Curve: 0.15, Sway: 0.4,
		Colors: []string{"#123a12", "#8dff3d", "#e9ffd1"}, Glow: 0.55},
	{ID: "whip", Name: "Magenta Whip", Fronds: 3, Height: 0.52, Width: 0.6, Curve: 0.95, Sway: 1.0,
		Colors: []string{"#4f0d2a", "#ff3d8f", "#ffe0ef"}, Glow: 0.95},
	{ID: "fan", Name: "Cyan Fan", Fronds: 8, Height: 0.24, Width: 1.1, Curve: 0.55, Sway: 0.55,
		Colors: []string{"#0a2c4f", "#3dc9ff", "#d9f6ff"}, Glow: 0.7},
}

var water = &contract.WaterPreset{
	ID: "deep-neon", Name: "Deep Neon",
	TopColor: "#17346e", BottomColor: "#04081c", Accent: "#00ffe1",
	Rays: 0.0, Caustics: 0.0, Bubbles: 0.5,
	PlantPalette: []string{"#00ffd0", "#c13dff", "#8dff3d"},
}

type demo struct {
	world    *sim.World
	bg       *render.Background
	trail    *render.Trail
	frame    int
	out      string
	prevMX   int
	prevMY   int
	mouseSpd float64
}

func (d *demo) Update() error {
	if d.frame == 1 {
		fmt.Println("shader compiled OK:", d.bg.OK())
	}
	d.frame++
	dFrame = d.frame
	mx, my := ebiten.CursorPosition()
	d.mouseSpd = hypot(float64(mx-d.prevMX), float64(my-d.prevMY)) * 60
	d.prevMX, d.prevMY = mx, my
	in := sim.Input{MouseX: float64(mx), MouseY: float64(my), MouseActive: mx > 0 || my > 0, MouseSpeed: d.mouseSpd}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		in.FeedAt = append(in.FeedAt, contract.Vec2{X: float64(mx), Y: float64(my)})
	}
	d.world.Update(1/60.0, in)
	return nil
}

func hypot(x, y float64) float64 { return math.Sqrt(x*x + y*y) }

func sqrtx(v float64) float64 { return math.Sqrt(v) }

func (d *demo) Draw(screen *ebiten.Image) {
	wl := d.world.WaterLive()
	d.bg.Draw(screen, render.WaterState{
		Time: wl.Time, DayFactor: wl.DayFactor, TOD: wl.TOD, Top: wl.Top, Bottom: wl.Bottom,
		Accent: wl.Accent, Rays: wl.Rays, Caustics: wl.Caustics,
	})
	d.trail.Fade(0.30)
	// d.trail.Blit(screen) // bisect: blit off

	d.trail.Fade(0.30)
	d.trail.Blit(screen)

	// plants
	night := wl.Night
	for _, pl := range d.world.Plants() {
		render.DrawPlant(screen, d.trail.Image(), pl.Def, pl.X, pl.Y, float64(h)*pl.Def.Height*pl.Sc, wl.Time, night)
	}

	// food + eggs
	for _, fd := range d.world.Foods() {
		render.DrawGlow(screen, fd.Pos.X, fd.Pos.Y, 5, "#ffe9a0", 0.7)
		render.DrawOrb(screen, fd.Pos.X, fd.Pos.Y, 1.8, "#fff6d8", 230)
	}
	for _, eg := range d.world.Eggs() {
		pulse := 3.5 + 1.5*math.Abs(math.Sin(wl.Time*4+float64(eg.Seed%10)))
		render.DrawOrb(screen, eg.Pos.X, eg.Pos.Y, 6+pulse*0.4, "#c9f7ff", 120)
		render.DrawGlow(d.trail.Image(), eg.Pos.X, eg.Pos.Y, 16+pulse*2, "#7fd4ff", 0.35)
	}

	// fish
	for _, f := range d.world.Fishes() {
		anim := render.FishAnim{Time: wl.Time, Speed01: clamp01(hypot2v(f.Vel) / 110)}
		render.DrawFish(screen, d.trail.Image(), f.Spine, f.Sp, &f.Pal, f.Stage, night, anim)
	}

	// sparkles + bubbles
	for _, p := range d.world.Particles() {
		a := p.Life / p.Max
		render.DrawGlow(screen, p.Pos.X, p.Pos.Y, p.Size*4, p.Color, 0.6*a)
	}
	for _, b := range d.world.Bubbles() {
		render.DrawOrb(screen, b.Pos.X, b.Pos.Y, b.R, "#bfe9ff", 90)
	}
	if d.out != "" && d.frame >= frameTarget() {
		if err := render.SavePNG(screen, d.out); err != nil {
			panic(err)
		}
		os.Exit(0)
	}
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
func hypot2v(v contract.Vec2) float64 { return sqrtx(v.X*v.X + v.Y*v.Y) }

func (d *demo) Layout(ow, oh int) (int, int) { return w, h }

var dFrame int

func frameTarget() int {
	if v := os.Getenv("FRAMES"); v != "" {
		n := 0
		for _, c := range v {
			n = n*10 + int(c-'0')
		}
		return n
	}
	return 120
}

func main() {
	out := flag.String("png", "", "save a frame to this path and exit")
	tod := flag.Float64("tod", -1, "time of day 0..1 (0 dawn, .25 noon, .5 dusk, .75 night); pins the clock")
	flag.Parse()
	cfg := contract.Config{MaxFish: 24, DaySeconds: 60}
	sps := make([]*contract.Species, len(species))
	for i := range species {
		sps[i] = &species[i]
	}
	pds := make([]*contract.PlantDesign, len(plants))
	for i := range plants {
		pds[i] = &plants[i]
	}
	d := &demo{
		world: sim.NewWorld(w, h, cfg, sps, pds, []*contract.WaterPreset{water}),
		bg:    render.NewBackground(),
		trail: render.NewTrail(w, h),
		out:   *out,
	}
	if *tod >= 0 { // F9: pin the clock over the full day+night cycle
		d.world.Clock = *tod * 2 * cfg.DaySeconds
	}
	ebiten.SetWindowTitle("NEON TANK — world demo")
	ebiten.SetWindowSize(w, h)
	if err := ebiten.RunGame(d); err != nil && err != ebiten.Termination {
		panic(err)
	}
}
