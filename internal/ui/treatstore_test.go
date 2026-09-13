// N8: Treat Store tray state-machine tests — pure, no ebiten/Draw, mirroring
// the menu test idiom. Geometry comes from the unexported rect helpers.
package ui

import "testing"

func trayCenter(r Rect) (int, int) { return r.X + r.W/2, r.Y + r.H/2 }

func TestTreatTrayButtonToggles(t *testing.T) {
	tr := NewTreatTray()
	if tr.Open {
		t.Fatal("tray starts closed")
	}
	bx, by := trayCenter(tr.buttonRect())
	tr.Update(bx, by, true, false) // press inside
	tr.Update(bx, by, true, true)  // release inside
	if !tr.Open {
		t.Fatal("completed click must open the tray")
	}
	// press inside, release outside: arm aborts, state must not flip
	tr.Update(bx, by, true, false)
	tr.Update(100, 300, true, true)
	if !tr.Open {
		t.Fatal("release outside must keep the tray open")
	}
	// button-up between press and release aborts the arm; the NEXT completed
	// click toggles afresh (closing the tray here)
	tr.Update(bx, by, true, false)
	tr.Update(bx, by, false, false)
	tr.Update(bx, by, true, true)
	if tr.Open {
		t.Fatal("fresh click after an aborted arm must toggle closed")
	}
}

func TestTreatTrayGrabRow(t *testing.T) {
	tr := NewTreatTray()
	bx, by := trayCenter(tr.buttonRect())
	tr.Update(bx, by, true, false)
	tr.Update(bx, by, false, true)      // open (release frame = level up)
	rx, ry := trayCenter(tr.rowRect(1)) // row 1 = worm
	// F26: the FIRST press inside a row selects the treat — no release needed
	if got := tr.Update(rx, ry, true, false); got != "grab:worm" {
		t.Fatalf("row press = %q, want %q", got, "grab:worm")
	}
	if tr.Open {
		t.Fatal("tray must close after a grab")
	}
	// the release of that same press must not grab again / do anything
	if got := tr.Update(rx, ry, false, true); got != "" {
		t.Fatalf("release after grab must be inert, got %q", got)
	}
	// a held button that was down before the panel opened must not refire on
	// repeat level frames (press edge only)
	tr2 := NewTreatTray()
	tr2.Update(bx, by, true, false)
	tr2.Update(bx, by, true, true) // open while button still down (level high)
	if got := tr2.Update(rx, ry, true, false); got != "" {
		t.Fatalf("stale level press must not grab, got %q", got)
	}
	tr2.Update(rx, ry, false, false) // button up
	if got := tr2.Update(rx, ry, true, false); got != "grab:worm" {
		t.Fatalf("fresh press = %q, want grab:worm", got)
	}
	// grabbing from a closed tray yields nothing
	tr3 := NewTreatTray()
	if got := tr3.Update(rx, ry, true, false); got != "" {
		t.Fatalf("closed tray must not grab, got %q", got)
	}
}

func TestTreatTrayHit(t *testing.T) {
	tr := NewTreatTray()
	bx, by := trayCenter(tr.buttonRect())
	if !tr.Hit(bx, by) {
		t.Fatal("button rect must hit")
	}
	if tr.Hit(100, 300) {
		t.Fatal("(100,300) is tank water, not tray")
	}
	px, py := trayCenter(tr.panelRect())
	if tr.Hit(px, py) {
		t.Fatal("panel must not hit while closed")
	}
	b2x, b2y := trayCenter(tr.buttonRect())
	tr.Update(b2x, b2y, true, false)
	tr.Update(b2x, b2y, true, true) // open
	if !tr.Hit(px, py) {
		t.Fatal("open panel must hit")
	}
	if !tr.Hit(tr.panelRect().X, tr.panelRect().Y) {
		t.Fatal("panel top-left corner must hit")
	}
}
