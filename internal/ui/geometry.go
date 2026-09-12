// F12/F19: shared UI geometry — the single source of truth for where every
// interactive control lives on the logical canvas. The menu, the treat tray,
// the -menu-smoke script and the -shot evidence manifests all derive from
// here, so tests and screenshots can never drift from what the player sees.
package ui

// Layout constants (F12 restyle: bigger, higher-contrast controls).
const (
	PanelW   = 480 // open panel width (px)
	HandleW  = 16  // visible handle width when closed (px)
	HandleH  = 110
	ChipW    = 58 // "MENU" label chip beside the handle
	ChipH    = 20
	TabH     = 34
	SlideSec = 0.25 // open/close animation time
	MenuPad  = 10

	// Treat tray (top-right, floats above the open menu).
	TrayBtnW   = 80
	TrayBtnH   = 30
	TrayBtnY   = 40
	TrayPanelW = 150
	TrayPanelH = 166
	TrayRowH   = 32
	TrayRowGap = 6 // pitch = TrayRowH + TrayRowGap
)

// Metrics is a snapshot of control rects for a w×h canvas.
type Metrics struct {
	W, H       int
	PanelX     int    // open panel left edge
	PanelTopY  int    // v0.3.7 (F28): open panel top edge — clears TREATS and X
	Handle     Rect   // menu grip (open or closed position per `open`)
	HandleChip Rect   // "MENU" chip beside the grip (part of the hit area)
	TabBar     Rect   // full tab strip
	Tabs       []Rect // one cell per tab
	Content    Rect   // tab content area
	TrayBtn    Rect
	TrayPanel  Rect
	TrayRows   []Rect
	CloseBtn   Rect // v0.3.3: always-visible quit button, top-right
}

// UIMetrics computes the layout for a w×h canvas. open places the handle at
// the panel edge (true) or the screen edge (false); the tab/content/tray
// fields are identical either way.
func UIMetrics(w, h int, open bool) *Metrics {
	m := &Metrics{W: w, H: h}
	m.PanelX = w - PanelW
	m.PanelTopY = TrayBtnY + TrayBtnH + 2 // F28: same anchor idiom as TrayPanel
	px := w
	if open {
		px = m.PanelX
	}
	m.Handle = Rect{X: px - HandleW, Y: h/2 - HandleH/2, W: HandleW, H: HandleH}
	m.HandleChip = Rect{X: m.Handle.X - ChipW - 4, Y: m.Handle.Y + m.Handle.H + 6, W: ChipW, H: ChipH}
	m.TabBar = Rect{X: m.PanelX + MenuPad, Y: m.PanelTopY + MenuPad, W: PanelW - 2*MenuPad, H: TabH}
	n := 5
	cw := m.TabBar.W / n
	for i := 0; i < n; i++ {
		m.Tabs = append(m.Tabs, Rect{X: m.TabBar.X + i*cw, Y: m.TabBar.Y, W: cw, H: TabH})
	}
	m.Content = Rect{
		X: m.PanelX + MenuPad,
		Y: m.PanelTopY + MenuPad + TabH + 6,
		W: PanelW - 2*MenuPad,
		H: h - (m.PanelTopY + MenuPad + TabH + 6) - MenuPad,
	}
	m.CloseBtn = Rect{X: w - 40, Y: 8, W: 32, H: 26}
	m.TrayBtn = Rect{X: w - TrayBtnW - 12, Y: TrayBtnY, W: TrayBtnW, H: TrayBtnH}
	m.TrayPanel = Rect{X: w - 12 - TrayPanelW, Y: TrayBtnY + TrayBtnH + 2, W: TrayPanelW, H: TrayPanelH}
	for i := 0; i < 4; i++ {
		m.TrayRows = append(m.TrayRows, Rect{
			X: m.TrayPanel.X + 8, Y: m.TrayPanel.Y + 8 + i*(TrayRowH+TrayRowGap),
			W: m.TrayPanel.W - 16, H: TrayRowH,
		})
	}
	return m
}
