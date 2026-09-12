// F19 capture side of the -shot harness: frozen-pixel PNG writes (Update
// freezes at the exact capture frame, Draw writes the file), the expected
// control-rect manifest and the final report. Split from shot.go (ceiling).
package game

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Rollingzzzzz/Fish-tank/internal/render"
	"github.com/Rollingzzzzz/Fish-tank/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

// clusterPct is the v0.3.8 roaming metric: share of fish hovering within
// 200px of any crag-door center — the "huddled at the towers" reading.
func (g *Game) clusterPct() int {
	holes := g.world.Holes()
	fishes := g.world.Fishes()
	if len(fishes) == 0 {
		return 0
	}
	near := 0
	for _, f := range fishes {
		for _, h := range holes {
			dx, dy := f.Pos.X-h.Center.X, f.Pos.Y-h.Center.Y
			if dx*dx+dy*dy < 200*200 {
				near++
				break
			}
		}
	}
	return near * 100 / len(fishes)
}

// shotCapture saves one queued frame PNG and records the expected-rect
// manifest entries (called from Draw, after everything else is rendered).
func (g *Game) shotCapture(screen *ebiten.Image) {
	s := g.shot
	if s == nil || len(s.queue) == 0 {
		return
	}
	name := s.queue[0]
	s.queue = s.queue[1:]
	if len(s.queue) == 0 {
		s.drained = true
	}
	shares := func() [4]float64 { p, r, c, o := g.world.FloorShares(); return [4]float64{p, r, c, o} }()
	info := s.pendInfo[name]
	if info == "" {
		info = fmt.Sprintf("canvas=%dx%d fish=%d alive=%d TPS=%.0f FPS=%.0f menu=%v tray=%v grab=%q treats=%d cluster=%d%%",
			ScreenW, ScreenH, len(g.world.Fishes()), g.world.AliveFishes(),
			ebiten.ActualTPS(), ebiten.ActualFPS(), g.menu.Open(), g.tray.Open, g.grabKind, len(g.world.Treats()), g.clusterPct())
	}
	fmt.Printf("shot: %-22s %s floor(plants/rocks/corals/open)=%.0f%%/%.0f%%/%.0f%%/%.0f%% plants=%d mites=%d cluster=%d%%\n",
		name, info, shares[0]*100, shares[1]*100, shares[2]*100, shares[3]*100,
		len(g.world.Plants()), len(g.world.Mites()), g.clusterPct())
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		fmt.Println("shot:", err)
		return
	}
	pix, ok := s.pending[name]
	if !ok {
		pix = make([]byte, 4*screen.Bounds().Dx()*screen.Bounds().Dy())
		screen.ReadPixels(pix) // fallback: no frozen snapshot available
	}
	if err := render.SavePNGBytes(pix, ScreenW, ScreenH, filepath.Join(s.dir, name)); err != nil {
		fmt.Println("shot:", err)
		return
	}
	delete(s.pending, name)
	delete(s.pendInfo, name)
	s.captured++
	s.manifest = append(s.manifest, g.manifestRects(name)...)
	fmt.Println("shot: captured", name)
}

