// v0.3: the Draw pass — full-scene compositing order (background, trails,
// rock back, plants, corals, nest, fish, rock front, fg flora, sand bed,
// food/treats/eggs/mites/critters, particles, bubbles, menu, tray, HUD,
// held treat) plus the debug overlay (split from game.go, line ceiling).
package game

import (
	"fmt"

	"math"
	"runtime"
	"sort"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/Rollingzzzzz/Fish-tank/internal/render"
	"github.com/Rollingzzzzz/Fish-tank/internal/sim"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// Draw renders the whole scene.
func (g *Game) Draw(screen *ebiten.Image) {
	wl := g.world.WaterLive()
	g.bg.Draw(screen, render.WaterState{
		Time: wl.Time, DayFactor: wl.DayFactor, TOD: wl.TOD, Top: wl.Top, Bottom: wl.Bottom,
		Accent: wl.Accent, Rays: wl.Rays, Caustics: wl.Caustics,
	})

	// phosphor trails: fade old, blit under the scene, stamp new as we draw
	g.trail.Fade(0.10)
	g.trail.Blit(screen)

	night := wl.Night
	// v0.3.8: Hide01 is strictly binary (0 face / 1 swallowed), so the old
	// pre-rock fish pass is gone — a transiting fish is simply not drawn
	// while inside the rock. The school renders through ONE FishBatch
	// (~2 draw calls, F29).
	g.rock.DrawBack(screen, night) // N2: cave silhouettes behind the scene

	for _, pl := range g.world.Plants() {
		render.DrawPlant(screen, g.trail.Image(), pl.Def, pl.X, pl.Y,
			float64(ScreenH)*pl.Def.Height*pl.Sc, wl.Time, night)
	}
	coralSpots := make([]render.CoralSpot, 0, len(g.world.Corals()))
	for _, c := range g.world.Corals() {
		coralSpots = append(coralSpots, render.CoralSpot{Def: c.Def, X: c.X, Y: c.Y, Sc: c.Sc})
	}
	render.DrawCorals(screen, coralSpots, wl.Time, night) // N1
	// v0.3.8: her oyster sits BEHIND the school now — plants render behind
	// the shell, fish (herself included) glide in front of it. Scenery, not
	// a wall; the closer she swims, the brighter the scene (v0.3.5 contact).
	contact := clamp01(1 - g.world.ChosenNestDist()/150)
	g.rock.DrawNest(screen, wl.Time, contact)
	for _, eg := range g.world.Eggs() {
		pulse := 3.5 + 1.5*math.Abs(math.Sin(wl.Time*4+float64(eg.Seed%10)))
		render.DrawOrb(screen, eg.Pos.X, eg.Pos.Y, 6+pulse*0.4, "#c9f7ff", 120)
		render.DrawGlow(g.trail.Image(), eg.Pos.X, eg.Pos.Y, 16+pulse*2, "#7fd4ff", 0.20)
	}
	// v1.1: painter's order by depth lane — far-lane fish pass behind the
	// big bodies, near-lane fish cross in front of them (the 3D read)
	fishes := g.world.Fishes()
	if cap(g.fishOrder) < len(fishes) {
		g.fishOrder = make([]*sim.Fish, len(fishes))
	}
	order := g.fishOrder[:len(fishes)]
	copy(order, fishes)
	sort.Slice(order, func(i, j int) bool { return order[i].Z() < order[j].Z() })
	if cap(g.casters) < len(order) {
		g.casters = make([]render.ShadowCaster, len(order))
	}
	casters := g.casters[:0]
	for _, f := range order {
		if f.Hide01 > 0 { // swallowed by a crag door — not drawn at all
			continue
		}
		// G74: the shadow projection rides the same loop — spine length,
		// forward x, dune lift and the portal fade all come straight off
		// the live body
		if !f.Dying {
			n := len(f.Spine)
			bl, hx := 0.0, 1.0
			if n >= 2 {
				dx, dy := 0.0, 0.0
				for j := 1; j < n; j++ {
					sx := f.Spine[j].X - f.Spine[j-1].X
					sy := f.Spine[j].Y - f.Spine[j-1].Y
					bl += math.Sqrt(sx*sx + sy*sy)
					dx += f.Spine[j-1].X - f.Spine[j].X
					dy += f.Spine[j-1].Y - f.Spine[j].Y
				}
				if l := math.Sqrt(dx*dx + dy*dy); l > 1 {
					hx = dx / l
				}
			}
			casters = append(casters, render.ShadowCaster{
				X: f.Pos.X, Hx: hx, BodyLen: bl, Z: f.Z(),
				Lift: math.Max(0, contract.SandSurfaceY(float64(ScreenH), f.Pos.X)-f.Pos.Y),
				Fade: f.PortalFade01(),
			})
		}
		chosen := f.Sp.Role == contract.RoleChosen
		anim := render.FishAnim{
			Time: wl.Time, Speed01: clamp01(hypot2v(f.Vel) / 110), ElderP: f.ElderP,
			Attached: f.Attached(), AttachSide: f.AttachWall(), // F18
			Hide01: f.Hide01, // v0.3.8: binary 0/1
			Z:      f.Z(),    // v1.1 depth lane
			Beat:   f.Beat(),   // G94: the silk rides the sim beat
		}
		if chosen {
			// G73: the wormhole pass — her body fades into the door and
			// the door itself breathes under her
			anim.PortalFade = f.PortalFade01()
			if p := f.PortalFrom(); p != nil {
				grow := 1.0
				if f.PortalPhase() == 1 {
					grow = clamp01(f.PortalClock() / 0.9)
				}
				render.DrawPortal(screen, p.X, p.Y, grow, wl.Time)
			}
			if f.PortalPhase() == 3 {
				render.DrawPortal(screen, f.Pos.X, f.Pos.Y, 1-clamp01(f.PortalClock()/0.6), wl.Time)
			}
		}
		if chosen || f.Attached() {
			// F16: her rim/shimmer/bioluminescence stay on the immediate path
			render.DrawFish(screen, g.trail.Image(), f.Spine, f.Sp, &f.Pal, f.Stage, night, anim)
		} else {
			g.fish.Draw(screen, g.trail.Image(), f.Spine, f.Sp, &f.Pal, f.Stage, night, anim)
		}
		// F16: the eat-flash sparkle is the Chosen's alone (normals keep their
		// particle bites from the sim, no additive neon)
		if f.EatFlash > 0 && chosen {
			render.DrawGlow(screen, f.Pos.X, f.Pos.Y, 14, f.Pal.Glow, 0.28*f.EatFlash)
		}
	}
	g.fish.Flush(screen, g.trail.Image())
	g.rock.DrawFront(screen, night) // N2: rock faces close over cave mouths
	// v0.3.8: the near-glass plane — dark flank silhouettes close the depth
	// sandwich (far crags → midground school/nest → foreground fronds)
	g.fgFlora.Draw(screen, wl.Time, float64(ScreenW), float64(ScreenH))
	// v1.1: the sand bed is the FRONT-most terrain — crags, flora and the
	// school all render behind it; only the floor dwellers and the food that
	// lands on its surface draw on top
	render.DrawSandBed(screen, g.world.Sand(), float64(ScreenW), float64(ScreenH),
		float64(ScreenH)*contract.FloorLineFrac, night)
	// G83: the strike pressure rings — one batched ring mesh for every
	// live shock, riding just under the floor dwellers
	if sv := g.world.Shocks(); len(sv) > 0 {
		if cap(g.shockView) < len(sv) {
			g.shockView = make([]render.ShockView, len(sv))
		}
		view := g.shockView[:0]
		for _, sh := range sv {
			view = append(view, render.ShockView{X: sh.Pos.X, Y: sh.Pos.Y,
				P: sh.T / contract.ShockRingSec})
		}
		render.DrawShocks(screen, view)
	}
	if contract.ExtrasEnabled {
		// v1.1 realism pass on the bed (G74/G75): sun first, shade second —
		// a shadow reads as blocked light when it dims the dapple under it
		render.DrawSandCaustics(screen, float64(ScreenW), float64(ScreenH), wl.Time, night)
	}
	render.DrawShadows(screen, casters, float64(ScreenW), float64(ScreenH), night)
	for _, fd := range g.world.Foods() {
		render.DrawGlow(screen, fd.Pos.X, fd.Pos.Y, 5, "#ffe9a0", 0.45)
		render.DrawOrb(screen, fd.Pos.X, fd.Pos.Y, 1.8, "#fff6d8", 230)
	}
	for _, tr := range g.world.Treats() { // N8 live treats
		render.DrawTreat(screen, tr.Kind, tr.Pos, tr.Phase)
	}
	for _, m := range g.world.Mites() { // N7 water mites — they graze the sand
		render.DrawMite(screen, m.Pos, m.Phase)
	}
	for _, c := range g.world.Critters() { // v1.1 floor critters — they live
		// on the sand surface, so they ride in front of the bed layer
		render.DrawCreature(screen, c.Kind, c.Pos, c.Phase, c.Dir, c.Burrow, c.Vulnerable())
	}
	for _, p := range g.world.Particles() {
		a := p.Life / p.Max
		render.DrawGlow(screen, p.Pos.X, p.Pos.Y, p.Size*4, p.Color, 0.45*a)
	}
	// v1.1 G76: one mesh for every bubble — the seep columns and the gill
	// breaths ride the same single draw call
	bubs := g.world.Bubbles()
	if cap(g.bubView) < len(bubs) {
		g.bubView = make([]render.BubbleView, len(bubs))
	}
	view := g.bubView[:0]
	for _, b := range bubs {
		view = append(view, render.BubbleView{X: b.Pos.X, Y: b.Pos.Y, R: b.R, Seed: b.Seed})
	}
	render.DrawBubbles(screen, view)

	g.menu.Draw(screen)
	g.tray.Draw(screen, g.grabKind) // N8: tray button + open store (above menu)
	drawHUD(screen, g.world)        // F8: clock chip, always visible
	// N11: the held treat writhes at the cursor like bait on a hook — the
	// crowding fish underneath are drawn by the sim pull.
	if g.grabKind != "" {
		render.DrawTreat(screen, g.grabKind, contract.Vec2{X: float64(g.lastMX), Y: float64(g.lastMY)}, wl.Time*6)
	}

	if g.debug {
		ebitenutil.DebugPrint(screen, fmt.Sprintf(
			"TPS %.0f FPS %.0f heap %.1fMB fish %d plants %d corals %d mites %d treats %d titan %d critters %d bubbles %d care %.0f",
			ebiten.ActualTPS(), ebiten.ActualFPS(), heapMB(),
			len(g.world.Fishes()), len(g.world.Plants()), len(g.world.Corals()),
			len(g.world.Mites()), len(g.world.Treats()), g.world.TitanCount(),
			len(g.world.Critters()), len(g.world.Bubbles()), g.world.Care))
	}

	drawClose(screen, &g.closeBtn) // v0.3.3: the X, above even the menu
	drawRec(screen, &g.recBtn, &g.rec) // G95: the REC 60s capture button
	g.shotCapture(screen)          // F19: evidence frames (no-op outside -shot)
	g.recCapture(screen)           // G95: REC frames (no-op while idle)
	if g.shot != nil {
		// F19 fix (v0.3.6): keep the last fully-drawn frame's pixels so the
		// next Update can freeze them at the exact capture frame
		if len(g.shot.snap) != 4*ScreenW*ScreenH {
			g.shot.snap = make([]byte, 4*ScreenW*ScreenH)
		}
		screen.ReadPixels(g.shot.snap)
	}
}

// heapMB reports current heap usage for the debug overlay.
func heapMB() float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return float64(m.HeapAlloc) / (1024 * 1024)
}
