# NOTES — Lane C (UI): G5.1 / G5.2 / G5.3

Running decision log for the ui lane, in the `docs/NOTES.md` block format (C3).
Append-only. See also the package header `internal/ui/doc.go`.

<!-- block template:
### Gx.y — <title>
- What:
- Why:
- Gotchas:
- Follow-ups:
-->

### G5.1 — Widget kit (theme, bitmap font, widgets, text input, scroll text)
- What: `internal/ui/{doc,theme,font,widget,textinput,scrolltext}.go` — neon-glass
  theme (panel rgba(8,12,32,0.82), cyan #00ffe1 border rgba(0,255,230,0.35),
  magenta #ff3df5, text #dfe8ff, dim #7a86b8), immediate-mode Panel / Button /
  Toggle / Slider / TabBar / TextInput (masked mode) / ScrollText (word-wrapped,
  auto-scroll, dimmed thought lines, 500-line cap), plus `cmd/demo-ui` with
  `--png <out>` (renders ~90 frames, dumps the last one, exits) and `--closed`
  (proves the handle-only frame). Frames committed: `docs/art/demo-ui-open.png`,
  `docs/art/demo-ui-closed.png`.
- Why: G5.1 acceptance; all hit-testing/state transitions are pure methods
  (unit-tested without ebiten), Draw methods only render — keeps the kit
  headless-testable and allocation-disciplined (C3).
- Gotchas:
  - **Font decision:** `golang.org/x/image` is in ebiten's module graph but NOT
    in this repo's `go.mod`/`go.sum`, and Lane C may not run `go get` — so there
    is no TTF/basicfont available. Implemented a built-in 5x7 bitmap font
    (`font.go`, ~90 glyphs: A-Z a-z 0-9 punctuation + Turkish ğığŞçöüİÖÜ and a
    fallback box for any other rune). Glyphs are baked into ONE white atlas on
    first Draw (never at package init, so tests stay GPU-free) and tinted per
    draw via `ColorScale` — color-independent, zero per-frame allocation.
  - Ebiten `ColorScale` operates on premultiplied alpha: to draw straight color
    (r,g,b,a) from an opaque white sprite, scale by (r*a, g*a, b*a, a).
  - `ebiten.SubImage` keeps the PARENT coordinate system — tab views draw with
    absolute coordinates into the clipped panel; ebiten also caches sub-images
    internally, so per-frame clipping does not allocate.
  - Ebiten has no clipboard API in the standard build: TextInput has no
    Ctrl+V paste in v0.1 (backspace hold-repeat, caret blink, masked mode and
    OS-layout runes are in).
  - PNG dump uses one bulk `ReadPixels` (per-pixel `At()` would do a GPU
    readback per call).
- Follow-ups: none for G5.1.

### G5.2 — Slide-in menu (immersion rule)
- What: `internal/ui/menu.go` (+ `action.go`) — right-edge handle 8 px wide
  (the ONLY thing drawn when closed, chevron + pulsing magenta dot when agent
  events arrived while closed), 0.25 s easeOutCubic slide to a 380 px panel,
  tabs Agents/Species/Plants/Log/Settings via the shared TabBar. API exactly as
  specified: `NewMenu/Update/Draw/Open/SetContentStore/PushEvent/Actions`, plus
  `SetOpen` (demo/game convenience) and `Wheel(dy, mx, my)` (game forwards
  wheel deltas to the active tab). Actions queue drained by the game (G6.1).
- Why: G5.2 — the tank stays pixel-pure when closed; menu never shifts it.
- Gotchas:
  - The specified `Update(mx, my, clicked, released)` carries no dt, so the
    slide/clock advance on wall-clock time inside Update (`time.Since`); tests
    drive `advance(dt)` directly for determinism. Input convention everywhere:
    `clicked` = left button held this frame, `released` = released this frame;
    pressables arm on press-inside and fire on release-inside (no click-through).
  - Tabs only receive input when the slide is >= 0.995; content is clipped to
    the panel via a SubImage so the animation never bleeds onto the tank.
  - `NewMenu` never touches the GPU (safe in tests); the font atlas, white
    sprite and glow sprite are lazily built on first Draw.
- Follow-ups: none for G5.2.

### G5.3 — Tab screens
- What: `internal/ui/tab_{agents,catalog,plants,log,settings}.go` — Agents: 3
  live cards (status badge colors idle/thinking/writing/done/error/simulated =
  gray/amber/cyan/green/red/purple, dimmed thinking line, streaming ScrollText,
  artifact preview: species = palette swatches + `render.DrawFishPreview` mini
  render, water = swatches or the shipped plant design via `render.DrawPlant`,
  pattern = before/after palette bars) + Run All / per-agent Run / Research New
  Species. Species/Plants: 2-column scrollable grids with live mini previews;
  clicks enqueue ActionSelectSpecies / ActionApplyPlant. Log: shared tank log
  (200-line cap per C3). Settings: form edited IN PLACE on `*contract.Config`
  (endpoint, model READ-ONLY per C6, masked key, proxy, 3 toggles, 3 sliders
  with frozen ranges, Import key from ZCode / Export Pack / Import Pack /
  Reset Tank) — all buttons enqueue actions for the game (G6.1).
- Why: G5.3 acceptance — every button performs its real action through its
  owning package; no tab crashes with an empty content folder.
- Gotchas:
  - **Demo-sample fallback:** with a nil store or empty folders, the catalog /
    plants / preview lookups fall back to small contract-valid samples
    (`source:"core"`-style, tab_catalog.go) so the demo and PNG frames stay
    meaningful. The real game always has >= 4 seed species (G3.1), so samples
    never appear in production.
  - Pattern before/after bars: the event bus (`contract.Event`) carries no
    palette, and the store exposes no recipe lookup, so both bars render as
    labeled placeholders unless a future event adds palettes (see follow-ups).
  - Settings has no scroll: the form is ~430 px tall and fits the 720 px
    design height; much smaller windows clip the bottom buttons.
  - Config writes are live (sliders/toggles/inputs mutate `cfg` immediately);
    persisting to `config.json` belongs to the game/config packages (G6.2).
  - The catalog grid caches preview images per species id (cleared above 64
    entries and on `SetContentStore`) to keep the draw path allocation-flat.
- Follow-ups:
  - Game (G6.1): call `menu.Wheel(ebiten wheel dy, mx, my)` and drain
    `menu.Actions()` each frame; map ActionKind values to agent/world calls;
    default pack paths when Action.Path == "".
  - If the events contract ever grows an optional palette field, wire the
    pattern card's before/after bars to real palettes (D3: README change first).
