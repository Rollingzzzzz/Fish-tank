// G5.1: immediate-mode widgets — Panel, Button, Toggle, Slider, TabBar.
// Hit-testing and state transitions are pure methods on the state structs
// (unit-testable without ebiten); Draw* methods only render. Callers feed
// (mx, my, pressed, released) every frame and react to the returned events.
package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// Rect is an axis-aligned pixel rectangle.
type Rect struct{ X, Y, W, H int }

// Hit reports whether (mx, my) lies inside the rect. Pure.
func Hit(x, y, w, h, mx, my int) bool {
	return mx >= x && mx < x+w && my >= y && my < y+h
}

// HitR is Hit for a Rect. Pure.
func HitR(r Rect, mx, my int) bool { return Hit(r.X, r.Y, r.W, r.H, mx, my) }

// ButtonState carries hover/press feedback between frames. Zero value ready.
type ButtonState struct {
	Hover bool
	Armed bool // press started inside; fires on release inside
}

// Update advances the button; returns true exactly once per completed click
// (pressed inside, then released inside). Pure — no ebiten. released is the
// button-up edge; it is implied by pressed=false but accepted for symmetry.
func (b *ButtonState) Update(x, y, w, h, mx, my int, pressed, _ bool) bool {
	inside := Hit(x, y, w, h, mx, my)
	b.Hover = inside
	fire := false
	if pressed {
		if !b.Armed && inside {
			b.Armed = true
		}
	} else if b.Armed { // button up this frame
		b.Armed = false
		fire = inside
	}
	return fire
}

// DrawButton renders label centered in a glass button; accent when hovered,
// brighter fill while armed (finger still down). F12: resting buttons carry a
// visible accent fill + frame — they must read as buttons at a glance, not
// melt into the panel.
func DrawButton(dst *ebiten.Image, b *ButtonState, x, y, w, h int, label, accent string) {
	frameA := 0.65
	fillA := 0.35
	if b.Armed {
		fillA = 0.55
		frameA = 0.95
	} else if b.Hover {
		frameA = 0.9
	}
	FillRect(dst, x+1, y+1, w-2, h-2, ColPanel, 0.9)
	FillRect(dst, x+1, y+1, w-2, h-2, accent, fillA)
	if b.Hover {
		Glow(dst, float64(x+w/2), float64(y+h/2), float64(w)/2, accent, 0.16)
	}
	FrameRect(dst, x+1, y+1, w-2, h-2, accent, frameA)
	drawCornerTicks(dst, x, y, w, h)
	sc := 2
	for TextWidth(label, 2) > w-10 && sc > 1 {
		sc = 1
	}
	lw := TextWidth(label, sc)
	DrawText(dst, label, x+(w-lw)/2, y+(h-LineHeight(sc))/2+1, sc, ColText, 1)
}

// ToggleState is a labeled on/off switch. Zero value is off.
type ToggleState struct {
	On    bool
	Hover bool
	armed bool
}

// Update advances the toggle; returns true on the frame it flips. Pure.
func (t *ToggleState) Update(x, y, w, h, mx, my int, pressed, released bool) bool {
	inside := Hit(x, y, w, h, mx, my)
	t.Hover = inside
	flip := false
	if pressed {
		if !t.armed && inside {
			t.armed = true
		}
	} else if t.armed {
		t.armed = false
		if inside {
			t.On = !t.On
			flip = true
		}
	}
	return flip
}

// DrawToggle renders label left, switch track right (w x h describes the
// whole row; the track sits at the right edge). F12: larger track + brighter
// resting frame so the on/off state is legible at a glance.
func DrawToggle(dst *ebiten.Image, t *ToggleState, x, y, w, h int, label string) {
	DrawText(dst, label, x, y+(h-LineHeight(2))/2+1, 2, ColText, 1)
	tw, th := 40, 16
	tx, ty := x+w-tw, y+(h-th)/2
	onHex := ColAccent
	if !t.On {
		onHex = ColDim
	}
	FillRect(dst, tx, ty, tw, th, ColPanel, 0.95)
	FillRect(dst, tx+1, ty+1, tw-2, th-2, onHex, map[bool]float64{true: 0.40, false: 0.18}[t.On])
	FrameRect(dst, tx, ty, tw, th, onHex, 0.6)
	knobX := tx + 2
	if t.On {
		knobX = tx + tw - th + 1
	}
	Glow(dst, float64(knobX+th/2-1), float64(ty+th/2), 7, onHex, 0.5)
	FillRect(dst, knobX, ty+1, th-2, th-2, onHex, 1)
	if t.Hover {
		FrameRect(dst, tx-1, ty-1, tw+2, th+2, onHex, 0.7)
	}
}

