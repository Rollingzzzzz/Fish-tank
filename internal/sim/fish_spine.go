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
	// G90: the visible body reads the EASED pace. Raw speed01 rescales the
	// wave amplitude and the beat in a single frame whenever a burst is
	// granted or spent — the tail hops sideways. One low-pass (4/s) keeps
	// the swim wave continuous through every speed change.
	f.speed01S += (speed01 - f.speed01S) * minF(1, 4*dt)
	speed01 = f.speed01S
	bend := contract.SpineBendNormal
	if f.Sp.Role == contract.RoleTitan || f.Sp.Role == contract.RoleShark {
		bend = contract.TitanSpineBend
		if f.turning > 0 {
			// G67: the arc opens wide mid-turn and TIGHTENS as it closes —
			// the chain relaxes back onto the line by itself instead of
			// staying wound after the turn
			p := 1 - f.turning/contract.TitanTurnWindow
			bend = contract.TitanSpineBendTurn * (1 - 0.75*p)
		}
	}
	// G67: the chain settles onto the line after the convoy arc — a leftover
	// C would linger for minutes at ponderous pace. Strong through the
	// arc's last stretch, then a light settle keeps the trailing body on
	// the axis while the turn clock runs; a few px per frame, never a snap.
	if f.Sp.Role == contract.RoleTitan && f.turnT > 0 {
		k := 0.0
		if f.turning > 0 {
			// eased in through the whole arc: the C opens gradually and the
			// convoy lands on its new line FINISHED, not still unwinding
			p := 1 - f.turning/contract.TitanTurnWindow
			k = minF(1, 2.2*dt) * (0.3 + 0.7*p)
		} else {
			k = minF(1, 4*dt)
		}
		if k > 0 {
			ax, ay := math.Cos(f.headingA), math.Sin(f.headingA)
			for j := range f.Spine {
				tx := f.Pos.X - ax*float64(j)*f.segLen
				ty := f.Pos.Y - ay*float64(j)*f.segLen
				f.Spine[j].X += (tx - f.Spine[j].X) * k
				f.Spine[j].Y += (ty - f.Spine[j].Y) * k
			}
		}
	}

	// the TRAILING axis the chain must hang from: opposite the velocity, or
	// the direction the chain already hangs when the fish is at a standstill.
	// (Sign matters: referenced against the forward velocity the clamp would
	// re-lay every segment AHEAD of the head — the end-for-end inversion
	// that made eyes trail and tail fins lead like clock hands.)
	px, py := 0.0, 0.0
	if f.Sp.Role == contract.RoleTitan {
		// the sweep heading owns the titan axis outright (G67): the arc and
		// the settle after it must not fight a velocity bent by a startle,
		// and straight-cruise heading follows the velocity anyway
		px, py = -math.Cos(f.headingA), -math.Sin(f.headingA)
	} else if v := hyp2(f.Vel); v > 1 {
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
				tx, ty := px*cb-py*s*sb, py*cb+px*s*sb
				// G90: a GRAZE outside the cone SLEWS toward the edge instead
				// of snapping onto it — the instant rotation swung a joint
				// ~segLen·sin(bend) in one frame (the ±3 px glitch at school
				// scale, worse on big bodies). A genuine FOLD (the head
				// overtook the tail — big external shoves) still returns in
				// one frame: a lingering fold reads as a broken rope.
				cur := math.Atan2(by, bx)
				tgt := math.Atan2(ty, tx)
				da := math.Mod(tgt-cur+3.14159, 6.28318) - 3.14159
				if da > -0.3 && da < 0.3 {
					cur += clampF(da, -contract.SpineBendSlew*dt, contract.SpineBendSlew*dt)
					bx, by = math.Cos(cur), math.Sin(cur)
				} else {
					bx, by = tx, ty
				}
			}
		}
		px, py = bx, by
		// constrained base vector
		dx, dy := bx*f.segLen, by*f.segLen
		// swimming wave displacement (perpendicular to the segment); a big
		// body carries a calmer tail (G67): with the eye glued to the line,
		// the trailing tip may only shimmer, not wander off it
		amp := f.segLen * 0.35 * (0.25 + 0.75*speed01)
		if f.Sp.Role == contract.RoleTitan || f.Sp.Role == contract.RoleShark {
			amp *= 0.30
		}
		wave := sin(f.phase-float64(i)*0.55) * amp * (float64(i) / float64(len(f.Spine)-1))
		nx, ny := -dy/f.segLen, dx/f.segLen
		vx, vy := dx+nx*wave, dy+ny*wave
		vl := maxF(sqrt(vx*vx+vy*vy), 1e-6)
		f.Spine[i] = v2(f.Spine[i-1].X+vx/vl*f.segLen, f.Spine[i-1].Y+vy/vl*f.segLen)
	}
}

