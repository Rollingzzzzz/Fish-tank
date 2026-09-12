// G1.2: phosphor trail buffer — previous bright pixels fade exponentially
// instead of clearing, giving every moving glow a soft comet trail.
// Implemented as a ping-pong pair: Fade scales the previous frame down
// into the spare buffer (alpha blend over empty = exponential decay).
package render

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// Trail is a double-buffered offscreen RGBA buffer.
type Trail struct {
	cur  *ebiten.Image
	next *ebiten.Image
	w    int
	h    int
}

// NewTrail creates a w×h trail buffer pair.
func NewTrail(w, h int) *Trail {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	return &Trail{cur: ebiten.NewImage(w, h), next: ebiten.NewImage(w, h), w: w, h: h}
}

// Resize recreates the buffers when the window changes.
func (t *Trail) Resize(w, h int) {
	if w < 1 || h < 1 || (w == t.w && h == t.h) {
		return
	}
	t.cur, t.next = ebiten.NewImage(w, h), ebiten.NewImage(w, h)
	t.w, t.h = w, h
}

// Image exposes the current buffer for stamping bright bits (additive draws).
func (t *Trail) Image() *ebiten.Image { return t.cur }

// Fade scales existing content down by (1 - frac) via ping-pong blit.
func (t *Trail) Fade(frac float64) {
	if frac <= 0 {
		return
	}
	if frac > 1 {
		frac = 1
	}
	t.next.Clear()
	opts := &ebiten.DrawImageOptions{}
	opts.ColorScale.Scale(float32(1-frac), float32(1-frac), float32(1-frac), float32(1-frac))
	t.next.DrawImage(t.cur, opts)
	t.cur, t.next = t.next, t.cur
}

// Blit composites the trail additively onto dst (1:1 pixels).
func (t *Trail) Blit(dst *ebiten.Image) {
	dst.DrawImage(t.cur, &ebiten.DrawImageOptions{Blend: ebiten.BlendLighter})
}
