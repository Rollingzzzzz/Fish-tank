// G95: the REC button shares the X's top row without stepping on it and
// stays on the canvas even at the narrowest legal logical width.
package ui

import "testing"

func TestRecButtonClearsTheChrome(t *testing.T) {
	for _, open := range []bool{false, true} {
		m := UIMetrics(1280, 720, open)
		if m.RecBtn.Y != m.CloseBtn.Y || m.RecBtn.H != m.CloseBtn.H {
			t.Fatalf("RecBtn row (y=%d h=%d) differs from the X row (y=%d h=%d)",
				m.RecBtn.Y, m.RecBtn.H, m.CloseBtn.Y, m.CloseBtn.H)
		}
		if m.RecBtn.X+m.RecBtn.W >= m.CloseBtn.X {
			t.Fatalf("RecBtn (%+v) overlaps the X (%+v)", m.RecBtn, m.CloseBtn)
		}
		if m.RecBtn.X < 0 || m.RecBtn.X+m.RecBtn.W > m.W {
			t.Fatalf("RecBtn outside the canvas: %+v", m.RecBtn)
		}
		mn := UIMetrics(1024, 720, open) // the narrowest legal canvas
		if mn.RecBtn.X < 0 {
			t.Fatalf("RecBtn clipped at min width: %+v", mn.RecBtn)
		}
	}
}