// glideCap eases a speed down to its ceiling at ≤ 8 px/s per frame —
// whichever path drops the ceiling (a spent chase, an elder stage
// crossing, a vanished mite mid-frenzy) the speed GLIDES down inside
// the envelope instead of snapping mid-flight (G88).
func glideCap(sp, cap float64) float64 {
	if excess := sp - cap; excess > 0 {
		return (cap + excess - minF(excess, 8)) / sp
	}
	return 1
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
	// G80: COHERENCE gates the turn. A committed turn (food behind the
	// back, a startle, a U-turn) holds its correction sign and swings at
	// full rate. A heading signal that flip-flops frame to frame is
	// noise, and noise may only turn a slow fish slowly — that is what
	// killed the "eyes stay put while the body flails every direction"
	// stationary whip without touching anyone's agility.
	if da*f.lastDa < 0 {
		f.noiseT = 0.35
	}
	f.lastDa = da
	f.noiseT = maxF(0, f.noiseT-dt)
	rate := contract.NormalTurnRate
	if f.noiseT > 0 && v < f.bodyLen {
		rate *= 0.35
	}
	f.headingA += clampF(da, -rate*dt, rate*dt)
	f.Vel = mulS(v2(cos(f.headingA), sin(f.headingA)), v)
}

// applyFrameBounds is the impenetrable tank bounds (v1.0) plus the G66
// body margin: a fish can never leave the water, and the big residents
// keep their tall frames below the top edge — the drawn body must fit.
// G81: the inbound velocity component now decays (10/s) instead of being
// zeroed in one frame — the instant kill read as collision-response
// physics; the wall itself still stops the position dead, the easing only
// governs how the speed reads after contact.
func (f *Fish) applyFrameBounds(w *World, dt float64) {
	soft := maxF(0, 1-10*dt)
	if f.Pos.X < 8 {
		f.Pos.X = 8
		if f.Vel.X < 0 {
			f.Vel.X *= soft
		}
	}
	if f.Pos.X > w.W-8 {
		f.Pos.X = w.W - 8
		if f.Vel.X > 0 {
			f.Vel.X *= soft
		}
	}
	if f.Pos.Y < 8 {
		f.Pos.Y = 8
		if f.Vel.Y < 0 {
			f.Vel.Y *= soft
		}
	}
	if f.Pos.Y > w.H-8 {
		f.Pos.Y = w.H - 8
		if f.Vel.Y > 0 {
			f.Vel.Y *= soft
		}
	}
	if f.Sp.Role != contract.RoleTitan && f.Sp.Role != contract.RoleShark {
		return // short chains fit once the head is inside
	}
	// the tall frames ride below the top edge; the trailing cone's worst
	// dive excursion is caught by the final canvas clamp, so the hard
	// margin keeps only the body proper inside (G66/G67 — leaving room
	// for the convoy's up-curl to exist at cruise height)
	myTop := f.bodyLen*0.30 + 6
	if f.Pos.Y < myTop {
		f.Pos.Y = myTop
		if f.Vel.Y < 0 {
			f.Vel.Y *= soft
		}
	}
}

// clampBodyInFrame is the G66 guarantee pass: every spine point of a big
// body stays inside the canvas, every frame. G87: when the trailing cone
// pokes past an edge the WHOLE FISH shifts rigidly (a few px per frame)
// instead of the tail being pinned while the head keeps cruising — the
// pin-and-slide read as a snag with the body oscillating around the still
// eye. The fish glides level under the ceiling; the clamp itself remains
// the last invisible guarantee.
func (f *Fish) clampBodyInFrame(w *World) {
	shiftX, shiftY := 0.0, 0.0
	for i := range f.Spine {
		if f.Spine[i].X < 3 {
			shiftX = maxF(shiftX, 3-f.Spine[i].X)
		}
		if f.Spine[i].X > w.W-3 {
			shiftX = minF(shiftX, w.W-3-f.Spine[i].X)
		}
		if f.Spine[i].Y < 3 {
			shiftY = maxF(shiftY, 3-f.Spine[i].Y)
		}
		if f.Spine[i].Y > w.H-3 {
			shiftY = minF(shiftY, w.H-3-f.Spine[i].Y)
		}
	}
	// the rigid shift itself is bounded per frame — an unbounded jump is
	// just another teleport wearing a fix's clothes. G90: 5 px/frame was
	// still a visible hop at school scale; the cap now sits at
	// BodyShiftCapPx and the edge deficit closes over a few gliding frames.
	shiftX = clampF(shiftX, -contract.BodyShiftCapPx, contract.BodyShiftCapPx)
	shiftY = clampF(shiftY, -contract.BodyShiftCapPx, contract.BodyShiftCapPx)
	if shiftX != 0 || shiftY != 0 {
		for i := range f.Spine {
			f.Spine[i].X += shiftX
			f.Spine[i].Y += shiftY
		}
		f.Pos.X += shiftX
		f.Pos.Y += shiftY
	}
	for i := range f.Spine {
		f.Spine[i].X = clampF(f.Spine[i].X, 3, w.W-3)
		f.Spine[i].Y = clampF(f.Spine[i].Y, 3, w.H-3)
	}
}
