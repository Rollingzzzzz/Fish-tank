// G5.1: widget gallery for the demo — exercises every ui widget (Panel,
// Button, Toggle, Slider, TabBar, TextInput normal + masked, ScrollText) with
// real input, so text input, hover and click behavior can be tried live.
package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/Rollingzzzzz/Fish-tank/internal/ui"
)

type gallery struct {
	btn    ui.ButtonState
	tgl    ui.ToggleState
	sld    ui.SliderState
	tabs   ui.TabBar
	inA    ui.TextInput
	inB    ui.TextInput
	stream *ui.ScrollText
	clicks int
	words  int
}

func newGallery() *gallery {
	g := &gallery{stream: ui.NewScrollText(120)}
	g.tgl.On = true
	g.sld.Value = 0.4
	g.tabs.Tabs = []string{"Alpha", "Beta", "Gamma"}
	g.inA.Placeholder = "type here..."
	g.inB.Placeholder = "masked key"
	g.inB.Masked = true
	return g
}

// update advances all gallery widgets with real input.
func (g *gallery) update(mx, my int, pressed, released bool, wheelY, dt float64) {
	if g.btn.Update(40, 100, 120, 26, mx, my, pressed, released) {
		g.clicks++
	}
	g.tgl.Update(246, 96, 86, 30, mx, my, pressed, released)
	g.sld.Update(40, 150, 240, 20, mx, my, pressed, released)
	g.tabs.Update(36, 200, 250, 26, mx, my, pressed, released)
	g.inA.Update(36, 240, 250, 24, mx, my, pressed, released, dt)
	g.inB.Update(36, 300, 250, 24, mx, my, pressed, released, dt)
	if wheelY != 0 {
		g.stream.ScrollWheel(int(wheelY) * 2)
	}
}

// feed streams demo text into the ScrollText so the capture shows live
// streaming (word-wrapped, dimmed thought lines, auto-scroll).
func (g *gallery) feed(frame int) {
	words := []string{"neon", "glow", "drift", "school", "ripple", "spark", "plankton", "caustic"}
	if frame%6 == 0 && g.words < 60 {
		w := words[g.words%len(words)]
		g.stream.AppendChunk(ui.KindText, w+" ")
		g.words++
		if g.words%9 == 0 {
			g.stream.AppendChunk(ui.KindText, "\n")
		}
	}
	if frame == 30 || frame == 60 {
		g.stream.AppendLine(ui.KindThought,
			"reasoning: balance school density against food drift...")
	}
	if frame == 45 || frame == 75 {
		g.stream.AppendLine(ui.KindLog,
			fmt.Sprintf("gallery log at frame %d", frame))
	}
}

// draw renders the gallery panel.
func (g *gallery) draw(dst *ebiten.Image) {
	ui.GlassPanel(dst, 20, 20, 320, 660)
	ui.DrawText(dst, "WIDGET KIT GALLERY", 36, 34, 2, ui.ColAccent, 1)
	ui.DrawText(dst, "G5.1 acceptance: every widget, live", 36, 54, 1, ui.ColDim, 0.9)

	ui.DrawButton(dst, &g.btn, 40, 96, 120, 26, "Press me", ui.ColAccent2)
	ui.DrawText(dst, fmt.Sprintf("clicks: %d", g.clicks), 170, 104, 1, ui.ColDim, 1)
	ui.DrawToggle(dst, &g.tgl, 246, 100, 86, 24, "Glow")
	ui.DrawText(dst, "slider:", 40, 138, 1, ui.ColDim, 1)
	ui.DrawSlider(dst, &g.sld, 40, 152, 240, 20)
	ui.DrawText(dst, "tabs:", 40, 186, 1, ui.ColDim, 1)
	g.tabs.DrawTabBar(dst, 36, 200, 250, 26)

	ui.DrawText(dst, "text input:", 40, 228, 1, ui.ColDim, 1)
	g.inA.Draw(dst, 36, 240, 250, 24)
	ui.DrawText(dst, "masked input:", 40, 288, 1, ui.ColDim, 1)
	g.inB.Draw(dst, 36, 300, 250, 24)

	ui.DrawText(dst, "streaming text:", 40, 340, 1, ui.ColDim, 1)
	g.stream.Draw(dst, 36, 354, 250, 300, 2)
	ui.DrawText(dst, "left of the tank: gallery | right: slide-in menu",
		36, 660, 1, ui.ColDim, 0.7)
}
