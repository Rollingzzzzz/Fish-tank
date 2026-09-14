// G95: the REC crop follows the living center of mass — giants anchor it,
// the school can pull it, the dead and hidden abstain.
package sim

import (
	"testing"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

func TestFocusXTracksTheBigFish(t *testing.T) {
	w := &World{W: 1000, H: 600}
	chain := func(x, y, n, seg float64) []contract.Vec2 {
		s := make([]contract.Vec2, int(n))
		for i := range s {
			s[i] = contract.Vec2{X: x + seg*float64(i), Y: y}
		}
		return s
	}
	if x := w.FocusX(); x != 500 {
		t.Fatalf("empty water reads %.0f, want the tank center 500", x)
	}
	// a giant (body 90) right of a minnow (body 20): weight 8100 vs 400
	big := &Fish{Pos: contract.Vec2{X: 800, Y: 300}, Spine: chain(750, 300, 10, 10)}
	minnow := &Fish{Pos: contract.Vec2{X: 200, Y: 300}, Spine: chain(180, 300, 5, 5)}
	w.fishes = []*Fish{big, minnow}
	if x := w.FocusX(); x < 700 {
		t.Fatalf("focus %.0f — the giant's weight was ignored", x)
	}
	minnow.Dying = true
	if x := w.FocusX(); x != 800 {
		t.Fatalf("a dying fish still voted: focus %.0f, want 800", x)
	}
	big.Hide01 = 1 // swallowed by a crag door — not drawn, not followed
	if x := w.FocusX(); x != 500 {
		t.Fatalf("a hidden fish still voted: focus %.0f, want 500", x)
	}
}
