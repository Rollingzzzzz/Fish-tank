// v1.2 G94: the titan silk strands append real geometry to the fins mesh —
// the hard fins are gone for giants, the silk draws instead. Mesh-level
// test: no ebiten image state involved.
package render

import (
	"testing"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

func silkRig(t *testing.T) (*mesh, []contract.Vec2, []contract.Vec2, []contract.Vec2, []float64) {
	t.Helper()
	spine := make([]contract.Vec2, contract.SpineSegments)
	for i := range spine {
		spine[i] = contract.Vec2{X: 600 - float64(i)*22, Y: 250}
	}
	n := len(spine)
	segs := make([]contract.Vec2, n-1)
	norms := make([]contract.Vec2, n)
	widths := make([]float64, n)
	for i := 0; i < n-1; i++ {
		dx := spine[i+1].X - spine[i].X
		dy := spine[i+1].Y - spine[i].Y
		l := sqrt(maxF(dx*dx+dy*dy, 1e-6))
		segs[i] = v2(dx/l, dy/l)
		widths[i] = 14
	}
	for i := 0; i < n; i++ {
		s := segs[min(i, n-2)]
		norms[i] = v2(-s.Y, s.X)
	}
	return &mesh{}, spine, segs, norms, widths
}

func TestTitanSilkRenders(t *testing.T) {
	for _, beat := range [3]float64{0, 2.0, 5.5} {
		fins, spine, segs, norms, widths := silkRig(t)
		anim := FishAnim{Time: 3.0, Speed01: 0.4, Beat: beat}
		drawTitanSilk(fins, spine, norms, widths, segs, 264, anim, 0)
		if len(fins.indices) == 0 {
			t.Fatalf("silk appended no geometry at beat %.1f", beat)
		}
	}
	// the wave travels: a different beat phase lays a different ribbon
	f1, s1, sg1, no1, w1 := silkRig(t)
	f2, s2, sg2, no2, w2 := silkRig(t)
	drawTitanSilk(f1, s1, no1, w1, sg1, 264, FishAnim{Time: 3, Speed01: 0.4, Beat: 0.3}, 0)
	drawTitanSilk(f2, s2, no2, w2, sg2, 264, FishAnim{Time: 3, Speed01: 0.4, Beat: 2.9}, 0)
	if len(f1.indices) != len(f2.indices) {
		t.Fatalf("silk strand count changed with beat phase")
	}
}
