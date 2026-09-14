// G2.1: steering forces (split from fish.go for the line ceiling).
package sim

import (
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// steer accumulates the behavior acceleration vector.
func (f *Fish) steer(dt, maxSp, night float64, w *World) contract.Vec2 {
	acc := v2(0, 0)
	// G81: a spent burst's speed allowance DECAYS (3.5/s) instead of
	// snapping back to cruise — the cap follows it down gently, so the
	// strict speed invariant holds while the deceleration reads as a
	// glide, never a mid-water brake (an eat that crosses Satiety 0.9
	// once snapped the cap in half mid-water — probe: 4244 px/s in a
	// single frame). A full fish simply never renews the drive; the
	// residual momentum glides off the same way.
	f.seekBonus = maxF(1.0, f.seekBonus-3.5*dt)
	addForce := func(desired contract.Vec2, weight float64) {
		acc.X += (desired.X - f.Vel.X) * weight
		acc.Y += (desired.Y - f.Vel.Y) * weight
	}

	// v1.1: pod members run their own ponderous mix entirely
	if f.Sp.Role == contract.RoleTitan {
		return f.steerTitan(dt, maxSp, w)
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
	// wander (smooth random heading) — quiet neighbors let schooling win.
	// G80: the noise's authority rides the fin flow (clamped to 15 % at a
	// standstill): a slow fish HOLDS its nose instead of being
	// weather-vaned by noise, so the "eyes fixed while the body turns
	// every direction" stationary spin is dead at the source — only a
	// coherent target force (food, scare, shelter) turns a slow fish, and
	// those turns read as intention.
	f.wanderA += (f.rng.Float64() - 0.5) * 2.4 * dt
	wAuth := clampF(hyp2(f.Vel)/(0.30*maxSp), 0.15, 1)
	wx, wy := cos(f.wanderA)*0.6*maxSp*wAuth, sin(f.wanderA)*0.6*maxSp*wAuth
	if f.Sp.Role == contract.RoleShark {
		wy *= contract.TitanHeadFlat // the hunter hugs the sand line too
	}
	addForce(v2(wx, wy), 0.55)

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
	var tgtCrit *Creature // v1.1: a struggling floor critter (G45)
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
		// v1.1 (G45): a struggling floor critter is living bait — hungry fish
		// feel it from CreatureLureRadius and it outranks mites and treats
		for _, c := range w.creatures {
			if !c.Vulnerable() {
				continue
			}
			if cd := hyp2(sub(c.Pos, f.Pos)); cd < tBestD && cd < contract.CreatureLureRadius {
				tBestD, treat = cd, (*Treat)(nil)
				best, tgtMite = nil, nil
				tgtCrit = c
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
	if best != nil || treat != nil || tgtMite != nil || tgtCrit != nil || held {
		tgt := f.Pos
		switch {
		case tgtCrit != nil:
			tgt = tgtCrit.Pos
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
		if treat != nil || held || tgtCrit != nil {
			foodW += 1.0 // live food triggers a mad dash
		}
		// G80: ARRIVE, but only where the target is a PLACE — the held
		// treat's keep-back ring. The old fixed 2×cruise desire overshot
		// the ring and chained U-turn flexes into a sustained low-speed
		// spin; a fish now settles onto the ring. Real food is hit at
		// speed — the dash never brakes for a flake it means to eat.
		approach := maxSp * 2.0
		if held && l < 130 {
			approach = maxF(approach*l/130, maxSp*0.55)
		}
		addForce(mulS(d, approach/l), foodW)
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

	// depth band preference + the shark's upper-80% preference (depth.go)
	f.depthBandSteer(w, night, maxSp, addForce)

	// N2: scared or tired fish shelter in the nearest cave (roam.go)
	if d, ok := f.shelterSteer(w, maxSp); ok {
		addForce(d, 1.0)
	}

	f.auraRepulsion(w, maxSp, addForce)

	// v1.1: flow around the big bodies — through a titan or the shark
	// nobody swims; the silhouette is gone around, never crossed
	f.avoidBigBodies(w, maxSp, addForce)

	f.wallMargins(w, maxSp, addForce)

	// clamp total force (seekBonus = F3's local allowance while food-chasing)
	l := hyp2(acc)
	cap := contract.MaxForce * f.seekBonus
	if l > cap {
		acc = mulS(acc, cap/l)
	}
	return acc
}
