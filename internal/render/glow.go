// G1.2: cached radial glow sprites + additive DrawGlow helper.
package render

import (
	"image"
	"image/color"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

const glowSize = 128

var (
	glowMu    sync.Mutex
	glowCache = map[string]*ebiten.Image{}
)

// glowSprite builds (once per color) a radial falloff sprite: bright core,
// exponential tail — the phosphor look (art direction §3).
func glowSprite(hex string) *ebiten.Image {
	glowMu.Lock()
	defer glowMu.Unlock()
	if img, ok := glowCache[hex]; ok {
		return img
	}
	c := hexRGBA(hex)
	src := image.NewRGBA(image.Rect(0, 0, glowSize, glowSize))
	center := float64(glowSize-1) / 2
	for y := 0; y < glowSize; y++ {
		for x := 0; x < glowSize; x++ {
			dx, dy := float64(x)-center, float64(y)-center
			d := (dx*dx + dy*dy) / (center * center) // 0 center .. 1 edge
			if d > 1 {
				d = 1
			}
			a := (1 - d)
			a = a * a * a // cubic falloff, soft tail
			// store PREMULTIPLIED pixels — ebiten samples image data as
			// premultiplied; straight rgb here bleeds as square halos
			src.SetRGBA(x, y, color.RGBA{
				R: uint8(float64(c.R) * a), G: uint8(float64(c.G) * a),
				B: uint8(float64(c.B) * a), A: uint8(255 * a),
			})
		}
	}
	img := ebiten.NewImageFromImage(src)
	glowCache[hex] = img
	return img
}

// GlowBudget is the global additive-brightness multiplier (F2): every glow
// stamp and every additive mesh draw is scaled by it, so the whole scene's
// neon intensity has one tuning knob. 1 = v0.1 intensity; 0.55 = the v0.2
// eye-friendly default.
var GlowBudget = float32(0.55)

// DrawGlow stamps an additive radial glow centered at (x, y).
// radius = visible halo radius in px; alpha 0..1. Safe on nil dst.
func DrawGlow(dst *ebiten.Image, x, y, radius float64, hex string, alpha float64) {
	if dst == nil || radius <= 0 || alpha <= 0 {
		return
	}
	if alpha > 1 {
		alpha = 1
	}
	alpha *= float64(GlowBudget) // F2: single choke point for all glow stamps
	img := glowSprite(hex)
	s := radius * 2 / glowSize
	opts := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear, Blend: ebiten.BlendLighter}
	opts.GeoM.Scale(s, s)
	opts.GeoM.Translate(x-radius, y-radius)
	opts.ColorScale.Scale(float32(alpha), float32(alpha), float32(alpha), float32(alpha))
	dst.DrawImage(img, opts)
}

// DrawOrb draws a soft filled circle (normal alpha blend) — bubbles, eggs.
func DrawOrb(dst *ebiten.Image, x, y, r float64, hex string, alpha uint8) {
	if dst == nil || r <= 0 {
		return
	}
	var m mesh
	m.fan(v2(x, y), r, withA(hexRGBA(hex), alpha), 14)
	m.draw(dst, false)
}
