// F13: logical-canvas sizing rules — height-locked, aspect-proportional,
// clamped to the safe band so no display gets an unusable or sparse tank.
package game

import (
	"testing"

	"github.com/Rollingzzzzz/Fish-tank/internal/ui"
)

func TestLogicalWidthFor(t *testing.T) {
	cases := []struct {
		name         string
		mw, mh, want int
		noMonitor    bool
	}{
		{name: "16:9 full-hd", mw: 1920, mh: 1080, want: 1280},
		{name: "16:9 4k", mw: 3840, mh: 2160, want: 1280},
		{name: "16:10", mw: 1920, mh: 1200, want: 1152},
		{name: "3:2", mw: 2160, mh: 1440, want: 1080},
		{name: "4:3 clamps up to floor", mw: 1024, mh: 768, want: contract_LogicalWMin},
		{name: "5:4 clamps up to floor", mw: 1280, mh: 1024, want: contract_LogicalWMin},
		{name: "super-ultrawide clamps to ceiling", mw: 5120, mh: 1440, want: contract_LogicalWMax},
		{name: "no monitor falls back", mw: 0, mh: 0, want: 1280, noMonitor: true},
	}
	for _, c := range cases {
		if got := LogicalWidthFor(c.mw, c.mh); got != c.want {
			t.Errorf("%s: LogicalWidthFor(%d,%d) = %d, want %d", c.name, c.mw, c.mh, got, c.want)
		}
	}
}

// aliased so the table stays literal if the contract band ever changes
const contract_LogicalWMin = 1024
const contract_LogicalWMax = 1920

func TestSetLogicalSize(t *testing.T) {
	oldW, oldH := ScreenW, ScreenH
	defer func() { ScreenW, ScreenH = oldW, oldH }()
	SetLogicalSize(1600, 720)
	if ScreenW != 1600 || ScreenH != 720 {
		t.Fatalf("SetLogicalSize(1600,720) got %dx%d", ScreenW, ScreenH)
	}
	SetLogicalSize(-5, 0) // junk input must not corrupt the canvas
	if ScreenW != 1600 || ScreenH != 720 {
		t.Fatalf("SetLogicalSize(-5,0) corrupted canvas: %dx%d", ScreenW, ScreenH)
	}
}

// v0.3.3: the quit button sits top-right, clear of the tray; the HUD moved
// top-left so the menu panel never tangles with it.
func TestCloseButtonGeometry(t *testing.T) {
	m := ui.UIMetrics(1280, 720, false)
	if m.CloseBtn.Y+m.CloseBtn.H > m.TrayBtn.Y {
		t.Fatalf("close button overlaps the tray: %+v vs %+v", m.CloseBtn, m.TrayBtn)
	}
	if m.CloseBtn.X+m.CloseBtn.W > 1280 || m.CloseBtn.X < 1280-60 {
		t.Fatalf("close button not pinned top-right: %+v", m.CloseBtn)
	}
	if m.CloseBtn.H < 20 || m.CloseBtn.W < 20 {
		t.Fatalf("close button too small to click: %+v", m.CloseBtn)
	}
}
