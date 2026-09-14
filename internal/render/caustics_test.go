// v1.1 G75: caustic-field law tests — bounded output, real interference
// nodes, visible drift, and the frozen day/night gain.
package render

import (
	"math"
	"testing"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

func TestCausticFieldIsBoundedAndAlive(t *testing.T) {
	distinct := map[float64]bool{}
	mx := 0.0
	for x := 0.0; x <= 1720; x += 40 {
		for _, tt := range []float64{0, 3.7, 11.2} {
			v := Caustic01(x, 7, tt)
			if v < 0 || v > 1 {
				t.Fatalf("field escapes [0,1]: %.3f", v)
			}
			distinct[math.Round(v*20)/20] = true
			mx = math.Max(mx, v)
		}
	}
	if len(distinct) < 12 {
		t.Fatalf("field too flat: %d distinct brightness buckets", len(distinct))
	}
	if mx < 0.55 {
		t.Fatalf("no bright caustic nodes: max %.3f", mx)
	}
}

func TestCausticDrifts(t *testing.T) {
	changed, n := 0, 0
	for x := 0.0; x <= 1720; x += 30 {
		if math.Abs(Caustic01(x, 7, 0)-Caustic01(x, 7, 6)) > 0.02 {
			changed++
		}
		n++
	}
	if changed*2 < n {
		t.Fatalf("pattern reads as static: %d/%d samples moved", changed, n)
	}
}

func TestCausticNightLaw(t *testing.T) {
	if g := causticGain(0); g != 1 {
		t.Fatalf("day gain %.3f, want 1", g)
	}
	if g := causticGain(1); math.Abs(g-contract.CausticNightK) > 1e-9 {
		t.Fatalf("deep-night gain %.3f, want the moon fraction %.3f", g, contract.CausticNightK)
	}
}
