// G5.1: neon-glass theme constants and drawing primitives — layered rects,
// glass panels, cached additive glow sprites, status colors. Self-contained
// (does NOT import internal/render, to keep the widget kit decoupled).
package ui

import (
	"image"
	"image/color"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

// Theme colors (hex) and panel opacities. Accent cyan #00ffe1, accent2
// magenta #ff3df5, text #dfe8ff, dim #7a86b8 (README G5.1).
const (
	ColPanel     = "#080c20" // panel background rgb(8,12,32)
	ColBorder    = "#00ffe1" // glass border
	ColAccent    = "#00ffe1" // cyan accent
	ColAccent2   = "#ff3df5" // magenta accent
	ColText      = "#dfe8ff" // primary text
	ColDim       = "#7a86b8" // dimmed text / thought lines
	ColLogText   = "#7fb2e0" // tank-log lines
	ColWarn      = "#ffb347" // warnings
	ColErr       = "#ff4d5e" // errors
	ColOk        = "#4dff88" // success
	panelAlpha   = 0.82      // rgba(8,12,32,0.82)
	borderAlpha  = 0.35      // rgba(0,255,230,0.35)
	cornerAlpha  = 0.85      // corner tick brightness
	glowSpritePx = 96        // radial glow sprite diameter
)

// StatusColor maps an agent status (contract.AgentStatus string value) to its
// badge color: idle gray, thinking amber, writing cyan, done green, error red,
// simulated purple.
func StatusColor(st string) string {
	switch st {
	case "thinking":
		return ColWarn
	case "writing":
		return ColAccent
	case "done":
		return ColOk
	case "error":
		return ColErr
	case "simulated":
		return ColAccent2
	default:
		return ColDim // idle
	}
}

var (
	primMu        sync.Mutex
	whiteSprite   *ebiten.Image // 1x1 white, lazy
	glowMu        sync.Mutex
	glowSpriteImg *ebiten.Image // premultiplied-white radial falloff, lazy
)

var white = color.RGBA{R: 255, G: 255, B: 255, A: 255}

func ensureWhite() {
	primMu.Lock()
	defer primMu.Unlock()
	if whiteSprite == nil {
		whiteSprite = ebiten.NewImage(1, 1)
		whiteSprite.Fill(white)
	}
}

// hexRGB parses "#rrggbb" (or "#rrggbbaa") to 0..1 RGBA; gray on error (D4).
func hexRGB(hex string) ([4]float32, bool) {
	var out [4]float32
	n := len(hex)
	if (n != 7 && n != 9) || hex[0] != '#' {
		out[0], out[1], out[2], out[3] = 0.5, 0.5, 0.5, 1
		return out, false
	}
	v := func(i int) float32 {
		hi := hexDigit(hex[i])
		lo := hexDigit(hex[i+1])
		return float32(hi*16+lo) / 255
	}
	out[0], out[1], out[2] = v(1), v(3), v(5)
	if n == 9 {
		out[3] = v(7)
	} else {
		out[3] = 1
	}
	return out, true
}

func hexDigit(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return 0
}

// FillRect draws a solid axis-aligned rect tinted to hex with alpha 0..1.
func FillRect(dst *ebiten.Image, x, y, w, h int, hex string, alpha float64) {
	if dst == nil || w <= 0 || h <= 0 || alpha <= 0 {
		return
	}
	ensureWhite()
	c, _ := hexRGB(hex)
	r, g, b, a := c[0], c[1], c[2], c[3]*float32(alpha)
	var op ebiten.DrawImageOptions // premultiplied: scale RGB by A
	op.GeoM.Scale(float64(w), float64(h))
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.Scale(r*a, g*a, b*a, a)
	dst.DrawImage(whiteSprite, &op)
}

// FrameRect draws a 1 px outline.
func FrameRect(dst *ebiten.Image, x, y, w, h int, hex string, alpha float64) {
	FillRect(dst, x, y, w, 1, hex, alpha)
	FillRect(dst, x, y+h-1, w, 1, hex, alpha)
	FillRect(dst, x, y, 1, h, hex, alpha)
	FillRect(dst, x+w-1, y, 1, h, hex, alpha)
}

// glowSprite builds the shared premultiplied-white radial falloff sprite
// once (color-independent: Glow tints per draw via ColorScale — one GPU
// image total, the C3-bounded alternative to render's per-color cache).
func glowSprite() *ebiten.Image {
	glowMu.Lock()
	defer glowMu.Unlock()
	if glowSpriteImg != nil {
		return glowSpriteImg
	}
	img := ebiten.NewImage(glowSpritePx, glowSpritePx)
	center := float64(glowSpritePx-1) / 2
	for py := 0; py < glowSpritePx; py++ {
		for px := 0; px < glowSpritePx; px++ {
			dx, dy := float64(px)-center, float64(py)-center
			d := (dx*dx + dy*dy) / (center * center)
			if d > 1 {
				d = 1
			}
			a := (1 - d)
			a = a * a * a // soft cubic tail
			if a < 1.0/255 {
				continue
			}
			img.Set(px, py, color.RGBA{
				R: uint8(255 * a), G: uint8(255 * a), B: uint8(255 * a),
				A: uint8(255 * a), // premultiplied white
			})
		}
	}
	glowSpriteImg = img
	return img
}

// Glow draws an additive radial halo centered at (cx, cy). Local stand-in for
// render.DrawGlow so the widget kit has no render dependency.
func Glow(dst *ebiten.Image, cx, cy, radius float64, hex string, alpha float64) {
	if dst == nil || radius <= 0 || alpha <= 0 {
		return
	}
	sprite := glowSprite()
	c, _ := hexRGB(hex)
	var op ebiten.DrawImageOptions
	half := radius
	op.GeoM.Scale(half*2/float64(glowSpritePx), half*2/float64(glowSpritePx))
	op.GeoM.Translate(cx-half, cy-half)
	a := float32(alpha)
	op.ColorScale.Scale(c[0]*a, c[1]*a, c[2]*a, a)
	op.CompositeMode = ebiten.CompositeModeLighter
	dst.DrawImage(sprite, &op)
}

// GlassPanel draws the layered neon-glass panel: soft accent halo, dark glass
// fill rgba(8,12,32,0.82), top highlight, cyan border rgba(0,255,230,0.35)
// with cut corners and bright corner ticks (rounded-ish look, no paths).
func GlassPanel(dst *ebiten.Image, x, y, w, h int) {
	if dst == nil || w < 4 || h < 4 {
		return
	}
	Glow(dst, float64(x+w/2), float64(y+h/2), float64(w)*0.75, ColAccent, 0.05)
	FillRect(dst, x+2, y, w-4, h, ColPanel, panelAlpha)        // body, cut corners
	FillRect(dst, x, y+2, w, h-4, ColPanel, panelAlpha)        //   both axes
	FillRect(dst, x+2, y+1, w-4, 1, "#ffffff", 0.06)           // top highlight
	FrameRect(dst, x+2, y+2, w-4, h-4, ColBorder, borderAlpha) // inner border
	FillRect(dst, x+1, y+2, w-2, 1, ColBorder, borderAlpha)    // outer border
	FillRect(dst, x+1, y+h-3, w-2, 1, ColBorder, borderAlpha)
	FillRect(dst, x+2, y+1, 1, h-2, ColBorder, borderAlpha)
	FillRect(dst, x+w-3, y+1, 1, h-2, ColBorder, borderAlpha)
	drawCornerTicks(dst, x, y, w, h)
}

// drawCornerTicks stamps short bright accent segments at the four corners.
func drawCornerTicks(dst *ebiten.Image, x, y, w, h int) {
	t := 7
	FillRect(dst, x+1, y+1, t, 1, ColAccent, cornerAlpha)
	FillRect(dst, x+1, y+1, 1, t, ColAccent, cornerAlpha)
	FillRect(dst, x+w-t-1, y+1, t, 1, ColAccent, cornerAlpha)
	FillRect(dst, x+w-2, y+1, 1, t, ColAccent, cornerAlpha)
	FillRect(dst, x+1, y+h-2, t, 1, ColAccent, cornerAlpha)
	FillRect(dst, x+1, y+h-t-1, 1, t, ColAccent, cornerAlpha)
	FillRect(dst, x+w-t-1, y+h-2, t, 1, ColAccent, cornerAlpha)
	FillRect(dst, x+w-2, y+h-t-1, 1, t, ColAccent, cornerAlpha)
}

// TriRight draws a small solid right-pointing triangle (play icon, D6: drawn
// shape, no emoji), apex at the right. size must be >= 1.
func TriRight(dst *ebiten.Image, x, y, size int, hex string, alpha float64) {
	for i := 0; i < size; i++ {
		hh := 2*(size-i) - 1
		FillRect(dst, x+i, y+i, 1, hh, hex, alpha)
	}
}

// ChevronRight draws a right/left (flip) chevron made of stepped rects.
func ChevronRight(dst *ebiten.Image, x, y, size int, hex string, alpha float64, flip bool) {
	half := size / 2
	for i := 0; i < size; i++ {
		col := i
		if flip {
			col = size - 1 - i
		}
		hh := abs(col-half) + 1
		FillRect(dst, x+i, y+half-hh, 1, hh*2, hex, alpha)
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// subImage returns a clipped view of dst in the PARENT's coordinate system
// (ebiten SubImage semantics), used to clip tab content to the sliding panel.
func subImage(dst *ebiten.Image, r image.Rectangle) (*ebiten.Image, bool) {
	if r.Empty() {
		return nil, false
	}
	sub, ok := dst.SubImage(r).(*ebiten.Image)
	return sub, ok && sub != nil
}

// imageRect converts a ui Rect to an image.Rectangle.
func imageRect(r Rect) image.Rectangle {
	return image.Rect(r.X, r.Y, r.X+r.W, r.Y+r.H)
}

// pulse returns a 0..1 eased ping-pong wave for the given time (activity dot,
// handle shimmer).
func pulse(t float64) float64 { return 0.5 - 0.5*math.Cos(t*2.4) }

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// easeOutCubic is the menu slide easing (G5.2: 0.25 s easeOutCubic).
func easeOutCubic(t float64) float64 {
	u := 1 - clamp01(t)
	return 1 - u*u*u
}
