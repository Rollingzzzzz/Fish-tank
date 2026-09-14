// G2.4: world-internal helpers — logging, care, lookups, ids, plants.
package sim

import (
	"math/rand"
	"time"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// contractRand aliases the stdlib RNG type used across the package.
type contractRand = rand.Rand

func contractRandFrom(r *rand.Rand) *rand.Rand { return r }

// SeedRng pins the world RNG — a determinism hook for tests (D5).
func (w *World) SeedRng(seed int64) {
	w.rng = contractRandFrom(contract.RandSeed(seed))
	// F4 timers stagger so the tank never opens with a burst of events
	w.evStartle = contract.StartleMeanSec * (0.6 + 0.4*w.rng.Float64())
	w.evChase = 70 * (0.6 + 0.4*w.rng.Float64())
	w.evZoom = 55 * (0.6 + 0.4*w.rng.Float64())
	w.evNudge = 30 * (0.6 + 0.4*w.rng.Float64())
	// F15: stagger the per-fish lounge countdowns the same way
	for _, f := range w.fishes {
		f.loungeNext = contract.LoungeMeanSec * (0.6 + 0.4*w.rng.Float64())
	}
}

// effectivePlantCap is the living plant budget: v0.3.2 gives plants ~20% of
// the floor line (base-width budget); F14 still applies — live fish always
// outnumber plants, so the cap is min(floor share, alive-1).
func (w *World) effectivePlantCap() int {
	hard := clampI(int(w.W*contract.FloorSharePlants/contract.PlantBasePx+0.5), 5, contract.PlantCapMax)
	return min(hard, len(w.aliveFishes())-1)
}

// enforcePlantMajority re-asserts F14 after any fish loss: the oldest plant
// (first placed) is culled per breach, each with its own log line.
func (w *World) enforcePlantMajority() {
	for len(w.plants) > 0 && len(w.plants) > w.effectivePlantCap() {
		w.plants = w.plants[1:]
		if len(w.PlantIDs) > len(w.plants) {
			w.PlantIDs = w.PlantIDs[1:]
		}
		w.logf("garden", "the garden thins")
	}
}

// sweepCorpses logs and removes fully faded fish, then re-checks the F14
// plant majority — deaths thin the garden one plant at a time.
func (w *World) sweepCorpses() {
	for _, f := range w.fishes {
		if f.Dying && f.Fade <= 0 {
			w.logf("life", f.Sp.Name+" drifted away ("+f.DieReason+")")
		}
	}
	kept := w.fishes[:0]
	for _, f := range w.fishes {
		if f.Dying && f.Fade <= 0 {
			continue
		}
		kept = append(kept, f)
	}
	w.fishes = kept
	w.enforcePlantMajority()
}

// nextID returns the next sequential fish id.
func (w *World) nextID() int {
	w.idSeq++
	return w.idSeq
}

// storeSpecies looks a species up by ID.
func (w *World) storeSpecies(id string) *contract.Species {
	for _, sp := range w.species {
		if sp.ID == id {
			return sp
		}
	}
	return nil
}

// aliveFishes lists fish that are not dying AND not titans (v1.1): the pod
// is an ambient visit, never part of the population the caps, breeding,
// behavior picks and the population floor govern.
func (w *World) aliveFishes() []*Fish {
	out := w.fishes[:0:0]
	for _, f := range w.fishes {
		if !f.Dying && f.Sp.Role != contract.RoleTitan && f.Sp.Role != contract.RoleShark {
			out = append(out, f)
		}
	}
	return out
}

// logf appends a bounded tank-log line (C3 cap 200).
func (w *World) logf(who, text string) {
	w.Log = append(w.Log, LogEntry{At: time.Now().Format("15:04:05"), Who: who, Text: text})
	if len(w.Log) > 200 {
		w.Log = w.Log[len(w.Log)-200:]
	}
}

// addCare raises the care score.
func (w *World) addCare(v float64) { w.Care += v }

// nearestPlantAnchor returns the closest plant base as a resting spot.
func (w *World) nearestPlantAnchor(p contract.Vec2) *contract.Vec2 {
	var best *contract.Vec2
	bestD := 1e18
	for _, pl := range w.plants {
		d := hyp2(sub(v2(pl.X, pl.Y-20), p))
		if d < bestD {
			bestD = d
			a := v2(pl.X, pl.Y-20)
			best = &a
		}
	}
	return best
}

// onStageChanged queues an agent repaint and logs the milestone.
func (w *World) onStageChanged(f *Fish) {
	w.logf("life", f.Sp.Name+" grew into a "+f.Stage)
	if w.cfg.MaxFish <= 0 {
		return
	}
	w.repaintQ = append(w.repaintQ, contract.PatternRequest{
		FishID: f.ID, Species: *f.Sp, Stage: f.Stage, OldPal: f.Pal, OldPat: f.Pat,
	})
	if len(w.repaintQ) > 64 {
		w.repaintQ = w.repaintQ[len(w.repaintQ)-64:]
	}
}

// spawnEggs lays n eggs near a fish.
func (w *World) spawnEggs(f *Fish, n int) {
	for k := 0; k < n; k++ {
		w.eggs = append(w.eggs, Egg{
			SpeciesID: f.Sp.ID,
			Pos:       add(f.CourtC, v2(w.rng.Float64()*16-8, w.rng.Float64()*10-5)),
			Progress:  0,
			Seed:      w.rng.Int63(),
		})
	}
}

// clampI is the integer sibling of clampF.
func clampI(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
