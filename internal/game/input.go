// G6.1: player input — feeding, watching (care), and scare swipes.
package game

import (
	"math"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/Rollingzzzzz/Fish-tank/internal/sim"
)

// readInput converts mouse state into the sim input struct.
// mx/my come from the caller (real cursor or the smoke script);
// prevMX/prevMY/mouseSpd are in/out state kept on the Game.
func readInput(mx, my int, prevMX, prevMY *int, mouseSpd *float64) sim.Input {
	dx, dy := float64(mx-*prevMX), float64(my-*prevMY)
	*prevMX, *prevMY = mx, my
	spd := math.Sqrt(dx*dx+dy*dy) * 60 // px/s at 60fps
	*mouseSpd = *mouseSpd*0.7 + spd*0.3

	in := sim.Input{
		MouseX:      float64(mx),
		MouseY:      float64(my),
		MouseActive: mx > 0 || my > 0,
		MouseSpeed:  *mouseSpd,
	}
	// F5: feeding is decided by the caller (game.Update) — the click may be
	// consumed by the menu or the treat tray, and carrying a treat drops it
	// instead of feeding.
	return in
}

// hypot2v / clamp01 tiny helpers.
func hypot2v(v contract.Vec2) float64 { return math.Sqrt(v.X*v.X + v.Y*v.Y) }
func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
