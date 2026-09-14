// G6.1: the NEON TANK game loop — owns world, menu, agent hub, audio and
// renders everything each frame.
package game

import (
	"runtime/pprof" // TEMP v0.3.8 profiling

	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Rollingzzzzz/Fish-tank/internal/agents"
	"github.com/Rollingzzzzz/Fish-tank/internal/audio"
	"github.com/Rollingzzzzz/Fish-tank/internal/content"
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/Rollingzzzzz/Fish-tank/internal/render"
	"github.com/Rollingzzzzz/Fish-tank/internal/sim"
	"github.com/Rollingzzzzz/Fish-tank/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Game is the ebiten Game implementation.
type Game struct {
	cfg     *contract.Config
	cfgPath string
	root    string

	store *content.Store
	world *sim.World
	menu  *ui.Menu
	hub   *agents.Hub

	bg      *render.Background
	trail   *render.Trail
	fish    render.FishBatch // F29: the school renders as one batch
	audioE  *audio.Engine
	persist *sim.Persist

	hubCancel context.CancelFunc
	lockPath  string
	touchLock func()
	lastTouch time.Time

	rock      *render.RockLayout      // volcanic decor + zones (N2/N3)
	fgFlora   *render.ForegroundStage // v0.3.8: near-glass depth plane
	tray      *ui.TreatTray           // treat store (N8)
	grabKind  string                  // treat currently carried by the mouse (F26 toggle)
	grabSec   float64                 // seconds since the grab — scent growth (F26)
	smoke     *smokeRun               // -menu-smoke scripted acceptance run
	shot      *shotRun                // -shot scripted evidence capture (F19)
	fishOrder []*sim.Fish             // v1.1: reused depth-sorted draw order
	casters   []render.ShadowCaster   // v1.1 G74: reused floor-shadow projections
	bubView   []render.BubbleView     // v1.1 G76: reused bubble batch view
	shockView []render.ShockView      // v1.1 G83: reused strike-ring batch view
	feedHeld  bool                    // left-button level state (F5 feed edge)
	scareHeld bool                    // right-button level state (N10 scare edge)
	closeBtn  ui.ButtonState          // v0.3.3: the always-visible X button
	quit      bool                    // v0.3.3: X or ESC pressed — exit next Update
	lastMX    int                     // cursor position for the held-treat draw
	lastMY    int

	frame    int
	prevMX   int
	prevMY   int
	mouseSpd float64
	debug    bool
}

// New assembles the full game from an already-bootstrapped root.
func New(cfg *contract.Config, cfgPath, root string, store *content.Store,
	world *sim.World, persist *sim.Persist, eng *audio.Engine) *Game {
	g := &Game{
		cfg: cfg, cfgPath: cfgPath, root: root,
		store: store, world: world, persist: persist, audioE: eng,
		bg: render.NewBackground(), trail: render.NewTrail(ScreenW, ScreenH),
	}
	g.rock = render.NewRockLayout(world.RockSeed(), ScreenW, ScreenH)
	world.SetZones(g.rock.Zones())
	world.SetHoles(g.rock.Holes())        // F25: door-to-door transit mouths
	world.SetRockBase(g.rock.BaseWidth()) // v0.3.2 composition budget
	g.fgFlora = render.NewForegroundStage(store.Plants(), world.RockSeed()+7)
	g.tray = ui.NewTreatTray()
	g.menu = ui.NewMenu(cfg)
	g.menu.SetContentStore(store)
	g.startAgents()
	return g
}

// startAgents (re)creates the agent hub — boot and after key changes.
func (g *Game) startAgents() {
	if g.hubCancel != nil {
		g.hubCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	g.hubCancel = cancel
	g.hub = agents.NewHub(*g.cfg, g.store, contract.AgentHooks{
		SpeciesNames:       func() []string { return g.store.Names("species") },
		RecentFingerprints: func(n int) []string { return g.store.RecentFingerprints(n) },
		NextRepaints:       func(max int) []contract.PatternRequest { return g.world.NextRepaints(max) },
		ApplyRecipe:        func(fishID string, r contract.PatternRecipe) { g.world.ApplyRecipe(fishID, r) },
		SpawnEgg:           func(speciesID string) { g.world.SpawnEgg(speciesID) },
	})
	g.hub.Start(ctx)
}

// Trigger runs an agent now.
func (g *Game) Trigger(id string) { g.hub.Trigger(id) }

// Update advances one frame.
func (g *Game) Update() error {
	g.frame++
	if ebiten.IsWindowBeingClosed() {
		g.shutdown()
		return ebiten.Termination
	}
	g.handleHotkeys()
	if g.quit { // v0.3.3: the X button / ESC asked to leave
		g.shutdown()
		return ebiten.Termination
	}

	// -menu-smoke / -shot: scripted input overrides the real mouse; when the
	// timeline finishes we report and terminate.
	if g.smoke != nil {
		mx, my, pressed, released, rPressed, rReleased, result, done := g.smokeInput()
		g.processInput(mx, my, pressed, released, rPressed, rReleased)
		if done {
			g.shutdown()
			fmt.Println(result)
			if len(g.smoke.fails) > 0 {
				os.Exit(1)
			}
			os.Exit(0)
		}
		g.persistTick()
		return nil
	}
	if g.shot != nil {
		mx, my, pressed, released, rPressed, rReleased, done := g.shotInput()
		g.processInput(mx, my, pressed, released, rPressed, rReleased)
		if done {
			g.shutdown()
			g.finishShot()
			pprof.StopCPUProfile() // TEMP v0.3.8 profiling
			os.Exit(0)
		}
		g.persistTick()
		return nil
	}

	mx, my, pressed, released, rPressed, rReleased := readMouse()
	g.processInput(mx, my, pressed, released, rPressed, rReleased)
	g.persistTick()
	return nil
}

// persistTick wraps the autosave cadence (split for the smoke path).
func (g *Game) persistTick() {
	// keep the single-instance lock fresh
	if time.Since(g.lastTouch) > 30*time.Second {
		g.lastTouch = time.Now()
		g.touchLock()
	}

	// autosave tiers
	if err := g.persist.Tick(g.world, time.Now()); err != nil {
		fmt.Println("persist:", err)
	}
}

// processInput advances one frame of input, simulation and reactions.
func (g *Game) processInput(mx, my int, pressed, released, rPressed, rReleased bool) {
	dt := 1.0 / float64(ebiten.TPS())
	clickEdge := pressed && !g.feedHeld // one click = one feed (F5)
	g.feedHeld = pressed
	scareEdge := rPressed && !g.scareHeld // one right click = one scare (N10)
	g.scareHeld = rPressed
	g.lastMX, g.lastMY = mx, my
	cb := ui.UIMetrics(ScreenW, ScreenH, g.menu.Open()).CloseBtn

	// F26 click-toggle carry: press #1 on a tray row picks the treat up and
	// it rides the cursor (button state irrelevant); the NEXT left press
	// drops it right there and is fully consumed — menu, tray, close button
	// and feeding all see a neutral frame. Right clicks still scare.
	wasCarrying := g.grabKind != ""
	if wasCarrying {
		g.grabSec += dt // F26: scent cloud grows while baited
		g.menu.Update(mx, my, false, false)
		g.tray.Update(mx, my, false, false)
		if clickEdge {
			g.world.DropTreat(g.grabKind, contract.Vec2{X: float64(mx), Y: float64(my)})
			g.grabKind = ""
			g.grabSec = 0
		}
	} else {
		g.menu.Update(mx, my, pressed, released)
		if g.closeBtn.Update(cb.X, cb.Y, cb.W, cb.H, mx, my, pressed, released) {
			g.quit = true // v0.3.3: the always-on-top X button
		}
		if ev := g.tray.Update(mx, my, pressed, released); strings.HasPrefix(ev, "grab:") {
			g.grabKind = strings.TrimPrefix(ev, "grab:")
			g.grabSec = 0
		}
	}
	_, scrollDy := ebiten.Wheel()
	g.menu.Wheel(int(scrollDy), mx, my)

	in := readInput(mx, my, &g.prevMX, &g.prevMY, &g.mouseSpd)
	// F5 click hygiene: one click = one effect. UI surfaces consume their
	// clicks; a grab press selects, a drop press releases — neither feeds.
	if clickEdge && !wasCarrying && g.grabKind == "" &&
		!g.menu.Consumes(mx, my) && !g.tray.Hit(mx, my) && !ui.HitR(cb, mx, my) {
		in.FeedAt = append(in.FeedAt, contract.Vec2{X: float64(mx), Y: float64(my)})
	}
	// N10: a right click in the water startles nearby fish — they bolt, then
	// forget (the same aquarium feeling as tapping the glass).
	if scareEdge && !g.menu.Consumes(mx, my) && !g.tray.Hit(mx, my) && !ui.HitR(cb, mx, my) {
		p := contract.Vec2{X: float64(mx), Y: float64(my)}
		in.ScareAt = &p
	}
	// N11/F26: while a treat is carried, the sim sees it — fish catch the
	// scent (radius grows with carry time), crowd the keep-back ring and
	// cannot bite until it is dropped.
	if g.grabKind != "" {
		in.HeldTreat = &sim.HeldTreat{
			Kind: g.grabKind,
			Pos:  contract.Vec2{X: float64(mx), Y: float64(my)},
			Age:  g.grabSec,
		}
	}
	g.world.Update(dt, in)

	// agent events → menu + world reactions (drain, never block)
	for drained := false; !drained; {
		select {
		case ev, ok := <-g.hub.Events():
			if !ok {
				drained = true
				continue
			}
			g.menu.PushEvent(ev)
			g.onAgentEvent(ev)
		default:
			drained = true
		}
	}

	// menu actions
	for _, a := range g.menu.Actions() {
		g.runAction(a)
	}
}

// resetTank and shutdown live in lifecycle.go (line ceiling).

// handleHotkeys processes global shortcuts.
func (g *Game) handleHotkeys() {
	if inpututil.IsKeyJustPressed(ebiten.KeyF) {
		g.debug = !g.debug
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF11) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) { // v0.3.3: ESC quits
		g.quit = true
	}
}

// Layout returns the internal resolution.
func (g *Game) Layout(ow, oh int) (int, int) { return ScreenW, ScreenH }
