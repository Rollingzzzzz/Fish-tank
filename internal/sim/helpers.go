// G2.1: package-local vector math and scalar helpers (stdlib only).
package sim

import (
	"math"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// v2 builds a Vec2.
func v2(x, y float64) contract.Vec2 { return contract.Vec2{X: x, Y: y} }

// sub returns a-b.
func sub(a, b contract.Vec2) contract.Vec2 { return v2(a.X-b.X, a.Y-b.Y) }

// mulS scales a vector.
func mulS(a contract.Vec2, s float64) contract.Vec2 { return v2(a.X*s, a.Y*s) }

// hyp2 is the length of a vector.
func hyp2(a contract.Vec2) float64 { return math.Sqrt(a.X*a.X + a.Y*a.Y) }

func sqrt(x float64) float64 { return math.Sqrt(x) }

// minF / maxF / clampF micro-helpers.
func minF(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
func maxF(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
func clampF(v, lo, hi float64) float64 { return contract.Clamp(v, lo, hi) }

// add returns p+v.
func add(p, v contract.Vec2) contract.Vec2 { return v2(p.X+v.X, p.Y+v.Y) }

// cos / sin radians aliases for local readability.
func cos(a float64) float64 { return math.Cos(a) }
func sin(a float64) float64 { return math.Sin(a) }

// norm2 returns the unit vector of a (safe for zero length).
func norm2(a contract.Vec2) contract.Vec2 {
	l := maxF(hyp2(a), 1e-6)
	return v2(a.X/l, a.Y/l)
}

// fishByID looks a fish up by its sequential id.
func (w *World) fishByID(id string) *Fish {
	for _, f := range w.fishes {
		if f.ID == id {
			return f
		}
	}
	return nil
}
