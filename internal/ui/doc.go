// Package ui implements the NEON TANK neon-glass widget kit and the right
// slide-in menu (G5.1-G5.3).
//
// Purpose: immediate-mode, hit-tested Ebiten widgets (Panel, Button, Toggle,
// Slider, TabBar, TextInput, ScrollText) plus the 380 px slide-in menu with
// the Agents/Species/Plants/Log/Settings tabs. The demo lives in
// cmd/demo-ui.
//
// Owns: internal/ui/*.go (this package only) — theme, bitmap font, widgets,
// menu, tab screens. Demo binary: cmd/demo-ui.
//
// Public API: theme drawing helpers (GlassPanel, fill/glow), DrawText family,
// Hit, ButtonState, ToggleState, SliderState, TabBar, TextInput, ScrollText,
// Menu (NewMenu/Update/Draw/Open/SetOpen/SetContentStore/PushEvent/Actions),
// Action + ActionKind (consumed by the game, G6.1).
//
// Invariants: hit-testing and buffer logic are pure (unit-tested without
// ebiten); when closed the menu draws ONLY the 8 px handle; no emoji anywhere
// (D6, bitmap font has no emoji); ScrollText caps at 500 lines, the tank log
// view at 200 (C3); atlases/caches are bounded and built lazily so tests and
// NewMenu never touch the GPU.
//
// Extension points: a new tab screen = a new tab_*.go file holding its own
// state struct, wired in menu.go; new widgets = new file, pure Update + a
// Draw method. Art direction §3 applies (glow accents, layered glass).
package ui
