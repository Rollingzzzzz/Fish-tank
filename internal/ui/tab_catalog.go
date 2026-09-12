// G5.3: Species catalog tab — scrollable 2-column grid of species cards with
// mini fish previews (render.DrawFishPreview); clicking a card enqueues
// ActionSelectSpecies. Also hosts the demo-sample content used as fallback so
// tabs and PNG frames stay meaningful with an empty content folder (the real
// game always has >= 4 seed species, G3.1).
package ui

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/Rollingzzzzz/Fish-tank/internal/content"
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/Rollingzzzzz/Fish-tank/internal/render"
)

const catalogCellH = 96

type catalogTab struct {
	scroll    float64
	maxScroll float64
	arm       bool
	previews  map[string]*ebiten.Image
	list      []string // visible ids, refreshed by draw (store order)
}

func newCatalogTab() *catalogTab {
	return &catalogTab{previews: map[string]*ebiten.Image{}}
}

func (c *catalogTab) reset() {
	c.previews = map[string]*ebiten.Image{}
	c.scroll, c.maxScroll, c.list = 0, 0, nil
}

func (c *catalogTab) wheel(dy int) {
	c.scroll -= float64(dy) * catalogCellH / 2
	if c.scroll > 0 {
		c.scroll = 0
	}
	if c.scroll < -c.maxScroll {
		c.scroll = -c.maxScroll
	}
}

// update returns a species id when a card was clicked.
func (c *catalogTab) update(area Rect, mx, my int, pressed, released bool) string {
	click := false
	if pressed && !c.arm && HitR(area, mx, my) {
		c.arm = true
	}
	if c.arm {
		if released {
			c.arm = false
			click = HitR(area, mx, my)
		} else if !pressed {
			c.arm = false
		}
	}
	if !click {
		return ""
	}
	cw := area.W/2 - 4
	for i, id := range c.list {
		if HitR(c.catalogCell(area, cw, i), mx, my) {
			return id
		}
	}
	return ""
}

// catalogCell is the rect of visible grid cell i (scroll applied).
func (c *catalogTab) catalogCell(area Rect, cw, i int) Rect {
	return Rect{
		X: area.X + (i%2)*(cw+8),
		Y: area.Y + (i/2)*catalogCellH + int(c.scroll),
		W: cw, H: catalogCellH - 6,
	}
}

// draw renders the grid from the store (or demo samples when empty).
func (c *catalogTab) draw(dst *ebiten.Image, m *Menu, area Rect) {
	list := catalogList(m.store)
	c.list = c.list[:0]
	if len(list) == 0 {
		DrawText(dst, "No species content loaded.", area.X+8, area.Y+8, 2, ColDim, 0.9)
		return
	}
	cw := area.W/2 - 4
	rows := (len(list) + 1) / 2
	c.maxScroll = float64(rows*catalogCellH - area.H)
	if c.maxScroll < 0 {
		c.maxScroll = 0
	}
	c.wheel(0) // clamp
	clip, ok := subImage(dst, imageRect(area))
	if !ok {
		return
	}
	for i, sp := range list {
		c.list = append(c.list, sp.ID)
		if r := c.catalogCell(area, cw, i); r.Y+catalogCellH >= area.Y && r.Y <= area.Y+area.H {
			drawSpeciesCell(clip, c, sp, r)
		}
	}
	DrawText(dst, "click a card to focus it (logged)", area.X+4, area.Y+area.H-12, 1, ColDim, 0.6)
}

// drawSpeciesCell renders one card: mini fish preview + name + origin tag.
func drawSpeciesCell(dst *ebiten.Image, c *catalogTab, sp *contract.Species, r Rect) {
	FillRect(dst, r.X+1, r.Y+1, r.W-2, r.H-2, ColPanel, 0.85)
	FrameRect(dst, r.X, r.Y, r.W, r.H, ColBorder, borderAlpha*0.7)
	const prevH = 52
	key := sp.ID
	img, ok := c.previews[key]
	if !ok {
		if len(c.previews) > 64 { // bounded cache (C3)
			c.previews = map[string]*ebiten.Image{}
		}
		img = ebiten.NewImage(r.W-8, prevH)
		render.DrawFishPreview(img, sp, r.W-8, prevH)
		c.previews[key] = img
	}
	var op ebiten.DrawImageOptions
	op.GeoM.Translate(float64(r.X+4), float64(r.Y+3))
	dst.DrawImage(img, &op)
	DrawText(dst, Ellipsis(sp.Name, r.W/6-2), r.X+6, r.Y+prevH+6, 1, ColText, 1)
	DrawText(dst, Ellipsis(sp.Latin+"  "+sp.Source, r.W/6-2), r.X+6, r.Y+prevH+18, 1, ColDim, 0.8)
}

