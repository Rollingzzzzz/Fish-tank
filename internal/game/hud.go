// F8: top-right HUD chip — tank day + in-day clock, always visible.
package game

import (
	"fmt"

	"github.com/Rollingzzzzz/Fish-tank/internal/sim"
	"github.com/Rollingzzzzz/Fish-tank/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

// drawHUD paints the non-interactive clock chip top-right (drawn after the
// menu so the open panel never hides it).
func drawHUD(dst *ebiten.Image, w *sim.World) {
	day, hh, mm, night := w.TimeOfDay()
	glyph := "DAY"
	if night {
		glyph = "NIGHT"
	}
	// ASCII only — the 5x7 bitmap font has no middle-dot glyph
	txt := fmt.Sprintf("%s - DAY %d - %02d:%02d", glyph, day, hh, mm)
	// v0.3.3: the clock lives top-LEFT — the menu panel slides over the old
	// top-right spot and the two tangled
	ui.ChipLeft(dst, txt, 10, 10)
}

// drawClose paints the always-visible quit button (v0.3.3): a small glass X
// in the top-right corner, drawn above the menu so it can never be hidden.
func drawClose(dst *ebiten.Image, b *ui.ButtonState) {
	r := ui.UIMetrics(ScreenW, ScreenH, false).CloseBtn
	ui.DrawButton(dst, b, r.X, r.Y, r.W, r.H, "X", ui.ColErr)
}
