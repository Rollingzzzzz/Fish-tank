// v0.2: reaction + input helpers split from game.go (line ceiling).
package game

import (
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// onAgentEvent reacts to agent artifacts that affect the world.
func (g *Game) onAgentEvent(ev contract.Event) {
	if ev.Kind != contract.EventArtifact {
		return
	}
	switch ev.Agent {
	case contract.AgentSpecies:
		g.world.SpawnEgg(ev.ArtifactID)
	case contract.AgentWater:
		for _, wp := range g.store.Waters() {
			if wp.ID == ev.ArtifactID {
				g.world.ApplyWater(wp)
			}
		}
	}
}

// readMouse reads the real cursor plus the left-button state (level + edge)
// and the right-button state (level + edge; N10 scare input).
func readMouse() (mx, my int, pressed, released, rPressed, rReleased bool) {
	mx, my = ebiten.CursorPosition()
	return mx, my,
		ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft),
		inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft),
		ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight),
		inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonRight)
}
