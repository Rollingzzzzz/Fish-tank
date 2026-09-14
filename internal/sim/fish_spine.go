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
	} else {
		// G90: EVERY body hangs from its heading, school fish included.
		// The old velocity-axis flopped milling fish: below the turn-cap
		// floor (v < 4) the velocity direction is force-driven and can
		// spin 180° between frames, re-laying the whole chain to the other
		// side (5-18 px flops in a food frenzy). headingA is the same
		// direction while truly swimming (capTurn keeps them in sync) and
		// a CONTINUOUS variable everywhere else — a resting fish keeps the
		// orientation it last swam in, like a real one.
		px, py = -math.Cos(f.headingA), -math.Sin(f.headingA)
	}
	for i := 1; i < len(f.Spine); i++ {
		d := sub(f.Spine[i], f.Spine[i-1])
		l := maxF(hyp2(d), 1e-6)
		bx, by := d.X/l, d.Y/l
		clamped := false
		// swimming wave displacement (perpendicular to the segment),
		// computed FIRST so the bend cone below can account for its tilt —
		// a big body carries a calmer tail (G67): with the eye glued to the
		// line, the trailing tip may only shimmer, not wander off it
		amp := f.segLen * 0.35 * (0.25 + 0.75*speed01)
		if f.Sp.Role == contract.RoleTitan || f.Sp.Role == contract.RoleShark {
			amp *= 0.30
		}
		wave := sin(f.phase-float64(i)*0.55) * amp * (float64(i) / float64(len(f.Spine)-1))
		waveTilt := absF(wave) / maxF(f.segLen, 1e-6)
		if px != 0 || py != 0 {
			// G90: the cone the BASE direction must respect shrinks by the
			// wave's own tilt, so base bend + wave tilt stays inside the
			// designed silhouette — the slew slack then costs nothing.
			// The BIG bodies slew nothing: their 0.085 cone is the G67
			// design itself and the wave tilt there is ~the whole cone —
			// they keep the exact instant clamp (their own twitch class
			// was tamed by the G87 laws).
			effBend := bend
			slew := true
			if f.Sp.Role == contract.RoleTitan || f.Sp.Role == contract.RoleShark {
				effBend = maxF(bend-waveTilt, bend*0.25)
				slew = false
			} else {
				effBend = maxF(bend-waveTilt, bend*0.25)
			}
			if dot := bx*px + by*py; dot < math.Cos(effBend) {
				s := 1.0
				if cross := px*by - py*bx; cross < 0 {
					s = -1.0
				}
				// G90: a near-opposite direction has a noisy cross — the
				// chosen edge flips frame to frame and the joint MIRRORS
				// across the axis (the stationary tail twitch). Pin the
				// fold side to the swim wave: the tail resolves to the
				// side it is already waving toward.
				if dot < -0.2 {
					if w := sin(f.phase - float64(i)*0.55); absF(w) > 0.05 {
						s = 1.0
						if w < 0 {
							s = -1.0
						}
					}
				}
				cb, sb := math.Cos(effBend), math.Sin(effBend)
				tx, ty := px*cb-py*s*sb, py*cb+px*s*sb
				cur := math.Atan2(by, bx)
				tgt := math.Atan2(ty, tx)
				da := math.Mod(tgt-cur+3.14159, 6.28318) - 3.14159
				// G90: a GRAZE outside the cone SLEWS toward the edge
				// instead of snapping onto it — the instant rotation swung
				// a joint ~segLen·sin(bend) in one frame. The window is
				// half the effective cone: the transient total (base bend +
				// wave tilt) can never cross the hairpin budget of the
				// timelapse law (0.85 rad). Beyond the window is a genuine
				// FOLD and still returns in one frame.
				if slew && absF(da) <= 0.5*effBend {
					cur += clampF(da, -contract.SpineBendSlew*dt, contract.SpineBendSlew*dt)
					bx, by = math.Cos(cur), math.Sin(cur)
				} else {
					bx, by = tx, ty
				}
				// the CHILD measures its own cone against the edge this
				// segment is bound for, not against the easing actual —
				// otherwise every joint adds its own partial turn and the
				// slew compounds into a tail whip down the chain (G90)
				px, py = tx, ty
				clamped = true
			}
		}
		if !clamped {
			px, py = bx, by
		}
		// constrained base vector + the wave, laid at exact segment length
		dx, dy := bx*f.segLen, by*f.segLen
		nx, ny := -dy/f.segLen, dx/f.segLen
		vx, vy := dx+nx*wave, dy+ny*wave
		vl := maxF(sqrt(vx*vx+vy*vy), 1e-6)
		f.Spine[i] = v2(f.Spine[i-1].X+vx/vl*f.segLen, f.Spine[i-1].Y+vy/vl*f.segLen)
		// G90: the LAYOUT kink cap — the angle between two LAID neighbors
		// is what the eye sees and what the timelapse law measures; cap it
		// directly whatever the clamp, the slew and the wave did upstream.
		// School scale only: the giants' 0.085 cone never produces hairpins,
		// and rotating their 448 px tail would re-trigger the ceiling drape
		// shifts the G87 law just tamed.
		if i >= 2 && f.Sp.Role == contract.RoleNormal {
			p2 := f.Spine[i]
			capKinkBetween(&f.Spine[i-2], &f.Spine[i-1], &p2)
			f.Spine[i] = p2
		}
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

// capKinkBetween caps the layout angle of the pair (prev1→prev2) versus
// (prev2→p): the observed neighbor kink may never exceed LayoutKinkMax,
// whatever the clamp, the slew and the wave did upstream (G90). Rotation
// preserves the segment length.
func capKinkBetween(prev2, prev1, p *contract.Vec2) {
	pd := sub(*prev1, *prev2)
	pl := hyp2(pd)
	d := sub(*p, *prev1)
	dl := hyp2(d)
	if pl < 1e-6 || dl < 1e-6 {
		return
	}
	cosA := (d.X*pd.X + d.Y*pd.Y) / (dl * pl)
	if cosA >= math.Cos(contract.LayoutKinkMax) {
		return
	}
	sign := 1.0
	if d.X*pd.Y-d.Y*pd.X < 0 {
		sign = -1.0
	}
	ca, sa := math.Cos(contract.LayoutKinkMax), math.Sin(contract.LayoutKinkMax)
	ux, uy := pd.X/pl, pd.Y/pl
	tx := ux*ca - uy*sign*sa
	ty := ux*sign*sa + uy*ca
	if d.X*tx+d.Y*ty < 0 { // the segment lay on the far side — take the near edge
		tx, ty = ux*ca+uy*sa, -ux*sa+uy*ca
	}
	*p = v2(prev1.X+tx*dl, prev1.Y+ty*dl)
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
