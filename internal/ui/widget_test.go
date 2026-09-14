// G5.1: pure widget logic tests — hit testing, button arm/fire, toggle flip,
// slider drag math, tab switching. No ebiten (no Draw calls).
package ui

import "testing"

func TestHit(t *testing.T) {
	if !Hit(10, 10, 50, 20, 10, 10) {
		t.Fatal("top-left corner should hit")
	}
	if !Hit(10, 10, 50, 20, 59, 29) {
		t.Fatal("bottom-right inner pixel should hit")
	}
	if Hit(10, 10, 50, 20, 60, 30) {
		t.Fatal("outside edge must not hit")
	}
	if Hit(10, 10, 50, 20, 9, 15) {
		t.Fatal("left of rect must not hit")
	}
	if !HitR(Rect{X: 1, Y: 2, W: 3, H: 4}, 3, 3) {
		t.Fatal("HitR should delegate to Hit")
	}
}

func TestButtonArmFire(t *testing.T) {
	var b ButtonState
	const x, y, w, h = 100, 100, 60, 24
	// Press inside, release inside -> exactly one fire.
	if b.Update(x, y, w, h, 120, 110, true, false) {
		t.Fatal("must not fire on press")
	}
	if !b.Armed {
		t.Fatal("press inside must arm")
	}
	if b.Update(x, y, w, h, 120, 110, true, false) {
		t.Fatal("repeated pressed frames must not fire")
	}
	if !b.Update(x, y, w, h, 120, 110, false, true) {
		t.Fatal("release inside must fire")
	}
	if b.Armed {
		t.Fatal("armed must clear after release")
	}
	// Release outside after press inside -> no fire.
	b.Update(x, y, w, h, 120, 110, true, false)
	if b.Update(x, y, w, h, 500, 500, false, true) {
		t.Fatal("release outside must not fire")
	}
	// Press outside, release inside -> no fire (never armed).
	b.Update(x, y, w, h, 0, 0, true, false)
	if b.Update(x, y, w, h, 120, 110, false, true) {
		t.Fatal("press outside then release inside must not fire")
	}
}

func TestToggleFlip(t *testing.T) {
	var tg ToggleState
	const x, y, w, h = 0, 0, 200, 24
	if tg.Update(x, y, w, h, 100, 12, true, false) {
		t.Fatal("no flip on press")
	}
	if !tg.Update(x, y, w, h, 100, 12, false, true) {
		t.Fatal("flip expected on release inside")
	}
	if !tg.On {
		t.Fatal("toggle should be on after one flip")
	}
	tg.Update(x, y, w, h, 100, 12, true, false)
	tg.Update(x, y, w, h, 900, 900, false, true)
	if !tg.On {
		t.Fatal("release outside must not change state")
	}
}

func TestSliderDrag(t *testing.T) {
	var s SliderState
	const x, y, w, h = 50, 50, 200, 20
	// Fresh press inside starts a drag and snaps to the pointer.
	if !s.Update(x, y, w, h, x+w/2, y+5, true, false) {
		t.Fatal("press inside should change value (drag start)")
	}
	if s.Value <= 0.4 || s.Value >= 0.6 {
		t.Fatalf("mid-press value ~0.5 expected, got %.2f", s.Value)
	}
	// Dragging to the right end clamps to 1.
	s.Update(x, y, w, h, x+w+40, y+5, true, false)
	if s.Value != 1 {
		t.Fatalf("clamped right value expected 1, got %.2f", s.Value)
	}
	// Release ends the drag; later moves without a press do nothing.
	s.Update(x, y, w, h, x, y, false, true)
	s.Update(x, y, w, h, x+w, y, false, false)
	if s.Value != 1 {
		t.Fatalf("value must not change while released, got %.2f", s.Value)
	}
}

func TestSliderClickThroughIgnored(t *testing.T) {
	var s SliderState
	// A release (end of a click that started on a button elsewhere) must not
	// grab the slider: only a fresh press inside starts a drag.
	s.Update(0, 0, 100, 20, 50, 10, false, true)
	if s.Drag {
		t.Fatal("release alone must not start a drag")
	}
}

func TestTabBarSwitch(t *testing.T) {
	tb := TabBar{Tabs: []string{"A", "B", "C", "D", "E"}}
	const x, y, tw, th = 10, 10, 380, 26
	// Clicking the already-active tab fires no change event.
	if tb.Update(x, y, tw, th, x+40, y+5, true, false) {
		t.Fatal("no switch on press")
	}
	if tb.Update(x, y, tw, th, x+40, y+5, false, true) {
		t.Fatal("no switch when clicking the active tab")
	}
	if tb.Active != 0 {
		t.Fatalf("active must stay 0, got %d", tb.Active)
	}
	// Click tab 1 region (index 1 -> 10+76..162): press + release switches.
	if tb.Update(x, y, tw, th, x+100, y+5, true, false) {
		t.Fatal("no switch on press")
	}
	if !tb.Update(x, y, tw, th, x+100, y+5, false, true) {
		t.Fatal("switch expected on release over a new tab")
	}
	if tb.Active != 1 {
		t.Fatalf("want active tab 1, got %d", tb.Active)
	}
	// Click outside -> nothing changes.
	tb.Update(x, y, tw, th, x+100, y+200, true, false)
	tb.Update(x, y, tw, th, x+100, y+200, false, true)
	if tb.Active != 1 {
		t.Fatalf("click outside must not switch; got %d", tb.Active)
	}
}
