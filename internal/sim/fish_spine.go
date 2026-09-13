// G2.1: the spine pass — the 14-point chain that makes every fish a fish
// (split from fish_steering.go for the line ceiling; v1.1 giants share it
// with everyone else).
package sim

import (
	"math"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

func mathSqrt(x float64) float64 { return math.Sqrt(x) }
func mathCos(a float64) float64  { return math.Cos(a) }
func mathSin(a float64) float64  { return math.Sin(a) }

// followSpine keeps segment lengths exactly and adds the swimming wave.
// v1.1: pod members and the hammerhead pair get a curvature clamp — each
// base segment may bend at most TitanSpineBend from its parent, so a large
// body sweeps in arcs and never folds into a hairpin (the v1 over-curl
// issue at shark/giant scale). Normals keep the v1 body untouched.
func (f *Fish) followSpine(dt float64) {
	f.Spine[0] = f.Pos
	maxSp := f.maxSpeed(f.curNight)
	speed01 := clampF(hyp2(f.Vel)/maxSp, 0, 1)
	clampBend := f.Sp.Role == contract.RoleTitan || f.Sp.Role == contract.RoleShark
	bend := contract.TitanSpineBend
	if f.turning > 0 {
		bend = contract.TitanSpineBendTurn // G58: the curl may use the full arc
	}
	px, py := 0.0, 0.0
	if v := hyp2(f.Vel); v > 1 {
		px, py = f.Vel.X/v, f.Vel.Y/v
	}
	for i := 1; i < len(f.Spine); i++ {
		d := sub(f.Spine[i], f.Spine[i-1])
		l := maxF(hyp2(d), 1e-6)
		bx, by := d.X/l, d.Y/l
		if clampBend && px != 0 {
			if dot := bx*px + by*py; dot < math.Cos(bend) {
				s := 1.0
				if cross := px*by - py*bx; cross < 0 {
					s = -1.0
				}
				cb, sb := math.Cos(bend), math.Sin(bend)
				bx, by = px*cb-py*s*sb, py*cb+px*s*sb
			}
		}
		px, py = bx, by
		// constrained base vector
		dx, dy := bx*f.segLen, by*f.segLen
		// swimming wave displacement (perpendicular to the segment)
		amp := f.segLen * 0.35 * (0.25 + 0.75*speed01)
		wave := sin(f.phase-float64(i)*0.55) * amp * (float64(i) / float64(len(f.Spine)-1))
		nx, ny := -dy/f.segLen, dx/f.segLen
		vx, vy := dx+nx*wave, dy+ny*wave
		vl := maxF(sqrt(vx*vx+vy*vy), 1e-6)
		f.Spine[i] = v2(f.Spine[i-1].X+vx/vl*f.segLen, f.Spine[i-1].Y+vy/vl*f.segLen)
	}
}
