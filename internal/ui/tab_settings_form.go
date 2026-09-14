// v0.3: settings form plumbing — input/toggle/slider/button lookups, the
// update/draw walkers, config write-through helpers, and the F19 manifest
// row-rect walker (split from tab_settings.go, line ceiling).
package ui

import (
	"fmt"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/hajimehoshi/ebiten/v2"
)

// inputByName resolves an input row key.
func (s *settingsTab) inputByName(k string) *TextInput {
	switch k {
	case "endpoint":
		return &s.endpoint
	case "key":
		return &s.apiKey
	default:
		return &s.proxy
	}
}

// update walks rows, mutating cfg live and queueing button actions on m.
func (s *settingsTab) update(area Rect, mx, my int, pressed, released bool, m *Menu) {
	cfg := m.cfg
	y := area.Y
	for _, r := range s.layout() {
		rect := Rect{X: area.X, Y: y, W: area.W, H: r.h}
		switch r.kind {
		case rInput:
			ti := s.inputByName(r.text)
			ti.Update(rect.X, rect.Y, rect.W, rect.H, mx, my, pressed, released, m.lastDt)
			s.commitInput(cfg, r.text, ti.Value)
		case rToggle:
			t := s.toggleByName(r.text)
			if t.Update(rect.X, rect.Y, rect.W, rect.H, mx, my, pressed, released) {
				applyToggle(cfg, r.text, t.On)
				if r.text == "Music" {
					m.enqueue(Action{Kind: ActionToggleSound, Flag: t.On})
				}
			}
		case rSlider:
			sl, min, max := s.sliderByName(r.text), 0.0, 1.0
			switch r.text {
			case "Agent frequency":
				min, max = agentFreqMin, agentFreqMax
			case "Day length":
				min, max = daySecMin, daySecMax
			default:
				min, max = maxFishMin, maxFishMax
			}
			if sl.Update(rect.X, rect.Y, rect.W, rect.H, mx, my, pressed, released) {
				applySlider(cfg, r.text, min+(max-min)*sl.Value)
			}
		case rButton:
			b, act := s.buttonByName(r.text), Action{}
			switch r.text {
			case "Import key from ZCode":
				act = Action{Kind: ActionImportKey}
			case "Export Pack":
				act = Action{Kind: ActionExportPack}
			case "Import Pack":
				act = Action{Kind: ActionImportPack}
			default:
				act = Action{Kind: ActionResetTank}
			}
			if b.Update(rect.X, rect.Y, rect.W, rect.H, mx, my, pressed, released) {
				m.enqueue(act)
			}
		}
		y += r.h
	}
}

func (s *settingsTab) commitInput(cfg *contract.Config, key, val string) {
	switch key {
	case "endpoint":
		cfg.Endpoint = val
	case "key":
		cfg.APIKey = val
	default:
		cfg.ProxyURL = val
	}
}

func applyToggle(cfg *contract.Config, label string, on bool) {
	switch label {
	case "Auto feed":
		cfg.AutoFeed = on
	case "Auto care":
		cfg.AutoCare = on
	case "Start fullscreen":
		cfg.FullscreenOnStart = on
	default:
		cfg.MusicOn = on
	}
}

func applySlider(cfg *contract.Config, label string, v float64) {
	switch label {
	case "Agent frequency":
		cfg.AgentFreq = v
	case "Day length":
		cfg.DaySeconds = v
	default:
		cfg.MaxFish = int(v + 0.5)
	}
}

// draw renders the form (absolute coords — SubImage keeps the parent space).
func (s *settingsTab) draw(dst *ebiten.Image, m *Menu, area Rect) {
	cfg := m.cfg
	clip, ok := subImage(dst, imageRect(area))
	if !ok {
		return
	}
	y := area.Y
	for _, r := range s.layout() {
		rect := Rect{X: area.X, Y: y, W: area.W, H: r.h}
		switch r.kind {
		case rCaption:
			DrawText(clip, r.text, rect.X+2, rect.Y, 1, ColDim, 0.9)
		case rInput:
			s.inputByName(r.text).Draw(clip, rect.X, rect.Y, rect.W, rect.H)
		case rModel:
			FillRect(clip, rect.X, rect.Y, rect.W, rect.H, ColPanel, 0.6)
			FrameRect(clip, rect.X, rect.Y, rect.W, rect.H, ColDim, 0.3)
			DrawText(clip, cfg.Model+"  (read-only)", rect.X+6,
				rect.Y+(rect.H-LineHeight(2))/2+1, 2, ColDim, 1)
		case rToggle:
			DrawToggle(clip, s.toggleByName(r.text), rect.X, rect.Y, rect.W, rect.H, r.text)
		case rSlider:
			DrawText(clip, r.text+": "+sliderCaption(cfg, r.text), rect.X+2, rect.Y, 1, ColText, 1)
			DrawSlider(clip, s.sliderByName(r.text), rect.X+4, rect.Y+10, rect.W-8, rect.H-10)
		case rButton:
			hex := ColAccent
			if r.text == "Reset Tank" {
				hex = ColErr
			}
			DrawButton(clip, s.buttonByName(r.text), rect.X, rect.Y, rect.W, rect.H, r.text, hex)
		}
		y += r.h
	}
	DrawText(clip, "changes apply live; the game persists config.json", area.X+2, y+2, 1, ColDim, 0.6)
}

func (s *settingsTab) toggleByName(name string) *ToggleState {
	switch name {
	case "Auto feed":
		return &s.autoFeed
	case "Auto care":
		return &s.autoCare
	case "Start fullscreen":
		return &s.fullscreen
	default:
		return &s.sound
	}
}

func (s *settingsTab) sliderByName(name string) *SliderState {
	switch name {
	case "Agent frequency":
		return &s.freq
	case "Day length":
		return &s.day
	default:
		return &s.fish
	}
}

func (s *settingsTab) buttonByName(name string) *ButtonState {
	switch name {
	case "Import key from ZCode":
		return &s.impKey
	case "Export Pack":
		return &s.exportB
	case "Import Pack":
		return &s.importB
	default:
		return &s.resetB
	}
}

func sliderCaption(cfg *contract.Config, name string) string {
	switch name {
	case "Agent frequency":
		return fmt.Sprintf("%.2fx", cfg.AgentFreq)
	case "Day length":
		return fmt.Sprintf("%d s", int(cfg.DaySeconds+0.5))
	default:
		return fmt.Sprintf("%d", cfg.MaxFish)
	}
}

// rowRect walks the form layout and returns the rect of the named row inside
// the content area (F19: the evidence manifests gate that key controls — the
// Auto feed toggle above all — are actually visible on screen).
func (s *settingsTab) rowRect(area Rect, text string) (Rect, bool) {
	y := area.Y
	for _, r := range s.layout() {
		if r.text == text {
			return Rect{X: area.X, Y: y, W: area.W, H: r.h}, true
		}
		y += r.h
	}
	return Rect{}, false
}
