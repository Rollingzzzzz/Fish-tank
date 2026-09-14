// G5.1: TextInput — click-to-focus single-line editor with blinking caret,
// masked mode (API keys), backspace with hold-repeat, max length. Keyboard
// reading uses ebiten only inside Update; Insert/Backspace are pure (tested).
// NOTE: Ebiten has no clipboard API in the standard build — Ctrl+V paste is
// not supported in v0.1 (documented in docs/NOTES-laneC.md).
package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// TextInput is a single-line editor bound to a string value.
type TextInput struct {
	Value       string
	Placeholder string
	MaxLen      int // 0 = unlimited (still capped at 512 for safety)
	Masked      bool

	Focused bool
	caret   float64 // blink phase, seconds
	bsHold  int     // frames Backspace held (repeat)
}

const (
	inputHardCap  = 512
	bsRepeatAfter = 30  // frames before hold-repeat kicks in
	bsRepeatEvery = 3   // frames between repeats
	blinkPeriod   = 1.1 // seconds, full blink cycle
)

// Insert appends runes respecting MaxLen. Pure.
func (t *TextInput) Insert(rs []rune) {
	max := t.MaxLen
	if max <= 0 || max > inputHardCap {
		max = inputHardCap
	}
	for _, r := range rs {
		if r == '\n' || r == '\r' {
			continue
		}
		if len([]rune(t.Value)) >= max {
			return
		}
		t.Value += string(r)
	}
}

// Backspace removes the last rune. Pure.
func (t *TextInput) Backspace() {
	rs := []rune(t.Value)
	if len(rs) > 0 {
		t.Value = string(rs[:len(rs)-1])
	}
}

// Set rebinds the field to an external value (e.g. Config on open).
func (t *TextInput) Set(v string) { t.Value = v }

// Update handles focus clicks and keyboard input; dt advances the caret
// blink. Call once per frame while the input is on screen.
func (t *TextInput) Update(x, y, w, h, mx, my int, pressed, released bool, dt float64) {
	t.caret += dt
	if pressed && !t.Focused && Hit(x, y, w, h, mx, my) {
		t.Focused = true
		t.caret = 0
	}
	if t.Focused && released && !Hit(x, y, w, h, mx, my) {
		t.Focused = false
	}
	if !t.Focused {
		return
	}
	// Enter / Escape defocus (commit is live — Value is always current).
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		t.Focused = false
		return
	}
	// Printable runes from the OS layout (Turkish layouts yield Turkish runes).
	t.Insert(ebiten.InputChars())
	// Backspace: instant, then hold-repeat.
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		t.Backspace()
		t.bsHold = 1
	} else if ebiten.IsKeyPressed(ebiten.KeyBackspace) {
		t.bsHold++
		if t.bsHold > bsRepeatAfter && t.bsHold%bsRepeatEvery == 0 {
			t.Backspace()
		}
	} else {
		t.bsHold = 0
	}
}

// Draw renders the field: glass slot, masked or plain text, placeholder when
// empty, caret line while focused.
func (t *TextInput) Draw(dst *ebiten.Image, x, y, w, h int) {
	hex := ColBorder
	a := borderAlpha
	if t.Focused {
		a = 0.8
		Glow(dst, float64(x+w/2), float64(y+h/2), float64(w)/2.4, ColAccent, 0.07)
	}
	FillRect(dst, x+1, y+1, w-2, h-2, ColPanel, 0.95)
	FrameRect(dst, x, y, w, h, hex, a)

	draw := t.Value
	if t.Masked {
		draw = ""
		for range t.Value {
			draw += "*"
		}
	}
	sc := 2
	for TextWidth(draw, sc) > w-12 && sc > 1 {
		sc = 1
	}
	if draw == "" && !t.Focused {
		DrawText(dst, Ellipsis(t.Placeholder, w/6-1), x+6, y+(h-LineHeight(1))/2+1, 1, ColDim, 0.7)
		return
	}
	tx := x + 6
	DrawText(dst, Ellipsis(draw, w/(6*sc)-1), tx, y+(h-LineHeight(sc))/2+1, sc, ColText, 1)
	if t.Focused && int(t.caret/blinkPeriod*2)%2 == 0 {
		cx := tx + TextWidth(Ellipsis(draw, w/(6*sc)-1), sc) + 2
		FillRect(dst, cx, y+4, 2, h-8, ColAccent, 0.95)
	}
}

// DrawLabeled renders a small dim caption above the field at (x, y).
func DrawLabeled(dst *ebiten.Image, caption string, x, y int) {
	DrawText(dst, caption, x, y, 1, ColDim, 0.9)
}
