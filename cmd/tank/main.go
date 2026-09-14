// G0.1/G6.1: NEON TANK entry point.
package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	_ "image/png"
	"os"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/Rollingzzzzz/Fish-tank/internal/game"
	"github.com/hajimehoshi/ebiten/v2"
)

// v1: the window and taskbar carry the neon-fish mark (the exe icon comes
// from resource.syso).
//
//go:embed icon.png
var iconPNG []byte

// cliArgs is the parsed launch line.
type cliArgs struct {
	smoke    bool
	shotDir  string
	probeDir string // v1.1: -probe evidence pass (ambient features)
	w, h     int    // logical size override (F13 aspect matrix), 0 = auto
}

func parseArgs(args []string) cliArgs {
	var c cliArgs
	for i, a := range args {
		switch a {
		case "-menu-smoke", "--menu-smoke":
			c.smoke = true
		case "-shot", "--shot":
			if i+1 < len(args) {
				c.shotDir = args[i+1]
			}
		case "-probe", "--probe":
			if i+1 < len(args) {
				c.probeDir = args[i+1]
			}
		case "-winsize", "--winsize":
			if i+1 < len(args) {
				var w, h int
				if _, err := fmt.Sscanf(args[i+1], "%dx%d", &w, &h); err == nil {
					c.w, c.h = w, h
				}
			}
		}
	}
	return c
}

func main() {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	cli := parseArgs(os.Args[1:])

	// v0.3.4: BACK to the fast 720-locked proportional canvas (the layout the
	// tank ran 60 FPS at). Rendering 1280-class pixels and letting the GPU
	// scale beats rendering millions of physical pixels per frame on this
	// machine. Must run BEFORE Bootstrap — the world sizes from it.
	winW, winH := 1280, 720
	if cli.w > 0 && cli.h > 0 {
		winW, winH = cli.w, cli.h
		game.SetLogicalSize(cli.w, cli.h)
	} else if mons := ebiten.AppendMonitors(nil); len(mons) > 0 {
		mw, mh := mons[0].Size()
		winW = game.LogicalWidthFor(mw, mh)
		winH = contract.LogicalH
		game.SetLogicalSize(winW, winH)
	}

	g, err := game.Bootstrap(wd)
	if err != nil {
		fmt.Println("NEON TANK:", err)
		os.Exit(1)
	}
	if cli.smoke {
		g.StartSmoke() // F5: scripted UI acceptance (exit 0 = pass)
	}
	if cli.shotDir != "" {
		g.StartShot(cli.shotDir) // F19: evidence capture harness
	}
	if cli.probeDir != "" {
		g.StartProbe(cli.probeDir) // v1.1: titan + critter evidence pass
	}
	ebiten.SetWindowTitle("NEON TANK")
	ebiten.SetWindowSize(winW, winH)
	if icon, _, err := image.Decode(bytes.NewReader(iconPNG)); err == nil {
		ebiten.SetWindowIcon([]image.Image{icon}) // v1: taskbar/window mark
	}
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if g.FullscreenStart() { // F13: open immersive, edge to edge
		ebiten.SetFullscreen(true)
	}
	if err := ebiten.RunGame(g); err != nil && err != ebiten.Termination {
		panic(err)
	}
}
