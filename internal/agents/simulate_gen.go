// G3.3: simulate generators — contract-valid Species / WaterPreset /
// PlantDesign / CoralDesign / PatternRecipe with neon palettes. Registry-aware:
// candidates are checked via the matching content.Store gate and re-rolled
// until unique (species also respect the FD8 lilac reservation). N9: species
// draws stay under the normal supremacy caps so inventions never rival the
// Chosen.
package agents

import (
	"fmt"
	"math/rand"

	"github.com/Rollingzzzzz/Fish-tank/internal/content"
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// gateAttempts bounds the re-roll loop; the fingerprint space (pattern type x
// density/size buckets x four 10-degree hue buckets) makes exhaustion
// practically impossible, and the last candidate is returned regardless.
const gateAttempts = 128

// Simulate species draw ranges (N9): inventions never reach the Chosen's
// scale — size/fin/tail top out at contract.NormalSizeMax/FinMax/TailMax
// (spSizeMin/spFinMin/spTailMin mirror content's README §4 floors).
const (
	simSizeMin = 0.6
	simFinMin  = 0.6
	simTailMin = 0.7
)

// SimulateSpecies produces one guaranteed-unique neon species.
func SimulateSpecies(rng *rand.Rand, st *content.Store) contract.Species {
	gen := func() string { return pick(rng, namePrefix) + " " + pick(rng, nameCore) + pick(rng, nameTail) }
	var last contract.Species
	for attempt := 0; attempt < gateAttempts; attempt++ {
		name := uniqueName(rng, st, content.KindSpecies, gen)
		hue := rng.Float64()*360 + float64(attempt)*37 // rotate on gate retries
		sp := contract.Species{
			Name:    name,
			Latin:   pick(rng, latinGenus) + " " + pick(rng, latinEpith),
			Role:    contract.RoleNormal,
			Size:    simSizeMin + rng.Float64()*(contract.NormalSizeMax-simSizeMin),
			Width:   0.7 + rng.Float64()*0.6,
			Fin:     simFinMin + rng.Float64()*(contract.NormalFinMax-simFinMin),
			Tail:    simTailMin + rng.Float64()*(contract.NormalTailMax-simTailMin),
			Palette: neonPaletteAt(rng, hue),
			Pattern: contract.Pattern{
				Type:    pick(rng, patternPool),
				Density: rng.Float64(),
				Size:    rng.Float64(),
			},
			Behavior: contract.Behavior{
				Speed:       0.5 + rng.Float64()*(contract.NormalSpeedMax-0.5), // F23: under the Chosen's worst hour
				Schooling:   rng.Float64(),
				Curiosity:   rng.Float64(),
				Skittish:    rng.Float64(),
				Depth:       rng.Float64(),
				NightActive: rng.Float64() < 0.3,
			},
			Note: fmt.Sprintf(pick(rng, speciesNotes), name),
		}
		sp.ID = uniqueID(st, content.KindSpecies, slugWithHeadroom(name))
		last = sp
		if ok, _ := st.GateSpecies(&sp); ok {
			return sp
		}
	}
	return last
}

// SimulateWater produces one guaranteed-unique water preset (with event).
func SimulateWater(rng *rand.Rand, st *content.Store) contract.WaterPreset {
	gen := func() string { return pick(rng, waterCore) + " " + pick(rng, waterTail) }
	var last contract.WaterPreset
	for attempt := 0; attempt < gateAttempts; attempt++ {
		name := uniqueName(rng, st, content.KindWater, gen)
		hue := rng.Float64()*360 + float64(attempt)*41
		w := contract.WaterPreset{
			Name:         name,
			TopColor:     hslHex(hue, 0.7+rng.Float64()*0.3, 0.22+rng.Float64()*0.14),
			BottomColor:  hslHex(hue+40, 0.85, 0.04+rng.Float64()*0.05),
			Accent:       hslHex(hue+150+rng.Float64()*60, 0.95, 0.6),
			Rays:         rng.Float64(),
			Caustics:     rng.Float64(),
			Bubbles:      rng.Float64(),
			PlantPalette: gradientColors(rng, hue+60+rng.Float64()*80, 2+rng.Intn(3)),
			Event: &contract.WaterEvent{
				Name:     pick(rng, waterCore) + " " + pick(rng, []string{"Pulse", "Drift", "Still", "Swell"}),
				Kind:     pick(rng, eventKindPool),
				Duration: 15 + rng.Float64()*25,
				Note:     pick(rng, waterNotes),
			},
		}
		w.ID = uniqueID(st, content.KindWater, slugWithHeadroom(name))
		last = w
		if ok, _ := st.Gate(content.KindWater, w.ID, w.Name, contract.Pattern{}, content.WaterPalette(&w)); ok {
			return w
		}
	}
	return last
}

// SimulatePlant produces one guaranteed-unique plant design.
func SimulatePlant(rng *rand.Rand, st *content.Store) contract.PlantDesign {
	gen := func() string { return pick(rng, plantCore) + pick(rng, plantTail) }
	var last contract.PlantDesign
	for attempt := 0; attempt < gateAttempts; attempt++ {
		name := uniqueName(rng, st, content.KindPlant, gen)
		hue := rng.Float64()*360 + float64(attempt)*43
		p := contract.PlantDesign{
			Name:   name,
			Fronds: 3 + rng.Intn(6),
			Height: 0.15 + rng.Float64()*0.4,
			Width:  0.5 + rng.Float64(),
			Curve:  rng.Float64(),
			Sway:   rng.Float64(),
			Colors: gradientColors(rng, hue, 3+rng.Intn(2)),
			Glow:   0.3 + rng.Float64()*0.7,
		}
		p.ID = uniqueID(st, content.KindPlant, slugWithHeadroom(name))
		last = p
		if ok, _ := st.Gate(content.KindPlant, p.ID, p.Name, contract.Pattern{}, content.PlantPalette(&p)); ok {
			return p
		}
	}
	return last
}

// SimulateCoral produces one guaranteed-unique coral design (kind-aware gate).
func SimulateCoral(rng *rand.Rand, st *content.Store) contract.CoralDesign {
	gen := func() string { return pick(rng, coralCore) + " " + pick(rng, coralTail) }
	var last contract.CoralDesign
	for attempt := 0; attempt < gateAttempts; attempt++ {
		name := uniqueName(rng, st, content.KindCoral, gen)
		hue := rng.Float64()*360 + float64(attempt)*47
		c := contract.CoralDesign{
			Name:   name,
			Kind:   pick(rng, coralKindPool),
			Fronds: 3 + rng.Intn(7),
			Height: 0.08 + rng.Float64()*0.27,
			Width:  0.5 + rng.Float64()*1.1,
			Curve:  rng.Float64(),
			Sway:   rng.Float64(),
			Colors: gradientColors(rng, hue, 3+rng.Intn(2)),
			Glow:   0.3 + rng.Float64()*0.7,
		}
		c.ID = uniqueID(st, content.KindCoral, slugWithHeadroom(name))
		last = c
		if ok, _ := st.GateCoral(&c); ok {
			return c
		}
	}
	return last
}

// SimulateRecipe produces one stage-appropriate repaint recipe with a unique
// ID; it re-rolls until the recipe passes GateRecipe (registry-aware).
func SimulateRecipe(rng *rand.Rand, stage string, st *content.Store) contract.PatternRecipe {
	lo, hi := stageMultipliers(stage)
	var last contract.PatternRecipe
	for attempt := 0; attempt < gateAttempts; attempt++ {
		r := contract.PatternRecipe{
			Stage:    stage,
			Name:     stageTitle(stage) + " " + pick(rng, []string{"Veil", "Spark", "Prime", "Fade", "Wash", "Coat", "Shine", "Mist"}),
			SatMul:   lo[0] + rng.Float64()*(hi[0]-lo[0]),
			AlphaMul: lo[1] + rng.Float64()*(hi[1]-lo[1]),
			GlowMul:  lo[2] + rng.Float64()*(hi[2]-lo[2]),
			Pattern: contract.Pattern{
				Type:    pick(rng, patternPool),
				Density: rng.Float64(),
				Size:    rng.Float64(),
			},
			Note: recipeNotes[stage],
		}
		r.ID = uniqueID(st, content.KindPattern, slugWithHeadroom(stage+"-"+r.Name))
		last = r
		if ok, _ := st.GateRecipe(&r); ok {
			return r
		}
	}
	return last
}

// stageMultipliers returns lo/hi triples for satMul, alphaMul, glowMul per
// stage look (fry translucent soft ... elder desaturated faded).
func stageMultipliers(stage string) ([3]float64, [3]float64) {
	switch stage {
	case "fry":
		return [3]float64{0.5, 0.45, 0.7}, [3]float64{0.9, 0.7, 1.0}
	case "juvenile":
		return [3]float64{0.9, 0.7, 1.0}, [3]float64{1.2, 0.9, 1.4}
	case "elder":
		return [3]float64{0.3, 0.45, 0.5}, [3]float64{0.6, 0.75, 0.9}
	default: // adult
		return [3]float64{1.0, 0.85, 1.2}, [3]float64{1.4, 1.0, 2.0}
	}
}

func stageTitle(stage string) string {
	switch stage {
	case "fry":
		return "Fry"
	case "juvenile":
		return "Juvenile"
	case "elder":
		return "Elder"
	default:
		return "Adult"
	}
}
