// v1.1: the hammerhead pair (G50) — a resident hunter that roams the whole
// tank, snapped to the same rules as the rest of the ambient life: outside
// popCap, breeding and aging, immune to the titan's hunt, capped at
// SharkMax, and never faster than the Chosen at any hour (supremacy test).
// The sharks persist in snapshots — they are residents, not visits.
package sim

import (
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// sharkSpecies finds the core-seeded hammerhead (nil when absent).
func (w *World) sharkSpecies() *contract.Species {
	for _, s := range w.species {
		if s.Role == contract.RoleShark {
			return s
		}
	}
	return nil
}

// sharkCount reports live hammerheads.
func (w *World) sharkCount() int {
	n := 0
	for _, f := range w.fishes {
		if f.Sp.Role == contract.RoleShark && !f.Dying {
			n++
		}
	}
	return n
}

// ensureSharks keeps the resident pair whole (G50): whatever sweeps them
// away, the hunter returns — the mirror of ensureChosen, capped at SharkMax.
func (w *World) ensureSharks() {
	sp := w.sharkSpecies()
	if sp == nil {
		return
	}
	for w.sharkCount() < contract.SharkMax {
		p := v2(w.W*(0.25+w.rng.Float64()*0.5), w.H*(0.30+w.rng.Float64()*0.35))
		f := newFish(sp, w.rng.Int63(), p, 6, w.nextID()) // frozen adult
		w.fishes = append(w.fishes, f)
		w.logf("nature", "a hammerhead glides out of the blue")
	}
}
