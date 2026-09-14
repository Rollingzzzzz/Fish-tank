// F5/N8/N10: -menu-smoke mode — a scripted input timeline that drives the real
// UI (handle, tabs, treat tray, right-click scare) frame by frame and asserts
// state transitions. Exit 0 = all assertions held. This is the automated
// acceptance for the menu interaction fix: it fails exactly like a human
// click would. All coordinates derive from ui.UIMetrics — the same geometry
// the renderer uses (F12/F19: zero drift between test and screen).
package game

import (
	"fmt"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/Rollingzzzzz/Fish-tank/internal/ui"
)

// smokeStep drives the cursor until frame `until` (exclusive); `check` runs
// once, at the end of the step, against the state the step produced. `act`
// (shot harness) fires once when the step first becomes active.
type smokeStep struct {
	until    int
	mx, my   int
	down     bool // left button LEVEL state
	rdown    bool // right button LEVEL state (N10)
	name     string
	check    func(g *Game) bool
	feedback string
	act      func(g *Game) // F25: one-shot side effect on step entry
}

// smokeRun is the active scripted session (nil = normal play).
type smokeRun struct {
	frame    int
	prevDown bool
	prevR    bool
	foodN    int // food count snapshot before the scare (restores carry food)
	steps    []smokeStep
	fails    []string
}

// startSmoke builds the acceptance timeline.
func (g *Game) startSmoke() {
	closed := ui.UIMetrics(ScreenW, ScreenH, false)
	open := ui.UIMetrics(ScreenW, ScreenH, true)
	hx := closed.Handle.X + closed.Handle.W/2
	hy := closed.Handle.Y + closed.Handle.H/2
	ohx := open.Handle.X + open.Handle.W/2 // open-position handle (panel edge)
	ohy := open.Handle.Y + open.Handle.H/2
	tabC := func(i int) (int, int) {
		t := open.Tabs[i]
		return t.X + t.W/2, t.Y + t.H/2
	}
	trayX := open.TrayBtn.X + open.TrayBtn.W/2
	trayY := open.TrayBtn.Y + open.TrayBtn.H/2
	worm := open.TrayRows[1]
	wormX, wormY := worm.X+worm.W/2, worm.Y+worm.H/2
	t1x, t1y := tabC(1)
	t2x, t2y := tabC(2)
	t4x, t4y := tabC(4)
	// the scripted timeline races the auto-feeder otherwise: a random feeder
	// drop inside the scare window looks exactly like "right click fed the
	// tank". Scripted runs feed by hand, never on the timer (in-memory only).
	g.cfg.AutoFeed = false
	g.smoke = &smokeRun{steps: []smokeStep{
		{until: 8, mx: hx, my: hy, name: "closed-at-start", check: func(g *Game) bool { return !g.menu.Open() }},
		{until: 14, mx: hx, my: hy, down: true, name: "press-handle"},
		{until: 16, mx: hx, my: hy, name: "release-handle"},
		{until: 45, mx: hx, my: hy, name: "menu-opens", check: func(g *Game) bool { return g.menu.Open() },
			feedback: "menu did not open on first click"},
		{until: 50, mx: t1x, my: t1y, down: true, name: "press-tab1"},
		{until: 52, mx: t1x, my: t1y, name: "release-tab1"},
		{until: 58, mx: t1x, my: t1y, name: "tab1-active", check: func(g *Game) bool { return g.menu.ActiveTab() == 1 },
			feedback: "species tab did not activate"},
		{until: 62, mx: t2x, my: t2y, down: true, name: "press-tab2"},
		{until: 64, mx: t2x, my: t2y, name: "release-tab2"},
		{until: 72, mx: t2x, my: t2y, name: "tab2-active", check: func(g *Game) bool { return g.menu.ActiveTab() == 2 },
			feedback: "plants tab did not activate"},
		{until: 76, mx: t4x, my: t4y, down: true, name: "press-tab4"},
		{until: 78, mx: t4x, my: t4y, name: "release-tab4"},
		{until: 92, mx: t4x, my: t4y, name: "tab4-active", check: func(g *Game) bool { return g.menu.ActiveTab() == 4 },
			feedback: "setup tab did not activate"},
		// Timing contract of this timeline: a step is active while frame <
		// until, and a step's check observes the state at the END of frame
		// `until`. So a press that would invalidate a check (the grab press
		// closes the tray) must start at least one frame AFTER the checked
		// step's until — hence the single-frame filler steps below.
		{until: 110, mx: trayX, my: trayY, down: true, name: "press-tray"},
		{until: 112, mx: trayX, my: trayY, name: "release-tray"},
		{until: 130, mx: trayX, my: trayY, name: "tray-opens", check: func(g *Game) bool { return g.tray.Open },
			feedback: "treat tray did not open"},
		{until: 131, mx: trayX, my: trayY, name: "tray-settle"}, // filler: keep f130 grab-free
		{until: 136, mx: wormX, my: wormY, down: true, name: "press-worm"},
		{until: 138, mx: wormX, my: wormY, name: "release-worm"},
		{until: 152, mx: wormX, my: wormY, name: "worm-grabbed", check: func(g *Game) bool { return g.grabKind == contract.TreatWorm },
			feedback: "worm treat was not grabbed"},
		{until: 153, mx: wormX, my: wormY, name: "carry-settle"}, // filler: keep f152 drop-free
		{until: 159, mx: 400, my: 300, down: true, name: "press-drop"},
		{until: 161, mx: 400, my: 300, name: "release-drop"},
		{until: 175, mx: 400, my: 300, name: "treat-dropped", check: func(g *Game) bool {
			return g.grabKind == "" && len(g.world.Treats()) > 0
		}, feedback: "treat was not dropped into the tank"},
		// N10: a right click in the water must reach the sim without feeding
		// or toggling UI (behavior itself is unit-tested in sim). Food count
		// is snapshotted first — a restored save may legitimately carry food.
		{until: 196, mx: 400, my: 300, name: "pre-scare", check: func(g *Game) bool {
			g.smoke.foodN = len(g.world.Foods())
			return true
		}},
		{until: 202, mx: 300, my: 350, rdown: true, name: "right-press"},
		{until: 206, mx: 300, my: 350, name: "right-release"},
		{until: 216, mx: 300, my: 350, name: "scare-wired", check: func(g *Game) bool {
			return len(g.world.Fishes()) > 0 && len(g.world.Foods()) <= g.smoke.foodN
		}, feedback: "right click fed the tank instead of scaring"},
		// close the menu by its OPEN-position handle (the panel edge)
		{until: 222, mx: ohx, my: ohy, down: true, name: "press-handle-2"},
		{until: 224, mx: ohx, my: ohy, name: "release-handle-2"},
		{until: 240, mx: ohx, my: ohy, name: "menu-closes", check: func(g *Game) bool { return !g.menu.Open() },
			feedback: "menu did not close on second click"},
		// F26: the same carry with the menu CLOSED — the exact case that used
		// to drop the treat on the very frame it was grabbed.
		{until: 246, mx: trayX, my: trayY, down: true, name: "press-tray-2"},
		{until: 248, mx: trayX, my: trayY, name: "release-tray-2"},
		{until: 256, mx: trayX, my: trayY, name: "tray-opens-2", check: func(g *Game) bool { return g.tray.Open },
			feedback: "tray did not reopen for the closed-menu carry"},
		{until: 257, mx: trayX, my: trayY, name: "tray-settle-2"}, // filler: keep f256 grab-free
		{until: 262, mx: wormX, my: wormY, down: true, name: "press-worm-2"},
		{until: 264, mx: wormX, my: wormY, name: "release-worm-2"},
		{until: 320, mx: 500, my: 420, name: "carry-menu-closed", check: func(g *Game) bool {
			return g.grabKind == contract.TreatWorm && !g.menu.Open()
		}, feedback: "carry did not persist with the menu closed"},
		{until: 321, mx: 500, my: 420, name: "carry-settle-2"}, // filler: keep f320 drop-free
		{until: 327, mx: 520, my: 430, down: true, name: "press-drop-2"},
		{until: 331, mx: 520, my: 430, name: "release-drop-2"},
		{until: 348, mx: 520, my: 430, name: "treat-dropped-2", check: func(g *Game) bool {
			return g.grabKind == "" && len(g.world.Treats()) >= 1
		}, feedback: "second click did not drop the carried treat (menu closed)"},
		{until: 350, mx: 520, my: 430, name: "drain"}, // lets the final check run
	}}
}

