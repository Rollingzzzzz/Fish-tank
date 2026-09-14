// G2.4: world accessors consumed by the renderer and game loop.
package sim

import (
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// Fishes returns live fish (including fading ones for the fade-out draw).
func (w *World) Fishes() []*Fish { return w.fishes }

// AliveFishes counts fish that are not fading out (the F29 evidence line
// reports both: rendered total vs the live population the cap governs).
func (w *World) AliveFishes() int { return len(w.aliveFishes()) }

// Plants returns placed plant instances.
func (w *World) Plants() []*Plant { return w.plants }

// Foods returns current flakes.
func (w *World) Foods() []Food { return w.foods }

// Eggs returns current eggs.
func (w *World) Eggs() []Egg { return w.eggs }

// Particles returns live sparkles.
func (w *World) Particles() []Particle { return w.particles }

// Bubbles returns live bubbles.
func (w *World) Bubbles() []Bubble { return w.bubbles }

// WaterLive returns the crossfaded water state.
func (w *World) WaterLive() WaterLive {
	wl := w.WaterCur
	wl.Time = w.time
	wl.DayFactor = w.dayFactor()
	wl.Night = 1 - wl.DayFactor
	// F9: TOD spans the FULL day+night cycle (0 dawn, .25 noon, .5 dusk,
	// .75 night) so the render LUT keys line up with the dayFactor curve.
	wl.TOD = mathMod(w.Clock, 2*maxF(w.cfg.DaySeconds, 5)) / (2 * maxF(w.cfg.DaySeconds, 5))
	return wl
}

// LogEntries returns the tank log (bounded ring).
func (w *World) LogEntries() []LogEntry { return w.Log }

// applyWaterLive updates the target water values from a preset.
func (w *World) applyWaterLive(p *contract.WaterPreset) {
	hx := func(s string) [3]float64 {
		r, g, b := contract.HexToRGB(s)
		return [3]float64{float64(r) / 255, float64(g) / 255, float64(b) / 255}
	}
	w.WaterTgt.Top = hx(p.TopColor)
	w.WaterTgt.Bottom = hx(p.BottomColor)
	w.WaterTgt.Accent = hx(p.Accent)
	w.WaterTgt.Rays = contract.Clamp(p.Rays, 0, 1)
	w.WaterTgt.Caustics = contract.Clamp(p.Caustics, 0, 1)
	w.WaterTgt.Bubbles = contract.Clamp(p.Bubbles, 0, 1)
}

// ApplyWater switches to a preset with a soft crossfade.
func (w *World) ApplyWater(p *contract.WaterPreset) {
	if p == nil {
		return
	}
	w.applyWaterLive(p)
	w.WaterID = p.ID
	w.logf("water", "atmosphere shifts: "+p.Name)
}

// SpawnEgg drops an egg of the species (agent births and catalog clicks).
func (w *World) SpawnEgg(speciesID string) {
	sp := w.storeSpecies(speciesID)
	if sp == nil {
		return
	}
	if sp.Role == contract.RoleChosen { // v0.3.5: the one and only — never spawnable
		w.logf("life", "the eternal one cannot be created")
		return
	}
	if sp.Role == contract.RoleTitan { // v1.1: visits cannot be summoned
		w.logf("nature", "the deep wanderer answers no egg")
		return
	}
	if sp.Role == contract.RoleShark { // v1.1: the resident pair answers no egg
		w.logf("nature", "the hunter answers no egg")
		return
	}
	w.eggs = append(w.eggs, Egg{
		SpeciesID: speciesID,
		Pos:       v2(w.W*(0.2+w.rng.Float64()*0.6), w.H*(0.15+w.rng.Float64()*0.25)),
		Seed:      w.rng.Int63(),
	})
	w.logf("agents", "a "+sp.Name+" egg appears")
}

// AddPlant places a new plant of the given design near the floor. F14: the
// effective cap (min(PlantHardCap, alive fish - 1)) is a silent no-op gate —
// agents may keep proposing plants forever without a log spam.
func (w *World) AddPlant(def *contract.PlantDesign) {
	if def == nil || len(w.plants) >= w.effectivePlantCap() {
		return
	}
	x := w.W * (0.08 + w.rng.Float64()*0.84)
	w.plants = append(w.plants, &Plant{Def: def, X: x, Y: w.H - 6, Sc: 0.85 + w.rng.Float64()*0.4})
	w.PlantIDs = append(w.PlantIDs, def.ID)
	w.logf("tank", def.Name+" takes root")
}

// DebugStageFlora replaces the plant field with the given designs, spaced
// evenly along the floor — evidence harness only (-shot): it ignores the
// placement caps and the F14 fish-majority gate on purpose.
func (w *World) DebugStageFlora(defs []*contract.PlantDesign) {
	w.plants = w.plants[:0]
	w.PlantIDs = w.PlantIDs[:0]
	n := len(defs)
	for i, def := range defs {
		if def == nil {
			continue
		}
		x := w.W * (0.12 + 0.76*float64(i)/float64(max(n-1, 1)))
		w.plants = append(w.plants, &Plant{Def: def, X: x, Y: w.H - 6, Sc: 1})
		w.PlantIDs = append(w.PlantIDs, def.ID)
	}
	w.logf("tank", "the flora is restaged for the camera")
}

// ---- public accessors used by the renderer and the game loop ----

// DropFood sprinkles 3 flakes at (x, y).
func (w *World) DropFood(x, y float64) {
	for k := 0; k < 3 && len(w.foods) < 64; k++ {
		w.foods = append(w.foods, Food{
			Pos:  v2(x+w.rng.Float64()*18-9, y+w.rng.Float64()*8),
			Vel:  v2(w.rng.Float64()*10-5, 10+w.rng.Float64()*10),
			Seed: w.rng.Float64() * 6.283,
		})
	}
}

// NextRepaints pops up to max repaint requests for the Pattern agent.
func (w *World) NextRepaints(max int) []contract.PatternRequest {
	n := min(max, len(w.repaintQ))
	out := make([]contract.PatternRequest, n)
	copy(out, w.repaintQ[:n])
	w.repaintQ = w.repaintQ[n:]
	return out
}

// ApplyRecipe repaints a fish and shows the before/after in the log.
func (w *World) ApplyRecipe(fishID string, r contract.PatternRecipe) {
	for _, f := range w.fishes {
		if f.ID != fishID {
			continue
		}
		f.Pal = applyRecipePal(f.Pal, r)
		f.Pat = r.Pattern
		w.logf("agents", f.Sp.Name+" was repainted: "+r.Name)
		return
	}
}

// SetRockBase records how much floor line the rock layout occupies (px) so
// FloorShares can report the composition (v0.3.2).
func (w *World) SetRockBase(bw float64) { w.rockBase = bw }

// FloorShares reports the tank-bottom composition as fractions of the floor
// width: plants, rocks, corals, and open water/sand left for the fish.
func (w *World) FloorShares() (plants, rocks, corals, open float64) {
	fw := maxF(w.W, 1)
	plants = float64(len(w.plants)) * contract.PlantBasePx / fw
	corals = float64(len(w.corals)) * contract.CoralBasePx / fw
	rocks = clampF(w.rockBase/fw, 0, 0.6)
	open = maxF(0, 1-plants-rocks-corals)
	return
}

// ChosenNestDist reports the Chosen's distance to her oyster mouth (px) —
// the renderer turns proximity into the touch-scene glow (v0.3.5). Returns
// a huge value when she is absent (immortal, but just in case).
func (w *World) ChosenNestDist() float64 {
	home := v2(w.W*0.62, w.H*0.5)
	best := 1e9
	for _, z := range w.zones {
		if z.Owner == "chosen" {
			home = z.Center
			break
		}
	}
	for _, f := range w.fishes {
		if f.Sp.Role == contract.RoleChosen && !f.Dying {
			if d := hyp2(sub(f.Pos, home)); d < best {
				best = d
			}
		}
	}
	return best
}

// DebugSetNight pins the clock to mid-night — evidence harness only
// (-probe): the night-state captures need the real dimming, not a mock.
func (w *World) DebugSetNight() {
	w.Clock = 1.5 * maxF(w.cfg.DaySeconds, 5)
}

// DebugSetDay pins the clock to mid-day — evidence harness only (-probe).
func (w *World) DebugSetDay() {
	w.Clock = 0.25 * maxF(w.cfg.DaySeconds, 5)
}

// DebugFeedFloor drops n flakes straight onto the dunes — evidence harness
// only (-probe): the resting-food composition with the school diving for it.
func (w *World) DebugFeedFloor(n int) {
	for i := 0; i < n; i++ {
		x := w.W * (0.30 + 0.18*float64(i))
		w.foods = append(w.foods, Food{
			Pos:  v2(x, contract.SandSurfaceY(w.H, x)-1),
			Seed: float64(i + 3),
			Age:  1,
		})
	}
}
