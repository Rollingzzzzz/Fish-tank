// F9: day-cycle LUT continuity + distinctness acceptance tests.
package render

import (
	"math"
	"testing"
)

func TestDayTintContinuous(t *testing.T) {
	// C0 check: consecutive samples of the LUT must never jump — no pops
	// between frames anywhere in the day cycle.
	const n = 4000
	pr, pg, pb, pk := dayTint(0)
	for i := 1; i <= n; i++ {
		r, g, b, k := dayTint(float64(i) / n)
		dr, dg, db, dk := math.Abs(r-pr), math.Abs(g-pg), math.Abs(b-pb), math.Abs(k-pk)
		if dr+dg+db+dk > 0.02 {
			t.Fatalf("LUT discontinuity at t=%.4f: delta %.4f", float64(i)/n, dr+dg+db+dk)
		}
		pr, pg, pb, pk = r, g, b, k
	}
}

func TestDayTintPhasesDistinct(t *testing.T) {
	// The four key phases must be clearly different moods.
	type rgb struct{ r, g, b float64 }
	phase := func(t float64) rgb {
		r, g, b, _ := dayTint(t)
		return rgb{r, g, b}
	}
	dist := func(a, b rgb) float64 {
		return math.Abs(a.r-b.r) + math.Abs(a.g-b.g) + math.Abs(a.b-b.b)
	}
	dawn, noon, dusk, night := phase(0), phase(0.25), phase(0.5), phase(0.75)
	pairs := map[string]float64{
		"dawn-noon": dist(dawn, noon), "dusk-night": dist(dusk, night),
		"dawn-night": dist(dawn, night), "noon-dusk": dist(noon, dusk),
	}
	for name, d := range pairs {
		if d < 0.3 {
			t.Fatalf("phases %s too similar: %.2f", name, d)
		}
	}
}
