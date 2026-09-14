// G5.3: Agents tab — three live agent cards (Species/Water/Pattern) with
// status badge, dimmed thinking line, streaming ScrollText and an artifact
// preview (species: palette swatches + mini fish render; water: swatches;
// pattern: before/after bars), plus Run All / per-agent Run / Research New
// Species buttons. All buttons enqueue Actions for the game (G6.1).
package ui

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/Rollingzzzzz/Fish-tank/internal/render"
)

const (
	agentsToolbarH = 34
	cardGap        = 6
	cardHeadH      = 20
	cardThinkH     = 12
	previewH       = 64
	thinkCap       = 96 // chars of thinking context kept per card (C3)
)

type agentCard struct {
	id         string
	name       string
	status     contract.AgentStatus
	think      string // latest reasoning tail (dimmed line, C6)
	stream     *ScrollText
	artifactID string
	note       string // artifact summary from the bus
	preview    *ebiten.Image
	previewKey string
	run        ButtonState
}

type agentsTab struct {
	cards    []*agentCard
	runAll   ButtonState
	research ButtonState
}

func newAgentsTab() *agentsTab {
	a := &agentsTab{cards: []*agentCard{
		{id: contract.AgentSpecies, name: "Species Agent"},
		{id: contract.AgentWater, name: "Water Agent"},
		{id: contract.AgentPattern, name: "Pattern Agent"},
	}}
	for _, c := range a.cards {
		c.stream = NewScrollText(60)
		c.status = contract.StatusIdle
	}
	return a
}

func (a *agentsTab) card(id string) *agentCard {
	for _, c := range a.cards {
		if c.id == id {
			return c
		}
	}
	return nil
}

func (a *agentsTab) resetPreviews() {
	for _, c := range a.cards {
		c.preview, c.previewKey = nil, ""
	}
}

// handle routes one bus event to its card (called from Menu.PushEvent).
func (a *agentsTab) handle(ev contract.Event) {
	c := a.card(ev.Agent)
	if c == nil {
		return
	}
	switch ev.Kind {
	case contract.EventThought:
		c.think = tailRunes(c.think+ev.Text, thinkCap)
	case contract.EventChunk:
		c.stream.AppendChunk(KindText, ev.Text)
	case contract.EventStatus:
		if ev.Status != "" {
			c.status = ev.Status
		}
		if ev.Text != "" {
			c.stream.AppendLine(KindText, ev.Text)
		}
	case contract.EventArtifact:
		c.artifactID = ev.ArtifactID
		c.note = ev.Text
		c.preview, c.previewKey = nil, ""
		c.stream.AppendLine(KindText, "artifact: "+ev.ArtifactID)
	}
}

// tailRunes keeps at most n trailing runes, prefixing ".." when trimmed.
func tailRunes(s string, n int) string {
	rs := []rune(s)
	if len(rs) <= n {
		return s
	}
	return ".." + string(rs[len(rs)-n:])
}

// update handles the tab's buttons; actions are enqueued on the menu.
func (a *agentsTab) update(area Rect, mx, my int, pressed, released bool, m *Menu) {
	cardsH := (area.H - agentsToolbarH - 2*cardGap) / 3
	for i, c := range a.cards {
		cr := Rect{X: area.X, Y: area.Y + i*(cardsH+cardGap), W: area.W, H: cardsH}
		run := Rect{X: cr.X + cr.W - 48, Y: cr.Y + 3, W: 44, H: 16}
		if c.run.Update(run.X, run.Y, run.W, run.H, mx, my, pressed, released) {
			m.enqueue(Action{Kind: ActionRunAgent, ID: c.id})
		}
	}
	tb := Rect{X: area.X, Y: area.Y + area.H - agentsToolbarH + 6, W: area.W, H: 22}
	if a.runAll.Update(tb.X, tb.Y, 92, 22, mx, my, pressed, released) {
		m.enqueue(Action{Kind: ActionRunAll})
	}
	if a.research.Update(tb.X+102, tb.Y, 168, 22, mx, my, pressed, released) {
		m.enqueue(Action{Kind: ActionResearch})
	}
}

// draw renders the three cards + toolbar.
func (a *agentsTab) draw(dst *ebiten.Image, m *Menu, area Rect) {
	cardsH := (area.H - agentsToolbarH - 2*cardGap) / 3
	for i, c := range a.cards {
		cr := Rect{X: area.X, Y: area.Y + i*(cardsH+cardGap), W: area.W, H: cardsH}
		drawAgentCard(dst, m, c, cr)
	}
	tb := Rect{X: area.X, Y: area.Y + area.H - agentsToolbarH + 6, W: area.W, H: 22}
	DrawButton(dst, &a.runAll, tb.X, tb.Y, 92, 22, "Run All", ColAccent2)
	DrawButton(dst, &a.research, tb.X+102, tb.Y, 168, 22, "Research New Species", ColAccent)
	DrawText(dst, "paced via Setup tab", tb.X+280, tb.Y+8, 1, ColDim, 0.7)
}