// SliderState is a horizontal 0..1 slider. Zero value = 0, not dragging.
type SliderState struct {
	Value float64
	Hover bool
	Drag  bool
	armed bool
}

// Update advances the slider; returns true when Value changed this frame.
// Pure — geometry and clamping are testable without ebiten.
func (s *SliderState) Update(x, y, w, h, mx, my int, pressed, released bool) bool {
	s.Hover = Hit(x, y, w, h, mx, my)
	changed := false
	if pressed && !s.armed && s.Hover {
		s.armed = true
		s.Drag = true
	}
	if s.armed {
		if pressed {
			nv := sliderNorm(x, w, mx)
			if nv != s.Value {
				s.Value = nv
				changed = true
			}
		}
		if released || !pressed {
			s.armed = false
			s.Drag = false
		}
	}
	return changed
}

// sliderNorm maps mx to 0..1 across the trough (with a knob-width margin).
func sliderNorm(x, w, mx int) float64 {
	const knob = 10
	span := w - knob
	if span <= 0 {
		return 0
	}
	v := (float64(mx-x-knob/2) / float64(span))
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	return v
}

// DrawSlider renders a neon trough with glow knob. Label is drawn by the
// caller (it owns formatting, e.g. "Day length: 60 s").
func DrawSlider(dst *ebiten.Image, s *SliderState, x, y, w, h int) {
	const knob = 10
	ty := y + h/2 - 2
	FillRect(dst, x+1, ty, w-2, 4, ColPanel, 1)
	FillRect(dst, x+1, ty, w-2, 4, ColAccent, 0.18)
	FrameRect(dst, x, ty-1, w, 6, ColBorder, borderAlpha)
	fillW := int(float64(w-knob) * s.Value)
	FillRect(dst, x+knob/2, ty+1, fillW, 2, ColAccent, 0.9)
	kx := x + knob/2 + fillW
	Glow(dst, float64(kx), float64(ty+2), 9, ColAccent, 0.6)
	FillRect(dst, kx-knob/2+1, ty-3, knob-2, 10, ColAccent, 1)
	FillRect(dst, kx-knob/2+3, ty-1, knob-6, 6, "#ffffff", 0.85)
}

// TabBar is a row of N equal tabs; Active indexes Tabs. Zero value ready
// after Tabs is set.
type TabBar struct {
	Tabs   []string
	Active int
	hover  int
	armed  bool
}

// Update advances the tab bar; returns true when Active changed. Pure.
func (t *TabBar) Update(x, y, tw, th, mx, my int, pressed, released bool) bool {
	n := len(t.Tabs)
	if n == 0 {
		return false
	}
	t.hover = -1
	for i := 0; i < n; i++ {
		if Hit(x+i*tw/n, y, tw/n, th, mx, my) {
			t.hover = i
			break
		}
	}
	changed := false
	if pressed && !t.armed && t.hover >= 0 {
		t.armed = true
	}
	if t.armed {
		if released {
			t.armed = false
			if t.hover >= 0 && t.hover != t.Active {
				t.Active = t.hover
				changed = true
			}
		} else if !pressed {
			t.armed = false
		}
	}
	return changed
}

// DrawTabBar renders the tab row; the active tab gets an accent underline.
// F12: labels render at scale 2 (auto-shrinking only if a cell is too
// narrow) with per-cell separators — tabs are the menu's primary controls.
func (t *TabBar) DrawTabBar(dst *ebiten.Image, x, y, tw, th int) {
	n := len(t.Tabs)
	if n == 0 {
		return
	}
	FillRect(dst, x, y, tw, th, ColPanel, 0.55)
	for i := 0; i < n; i++ {
		cx, cw := x+i*tw/n, tw/n
		hex := "#ffffff" // F12: inactive tabs read at full brightness
		a := 0.95
		if i == t.Active {
			hex = ColAccent
			a = 1
		} else if i == t.hover {
			a = 1
		}
		if i > 0 {
			FillRect(dst, cx, y+4, 1, th-8, ColBorder, 0.4)
		}
		sc := 2
		for TextWidth(t.Tabs[i], 2) > cw-6 && sc > 1 {
			sc = 1
		}
		lw := TextWidth(t.Tabs[i], sc)
		DrawText(dst, t.Tabs[i], cx+(cw-lw)/2, y+(th-LineHeight(sc))/2+1, sc, hex, a)
		if i == t.Active {
			FillRect(dst, cx+6, y+th-3, cw-12, 3, ColAccent, 0.95)
			Glow(dst, float64(cx+cw/2), float64(y+th-2), 12, ColAccent, 0.30)
		}
	}
	FrameRect(dst, x, y, tw, th, ColBorder, borderAlpha)
}
