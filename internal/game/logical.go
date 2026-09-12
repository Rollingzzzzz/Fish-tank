// F13: logical-canvas sizing — the tank is height-locked at 720 and the
// width follows the monitor aspect, so every display gets the same
// proportions with no letterbox bars (split from game.go, line ceiling).
package game

import (
	"math"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// ScreenW and ScreenH are the internal (logical) resolution. Tests and demos
// keep the 1280x720 default; the app sets the width from the monitor aspect
// at launch via SetLogicalSize.
var (
	ScreenW = 1280
	ScreenH = 720
)

// LogicalWidthFor derives the logical canvas width from a monitor size
// (F13/D2): 720-locked proportional width, clamped to a safe band.
func LogicalWidthFor(mw, mh int) int {
	if mw <= 0 || mh <= 0 {
		return 1280
	}
	w := int(math.Round(float64(contract.LogicalH) * float64(mw) / float64(mh)))
	if w < contract.LogicalWMin {
		w = contract.LogicalWMin
	}
	if w > contract.LogicalWMax {
		w = contract.LogicalWMax
	}
	return w
}

// SetLogicalSize fixes the logical canvas — must run before Bootstrap (the
// world, rock layout and trail are sized from these values).
func SetLogicalSize(w, h int) {
	if w > 0 {
		ScreenW = w
	}
	if h > 0 {
		ScreenH = h
	}
}

// FullscreenStart reports the persisted F13 launch preference.
func (g *Game) FullscreenStart() bool { return g.cfg.FullscreenOnStart }
