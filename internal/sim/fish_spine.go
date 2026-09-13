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
// v1.1 G63: EVERY body is a rope in a cone — each base segment is clamped
// within its class bend of the swim axis (the velocity, or the face
// direction at a near standstill), so a fish may arc from tail to head but
// can never coil onto itself or whirl its bones like clock hands. Titans
// and the hammerheads keep their stiffer giant caps (the v1 over-curl
// issue at giant scale); the school closes its own v1 known issue here.
func (f *Fish) followSpine(dt float64) {
	f.Spine[0] = f.Pos
	maxSp := f.maxSpeed(f.curNight)
	speed01 := clampF(hyp2(f.Vel)/maxSp, 0, 1)
	bend := contract.SpineBendNormal
	if f.Sp.Role == contract.RoleTitan || f.Sp.Role == contract.RoleShark {
		bend = contract.TitanSpineBend
		if f.turning > 0 {
			bend = contract.TitanSpineBendTurn // G58: the curl may use the full arc
		}
	}
	// the TRAILING axis the chain must hang from: opposite the velocity, or
	// the direction the chain already hangs when the fish is at a standstill.
	// (Sign matters: referenced against the forward velocity the clamp would
	// re-lay every segment AHEAD of the head — the end-for-end inversion
	// that made eyes trail and tail fins lead like clock hands.)
	px, py := 0.0, 0.0
	if v := hyp2(f.Vel); v > 1 {
		px, py = -f.Vel.X/v, -f.Vel.Y/v
	} else if d := sub(f.Spine[1], f.Spine[0]); hyp2(d) > 1e-3 {
		u := 1 / hyp2(d)
		px, py = d.X*u, d.Y*u
	}
	for i := 1; i < len(f.Spine); i++ {
		d := sub(f.Spine[i], f.Spine[i-1])
		l := maxF(hyp2(d), 1e-6)
		bx, by := d.X/l, d.Y/l
		if px != 0 || py != 0 {
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

// capTurn is the forward law for the generic steering mix (G63): no fish
// may swim tail-first. The velocity direction may swing at most
// NormalTurnRate — a rear target or a startle bends the path into an arc
// instead of flipping it, so the spine, riding the motion, always reads
// nose-first. Titans keep their heading ease and the hammerhead its body
// axis pin; this gate covers every other fish in the tank.
func (f *Fish) capTurn(dt float64) {
	v := hyp2(f.Vel)
	if v < 4 {
		return // drifting — nothing to cap
	}
	va := math.Atan2(f.Vel.Y, f.Vel.X)
	da := math.Mod(va-f.headingA+3.14159, 6.28318) - 3.14159
	f.headingA += clampF(da, -contract.NormalTurnRate*dt, contract.NormalTurnRate*dt)
	f.Vel = mulS(v2(cos(f.headingA), sin(f.headingA)), v)
}
