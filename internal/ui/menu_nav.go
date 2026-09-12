// v0.2: menu navigation helpers split from menu.go (line ceiling).
package ui

// Wheel routes a mouse-wheel delta to the active tab (extra API; the game
// forwards ebiten wheel deltas).
func (m *Menu) Wheel(dy int, mx, my int) {
	area, ok := m.contentArea()
	if !ok || !HitR(area, mx, my) {
		return
	}
	switch m.tabbar.Active {
	case 1:
		m.specT.wheel(dy)
	case 2:
		m.plantT.wheel(dy)
	case 3:
		m.logT.wheel(dy)
	case 4:
		m.setT.wheel(dy)
	}
}

// Consumes reports whether the point sits over the handle (or its MENU
// chip, F12) or the slid-in panel — clicks there belong to the UI, never to
// the tank (F5 click hygiene). The caller feeds the world only when this is
// false.
func (m *Menu) Consumes(mx, my int) bool {
	p := m.progress()
	hr := m.handleRect(p)
	if HitR(hr, mx, my) || HitR(m.handleChipRect(hr), mx, my) {
		return true
	}
	if p > 0.5 && mx >= m.panelX(p) && my >= m.panelTop() { // F28: TREATS/X stay clickable above the panel
		return true
	}
	return false
}

// ActiveTab returns the index of the active tab (0..4) — used by the
// -menu-smoke acceptance run.
func (m *Menu) ActiveTab() int { return m.tabbar.Active }

// Slide exposes the raw slide progress 0..1 for diagnostics.
func (m *Menu) Slide() float64 { return m.slide }

// SettingsRowRect returns the on-screen rect of a named settings row inside
// the given content area (F19: the evidence manifests gate that key toggles
// — Auto feed above all — are actually visible in the captured PNGs).
func (m *Menu) SettingsRowRect(area Rect, row string) (Rect, bool) {
	return m.setT.rowRect(area, row)
}
