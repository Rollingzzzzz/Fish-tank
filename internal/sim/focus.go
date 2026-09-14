// G95: the REC capture pans with the action — FocusX is the screen-space
// x of the tank's living center of mass. Every visible fish votes at its
// body length squared, so the giants anchor the frame while a feeding
// school can still pull the window; hidden and dying fish abstain.
package sim

import "math"

// FocusX returns the weighted centroid x of the living school (the tank
// center when the water is empty).
func (w *World) FocusX() float64 {
	var sum, wsum float64
	for _, f := range w.fishes {
		if f.Dying || f.Hide01 > 0 {
			continue
		}
		bl := 0.0
		for i := 1; i < len(f.Spine); i++ {
			dx := f.Spine[i].X - f.Spine[i-1].X
			dy := f.Spine[i].Y - f.Spine[i-1].Y
			bl += math.Sqrt(dx*dx + dy*dy)
		}
		wt := bl * bl
		if wt < 1 {
			wt = 1
		}
		sum += f.Pos.X * wt
		wsum += wt
	}
	if wsum == 0 {
		return w.W / 2
	}
	return sum / wsum
}
