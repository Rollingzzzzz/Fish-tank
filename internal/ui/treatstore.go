// N8: Treat Store tray — a glass button next to the HUD that opens a tray of
// four live treats. F12/F13: geometry is canvas-width-derived via UIMetrics
// (shared source with the smoke script and evidence manifests); the tray
// learns the canvas width from Draw. Update/Hit are pure (unit-tested
// without ebiten); Draw only renders.
package ui

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/Rollingzzzzz/Fish-tank/internal/render"
)

// treatKinds is the tray order, top to bottom.
var treatKinds = []string{
	contract.TreatBug, contract.TreatWorm, contract.TreatShrimp, contract.TreatChicken,
}

var treatLabels = map[string]string{
	contract.TreatBug:     "Bug",
	contract.TreatWorm:    "Worm",
	contract.TreatShrimp:  "Shrimp",
	contract.TreatChicken: "Chicken",
}

// TreatTray is the top-right treat store UI.
type TreatTray struct {
	Open bool

	w       int  // canvas width, learned from Draw (F13 proportional layout)
	btnArm  bool // press armed inside the button (menu handle idiom)
	rowDown bool // left-button level seen by the rows last frame (F26)
}

// NewTreatTray builds the tray (closed).
func NewTreatTray() *TreatTray { return &TreatTray{w: 1280} }

func (t *TreatTray) metrics() *Metrics { return UIMetrics(t.w, 720, false) }

func (t *TreatTray) buttonRect() Rect { return t.metrics().TrayBtn }
func (t *TreatTray) panelRect() Rect  { return t.metrics().TrayPanel }

func (t *TreatTray) rowRect(i int) Rect {
	r := t.metrics().TrayRows[i]
	return Rect{X: r.X + 4, Y: r.Y, W: r.W - 8, H: r.H - 2}
}

// Update processes one input frame. pressed is the LEVEL state of the left
// button, released its up edge (widget convention). Returns "" or
// "grab:<kind>" when a treat row is PRESSED (F26: the first press selects —
// the treat rides the cursor until the next left press drops it), which also
// closes the tray.
func (t *TreatTray) Update(mx, my int, pressed, released bool) string {
	t.updateButton(mx, my, pressed, released)
	if !t.Open {
		t.rowDown = pressed
		return ""
	}
	pressEdge := pressed && !t.rowDown
	t.rowDown = pressed
	for i, kind := range treatKinds {
		if pressEdge && HitR(t.rowRect(i), mx, my) {
			t.Open = false
			return "grab:" + kind
		}
	}
	return ""
}

// updateButton toggles Open on a completed click (arm on press inside, fire
// on release inside — same state machine as the menu handle).
func (t *TreatTray) updateButton(mx, my int, pressed, released bool) {
	r := t.buttonRect()
	if !t.btnArm && pressed && HitR(r, mx, my) {
		t.btnArm = true
	}
	if t.btnArm {
		if released {
			t.btnArm = false
			if HitR(r, mx, my) {
				t.Open = !t.Open
			}
		} else if !pressed {
			t.btnArm = false
		}
	}
}

// Hit reports whether the point is over the tray button, or over the open
// tray panel (clicks there never feed the tank).
func (t *TreatTray) Hit(mx, my int) bool {
	if HitR(t.buttonRect(), mx, my) {
		return true
	}
	return t.Open && HitR(t.panelRect(), mx, my)
}

// Draw paints the tray button, the open store, and highlights the currently
// grabbed treat. Draw is also where the tray learns the canvas width (F13).
func (t *TreatTray) Draw(dst *ebiten.Image, grabKind string) {
	if dst == nil {
		return
	}
	if w := dst.Bounds().Dx(); w > 0 {
		t.w = w
	}
	t.drawButton(dst)
	if t.Open {
		t.drawPanel(dst, grabKind)
	}
}

// drawButton renders the glass chip with a centered TREATS label (F12: 80x30
// with a scale-2 label — the store entrance must be obvious).
func (t *TreatTray) drawButton(dst *ebiten.Image) {
	r := t.buttonRect()
	fillA, frameA := 0.35, 0.65
	if t.Open {
		fillA, frameA = 0.5, 0.95
	}
	FillRect(dst, r.X, r.Y, r.W, r.H, ColPanel, 0.85)
	FillRect(dst, r.X+1, r.Y+1, r.W-2, r.H-2, ColAccent, fillA)
	FrameRect(dst, r.X, r.Y, r.W, r.H, ColAccent, frameA)
	drawCornerTicks(dst, r.X, r.Y, r.W, r.H)
	tw := TextWidth("TREATS", 2)
	DrawText(dst, "TREATS", r.X+(r.W-tw)/2, r.Y+(r.H-LineHeight(2))/2+1, 2, "#ffffff", 1)
}

// drawPanel renders the open tray: one clickable row per treat kind, each a
// procedural glyph plus a scale-2 label; the grabbed kind's row is
// highlighted.
func (t *TreatTray) drawPanel(dst *ebiten.Image, grabKind string) {
	p := t.panelRect()
	GlassPanel(dst, p.X, p.Y, p.W, p.H)
	for i, kind := range treatKinds {
		r := t.rowRect(i)
		if kind == grabKind {
			FillRect(dst, r.X, r.Y, r.W, r.H, ColAccent, 0.30)
			FrameRect(dst, r.X, r.Y, r.W, r.H, ColAccent, 0.8)
		} else {
			FillRect(dst, r.X, r.Y, r.W, r.H, ColPanel, 0.6)
			FrameRect(dst, r.X, r.Y, r.W, r.H, ColBorder, 0.5)
		}
		drawTreatGlyph(dst, kind, float64(r.X+16), float64(r.Y+r.H/2))
		DrawText(dst, treatLabels[kind], r.X+30, r.Y+(r.H-LineHeight(2))/2+1, 2, "#ffffff", 1)
	}
}

// drawTreatGlyph paints a small procedural preview of the treat kind at
// (x, y). Glow alphas stay <= 0.15, orb alphas <= 160 (sparing accents).
func drawTreatGlyph(dst *ebiten.Image, kind string, x, y float64) {
	switch kind {
	case contract.TreatBug: // tiny magenta orb + wing dots
		render.DrawGlow(dst, x, y, 7, ColAccent2, 0.15)
		render.DrawOrb(dst, x, y, 2.6, ColAccent2, 160)
		FillRect(dst, int(x)-7, int(y)-3, 4, 1, ColText, 0.6)
		FillRect(dst, int(x)+3, int(y)-3, 4, 1, ColText, 0.6)
	case contract.TreatWorm: // short wiggling lime line of 3 orbs
		for i := 0; i < 3; i++ {
			oy := math.Sin(float64(i)*2.1) * 3
			render.DrawOrb(dst, x+float64(i-1)*6, y+oy, 2.2, ColOk, 160)
		}
	case contract.TreatShrimp: // small cyan crescent of 3 orbs
		for i := 0; i < 3; i++ {
			a := math.Pi*0.75 + math.Pi*0.5*float64(i)/2
			render.DrawOrb(dst, x+math.Cos(a)*5, y+math.Sin(a)*4, 2, "#7fd4ff", 160)
		}
	default: // chicken: warm orb cluster
		render.DrawGlow(dst, x, y, 8, "#ffd9a0", 0.12)
		render.DrawOrb(dst, x, y, 3, "#ffd9a0", 160)
		render.DrawOrb(dst, x-4, y+3, 1.8, "#ffd9a0", 120)
		render.DrawOrb(dst, x+4, y+2, 1.6, "#ffe9a0", 120)
	}
}
