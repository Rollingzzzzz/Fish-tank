// v0.2: decor + zones — placed corals (N1), sim zones (N2 caves / N3 aura),
// wild water mites (N7), rock seed and day counter. v0.3.8: the crustaceans
// (N4) were retired — their dark floor silhouettes read as clutter.
package sim

import (
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// absF is the package-local float absolute value.
func absF(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// Coral is a placed coral instance (mirrors Plant).
type Coral struct {
	Def *contract.CoralDesign
	X   float64
	Y   float64
	Sc  float64
}

// Mite is a wild water mite (N7) — ephemeral, never saved.

// miteCap scales the wild-mite budget with the tank area (v0.3.1).
func (w *World) miteCap() int {
	return min(int(float64(contract.MiteCap*w.density)+0.5), contract.MiteCapMax)
}

// coralCap gives the reef ~20% of the floor line (v0.3.2 composition).
func (w *World) coralCap() int {
	return clampI(int(w.W*contract.FloorShareCorals/contract.CoralBasePx+0.5), 4, 24)
}
