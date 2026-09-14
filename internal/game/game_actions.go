// G6.1: menu action execution (split from game.go for the line ceiling).
package game

import (
	"fmt"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/Rollingzzzzz/Fish-tank/internal/ui"
)

// runAction executes one queued menu action.
func (g *Game) runAction(a ui.Action) {
	switch a.Kind {
	case ui.ActionResearch:
		g.hub.Trigger(contract.AgentSpecies)
	case ui.ActionRunAgent:
		g.hub.Trigger(a.ID)
	case ui.ActionRunAll:
		g.hub.Trigger(contract.AgentSpecies)
		g.hub.Trigger(contract.AgentWater)
		g.hub.Trigger(contract.AgentPattern)
	case ui.ActionExportPack:
		path := a.Path
		if path == "" {
			path = "tank-pack.zip"
		}
		fmt.Println("pack export:", g.store.ExportPack(path))
	case ui.ActionImportPack:
		path := a.Path
		if path == "" {
			path = "tank-pack.zip"
		}
		if err := g.store.ImportPack(path); err != nil {
			fmt.Println("pack import:", err)
		} else {
			g.menu.SetContentStore(g.store)
		}
	case ui.ActionResetTank:
		g.resetTank()
	case ui.ActionToggleSound:
		g.audioE.SetOn(a.Flag)
	case ui.ActionImportKey:
		if key, ok := findKey(); ok {
			g.cfg.APIKey = key
			_ = saveConfig(g.cfgPath, g.cfg)
			g.startAgents()
			fmt.Println("GLM key imported:", maskForLog(key))
		} else {
			fmt.Println("no ZCode key found")
		}
	case ui.ActionSelectSpecies:
		g.world.SpawnEgg(a.ID)
	case ui.ActionApplyPlant:
		for _, pd := range g.store.Plants() {
			if pd.ID == a.ID {
				g.world.AddPlant(pd)
				break
			}
		}
	}
}
