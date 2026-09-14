// N9: simulate generator supremacy test — simulated species inventions stay
// under the normal caps (size <= 1.16 across 500 seeded draws) so nothing the
// agents invent ever rivals the Chosen Lilastar.
package agents

import (
	"math"
	"math/rand"
	"path/filepath"
	"testing"

	"github.com/Rollingzzzzz/Fish-tank/internal/content"
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

func TestSimulateSpeciesUnderNormalCaps(t *testing.T) {
	st, err := content.Load(filepath.Join(t.TempDir(), "content"))
	if err != nil {
		t.Fatalf("content.Load: %v", err)
	}
	rng := rand.New(rand.NewSource(0xC0FFEE))
	maxSize, maxFin, maxTail := 0.0, 0.0, 0.0
	for i := 0; i < 500; i++ {
		sp := SimulateSpecies(rng, st)
		if sp.Size > contract.NormalSizeMax || sp.Size > 1.16 {
			t.Fatalf("draw %d size %v exceeds the normal cap %v", i, sp.Size, contract.NormalSizeMax)
		}
		if sp.Fin > contract.NormalFinMax {
			t.Fatalf("draw %d fin %v exceeds %v", i, sp.Fin, contract.NormalFinMax)
		}
		if sp.Tail > contract.NormalTailMax {
			t.Fatalf("draw %d tail %v exceeds %v", i, sp.Tail, contract.NormalTailMax)
		}
		if sp.Behavior.Speed > contract.NormalSpeedMax { // F23
			t.Fatalf("draw %d speed %v exceeds %v", i, sp.Behavior.Speed, contract.NormalSpeedMax)
		}
		maxSize = math.Max(maxSize, sp.Size)
		maxFin = math.Max(maxFin, sp.Fin)
		maxTail = math.Max(maxTail, sp.Tail)
	}
	t.Logf("max draws over 500: size=%.3f fin=%.3f tail=%.3f", maxSize, maxFin, maxTail)
}
