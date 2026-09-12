// G2.3: life cycle — aging, stages, day/night clock, care score, courtship
// breeding and natural elder death.
package sim

import (
	"math"
	"time"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// mathMod keeps positive results for the clock wrap.
func mathMod(a, b float64) float64 { return math.Mod(math.Mod(a, b)+b, b) }

// StageFor maps tank-days to a life stage name (entry ages: fry 0, juvenile 2,
// adult 5, elder 12).
func StageFor(ageDays float64) string {
	switch {
	case ageDays < contract.StageDays["juvenile"]:
		return "fry"
	case ageDays < contract.StageDays["adult"]:
		return "juvenile"
	case ageDays < contract.StageDays["elder"]:
		return "adult"
	default:
		return "elder"
	}
}

// StageSpeed is the speed multiplier of a life stage.
func StageSpeed(stage string) float64 {
	if v, ok := contract.StageSpeedMul[stage]; ok {
		return v
	}
	return 1
}

// StageScale is the body-size multiplier of a life stage.
func StageScale(stage string) float64 {
	switch stage {
	case "fry":
		return 0.48
	case "juvenile":
		return 0.72
	case "elder":
		return 1.04
	default:
		return 1.0
	}
}

// tickLife advances aging, stage transitions, death and care for one fish.
func (w *World) tickLife(f *Fish, dt float64) {
	chosen := f.Sp.Role == contract.RoleChosen // FD9: the eternal one
	if !chosen {
		f.AgeDays += dt / maxF(w.cfg.DaySeconds, 1)
		stage := StageFor(f.AgeDays)
		if stage != f.Stage {
			f.prevStage = f.Stage
			f.Stage = stage
			w.onStageChanged(f)
		}
	}
	// F7: elder fade progress (0 until elder, →1 at the natural death age).
	// F23: the Chosen never fades — elder progress is mortal business.
	if f.Stage == "elder" && !chosen {
		f.ElderP = clampF((f.AgeDays-contract.StageDays["elder"])/contract.DeathAfterElderDays, 0, 1)
	}
	// natural death: elder + DeathAfterElderDays, stretched by care (F11),
	// never below the population floor (FD11)
	if !chosen && !f.Dying && f.Stage == "elder" && f.AgeDays >= w.elderThreshold() {
		if len(w.aliveFishes()) <= contract.MinPopulation {
			if !f.SparedOnce {
				f.SparedOnce = true
				w.logf("care", f.Sp.Name+" is spared — the tank needs its elders")
			}
		} else {
			f.Dying = true
			f.DieReason = "old age"
			w.logf("care", f.Sp.Name+" has grown very old")
		}
	}
	if f.Dying {
		f.Fade -= dt / contract.DeathFadeSec
	}
	// eat flash decay
	if f.EatFlash > 0 {
		f.EatFlash -= dt
	}
}

// elderThreshold is the age (tank-days) at which an elder naturally dies:
// 12 + DeathAfterElderDays, plus up to ElderLifespanBonus extra days for a
// well-cared-for tank (F11: good feeding → fewer deaths).
func (w *World) elderThreshold() float64 {
	care01 := clampF(w.Care/80, 0, 1)
	return contract.StageDays["elder"] + contract.DeathAfterElderDays + care01*contract.ElderLifespanBonus
}

// cullOldestElder makes room for a stalled egg (F6): the oldest elder past
// the death threshold dies — but never below the population floor.
func (w *World) cullOldestElder() {
	if len(w.aliveFishes()) <= contract.MinPopulation {
		return
	}
	var best *Fish
	for _, f := range w.aliveFishes() {
		// F23: the Chosen is unculled — hatching pressure never touches her
		if f.Sp.Role == contract.RoleChosen {
			continue
		}
		if f.Stage == "elder" && !f.Dying && f.AgeDays >= w.elderThreshold() {
			if best == nil || f.AgeDays > best.AgeDays {
				best = f
			}
		}
	}
	if best != nil {
		best.Dying = true
		best.DieReason = "old age"
		w.logf("life", best.Sp.Name+" has grown very old")
	}
}

// tickCare accrues the care score from watching (or AutoCare, F6) and fires
// courtship breeding: the threshold ladder for early game, then a recurring
// cooldown while care stays above tier 2 — breeding never runs out.
func (w *World) tickCare(dt float64) {
	if w.input.MouseActive || w.cfg.AutoCare {
		w.watchAcc += dt
		if w.watchAcc >= contract.CareWatchTick {
			w.watchAcc -= contract.CareWatchTick
			w.addCare(1)
		}
	}
	for w.careThreshIdx < len(contract.CareThresholds) &&
		w.Care >= contract.CareThresholds[w.careThreshIdx] {
		w.careThreshIdx++
		w.courtCD = contract.CourtshipCooldownSec
		w.startCourtship(false)
	}
	// recurring courtships (F6): well-fed tanks keep breeding
	w.courtCD -= dt
	if w.courtCD > 0 {
		return
	}
	switch {
	case len(w.aliveFishes()) < contract.MinPopulation:
		// FD11 self-heal: below the floor the tank calls for new life —
		// elders may breed too (desperate times relax the age rule)
		if w.startCourtship(true) {
			w.logf("life", "the tank calls for new life")
		}
		w.courtCD = 12
	case w.Care >= contract.CareThresholds[1] && len(w.aliveFishes()) < w.popCap:
		care01 := clampF(w.Care/80, 0, 1)
		if w.startCourtship(false) {
			w.courtCD = contract.CourtshipCooldownSec * (1.6 - 0.8*care01)
		} else {
			w.courtCD = 20 // no free pair right now — check again soon
		}
	case w.Care >= contract.CareThresholds[1]:
		// v0.3.8: a FULL tank breeds no more — the old hatch→cull→fade churn
		// kept dozens of corpses dissolving on screen and cost real frames
		w.courtCD = 15
	default:
		w.courtCD = 8
	}
}

// startCourtship pairs two conspecific adults and lets them circle.
// relaxed allows elders (F11 self-heal below the population floor).
// Returns true when a pair formed.
func (w *World) startCourtship(relaxed bool) bool {
	bySpecies := map[string][]*Fish{}
	for _, f := range w.aliveFishes() {
		ageOK := f.Stage == "adult" || f.Stage == "juvenile" ||
			(relaxed && f.Stage == "elder")
		if f.CourtID == "" && ageOK && !f.Dying &&
			f.Sp.Role != contract.RoleChosen { // FD9: the Chosen never breeds
			bySpecies[f.Sp.ID] = append(bySpecies[f.Sp.ID], f)
		}
	}
	for _, pair := range bySpecies {
		if len(pair) >= 2 {
			a, b := pair[0], pair[1]
			c := v2((a.Pos.X+b.Pos.X)/2, (a.Pos.Y+b.Pos.Y)/2)
			a.loungeT, b.loungeT = 0, 0 // F15: love evicts the cave
			a.CourtID, b.CourtID = b.ID, a.ID
			a.CourtC, b.CourtC = c, c
			a.CourtT, b.CourtT = contract.CourtshipSec, contract.CourtshipSec
			a.CourtAng, b.CourtAng = 0, 3.14159
			w.courtships = append(w.courtships, &Courtship{a: a, b: b})
			w.logf("life", a.Sp.Name+" pair is courting")
			return true
		}
	}
	return false
}

// tickCourtships advances circling pairs and spawns eggs at completion.
func (w *World) tickCourtships(dt float64) {
	kept := w.courtships[:0]
	for _, c := range w.courtships {
		c.t += dt
		a, b := c.a, c.b
		if a.CourtID == "" || b.CourtID == "" || a.Dying || b.Dying {
			a.CourtID, b.CourtID = "", ""
			continue
		}
		a.CourtT -= dt
		b.CourtT -= dt
		if a.CourtT <= 0 {
			a.CourtID, b.CourtID = "", ""
			w.spawnEggs(a, 1+int(w.rng.Float64()*2))
			w.logf("life", a.Sp.Name+" laid eggs")
			continue
		}
		kept = append(kept, c)
	}
	w.courtships = kept
}

// dayFactor is 1 at day, 0 at night with smooth 20% transitions.
func (w *World) dayFactor() float64 {
	cycle := 2 * maxF(w.cfg.DaySeconds, 5)
	phase := mathMod(w.Clock, cycle)
	day := float64(cycle) * 0.5
	tr := day * 0.2
	switch {
	case phase < day-tr:
		return 1
	case phase < day+tr:
		u := (phase - (day - tr)) / (2 * tr) // 1→0
		return 1 - u
	case phase < cycle-tr:
		return 0
	default:
		u := (phase - (cycle - tr)) / (2 * tr) // 0→1
		return u
	}
}

// TimeOfDay reports the HUD clock: full tank days elapsed, in-cycle hours
// (0..24 over the day+night cycle) and minutes, and whether it is night.
func (w *World) TimeOfDay() (day, hh, mm int, night bool) {
	day = w.Day
	cycle := 2 * maxF(w.cfg.DaySeconds, 5)
	phase := mathMod(w.Clock, cycle) / cycle // 0..1 over day+night
	mins := int(phase * 24 * 60)
	hh, mm = mins/60, mins%60
	night = w.dayFactor() < 0.5
	return day, hh, mm, night
}

// rolloverDates tracks calendar day / week changes for the save tiers.
func rolloverDates(now time.Time) (day, week string) {
	day = now.Format("2006-01-02")
	year, wk := now.ISOWeek()
	week = now.Format("2006") + "-W" + pad2(wk)
	_ = year
	return day, week
}

func pad2(v int) string {
	if v < 10 {
		return "0" + itoa(v)
	}
	return itoa(v)
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var b [20]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
