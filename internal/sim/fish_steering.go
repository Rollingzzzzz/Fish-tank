// G2.1: steering forces (split from fish.go for the line ceiling).
package sim

import (
	"math"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

func mathSqrt(x float64) float64 { return math.Sqrt(x) }
func mathCos(a float64) float64  { return math.Cos(a) }
func mathSin(a float64) float64  { return math.Sin(a) }

// steer accumulates the behavior acceleration vector.
func (f *Fish) steer(dt, maxSp, night float64, w *World) contract.Vec2 {
	acc := v2(0, 0)
	f.seekBonus = 1.0
	addForce := func(desired contract.Vec2, weight float64) {
		acc.X += (desired.X - f.Vel.X) * weight
		acc.Y += (desired.Y - f.Vel.Y) * weight
	}

	// F15: lounging replaces the whole steering mix with the cave hold —
	// courtship, chases and zoomies cut through it when they start
	if f.loungeT > 0 && f.CourtID == "" && f.chaseT <= 0 && f.zoomT <= 0 {
		return f.steerLounge(maxSp, w)
	}

	// F25: swimming into a volcanic door replaces the mix the same way
	if f.transiting && f.transitPh == 0 && f.CourtID == "" && f.chaseT <= 0 && f.zoomT <= 0 {
		return f.steerTransit(maxSp)
	}
	// wander (smooth random heading) — quiet neighbors let schooling win
	f.wanderA += (f.rng.Float64() - 0.5) * 2.4 * dt
	addForce(v2(cos(f.wanderA)*0.6*maxSp, sin(f.wanderA)*0.6*maxSp), 0.55)

	// schooling (boids, same species)
	if f.Sp.Behavior.Schooling > 0.05 {
		var coh, ali, sep contract.Vec2
		cnt, sepCnt := 0, 0
		gatherR := contract.SchoolRadius * 2.5 // distant kin still pull together
		for _, o := range w.fishes {
			if o == f || o.Sp.ID != f.Sp.ID || o.Dying {
				continue
			}
			// v0.3.8: fish parked in a cave or mid-transit don't drag the
			// school toward the towers — the centroid counts free swimmers
			if o.loungeT > 0 || (o.transiting && o.transitPh == 1) {
				continue
			}
			dx, dy := o.Pos.X-f.Pos.X, o.Pos.Y-f.Pos.Y
			d := hyp2(v2(dx, dy))
			if d > gatherR || d < 1e-3 {
				continue
			}
			coh.X += dx
			coh.Y += dy
			cnt++
			if d <= contract.SchoolRadius {
				ali.X += o.Vel.X
				ali.Y += o.Vel.Y
			}
			if d < 26*contract.Clamp(f.Sp.Size, 0.5, 1.5) {
				sep.X -= dx / d
				sep.Y -= dy / d
				sepCnt++
			}
		}
		s := contract.Clamp(f.Sp.Behavior.Schooling, 0, 1)
		if cnt > 0 {
			// head for the centroid at cruise speed (normalized, strong)
			toC := mulS(coh, 1/float64(cnt))
			lc := maxF(hyp2(toC), 1)
			addForce(mulS(toC, maxSp*0.85/lc), 0.95*s) // v0.3.8: was 1.15 — roam beats huddle
			addForce(mulS(ali, 1/float64(cnt)), 0.5*s)
		}
		if sepCnt > 0 {
			addForce(mulS(sep, maxSp), 1.5)
		}
	}

	// food seek — the nearest flake inside sense radius wins the race.
	// F3: hunger raises urgency (weight + desired speed + force allowance),
	// so a starving fish visibly sprints for food instead of drifting.
	foodW, treat := 0.0, (*Treat)(nil)
	var best *Food
	var tgtMite *Mite
	if f.Satiety < 0.9 {
		hunger := clampF(1-f.Satiety, 0, 1)
		// F3: perception widens with curiosity and hunger — a starving fish
		// smells flakes from across the tank.
		bestD := contract.FoodSenseFloor + 60*contract.Clamp(f.Sp.Behavior.Curiosity, 0, 1) + 190*hunger
		for i := range w.foods {
			fd := hyp2(sub(w.foods[i].Pos, f.Pos))
			if fd < bestD {
				bestD, best = fd, &w.foods[i]
			}
		}
		// live treats (N8) outrank flakes and pull from further away
		tBestD := bestD * 1.5
		for _, t := range w.treats {
			td := hyp2(sub(t.Pos, f.Pos))
			if td < tBestD {
				tBestD, treat = td, t
			}
		}
		// water mites (N7) trigger the strongest frenzy of all
		for _, m := range w.mites {
			md := hyp2(sub(m.Pos, f.Pos))
			if md < tBestD {
				tBestD, treat = md, (*Treat)(nil)
				best = nil
				tgtMite = m
			}
		}
		if treat != nil {
			best = nil
		} else if tgtMite == nil {
			// keep flake target
		} else {
			best = nil
		}
	}
	// N11: the hook-treat — a wriggling treat held at the cursor pulls hungry
	// fish onto the keep-back ring. It outranks flakes and tank treats, but a
	// water mite in reach may still win the fish's attention.
	heldPt, held := w.heldTreatTarget(f)
	if held && tgtMite == nil {
		best, treat = nil, nil
	}
	if best != nil || treat != nil || tgtMite != nil || held {
		tgt := f.Pos
		switch {
		case tgtMite != nil:
			tgt = tgtMite.Pos
		case held:
			tgt = *heldPt
		case treat != nil:
			tgt = treat.Pos
		default:
			tgt = best.Pos
		}
		hunger := clampF(1-f.Satiety, 0, 1)
		foodW = 3.0 + 1.5*hunger
		d := sub(tgt, f.Pos)
		l := maxF(hyp2(d), 1)
		if treat != nil || held {
			foodW += 1.0 // live food triggers a mad dash
		}
		addForce(mulS(d, maxSp*2.0/l), foodW)
		f.seekBonus = 2.0 // F3: local force + speed allowance while chasing
		if held {
			// N11: excited — the brief speed raise caps at ×HeldTreatSpeed
			f.seekBonus = contract.HeldTreatSpeed
		}
	}

	// cursor curiosity — a gentle approach when not starving
	if w.input.MouseActive && f.Satiety > 0.35 && f.Sp.Behavior.Curiosity > 0.1 {
		d := sub(v2(w.input.MouseX, w.input.MouseY), f.Pos)
		dist := hyp2(d)
		if dist < 170 && dist > 46 {
			addForce(mulS(d, maxSp*0.5/maxF(dist, 1)), 0.55*contract.Clamp(f.Sp.Behavior.Curiosity, 0, 1))
		} else if dist <= 46 {
			// orbit the cursor playfully
			tan := v2(-d.Y, d.X)
			addForce(mulS(tan, maxSp*0.45/maxF(hyp2(tan), 1)), 0.5)
		}
	}

	// rest near plants when tired and not hungry
	if f.Energy < 0.28 && f.Satiety > 0.3 {
		if f.restTarget == nil {
			f.restTarget = w.nearestPlantAnchor(f.Pos)
		}
		if f.restTarget != nil {
			d := sub(*f.restTarget, f.Pos)
			if hyp2(d) > 30 {
				addForce(mulS(d, maxSp*0.35/maxF(hyp2(d), 1)), 1.0)
			} else {
				f.Resting = true
			}
		}
	}
	if f.Resting {
		if f.Energy > 0.92 || f.Satiety < 0.3 {
			f.Resting = false
			f.restTarget = nil
		}
		bob := sin(w.time*1.4+float64(f.Seed%7)) * 6
		addForce(v2(0, bob-f.Vel.Y*2), 0.4)
	}

	// courtship circling
	if f.CourtID != "" {
		toC := sub(f.CourtC, f.Pos)
		r := hyp2(toC)
		tan := v2(-toC.Y, toC.X)
		desired := add(mulS(toC, 1.2), mulS(tan, maxSp*0.55/maxF(r, 20)*20))
		addForce(desired, 2.0)
	}

	// F4: bully chase — pursue hard, break off before contact
	if f.chaseT > 0 {
		if o := w.fishByID(f.chaseID); o != nil {
			d := sub(o.Pos, f.Pos)
			l := maxF(hyp2(d), 1)
			if l < 26 {
				f.chaseT = 0 // break-off point
			} else {
				addForce(mulS(d, maxSp*1.6/l), 1.8)
				f.seekBonus = maxF(f.seekBonus, 1.4)
			}
		}
	}
	// F4: zoomies — a fast celebratory loop
	if f.zoomT > 0 {
		f.zoomAng += dt * 5.5
		tgt := v2(f.zoomC.X+cos(f.zoomAng)*46, f.zoomC.Y+sin(f.zoomAng)*46)
		addForce(mulS(sub(tgt, f.Pos), 2.2), 1.4)
		f.seekBonus = maxF(f.seekBonus, 1.5)
	}

	// v0.3.8 tour: idle cruisers sweep the whole tank on soft random waypoints
	// (gating in roam.go). A fish chasing food skips the tour — the chase is
	// a straight, committed rush.
	if foodW == 0 {
		if d, ok := f.tourSteer(w, dt, maxSp); ok {
			addForce(d, 0.9)
		}
	}

	// depth band preference (shifts up at night for night-active species)
	band := contract.Clamp(f.Sp.Behavior.Depth, 0, 1)
	if f.Sp.Behavior.NightActive {
		band = clampF(band-0.25*night, 0, 1)
	}
	prefY := w.H * (0.18 + 0.62*band)
	addForce(v2(0, (prefY-f.Pos.Y)*0.25), 0.25)

	// N2: scared or tired fish shelter in the nearest cave (roam.go)
	if d, ok := f.shelterSteer(w, maxSp); ok {
		addForce(d, 1.0)
	}

	// N3 aura repulsion. F27: the Chosen is exempt — her own nest never
	// pushes her out.
	for _, z := range w.zones {
		if z.Owner != "chosen" || f.Sp.Role == contract.RoleChosen {
			continue
		}
		d := sub(f.Pos, z.Center)
		if l := hyp2(d); l < z.Radius+40 && l > 1 {
			addForce(mulS(d, maxSp/l), 1.5)
		}
	}

	// wall margins
	const mgn = 60.0
	if f.Pos.X < mgn {
		addForce(v2(maxSp, 0), 1.2)
	}
	if f.Pos.X > w.W-mgn {
		addForce(v2(-maxSp, 0), 1.2)
	}
	if f.Pos.Y < mgn*0.7 {
		addForce(v2(0, maxSp), 1.2)
	}
	if f.Pos.Y > w.H-mgn*0.6 {
		addForce(v2(0, -maxSp), 1.2)
	}

	// clamp total force (seekBonus = F3's local allowance while food-chasing)
	l := hyp2(acc)
	cap := contract.MaxForce * f.seekBonus
	if l > cap {
		acc = mulS(acc, cap/l)
	}
	return acc
}

// followSpine keeps segment lengths exactly and adds the swimming wave.
func (f *Fish) followSpine(dt float64) {
	f.Spine[0] = f.Pos
	maxSp := f.maxSpeed(f.curNight)
	speed01 := clampF(hyp2(f.Vel)/maxSp, 0, 1)
	for i := 1; i < len(f.Spine); i++ {
		d := sub(f.Spine[i], f.Spine[i-1])
		l := maxF(hyp2(d), 1e-6)
		// constrained base vector
		dx, dy := d.X/l*f.segLen, d.Y/l*f.segLen
		// swimming wave displacement (perpendicular to the segment)
		amp := f.segLen * 0.35 * (0.25 + 0.75*speed01)
		wave := sin(f.phase-float64(i)*0.55) * amp * (float64(i) / float64(len(f.Spine)-1))
		nx, ny := -dy/f.segLen, dx/f.segLen
		vx, vy := dx+nx*wave, dy+ny*wave
		vl := maxF(sqrt(vx*vx+vy*vy), 1e-6)
		f.Spine[i] = v2(f.Spine[i-1].X+vx/vl*f.segLen, f.Spine[i-1].Y+vy/vl*f.segLen)
	}
}