func drawAgentCard(dst *ebiten.Image, m *Menu, c *agentCard, cr Rect) {
	GlassPanel(dst, cr.X, cr.Y, cr.W, cr.H)
	// Header: name + status badge + Run button.
	DrawText(dst, c.name, cr.X+8, cr.Y+5, 1, ColText, 1)
	stHex := StatusColor(string(c.status))
	Glow(dst, float64(cr.X+96), float64(cr.Y+9), 5, stHex, 0.8)
	FillRect(dst, cr.X+94, cr.Y+7, 4, 4, stHex, 1)
	DrawText(dst, string(c.status), cr.X+102, cr.Y+6, 1, stHex, 1)
	run := Rect{X: cr.X + cr.W - 48, Y: cr.Y + 3, W: 44, H: 16}
	DrawButton(dst, &c.run, run.X, run.Y, run.W, run.H, "Run", stHex)
	// Dimmed thinking line (C6: reasoning never mixes into output).
	thinkY := cr.Y + cardHeadH
	FillRect(dst, cr.X+6, thinkY, cr.W-12, 1, ColBorder, 0.15)
	DrawText(dst, Ellipsis(c.think, cr.W/6-2), cr.X+8, thinkY+3, 1, ColDim, 0.8)
	// Streaming area + artifact preview strip.
	body := Rect{X: cr.X + 6, Y: thinkY + cardThinkH, W: cr.W - 12, H: cr.H - cardHeadH - cardThinkH - 8}
	if c.artifactID != "" && cr.H > previewH+70 {
		body.H -= previewH + 4
		pr := Rect{X: body.X, Y: body.Y + body.H + 4, W: body.W, H: previewH}
		drawArtifactPreview(dst, m, c, pr)
	}
	c.stream.Draw(dst, body.X, body.Y, body.W, body.H, 1)
}

// drawArtifactPreview renders the artifact strip per agent kind.
func drawArtifactPreview(dst *ebiten.Image, m *Menu, c *agentCard, r Rect) {
	FillRect(dst, r.X, r.Y, r.W, r.H, ColPanel, 0.6)
	FrameRect(dst, r.X, r.Y, r.W, r.H, ColBorder, borderAlpha*0.6)
	switch c.id {
	case contract.AgentSpecies:
		spec := lookupSpecies(m.store, c.artifactID)
		if spec == nil {
			drawPreviewLabel(dst, r, "artifact not in catalog: "+c.artifactID)
			return
		}
		drawSwatches(dst, []string{spec.Palette.Body, spec.Palette.Belly,
			spec.Palette.Accent, spec.Palette.Glow}, Rect{X: r.X + r.W - 66, Y: r.Y + 6, W: 60, H: r.H - 12})
		drawSpeciesPreview(dst, c, spec, Rect{X: r.X + 4, Y: r.Y + 4, W: r.W - 74, H: r.H - 8})
	case contract.AgentWater:
		w := lookupWater(m.store, c.artifactID)
		p := lookupPlant(m.store, c.artifactID)
		switch {
		case w != nil:
			drawSwatches(dst, []string{w.TopColor, w.BottomColor, w.Accent}, r)
		case p != nil: // water agent also ships one plant design per run
			render.DrawPlant(dst, dst, p, float64(r.X+r.W/2), float64(r.Y+r.H-4),
				float64(r.H-14), m.clock, 0)
		default:
			drawPreviewLabel(dst, r, "artifact not in catalog: "+c.artifactID)
		}
	default: // pattern agent: before/after palette bars (before unknown -> dim)
		half := (r.H - 16) / 2
		drawPalBar(dst, Rect{X: r.X + 52, Y: r.Y + 6, W: r.W - 60, H: half}, nil, "before")
		drawPalBar(dst, Rect{X: r.X + 52, Y: r.Y + 10 + half, W: r.W - 60, H: half}, nil, "after")
		DrawText(dst, Ellipsis(c.note, r.W/6-10), r.X+6, r.Y+r.H-9, 1, ColDim, 0.9)
	}
}

// drawPalBar draws one labeled palette bar (empty shows a dim placeholder).
func drawPalBar(dst *ebiten.Image, r Rect, pal []string, label string) {
	DrawText(dst, label, r.X-46, r.Y+r.H/2-3, 1, ColDim, 0.9)
	FillRect(dst, r.X, r.Y, r.W, r.H, ColPanel, 0.7)
	if len(pal) == 0 {
		FrameRect(dst, r.X, r.Y, r.W, r.H, ColDim, 0.3)
		DrawText(dst, "awaiting recipe", r.X+8, r.Y+r.H/2-3, 1, ColDim, 0.6)
		return
	}
	seg := r.W / len(pal)
	for i, hex := range pal {
		FillRect(dst, r.X+i*seg, r.Y, seg, r.H, hex, 1)
	}
	FrameRect(dst, r.X, r.Y, r.W, r.H, ColBorder, borderAlpha)
}

// drawSwatches renders hex colors as rounded-ish squares, 3 per row.
func drawSwatches(dst *ebiten.Image, hexes []string, r Rect) {
	n := len(hexes)
	if n == 0 {
		return
	}
	const sw = 18
	cols := 3
	for i, hex := range hexes {
		cx := r.X + (i%cols)*(sw+2)
		cy := r.Y + (i/cols)*(sw+2)
		Glow(dst, float64(cx+sw/2), float64(cy+sw/2), 9, hex, 0.35)
		FillRect(dst, cx, cy, sw, sw, hex, 1)
		FrameRect(dst, cx, cy, sw, sw, "#ffffff", 0.25)
	}
}

// drawSpeciesPreview renders the cached mini fish via render.DrawFishPreview.
func drawSpeciesPreview(dst *ebiten.Image, c *agentCard, spec *contract.Species, r Rect) {
	key := spec.ID
	if c.preview == nil || c.previewKey != key {
		c.preview = ebiten.NewImage(r.W, r.H)
		render.DrawFishPreview(c.preview, spec, r.W, r.H)
		c.previewKey = key
	}
	var op ebiten.DrawImageOptions
	op.GeoM.Translate(float64(r.X), float64(r.Y))
	dst.DrawImage(c.preview, &op)
}

func drawPreviewLabel(dst *ebiten.Image, r Rect, text string) {
	DrawText(dst, Ellipsis(text, r.W/6-2), r.X+6, r.Y+r.H/2-3, 1, ColDim, 0.8)
}
