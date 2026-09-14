package ui

import (
	"testing"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

func TestMenuPhysicalClickOpens(t *testing.T) {
	cfg := &contract.Config{Model: "glm-5.3-flash"}
	m := NewMenu(cfg)
	// a physical click: ~6 frames held, then release
	for i := 0; i < 6; i++ {
		m.Update(1276, 360, true, false)
	}
	m.Update(1276, 360, false, true)
	if !m.Open() {
		t.Fatal("physical click did not open the menu")
	}
}

// F12: the MENU chip beside the handle is part of the click target — the
// closed menu must be discoverable, not just the grip strip.
func TestMenuChipClickOpens(t *testing.T) {
	cfg := &contract.Config{Model: "glm-5.3-flash"}
	m := NewMenu(cfg)
	ch := UIMetrics(1280, 720, false).HandleChip
	cx, cy := ch.X+ch.W/2, ch.Y+ch.H/2
	for i := 0; i < 4; i++ {
		m.Update(cx, cy, true, false)
	}
	m.Update(cx, cy, false, true)
	if !m.Open() {
		t.Fatal("MENU chip click did not open the menu")
	}
}