// catalogList returns store species or demo samples when absent.
func catalogList(s *content.Store) []*contract.Species {
	if s != nil {
		l := s.Species()
		// v0.3.5: the lilac Chosen is one of a kind, never spawnable from the
		// menu — she must not appear in the species catalog at all
		out := make([]*contract.Species, 0, len(l))
		for _, sp := range l {
			if sp.Role == contract.RoleChosen {
				continue
			}
			out = append(out, sp)
		}
		if len(out) > 0 {
			return out
		}
	}
	return sampleSpecies()
}

// --- content lookups with demo fallback (used by previews) ---

func lookupSpecies(s *content.Store, id string) *contract.Species {
	if s != nil {
		if v := s.SpeciesByID(id); v != nil {
			return v
		}
	}
	return sampleSpeciesByID(id)
}

func lookupPlant(s *content.Store, id string) *contract.PlantDesign {
	if s != nil {
		for _, p := range s.Plants() {
			if p.ID == id {
				return p
			}
		}
	}
	return samplePlantByID(id)
}

func lookupWater(s *content.Store, id string) *contract.WaterPreset {
	if s != nil {
		for _, w := range s.Waters() {
			if w.ID == id {
				return w
			}
		}
	}
	return sampleWaterByID(id)
}

// --- demo samples (contract-valid; source "core" style, neon per §3) ---

func sampleSpecies() []*contract.Species {
	mk := func(id, name, latin, body, belly, accent, glow, pat string, d, sz float64) *contract.Species {
		return &contract.Species{ID: id, Name: name, Latin: latin, Size: sz, Width: 1.0,
			Fin: 1.1, Tail: 1.2, Palette: contract.Palette{Body: body, Belly: belly, Accent: accent, Glow: glow},
			Pattern:  contract.Pattern{Type: pat, Density: d, Size: 0.5},
			Behavior: contract.Behavior{Speed: 1.0, Schooling: 0.6, Curiosity: 0.5, Skittish: 0.4, Depth: 0.4},
			Note:     "Demo sample used only when no content is loaded.", Source: "core"}
	}
	return []*contract.Species{
		mk("ember-tetra", "Ember Tetra", "Hyphessobrycon amandae", "#ff5a3c", "#ffd28a", "#ffe14d", "#ff7a2f", "stripe", 0.6, 1.0),
		mk("aqua-wisp", "Aqua Wisp", "Najas lucens", "#35e0ff", "#c9fbff", "#7df9ff", "#00ffe1", "spot", 0.5, 0.8),
		mk("violet-drifter", "Violet Drifter", "Noctiluca viol", "#8a5cff", "#e6d9ff", "#ff3df5", "#b085ff", "wave", 0.4, 1.2),
		mk("lime-dart", "Lime Dart", "Viridis celer", "#7dff5e", "#eaffe0", "#d6ff4d", "#58ff9b", "koi", 0.7, 0.9),
	}
}

func sampleSpeciesByID(id string) *contract.Species {
	for _, s := range sampleSpecies() {
		if s.ID == id {
			return s
		}
	}
	return nil
}

func samplePlants() []*contract.PlantDesign {
	return []*contract.PlantDesign{
		{ID: "neon-kelp", Name: "Neon Kelp", Fronds: 5, Height: 0.42, Width: 1.0, Curve: 0.3, Sway: 0.6,
			Colors: []string{"#0b6b4f", "#19e0a0", "#a8ffe0"}, Glow: 0.7, Source: "core"},
		{ID: "spiral-frond", Name: "Spiral Frond", Fronds: 4, Height: 0.5, Width: 1.1, Curve: 0.8, Sway: 0.4,
			Colors: []string{"#3c1e78", "#8a5cff", "#e6d9ff"}, Glow: 0.8, Source: "core"},
		{ID: "bubble-tendril", Name: "Bubble Tendril", Fronds: 6, Height: 0.3, Width: 0.9, Curve: 0.5, Sway: 0.9,
			Colors: []string{"#0d3b66", "#35e0ff", "#c9fbff"}, Glow: 0.6, Source: "core"},
	}
}

func samplePlantByID(id string) *contract.PlantDesign {
	for _, p := range samplePlants() {
		if p.ID == id {
			return p
		}
	}
	return nil
}

func sampleWaterByID(id string) *contract.WaterPreset {
	waters := []*contract.WaterPreset{
		{ID: "midnight-lagoon", Name: "Midnight Lagoon", TopColor: "#062a4a", BottomColor: "#02101f",
			Accent: "#00ffe1", Rays: 0.3, Caustics: 0.5, Bubbles: 0.4,
			PlantPalette: []string{"#0b6b4f", "#19e0a0"}, Source: "core"},
		{ID: "sunset-reef", Name: "Sunset Reef", TopColor: "#2a1055", BottomColor: "#ff7a2f",
			Accent: "#ff3df5", Rays: 0.6, Caustics: 0.7, Bubbles: 0.3,
			PlantPalette: []string{"#8a5cff", "#ff3df5"}, Source: "core"},
	}
	for _, w := range waters {
		if w.ID == id {
			return w
		}
	}
	return nil
}
