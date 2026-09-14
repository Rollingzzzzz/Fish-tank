// G5.3: Plants tab — scrollable grid of plant designs from the content store
// (demo samples as fallback), each rendered live with render.DrawPlant so
// sway/glow animate; clicking a card enqueues ActionApplyPlant (G6.1).
package ui

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/Rollingzzzzz/Fish-tank/internal/content"
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/Rollingzzzzz/Fish-tank/internal/render"
)

const (
	plantCellH = 84
	plantPrevH = 52
)

type plantsTab struct {
	scroll    float64
	maxScroll float64
	arm       bool
	list      []string // visible plant ids, refreshed by draw
}

func newPlantsTab() *plantsTab { return &plantsTab{} }

func (p *plantsTab) reset() { p.scroll, p.maxScroll, p.list = 0, 0, nil }

func (p *plantsTab) wheel(dy int) {
	p.scroll -= float64(dy) * plantCellH / 2
	if p.scroll > 0 {
		p.scroll = 0
	}
	if p.scroll < -p.maxScroll {
		p.scroll = -p.maxScroll
	}
}

// update returns a plant id when a card was clicked.
func (p *plantsTab) update(area Rect, mx, my int, pressed, released bool) string {
	click := false
	if pressed && !p.arm && HitR(area, mx, my) {
		p.arm = true
	}
	if p.arm {
		if released {
			p.arm = false
			click = HitR(area, mx, my)
		} else if !pressed {
			p.arm = false
		}
	}
	if !click {
		return ""
	}
	cw := area.W/2 - 4
	for i, id := range p.list {
		if HitR(p.cell(area, cw, i), mx, my) {
			return id
		}
	}
	return ""
}

func (p *plantsTab) cell(area Rect, cw, i int) Rect {
	return Rect{
		X: area.X + (i%2)*(cw+8),
		Y: area.Y + (i/2)*plantCellH + int(p.scroll),
		W: cw, H: plantCellH - 6,
	}
}

// draw renders the grid; plants are drawn live (animated sway, night = 0 in
// the menu context — the tank itself owns the day cycle).
func (p *plantsTab) draw(dst *ebiten.Image, m *Menu, area Rect) {
	list := plantList(m.store)
	p.list = p.list[:0]
	if len(list) == 0 {
		DrawText(dst, "No plant content loaded.", area.X+8, area.Y+8, 2, ColDim, 0.9)
		return
	}
	cw := area.W/2 - 4
	rows := (len(list) + 1) / 2
	p.maxScroll = float64(rows*plantCellH - area.H)
	if p.maxScroll < 0 {
		p.maxScroll = 0
	}
	p.wheel(0) // clamp
	clip, ok := subImage(dst, imageRect(area))
	if !ok {
		return
	}
	for i, pl := range list {
		p.list = append(p.list, pl.ID)
		r := p.cell(area, cw, i)
		if r.Y+plantCellH < area.Y || r.Y > area.Y+area.H {
			continue
		}
		FillRect(clip, r.X+1, r.Y+1, r.W-2, r.H-2, ColPanel, 0.85)
		FrameRect(clip, r.X, r.Y, r.W, r.H, ColBorder, borderAlpha*0.7)
		render.DrawPlant(clip, clip, pl, float64(r.X+r.W/2), float64(r.Y+plantPrevH-2),
			float64(plantPrevH-8), m.clock, 0)
		DrawText(clip, Ellipsis(pl.Name, r.W/6-2), r.X+6, r.Y+plantPrevH+4, 1, ColText, 1)
		DrawText(clip, Ellipsis(pl.Source, r.W/6-2), r.X+6, r.Y+plantPrevH+16, 1, ColDim, 0.8)
	}
	DrawText(dst, "click a card to apply the plant set", area.X+4, area.Y+area.H-12, 1, ColDim, 0.6)
}

// plantList returns store plants or demo samples when absent.
func plantList(s *content.Store) []*contract.PlantDesign {
	if s != nil {
		if l := s.Plants(); len(l) > 0 {
			return l
		}
	}
	return samplePlants()
}
