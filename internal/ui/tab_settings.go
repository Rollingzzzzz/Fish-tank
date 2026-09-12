// G5.3: Settings tab — form bound IN PLACE to *contract.Config: endpoint,
// model (READ-ONLY per C6), masked API key, proxy, autoFeed/autoCare/musicOn
// toggles, agentFreq (0.5-2) / daySeconds (20-180) / maxFish (8-100) sliders,
// plus Import key from ZCode / Export Pack / Import Pack / Reset Tank buttons
// (queued as Actions for the game, G6.1). The key is never logged (D9).
package ui

import (
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// Slider ranges from the frozen contract.Config doc comments (README §4).
const (
	agentFreqMin, agentFreqMax = 0.5, 2.0
	daySecMin, daySecMax       = 20.0, 180.0
	maxFishMin, maxFishMax     = 8, 100 // v0.3.7 F29: the player may stock up to 100
	settingsRowH               = 30
)

type rowKind uint8

const (
	rCaption rowKind = iota
	rInput
	rModel
	rToggle
	rSlider
	rButton
	rGap
)

type rowSpec struct {
	kind rowKind
	text string // caption text / widget key / button label
	h    int
}

type settingsTab struct {
	endpoint TextInput
	apiKey   TextInput
	proxy    TextInput

	autoFeed   ToggleState
	autoCare   ToggleState
	sound      ToggleState
	fullscreen ToggleState // F13: start borderless-fullscreen (takes effect next launch)

	freq, day, fish SliderState

	impKey, exportB, importB, resetB ButtonState

	rows []rowSpec
}

func newSettingsTab() *settingsTab {
	s := &settingsTab{}
	s.endpoint.Placeholder = "https://api.z.ai/.."
	s.endpoint.MaxLen = 200
	s.apiKey.Placeholder = "empty = simulate mode"
	s.apiKey.MaxLen = 200
	s.apiKey.Masked = true
	s.proxy.Placeholder = "optional http proxy"
	s.proxy.MaxLen = 200
	return s
}

// bind seeds the inputs and sliders from the live config (NewMenu time).
func (s *settingsTab) bind(cfg *contract.Config) {
	s.endpoint.Set(cfg.Endpoint)
	s.apiKey.Set(cfg.APIKey)
	s.proxy.Set(cfg.ProxyURL)
	s.autoFeed.On = cfg.AutoFeed
	s.autoCare.On = cfg.AutoCare
	s.sound.On = cfg.MusicOn
	s.fullscreen.On = cfg.FullscreenOnStart
	s.freq.Value = (cfg.AgentFreq - agentFreqMin) / (agentFreqMax - agentFreqMin)
	s.day.Value = (cfg.DaySeconds - daySecMin) / (daySecMax - daySecMin)
	s.fish.Value = float64(cfg.MaxFish-maxFishMin) / float64(maxFishMax-maxFishMin)
	s.rows = nil // captions show live values; force relayout on bind
}

// layout builds the row list (built once; heights are static).
func (s *settingsTab) layout() []rowSpec {
	if s.rows != nil {
		return s.rows
	}
	capRow := func(t string) rowSpec { return rowSpec{kind: rCaption, text: t, h: 12} }
	btnRow := func(t string) rowSpec { return rowSpec{kind: rButton, text: t, h: 26} }
	s.rows = []rowSpec{
		capRow("ENDPOINT"),
		{kind: rInput, text: "endpoint", h: 24},
		capRow("MODEL (FROZEN PER C6, READ-ONLY)"),
		{kind: rModel, h: 24},
		capRow("API KEY (MASKED, NEVER LOGGED)"),
		{kind: rInput, text: "key", h: 24},
		capRow("PROXY URL"),
		{kind: rInput, text: "proxy", h: 24},
		{kind: rGap, h: 8},
		{kind: rToggle, text: "Auto feed", h: 24},
		{kind: rToggle, text: "Auto care", h: 24},
		{kind: rToggle, text: "Music", h: 24},
		{kind: rToggle, text: "Start fullscreen", h: 24},
		{kind: rGap, h: 4},
		{kind: rSlider, text: "Agent frequency", h: settingsRowH},
		{kind: rSlider, text: "Day length", h: settingsRowH},
		{kind: rSlider, text: "Max fish", h: settingsRowH},
		{kind: rGap, h: 10},
		btnRow("Import key from ZCode"),
		btnRow("Export Pack"),
		btnRow("Import Pack"),
		btnRow("Reset Tank"),
	}
	return s.rows
}

// wheel is a no-op: the form fits the 720 px design height; smaller windows
// clip the bottom (documented in docs/NOTES-laneC.md).
func (s *settingsTab) wheel(int) {}
