// F19: -shot mode — the evidence harness. A scripted timeline drives the real
// game (menu, every tab, tray, a held wriggling worm, a drop, a scare) and
// captures numbered PNGs plus a JSON manifest of expected control rects
// sourced from ui.UIMetrics — the same geometry the renderer uses, so the
// audit can never drift from the screen. uiaudit consumes the pair.
package game

import (
	"fmt"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"

	"github.com/Rollingzzzzz/Fish-tank/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

// shotRect is one manifest entry: a control that must be visible in Image.
// Kind "control" (default) must show diff + bright label pixels; "scene"
// rects only need meaningful content (diff) — their pixels are the tank's.
type shotRect struct {
	Name  string `json:"name"`
	X     int    `json:"x"`
	Y     int    `json:"y"`
	W     int    `json:"w"`
	H     int    `json:"h"`
	Image string `json:"image"`
	Kind  string `json:"kind,omitempty"`
}

type shotRun struct {
	dir      string
	frame    int
	prevDown bool
	prevR    bool
	steps    []smokeStep    // reuse the smoke step driver
	caps     map[int]string // frame → capture name
	queue    []string       // pending captures (Update enqueues, Draw writes)
	drained  bool           // queue empty at least once after the last step
	holdX    int            // where the worm is held for the crowd shot
	holdY    int
	carry2X  int // F26: menu-closed carry position
	carry2Y  int
	manifest []shotRect
	captured int
	actIdx   int               // F25: last step whose act fired (one-shot)
	snap     []byte            // last frame's pixels (refreshed each Draw)
	pending  map[string][]byte // capture name -> pixels frozen at enqueue
	pendInfo map[string]string // capture name -> state line frozen at enqueue
}

// StartShot arms the -shot evidence run (F19).
func (g *Game) StartShot(dir string) {
	g.cfg.AutoFeed = false // the scripted scene must not gain random flakes
	g.shot = &shotRun{dir: dir, caps: map[int]string{},
		pending: map[string][]byte{}, pendInfo: map[string]string{}}
	g.shot.build()
	// a freshly restored tank has a near-empty log; give the Log tab real
	// content so its evidence shot shows what the player actually sees
	for _, line := range []string{
		"the tank wakes up",
		"life: the eternal one glides into view",
		"life: a ribbon-streamer pair begins a courtship circle",
		"garden: the garden thins",
		"behavior: the Glass Sucker clamps onto the glass",
		"behavior: a puff-orbit zooms a lap around the nest",
		"mites: the school devours a wild water mite",
		"treats: a worm wriggles loose into the water",
		"time: day 53 begins",
		"agents: species agent produced dart-spindle",
	} {
		g.menu.PushEvent(contract.Event{Kind: contract.EventLog, Text: line})
	}
}

// build assembles the timeline. Frames (60 tps): open the menu, visit every
// tab, open the tray, grab the worm and let the fish crowd it, drop it, then
// a right-click scare.
func (s *shotRun) build() {
	open := ui.UIMetrics(ScreenW, ScreenH, true)
	closed := ui.UIMetrics(ScreenW, ScreenH, false)
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
	s.holdX, s.holdY = ScreenW/2, ScreenH/2
	s.carry2X, s.carry2Y = ScreenW/2-120, ScreenH/2+60

	type step = smokeStep
	var st []step
	st = append(st,
		step{until: 8, mx: hx, my: hy, name: "idle"},
		step{until: 14, mx: hx, my: hy, down: true, name: "press-handle"},
		step{until: 16, mx: hx, my: hy, name: "release-handle"},
	)
	f := 60
	for i := 0; i < 5; i++ {
		tx, ty := tabC(i)
		st = append(st,
			step{until: f + 6, mx: tx, my: ty, down: true, name: fmt.Sprintf("press-tab%d", i)},
			step{until: f + 8, mx: tx, my: ty, name: fmt.Sprintf("release-tab%d", i)},
			step{until: f + 22, mx: tx, my: ty, name: fmt.Sprintf("settle-tab%d", i)},
		)
		s.caps[f+20] = fmt.Sprintf("0%d-tab-%s.png", i+2, uiTabSlug(i))
		f += 24
	}
	st = append(st,
		step{until: f + 6, mx: trayX, my: trayY, down: true, name: "press-tray"},
		step{until: f + 8, mx: trayX, my: trayY, name: "release-tray"},
		step{until: f + 22, mx: trayX, my: trayY, name: "tray-open"}, // idle: the capture frame must show the OPEN tray, pre-grab
	)
	s.caps[f+20] = "07-tray-open.png"
	f += 24
	st = append(st,
		step{until: f + 6, mx: wormX, my: wormY, down: true, name: "press-worm"},
		step{until: f + 8, mx: wormX, my: wormY, name: "release-worm"},
		step{until: f + 150, mx: s.holdX, my: s.holdY, name: "hold-worm"}, // fish crowd the hook
	)
	s.caps[f+140] = "08-held-worm.png"
	st = append(st,
		step{until: f + 156, mx: s.holdX, my: s.holdY, down: true, name: "press-drop"},
		step{until: f + 158, mx: s.holdX, my: s.holdY, name: "release-drop"},
		step{until: f + 190, mx: s.holdX, my: s.holdY, name: "frenzy"},
	)
	s.caps[f+185] = "09-frenzy.png"
	st = append(st,
		step{until: f + 196, mx: 300, my: 350, rdown: true, name: "right-press"},
		step{until: f + 200, mx: 300, my: 350, name: "right-release"},
		step{until: f + 215, mx: 300, my: 350, name: "scatter"},
	)
	// F26: click-toggle carry with the menu CLOSED — the case that used to
	// drop the treat on the grab frame. The worm rides the cursor (button is
	// UP) until the next press drops it.
	st = append(st,
		step{until: f + 221, mx: ohx, my: ohy, down: true, name: "press-handle-close"},
		step{until: f + 223, mx: ohx, my: ohy, name: "release-handle-close"},
		step{until: f + 235, mx: ohx, my: ohy, name: "menu-closed"},
		step{until: f + 241, mx: trayX, my: trayY, down: true, name: "press-tray-2"},
		step{until: f + 243, mx: trayX, my: trayY, name: "release-tray-2"},
		step{until: f + 255, mx: trayX, my: trayY, name: "tray-open-2"},
		step{until: f + 261, mx: wormX, my: wormY, down: true, name: "press-worm-2"},
		step{until: f + 263, mx: wormX, my: wormY, name: "release-worm-2"},
		step{until: f + 345, mx: s.carry2X, my: s.carry2Y, name: "carry-2"},
	)
	s.caps[f+340] = "10-carry-menu-closed.png"
	st = append(st,
		step{until: f + 351, mx: s.carry2X, my: s.carry2Y, down: true, name: "press-drop-2"},
		step{until: f + 353, mx: s.carry2X, my: s.carry2Y, name: "release-drop-2"},
		step{until: f + 368, mx: s.carry2X, my: s.carry2Y, name: "dropped-2"},
	)
	// F25: door-to-door transit — a fish is parked by the left crag's low
	// door, slips in, hides inside the rock and bursts out of another mouth.
	st = append(st,
		step{until: f + 380, mx: 260, my: 480, name: "pre-transit", act: func(g *Game) {
			g.world.DebugParkByDoor(0)
			g.world.DebugTransit()
		}},
		step{until: f + 460, mx: 260, my: 480, name: "transit-run"},
		step{until: f + 560, mx: 260, my: 480, name: "transit-out"},
	)
	s.caps[f+400] = "11-transit-enter.png"  // opaque at the mouth / just swallowed
	s.caps[f+440] = "12-transit-inside.png" // gone into the rock
	s.caps[f+530] = "13-transit-exit.png"   // clear of the far mouth, opaque
	// F30: the silky flora — the six new feather/silk seeds staged in a row
	st = append(st,
		step{until: f + 575, mx: 640, my: 400, name: "flora-set", act: func(g *Game) {
			want := []string{"ghost-pen-feather", "ember-plume-feather", "twilight-plume-feather",
				"moon-silk-grass", "abyss-silkthread", "pearl-veil-silk"}
			var defs []*contract.PlantDesign
			for _, id := range want {
				for _, pl := range g.store.Plants() {
					if pl.ID == id {
						defs = append(defs, pl)
						break
					}
				}
			}
			g.world.DebugStageFlora(defs)
		}},
		step{until: f + 592, mx: 640, my: 400, name: "flora-run"},
	)
	s.caps[f+588] = "14-flora.png"
	// F29: population pressure at the new cap — 70 eggs land at once and
	// hatch within 6 s; the crowd frame's info line records the fish count
	// and finishShot records the frame rate the load produced.
	st = append(st,
		step{until: f + 600, mx: 640, my: 400, name: "egg-bomb", act: func(g *Game) {
			var ids []string
			for _, sp := range g.store.Species() {
				if sp.Role != contract.RoleChosen {
					ids = append(ids, sp.ID)
				}
			}
			for i := 0; i < 70; i++ {
				g.world.SpawnEgg(ids[i%len(ids)])
			}
		}},
		step{until: f + 1090, mx: 640, my: 400, name: "hatch-run"},
	)
	s.caps[f+1085] = "15-crowd.png"
	s.caps[8] = "01-closed.png"
	s.steps = st
}

func uiTabSlug(i int) string {
	switch i {
	case 0:
		return "agents"
	case 1:
		return "species"
	case 2:
		return "plants"
	case 3:
		return "log"
	default:
		return "setup"
	}
}

// shotInput mirrors smokeInput for the capture timeline (no assertions —
// uiaudit judges the pixels afterwards). Capture frames enqueue their PNG
// name here (Update always runs) and the next Draw writes it — under scene
// load FPS can dip below TPS, so a frame-exact capture would drop shots.
func (g *Game) shotInput() (mx, my int, pressed, released, rPressed, rReleased bool, done bool) {
	s := g.shot
	s.frame++
	if name, ok := s.caps[s.frame]; ok {
		s.queue = append(s.queue, name)
		// v0.3.6 fix: freeze THIS frame's pixels now — the file is written at
		// the next Draw, which under scene load can land many frames later
		// and used to save the WRONG game state (e.g. a tray already closed
		// by the grab). snap holds the previous Draw = state at N-1.
		if len(s.snap) > 0 {
			s.pending[name] = append([]byte(nil), s.snap...)
			s.pendInfo[name] = fmt.Sprintf("canvas=%dx%d fish=%d alive=%d TPS=%.0f FPS=%.0f menu=%v tray=%v grab=%q treats=%d cluster=%d%%",
				ScreenW, ScreenH, len(g.world.Fishes()), g.world.AliveFishes(),
				ebiten.ActualTPS(), ebiten.ActualFPS(), g.menu.Open(), g.tray.Open, g.grabKind, len(g.world.Treats()), g.clusterPct())
		}
	}
	// find the active step
	for i, st := range s.steps {
		if s.frame < st.until {
			if st.act != nil && s.actIdx != i { // F25: one-shot on step entry
				s.actIdx = i
				st.act(g)
			}
			pressed = st.down
			released = !st.down && s.prevDown
			rPressed = st.rdown
			rReleased = !st.rdown && s.prevR
			s.prevDown = st.down
			s.prevR = st.rdown
			return st.mx, st.my, pressed, released, rPressed, rReleased, false
		}
	}
	done = true
	if len(s.queue) == 0 {
		return 0, 0, false, false, false, false, true
	}
	// let the final queued capture drain through Draw before finishing
	return 0, 0, false, false, false, false, s.drained
}
