// F8: HUD chip — a small glass label the game draws above everything.
package ui

import "github.com/hajimehoshi/ebiten/v2"

// Chip draws a right-aligned glass chip with text at (right, y).
func Chip(dst *ebiten.Image, text string, right, y int) {
	w := TextWidth(text, 2) + 24
	x := right - w
	if x < 4 {
		x = 4
	}
	GlassPanel(dst, x, y, w, 24)
	DrawText(dst, text, x+12, y+6, 2, "#cfeaff", 200)
}

// ChipLeft draws a left-anchored glass chip at (x, y) — the day/clock HUD
// moved to the TOP-LEFT corner (v0.3.3) so it never tangles with the
// right-side menu panel or the treat tray.
func ChipLeft(dst *ebiten.Image, text string, x, y int) {
	w := TextWidth(text, 2) + 24
	GlassPanel(dst, x, y, w, 24)
	DrawText(dst, text, x+12, y+6, 2, "#eaf6ff", 1)
}
