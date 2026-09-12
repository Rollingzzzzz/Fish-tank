// v0.2: tank lifecycle — reset + shutdown (split from game.go, line ceiling).
package game

import (
	"fmt"
	"os"
	"time"

	"github.com/Rollingzzzzz/Fish-tank/internal/render"
	"github.com/Rollingzzzzz/Fish-tank/internal/sim"
)

// resetTank wipes saves and rebuilds a fresh world (rock layout + zones too).
func (g *Game) resetTank() {
	removeSave(g.root, "save_minute.json")
	removeSave(g.root, "save_daily.json")
	removeSave(g.root, "save_weekly.json")
	fresh := sim.NewWorld(float64(ScreenW), float64(ScreenH), *g.cfg, g.store.Species(),
		g.store.Plants(), g.store.Waters())
	fresh.SetCorals(g.store.Corals())
	*g.world = *fresh
	g.world.SeedRng(time.Now().UnixNano())
	g.rock = render.NewRockLayout(g.world.RockSeed(), ScreenW, ScreenH)
	g.world.SetZones(g.rock.Zones())
	g.world.SetHoles(g.rock.Holes()) // F25
	g.world.SetRockBase(g.rock.BaseWidth())
	fmt.Println("tank reset")
}

// shutdown saves once more, releases the hub and the lock. Settings changes
// (toggles, sliders — including F13 fullscreen-on-start) are persisted here:
// they mutate cfg in place, so this is the single flush point.
func (g *Game) shutdown() {
	_ = g.persist.SnapshotNow(g.world)
	_ = saveConfig(g.cfgPath, g.cfg)
	if g.hubCancel != nil {
		g.hubCancel()
	}
	if g.lockPath != "" {
		_ = os.Remove(g.lockPath)
	}
}

// StartSmoke arms the scripted -menu-smoke acceptance run.
func (g *Game) StartSmoke() { g.startSmoke() }
