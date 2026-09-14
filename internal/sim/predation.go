// v1.1: the rare predation gate (G42) — split from titan.go for the line
// ceiling. The hunt belongs to the giant alone: starved for
// PredationSustain seconds, a tank-wide cooldown, a population floor, and
// the Chosen forever immune.
package sim

import (
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// tickPredation is the rare-hunt gate (G42): the giant alone, starving for
// PredationSustain seconds, with a tank-wide cooldown, a population floor
// and the Chosen forever immune.
func (w *World) tickPredation(dt float64) {
	if w.predCD > 0 {
		w.predCD -= dt
	}
	g := w.titanGiant()
	if g == nil {
		w.resetPredation()
		return
	}
	if w.predTgtID != "" {
		tgt := w.fishByID(w.predTgtID)
		if tgt == nil || tgt.Dying || tgt.Sp.Role != contract.RoleNormal {
			w.resetPredation()
			return
		}
		w.predT -= dt
		if hyp2(sub(tgt.Pos, g.Pos)) < g.bodyLen*0.16*1.2 {
			// the swallow: no corpse lingers, one log line, the belly resets
			tgt.Dying = true
			tgt.DieReason = "swallowed"
			tgt.fadeFast = true
			w.burst(tgt.Pos, g.Pal.Accent, 10)
			w.logf("nature", "the abyss feeds — a shadow swallows "+tgt.Sp.Name)
			w.strikeShock(tgt.Pos, contract.StrikeScareR2, "") // the impact reaches further
			g.eat()
			w.resetPredation()
			w.predCD = contract.PredationCD
		} else if w.predT <= 0 {
			w.resetPredation() // broke off — a shorter wait before the next try
			w.predCD = 60
		}
		return
	}
	if g.Satiety < contract.PredationSatiety {
		w.predHunger += dt
	} else {
		w.predHunger = 0
	}
	if w.predHunger < contract.PredationSustain || w.predCD > 0 {
		return
	}
	live := w.aliveFishes()
	if len(live) <= contract.PredationPopFloor {
		w.predCD = 60 // too few neighbors — the deep waits
		return
	}
	var best *Fish
	bestD := 1e18
	for _, f := range live {
		if f.Sp.Role != contract.RoleNormal {
			continue
		}
		if d := hyp2(sub(f.Pos, g.Pos)); d < bestD {
			bestD, best = d, f
		}
	}
	if best != nil {
		w.predTgtID = best.ID
		w.predT = contract.PredationPursuit
		w.strikeShock(g.Pos, contract.StrikeScareR, "the hunt opens — the water splits")
	}
}

func (w *World) resetPredation() {
	w.predTgtID, w.predT, w.predHunger = "", 0, 0
}

// satietyDecay is the seconds it takes a full belly to drain: 40 for normal
// fish, TitanSatietySec for giants (v1.1 — hunger drives the lunge).
func (f *Fish) satietyDecay() float64 {
	if f.Sp.Role == contract.RoleTitan {
		return contract.TitanSatietySec
	}
	return 40.0
}

// deathFadeRate is the seconds a dying fish takes to fade: swallowed prey
// (v1.1 predation) dissolves almost at once — no corpse lingers on screen.
func (f *Fish) deathFadeRate() float64 {
	if f.fadeFast {
		return 0.5
	}
	return contract.DeathFadeSec
}

// lungeSteer runs the per-fish hunger lunge (G41, retuned G82): a burst at
// TitanLungeMul × cruise — but ONLY at living food. The old version also
// struck at dead flakes and, with nothing in sight, threw a raw random
// direction dart (probe: ten darts in four hungry minutes — exactly the
// "sudden fast movement in place" glitch). A hungry elder with no live food
// simply holds the sweep; starvation eventually opens the rare hunt instead.
func (f *Fish) lungeSteer(w *World, dt, maxSp float64, addForce func(contract.Vec2, float64)) {
	if f.lungeCD > 0 {
		f.lungeCD -= dt
	}
	if f.lungeT > 0 {
		f.lungeT -= dt
		d := sub(f.lungePt, f.Pos)
		addForce(mulS(d, maxSp*f.burstMul()/maxF(hyp2(d), 1)), 4.5)
		f.seekBonus = f.burstMul()
		return
	}
	if f.lungeCD > 0 || f.Satiety >= 0.35 {
		return
	}
	var pt contract.Vec2
	found, bestD := false, 620.0
	for _, t := range w.treats {
		if d := hyp2(sub(t.Pos, f.Pos)); d < bestD {
			bestD, pt, found = d, t.Pos, true
		}
	}
	if !found {
		return // no living food in sight — the sweep holds its gravity
	}
	f.lungeCD = contract.TitanLungeCD
	if f.sizeMul >= 0.99 {
		w.logf("nature", "the giant dives — the water shivers")
	}
	f.lungePt = pt
	f.lungeT = contract.TitanLungeSec
	w.strikeShock(pt, contract.StrikeScareR, "the school parts before the dive")
}

// DebugForceTitanVisit pins the pod's next arrival to now — evidence
// harness only (-probe): it skips the visit cadence on purpose.
func (w *World) DebugForceTitanVisit() {
	w.titanPhase = 0
	w.titanT = 0
}
