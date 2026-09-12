// F26: click-toggle treat carry — game-layer acceptance. Press #1 on a tray
// row picks the treat up; it rides the cursor with the button UP; the next
// left press drops it exactly there, feeds nothing and leaves the UI inert.
// Pure headless: processInput only (no Draw, no ebiten Run).
package game

import (
	"testing"

	"github.com/Rollingzzzzz/Fish-tank/internal/agents"
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/Rollingzzzzz/Fish-tank/internal/sim"
	"github.com/Rollingzzzzz/Fish-tank/internal/ui"
)

func newCarryGame() *Game {
	cfg := &contract.Config{Endpoint: "http://localhost", Model: "glm-5.3-flash"}
	return &Game{
		cfg:   cfg,
		world: sim.NewWorld(1280, 720, *cfg, nil, nil, nil),
		menu:  ui.NewMenu(cfg),
		tray:  ui.NewTreatTray(),
		hub:   agents.NewHub(*cfg, nil, contract.AgentHooks{}),
	}
}

// grabWormFixture grabs a worm with the menu closed (the case that used to
// drop the treat on the very grab frame).
func (g *Game) grabWormFixture(t *testing.T) {
	t.Helper()
	m := ui.UIMetrics(ScreenW, ScreenH, false)
	bx, by := m.TrayBtn.X+m.TrayBtn.W/2, m.TrayBtn.Y+m.TrayBtn.H/2
	g.processInput(bx, by, true, false, false, false)
	g.processInput(bx, by, false, true, false, false)
	if !g.tray.Open {
		t.Fatal("tray did not open")
	}
	r := m.TrayRows[1]
	g.processInput(r.X+r.W/2, r.Y+r.H/2, true, false, false, false)
	g.processInput(r.X+r.W/2, r.Y+r.H/2, false, true, false, false)
	if g.grabKind != contract.TreatWorm {
		t.Fatalf("press on worm row must grab, got %q", g.grabKind)
	}
	if g.tray.Open {
		t.Fatal("tray must close on grab")
	}
}

func TestCarryToggleMenuClosed(t *testing.T) {
	g := newCarryGame()
	g.grabWormFixture(t)

	// carry persists with the button UP, menu closed, for 60+ frames
	for i := 0; i < 60; i++ {
		g.processInput(600+i, 300, false, false, false, false)
	}
	if g.grabKind != contract.TreatWorm || g.tray.Open {
		t.Fatalf("carry lost: grab=%q trayOpen=%v", g.grabKind, g.tray.Open)
	}

	// the next press drops exactly there and feeds nothing
	foods := len(g.world.Foods())
	g.processInput(500, 400, true, false, false, false)
	tr := g.world.Treats()
	if g.grabKind != "" || len(tr) != 1 {
		t.Fatalf("drop failed: grab=%q treats=%d", g.grabKind, len(tr))
	}
	if d := tr[0].Pos.X - 500; d < -2 || d > 2 {
		t.Fatalf("treat dropped at x=%.1f, want 500", tr[0].Pos.X)
	}
	if d := tr[0].Pos.Y - 400; d < -2 || d > 2 {
		t.Fatalf("treat dropped at y=%.1f, want 400", tr[0].Pos.Y)
	}
	if len(g.world.Foods()) != foods {
		t.Fatal("the drop click must not also feed the tank")
	}
}

func TestCarrySwallowsCloseButton(t *testing.T) {
	g := newCarryGame()
	g.grabWormFixture(t)
	cb := ui.UIMetrics(ScreenW, ScreenH, false).CloseBtn
	g.processInput(cb.X+cb.W/2, cb.Y+cb.H/2, true, false, false, false)
	if g.quit {
		t.Fatal("pressing X while carrying must not quit the app")
	}
	if g.grabKind != "" || len(g.world.Treats()) != 1 {
		t.Fatal("pressing X while carrying must drop the treat")
	}
}

func TestCarryScentGrows(t *testing.T) {
	g := newCarryGame()
	g.grabWormFixture(t)
	for i := 0; i < 300; i++ { // 5 s of carrying
		g.processInput(640, 360, false, false, false, false)
	}
	if g.grabSec < 4.5 {
		t.Fatalf("carry age = %.1fs, want ~5", g.grabSec)
	}
}