// smokeInput overrides real input while the scripted run is active and runs
// the per-step assertions. The returned result is non-empty once finished.
func (g *Game) smokeInput() (mx, my int, pressed, released, rPressed, rReleased bool, result string, done bool) {
	s := g.smoke
	s.frame++
	// run any assertions scheduled for this frame (state from prior steps)
	for _, st := range s.steps {
		if st.until == s.frame-1 && st.check != nil && !st.check(g) {
			msg := st.feedback
			if msg == "" {
				msg = "assertion failed"
			}
			s.fails = append(s.fails, fmt.Sprintf("%s: %s", st.name, msg))
		}
	}
	// find the active step
	for _, st := range s.steps {
		if s.frame < st.until {
			pressed = st.down                 // LEVEL state — the UI and the feed edge are
			released = !st.down && s.prevDown // derived in processInput
			rPressed = st.rdown
			rReleased = !st.rdown && s.prevR
			s.prevDown = st.down
			s.prevR = st.rdown
			return st.mx, st.my, pressed, released, rPressed, rReleased, "", false
		}
	}
	// timeline exhausted — report
	if len(s.fails) > 0 {
		return 0, 0, false, false, false, false, fmt.Sprintf("SMOKE FAIL (%d): %v", len(s.fails), s.fails), true
	}
	return 0, 0, false, false, false, false, "SMOKE OK: menu, tabs, treat store, scare all pass", true
}
