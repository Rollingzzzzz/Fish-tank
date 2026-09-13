// v1.1 split (300-line ceiling): the resident ensure passes live beside their
// concerns -- ensureSharks in shark.go, ensurePod in titan.go, and the
// original below.
package sim

import (
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// ensureChosen guarantees exactly one Chosen fish exists (FD9): immortal,
// outside the MaxFish economy, re-spawned whenever missing.
func (w *World) ensureChosen() {
	var sp *contract.Species
	for _, s := range w.species {
		if s.Role == contract.RoleChosen {
			sp = s
			break
		}
	}
	if sp == nil {
		return
	}
	for _, f := range w.fishes {
		if f.Sp.Role == contract.RoleChosen && !f.Dying {
			return // the eternal one endures
		}
	}
	// home is the aura center when the rock layout is installed
	home := v2(w.W*0.62, w.H*0.5)
	for _, z := range w.zones {
		if z.Owner == "chosen" {
			home = add(z.Center, v2(0, -60))
		}
	}
	w.fishes = append(w.fishes, newFish(sp, w.rng.Int63(), home, 8, w.nextID()))
	w.logf("life", "the eternal one glides into view")
}
