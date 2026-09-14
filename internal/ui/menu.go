// G5.2: right slide-in menu — 8 px edge handle (the ONLY thing drawn when
// closed, plus a pulsing dot when agent events arrived), 0.25 s easeOutCubic
// slide to a 380 px panel, tabs Agents/Species/Plants/Log/Settings. The menu
// never resizes or shifts the tank (immersion rule). Actions requested by tab
// buttons are queued here and drained by the game (G6.1) via Actions().
package ui

import (
	"image"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/Rollingzzzzz/Fish-tank/internal/content"
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// Menu layout + animation constants live in geometry.go (F12/F19 shared
// source with the smoke script and the evidence manifests).

// Menu owns the slide-in panel and all tab state.
type Menu struct {
	cfg   *contract.Config
	store *content.Store // may be nil until content loads

	open    bool    // target state
	slide   float64 // raw animation progress 0..1
	lastDt  float64 // seconds, dt of the most recent Update (input passthrough)
	w, h    int     // last known canvas size (from Draw)
	pending bool    // agent activity arrived while closed -> handle dot
	hArm    bool    // handle press arming
	pd      bool    // previous-frame button state (same-frame click fallback)
	clock   float64 // seconds, UI animation clock (pulse, plant sway)
	last    time.Time
	hasLast bool

	tabbar  TabBar
	agentsT *agentsTab
	specT   *catalogTab
	plantT  *plantsTab
	logT    *logTab
	setT    *settingsTab

	actions []Action
}

// NewMenu builds the menu around the live config (edited in place by
// Settings). Never touches the GPU — safe in tests.
func NewMenu(cfg *contract.Config) *Menu {
	if cfg == nil {
		cfg = &contract.Config{}
	}
	m := &Menu{
		cfg:     cfg,
		agentsT: newAgentsTab(),
		specT:   newCatalogTab(),
		plantT:  newPlantsTab(),
		logT:    newLogTab(),
		setT:    newSettingsTab(),
	}
	m.w, m.h = 1280, 720 // sane default until the first Draw sizes us
	m.tabbar.Tabs = []string{"Agents", "Species", "Plants", "Log", "Setup"}
	m.setT.bind(cfg)
	m.logT.append("NEON TANK menu ready.")
	return m
}

// Open reports whether the menu is (fully or partially) visible.
func (m *Menu) Open() bool { return m.slide > 0.0001 || m.open }

// SetOpen forces the menu open/closed (demo + game convenience).
func (m *Menu) SetOpen(open bool) {
	m.open = open
	if open {
		m.pending = false
	}
}

// SetContentStore attaches the content store (may be nil until loaded);
// tab preview caches are invalidated.
func (m *Menu) SetContentStore(s *content.Store) {
	m.store = s
	m.agentsT.resetPreviews()
	m.specT.reset()
	m.plantT.reset()
}

// PushEvent feeds one agent-bus event (thought/chunk/status/artifact/log).
func (m *Menu) PushEvent(ev contract.Event) {
	switch ev.Kind {
	case contract.EventLog:
		m.logT.append(ev.Text)
	case contract.EventThought, contract.EventChunk,
		contract.EventStatus, contract.EventArtifact:
		m.agentsT.handle(ev)
		if ev.Kind == contract.EventArtifact {
			m.logT.append(ev.Agent + " agent produced " + ev.ArtifactID)
		}
	}
	if !m.open {
		m.pending = true
	}
}

// Actions drains and returns the queued actions (G6.1 consumption).
func (m *Menu) Actions() []Action {
	out := m.actions
	m.actions = nil
	return out
}

func (m *Menu) enqueue(as ...Action) { m.actions = append(m.actions, as...) }

// progress returns the eased slide progress 0..1.
func (m *Menu) progress() float64 { return easeOutCubic(clamp01(m.slide)) }

// advance moves the slide animation toward the open target. Pure given dt.
func (m *Menu) advance(dt float64) {
	if dt < 0 {
		dt = 0
	}
	m.lastDt = dt
	step := dt / SlideSec
	if m.open {
		m.slide += step
	} else {
		m.slide -= step
	}
	m.slide = clamp01(m.slide)
	m.clock += dt
}

// Update processes one input frame. clicked = left button held this frame,
// released = left button released this frame (widget convention, see doc.go).
func (m *Menu) Update(mx, my int, clicked, released bool) {
	nowT := time.Now()
	if !m.hasLast {
		m.hasLast, m.last = true, nowT
	}
	dt := nowT.Sub(m.last).Seconds()
	m.last = nowT
	m.advance(dt)

	p := m.progress()
	hr := m.handleRect(p)
	inside := HitR(hr, mx, my) || HitR(m.handleChipRect(hr), mx, my)
	// Handle click toggles the menu (arm on press inside, fire on release).
	if !m.hArm && clicked && inside {
		m.hArm = true
	}
	if m.hArm {
		if released {
			m.hArm = false
			if inside {
				m.SetOpen(!m.open)
			}
		} else if !clicked {
			m.hArm = false
		}
	} else if released && !m.pd && inside {
		// Same-frame press+release (fast drivers / automation) never arms —
		// treat it as a full click anyway so the menu always responds.
		m.SetOpen(!m.open)
	}
	m.pd = clicked

	area, ok := m.contentArea()
	if ok && p > 0.6 { // tabs interactive once mostly slid in (F5)
		m.tabbar.Update(area.X, m.panelTop()+MenuPad, area.W, TabH, mx, my, clicked, released)
		switch m.tabbar.Active {
		case 0:
			m.agentsT.update(area, mx, my, clicked, released, m)
		case 1:
			if id := m.specT.update(area, mx, my, clicked, released); id != "" {
				m.enqueue(Action{Kind: ActionSelectSpecies, ID: id})
			}
		case 2:
			if id := m.plantT.update(area, mx, my, clicked, released); id != "" {
				m.enqueue(Action{Kind: ActionApplyPlant, ID: id})
			}
		case 3:
			m.logT.update(area, mx, my, clicked, released)
		case 4:
			m.setT.update(area, mx, my, clicked, released, m)
		}
	}
}

// handleRect is the current handle position: glued to the panel's left edge
// while animating, at the screen's right edge when closed.
func (m *Menu) handleRect(p float64) Rect {
	px := m.panelX(p)
	return Rect{X: px - HandleW, Y: m.h/2 - HandleH/2, W: HandleW, H: HandleH}
}

// handleChipRect is the "MENU" label chip below-left of the grip (F12: the
// closed menu must be discoverable — an 8 px strip was invisible).
func (m *Menu) handleChipRect(hr Rect) Rect {
	return Rect{X: hr.X - ChipW - 4, Y: hr.Y + hr.H + 6, W: ChipW, H: ChipH}
}

func (m *Menu) panelX(p float64) int { return m.w - int(float64(PanelW)*p) }

// panelTop is the open panel's top edge — shared geometry with the smoke
// script and the manifests (F28: it clears TREATS and the close button).
func (m *Menu) panelTop() int { return UIMetrics(m.w, m.h, true).PanelTopY }

// contentArea returns the inner area of the active tab view, if visible.
// Geometry comes from UIMetrics — the same source the smoke script and the
// evidence manifests use (F12/F19: no drift possible).
func (m *Menu) contentArea() (Rect, bool) {
	if m.progress() <= 0.0001 {
		return Rect{}, false
	}
	return UIMetrics(m.w, m.h, true).Content, true
}

// Draw renders the handle always; panel + tabs only while opening/open.
func (m *Menu) Draw(dst *ebiten.Image) {
	if dst == nil {
		return
	}
	b := dst.Bounds()
	m.w, m.h = b.Dx(), b.Dy()
	p := m.progress()
	hr := m.handleRect(p)
	if p <= 0.0001 && !m.open {
		m.drawHandle(dst, hr) // G5.2: pixel-pure tank — handle only
		return
	}
	px := m.panelX(p)
	vw := m.w - px
	if vw > PanelW {
		vw = PanelW
	}
	top := m.panelTop() // F28: the panel opens BELOW TREATS and the X
	GlassPanel(dst, px, top, vw, m.h-top)
	area, _ := m.contentArea()
	// Clip content to the panel so the slide never bleeds over the tank.
	// The draw helpers take GLOBAL coordinates — the clip rect only bounds
	// where pixels may land — so the tab strip is drawn at its global Y.
	if clip, ok := subImage(dst, image.Rect(px+1, top+1, m.w, m.h-1)); ok {
		m.tabbar.DrawTabBar(clip, area.X, top+MenuPad, area.W, TabH)
		switch m.tabbar.Active {
		case 0:
			m.agentsT.draw(clip, m, area)
		case 1:
			m.specT.draw(clip, m, area)
		case 2:
			m.plantT.draw(clip, m, area)
		case 3:
			m.logT.draw(clip, m, area)
		case 4:
			m.setT.draw(clip, m, area)
		}
	}
	m.drawHandle(dst, hr)
}

// drawHandle renders the grip + its MENU chip: a visible affordance with a
// chevron, and a pulsing magenta activity dot when events arrived while
// closed (G5.2 acceptance; F12: was an invisible 8 px strip).
func (m *Menu) drawHandle(dst *ebiten.Image, hr Rect) {
	hex := ColAccent
	a := 0.65 + 0.25*pulse(m.clock)
	FillRect(dst, hr.X, hr.Y, hr.W, hr.H, ColPanel, 0.88)
	FillRect(dst, hr.X, hr.Y, 2, hr.H, hex, a)
	FillRect(dst, hr.X+hr.W-1, hr.Y, 1, hr.H, hex, borderAlpha)
	FillRect(dst, hr.X, hr.Y, hr.W, 1, hex, a)
	FillRect(dst, hr.X, hr.Y+hr.H-1, hr.W, 1, hex, a)
	ChevronRight(dst, hr.X+4, hr.Y+hr.H/2-6, 8, hex, 0.95, m.open)
	// grip ridges
	for i := 0; i < 3; i++ {
		FillRect(dst, hr.X+4, hr.Y+hr.H/2+8+i*4, hr.W-8, 1, hex, 0.35)
	}
	// MENU chip below-left — the discoverable label
	ch := m.handleChipRect(hr)
	FillRect(dst, ch.X, ch.Y, ch.W, ch.H, ColPanel, 0.88)
	FrameRect(dst, ch.X, ch.Y, ch.W, ch.H, hex, a*0.8)
	lw := TextWidth("MENU", 2)
	DrawText(dst, "MENU", ch.X+(ch.W-lw)/2, ch.Y+(ch.H-LineHeight(2))/2+1, 2, "#ffffff", 1)
	if m.pending && !m.open {
		Glow(dst, float64(hr.X+hr.W/2), float64(hr.Y+8), 6+4*pulse(m.clock*2.1), ColAccent2, 0.9)
		FillRect(dst, hr.X+hr.W/2-2, hr.Y+6, 4, 4, ColAccent2, 1)
	}
}