// manifestRects lists the controls that must be visible in the named shot.
func (g *Game) manifestRects(name string) []shotRect {
	var out []shotRect
	add := func(n string, r ui.Rect, img string) {
		out = append(out, shotRect{Name: n, X: r.X, Y: r.Y, W: r.W, H: r.H, Image: img})
	}
	addScene := func(n string, r ui.Rect, img string) {
		out = append(out, shotRect{Name: n, X: r.X, Y: r.Y, W: r.W, H: r.H, Image: img, Kind: "scene"})
	}
	open := ui.UIMetrics(ScreenW, ScreenH, true)
	add("tray-button", open.TrayBtn, name)
	add("close-button", open.CloseBtn, name) // v0.3.3: the X must always be visible
	if name == "01-closed.png" {
		closed := ui.UIMetrics(ScreenW, ScreenH, false)
		add("menu-handle", closed.Handle, name)
		add("menu-chip", closed.HandleChip, name)
		return out
	}
	if name == "10-carry-menu-closed.png" {
		// F26: menu is closed here — only the always-visible chrome plus the
		// wriggling carried worm and its scent-crowd are expected
		closed := ui.UIMetrics(ScreenW, ScreenH, false)
		add("menu-handle", closed.Handle, name)
		add("menu-chip", closed.HandleChip, name)
		addScene("held-treat", ui.Rect{X: g.shot.carry2X - 24, Y: g.shot.carry2Y - 24, W: 48, H: 48}, name)
		return out
	}
	if name == "11-transit-enter.png" || name == "12-transit-inside.png" || name == "13-transit-exit.png" {
		// F25: the left volcanic crag carries the door-to-door sequence
		closed := ui.UIMetrics(ScreenW, ScreenH, false)
		add("menu-handle", closed.Handle, name)
		add("menu-chip", closed.HandleChip, name)
		addScene("crag-left", ui.Rect{X: 30, Y: 280, W: 380, H: 440}, name)
		return out
	}
	if name == "14-flora.png" || name == "15-crowd.png" {
		// F30: the staged silky flora row across the floor; F29: the crowd
		// frame also carries the population-pressure evidence
		closed := ui.UIMetrics(ScreenW, ScreenH, false)
		add("menu-handle", closed.Handle, name)
		add("menu-chip", closed.HandleChip, name)
		if name == "14-flora.png" {
			addScene("flora-band", ui.Rect{X: 120, Y: ScreenH - 340, W: ScreenW - 240, H: 320}, name)
		} else {
			addScene("water-column", ui.Rect{X: 120, Y: 80, W: ScreenW - 240, H: ScreenH - 260}, name)
		}
		return out
	}
	add("menu-handle", open.Handle, name)
	add("menu-chip", open.HandleChip, name)
	for i, t := range open.Tabs {
		add(fmt.Sprintf("tab-%s", uiTabSlug(i)), t, name)
	}
	addScene("tab-content", open.Content, name)
	if name == "06-tab-setup.png" {
		// the toggles the player actually reaches for — Auto feed is the one
		// with an explicit proof requirement in the acceptance list
		for _, row := range []string{"Auto feed", "Music", "Start fullscreen"} {
			if r, ok := g.menu.SettingsRowRect(open.Content, row); ok {
				add("settings-"+row, r, name)
			}
		}
	}
	switch {
	case name == "07-tray-open.png":
		add("tray-panel", open.TrayPanel, name)
		for i, r := range open.TrayRows {
			add(fmt.Sprintf("tray-row-%d", i), r, name)
		}
	case name == "08-held-worm.png":
		addScene("held-treat", ui.Rect{X: g.shot.holdX - 24, Y: g.shot.holdY - 24, W: 48, H: 48}, name)
	}
	return out
}

// finishShot writes the manifest and reports (F19 exit path).
func (g *Game) finishShot() {
	s := g.shot
	if s == nil {
		return
	}
	b, err := json.MarshalIndent(s.manifest, "", "  ")
	if err != nil {
		fmt.Println("shot:", err)
		return
	}
	if err := os.WriteFile(filepath.Join(s.dir, "manifest.json"), b, 0o644); err != nil {
		fmt.Println("shot:", err)
		return
	}
	// F29 evidence: the frame rate the full scripted load produced.
	fmt.Printf("SHOT RATE: fish=%d TPS=%.0f FPS=%.0f\n",
		len(g.world.Fishes()), ebiten.ActualTPS(), ebiten.ActualFPS())
	// v0.3.8 evidence: how much of the school is huddled at the crags.
	fmt.Printf("SHOT CLUSTER: nearDoor=%d%%\n", g.clusterPct())
	fmt.Printf("SHOT OK: %d pngs + manifest.json in %s\n", s.captured, s.dir)
}
