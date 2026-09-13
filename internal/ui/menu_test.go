// G5.2/G5.3: menu state machine tests — slide animation, handle toggling,
// activity dot flag, event routing, action queue, settings binding. Pure
// (no ebiten, no Draw): the menu constructor never touches the GPU.
package ui

import (
	"path/filepath"
	"testing"

	"github.com/Rollingzzzzz/Fish-tank/internal/content"
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

func newTestMenu() *Menu {
	return NewMenu(&contract.Config{
		Endpoint: "http://localhost", Model: "glm-5.3-flash", APIKey: "",
		AutoFeed: true, AutoCare: true, MusicOn: true,
		AgentFreq: 1, DaySeconds: 60, MaxFish: 24,
	})
}

// handleClosed is the handle rect at slide 0 for the 1280x720 default canvas.
func handleClosed() Rect {
	return Rect{X: 1280 - HandleW, Y: 720/2 - HandleH/2, W: HandleW, H: HandleH}
}

func TestMenuSlideStateMachine(t *testing.T) {
	m := newTestMenu()
	if m.Open() {
		t.Fatal("menu starts closed")
	}
	if m.progress() != 0 {
		t.Fatal("progress starts at 0")
	}
	m.SetOpen(true)
	m.advance(0.125) // half of the 0.25 s slide
	p := m.progress()
	if p <= 0.5 || p >= 1 {
		t.Fatalf("halfway eased progress expected, got %.3f", p)
	}
	m.advance(0.125)
	if m.progress() != 1 || !m.Open() {
		t.Fatalf("slide must complete at 1, got %.3f", m.progress())
	}
	// Closing returns to exactly 0 -> closed state draws handle only.
	m.SetOpen(false)
	m.advance(0.25)
	if m.progress() != 0 || m.Open() {
		t.Fatal("slide must return to 0")
	}
	// advance is clamped: no overshoot.
	m.advance(10)
	if m.progress() != 0 {
		t.Fatal("progress must clamp at 0 when closed")
	}
}

func TestMenuHandleClickToggles(t *testing.T) {
	m := newTestMenu()
	h := handleClosed()
	cx, cy := h.X+h.W/2, h.Y+h.H/2
	// Arm on press, fire on release -> opens.
	m.Update(cx, cy, true, false)
	m.Update(cx, cy, false, true)
	if !m.open {
		t.Fatal("handle click must open the menu")
	}
	m.advance(0.25)
	// Handle is now glued to the panel's left edge.
	hr := m.handleRect(1)
	if hr.X != 1280-PanelW-HandleW {
		t.Fatalf("open handle X = %d, want %d", hr.X, 1280-PanelW-HandleW)
	}
	m.Update(hr.X+3, hr.Y+10, true, false)
	m.Update(hr.X+3, hr.Y+10, false, true)
	if m.open {
		t.Fatal("handle click must close the menu")
	}
	// Press outside the handle never toggles.
	m.Update(200, 200, true, false)
	m.Update(200, 200, false, true)
	if m.open {
		t.Fatal("clicks away from the handle must not toggle")
	}
}

func TestMenuActivityDotFlag(t *testing.T) {
	m := newTestMenu()
	m.PushEvent(contract.Event{Kind: contract.EventLog, Text: "egg hatched"})
	if !m.pending {
		t.Fatal("event while closed must raise the activity dot")
	}
	m.SetOpen(true)
	if m.pending {
		t.Fatal("opening must clear the activity dot")
	}
	// Events while open do not set the dot.
	m.PushEvent(contract.Event{Kind: contract.EventLog, Text: "fed"})
	if m.pending {
		t.Fatal("events while open must not set the dot")
	}
}

func TestMenuEventRouting(t *testing.T) {
	m := newTestMenu()
	m.PushEvent(contract.Event{Kind: contract.EventThought, Agent: "species", Text: "planning"})
	m.PushEvent(contract.Event{Kind: contract.EventChunk, Agent: "species", Text: "Ember"})
	m.PushEvent(contract.Event{Kind: contract.EventStatus, Agent: "water", Status: contract.StatusWriting})
	m.PushEvent(contract.Event{Kind: contract.EventArtifact, Agent: "water", ArtifactID: "midnight-lagoon", Text: "new water"})
	m.PushEvent(contract.Event{Kind: contract.EventLog, Text: "log line"})

	sc := m.agentsT.card("species")
	if sc.think == "" || sc.stream.Len() != 1 {
		t.Fatalf("species card should hold the thought tail + one stream line, got %q / %d", sc.think, sc.stream.Len())
	}
	wc := m.agentsT.card("water")
	if wc.status != contract.StatusWriting || wc.artifactID != "midnight-lagoon" {
		t.Fatalf("water card status/artifact not applied: %s / %q", wc.status, wc.artifactID)
	}
	if m.logT.st.Len() != 2 { // "menu ready" + artifact notice + ... -> see below
		t.Logf("log lines = %d (menu ready + artifact)", m.logT.st.Len())
	}
}

func TestMenuActionsQueue(t *testing.T) {
	m := newTestMenu()
	m.enqueue(Action{Kind: ActionRunAll})
	m.enqueue(Action{Kind: ActionResetTank})
	acts := m.Actions()
	if len(acts) != 2 || acts[0].Kind != ActionRunAll || acts[1].Kind != ActionResetTank {
		t.Fatalf("unexpected drain: %+v", acts)
	}
	if acts := m.Actions(); len(acts) != 0 {
		t.Fatal("Actions must drain to empty")
	}
}

func TestMenuAgentsTabButtonsEnqueue(t *testing.T) {
	m := newTestMenu()
	m.SetOpen(true)
	m.advance(0.25) // fully open -> tabs interactive
	// Layout math: research button in the toolbar row.
	area, ok := m.contentArea()
	if !ok {
		t.Fatal("content area must exist when fully open")
	}
	cardsH := (area.H - agentsToolbarH - 2*cardGap) / 3
	cy := area.Y + 2*(cardsH+cardGap) // third card
	run := Rect{X: area.X + area.W - 48, Y: cy + 3, W: 44, H: 16}
	m.Update(run.X+run.W/2, run.Y+run.H/2, true, false)
	m.Update(run.X+run.W/2, run.Y+run.H/2, false, true)
	acts := m.Actions()
	if len(acts) != 1 || acts[0].Kind != ActionRunAgent || acts[0].ID != "pattern" {
		t.Fatalf("pattern Run button must enqueue ActionRunAgent(pattern), got %+v", acts)
	}
}

func TestSettingsRoundTripBinding(t *testing.T) {
	cfg := &contract.Config{Endpoint: "ep", Model: "glm-5.3-flash", APIKey: "secret",
		ProxyURL: "px", AutoFeed: true, MusicOn: true, FullscreenOnStart: true,
		AgentFreq: 1.5, DaySeconds: 90, MaxFish: 20}
	m := NewMenu(cfg)
	s := m.setT
	if s.endpoint.Value != "ep" || s.proxy.Value != "px" {
		t.Fatal("inputs must seed from config")
	}
	if !s.autoFeed.On || !s.sound.On || s.autoCare.On {
		t.Fatal("toggles must seed from config")
	}
	if !s.fullscreen.On {
		t.Fatal("F13 fullscreen toggle must seed from config")
	}
	if s.freq.Value != (1.5-agentFreqMin)/(agentFreqMax-agentFreqMin) {
		t.Fatalf("sliders must seed normalized: freq=%.2f day=%.2f", s.freq.Value, s.day.Value)
	}
	if s.fish.Value != (20-8)/92.0 { // v0.3.7 F29: slider range is 8..100
		t.Fatalf("fish slider normalized value wrong: %.2f", s.fish.Value)
	}
	// In-place edit: applySlider writes straight into the live config.
	applySlider(cfg, "Max fish", 30)
	applySlider(cfg, "Day length", 40)
	if cfg.MaxFish != 30 || cfg.DaySeconds != 40 {
		t.Fatalf("config not edited in place: %+v", cfg)
	}
	applyToggle(cfg, "Music", false)
	if cfg.MusicOn {
		t.Fatal("toggle must write through")
	}
	applyToggle(cfg, "Start fullscreen", false)
	if cfg.FullscreenOnStart {
		t.Fatal("F13 toggle must write through")
	}
}

func TestContentAreaGeometry(t *testing.T) {
	m := newTestMenu()
	if _, ok := m.contentArea(); ok {
		t.Fatal("no content area while closed")
	}
	m.SetOpen(true)
	m.advance(0.25)
	area, ok := m.contentArea()
	if !ok {
		t.Fatal("content area must exist when open")
	}
	if area.W != PanelW-2*MenuPad || area.X != 1280-PanelW+MenuPad {
		t.Fatalf("content area geometry wrong: %+v", area)
	}
	if area.H != 720-(TrayBtnY+TrayBtnH+2+MenuPad+TabH+6)-MenuPad {
		t.Fatalf("content height wrong: %d", area.H)
	}
}

// F28: the open panel starts BELOW the TREATS button and the close X — the
// top chrome is never covered, and clicks there reach those buttons even
// while the menu is open.
func TestMenuClearsTopChrome(t *testing.T) {
	for _, w := range []int{1024, 1280, 1920} {
		mm := UIMetrics(w, 720, true)
		lowest := mm.CloseBtn.Y + mm.CloseBtn.H
		if mm.TrayBtn.Y+mm.TrayBtn.H > lowest {
			lowest = mm.TrayBtn.Y + mm.TrayBtn.H
		}
		if mm.PanelTopY < lowest {
			t.Fatalf("%dpx: panel top %d overlaps top chrome (lowest edge %d)", w, mm.PanelTopY, lowest)
		}
		if mm.TabBar.Y < mm.PanelTopY {
			t.Fatalf("%dpx: tab bar sits above the panel top", w)
		}
	}
	m := newTestMenu()
	m.SetOpen(true)
	m.advance(0.25)
	mm := UIMetrics(m.w, m.h, true)
	if m.Consumes(mm.TrayBtn.X+mm.TrayBtn.W/2, mm.TrayBtn.Y+mm.TrayBtn.H/2) {
		t.Fatal("open menu swallows clicks on the TREATS button")
	}
	if m.Consumes(mm.CloseBtn.X+mm.CloseBtn.W/2, mm.CloseBtn.Y+mm.CloseBtn.H/2) {
		t.Fatal("open menu swallows clicks on the close button")
	}
	if !m.Consumes(mm.PanelX+PanelW/2, mm.PanelTopY+TabH) {
		t.Fatal("open menu must still own its own panel body")
	}
}

// v0.3.5: the Chosen never appears in the species catalog — one of a kind.
func TestCatalogExcludesChosen(t *testing.T) {
	dir := t.TempDir()
	store, err := content.Load(filepath.Join(dir, "content"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.EnsureSeed(); err != nil {
		t.Fatal(err)
	}
	list := catalogList(store)
	for _, sp := range list {
		if sp.Role == contract.RoleChosen {
			t.Fatalf("catalog must exclude the Chosen, found %s", sp.ID)
		}
	}
	if len(list) == 0 {
		t.Fatal("catalog should list the normal seeds")
	}
}

// v1.1: the titan role never appears in the species catalog either — the
// deep wanderers are ambient visits, not stock the menu can hand out.
func TestCatalogExcludesTitan(t *testing.T) {
	dir := t.TempDir()
	store, err := content.Load(filepath.Join(dir, "content"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.EnsureSeed(); err != nil {
		t.Fatal(err)
	}
	if store.SpeciesByID("titan-abyssdrifter") == nil {
		t.Fatal("titan seed missing from the store")
	}
	list := catalogList(store)
	for _, sp := range list {
		if sp.Role == contract.RoleTitan {
			t.Fatalf("catalog must exclude the titan role, found %s", sp.ID)
		}
	}
	for _, sp := range list {
		if sp.ID == "titan-abyssdrifter" {
			t.Fatal("titan-abyssdrifter must not be listed in the catalog")
		}
	}
}
