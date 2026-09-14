// G2.5: 3-tier rotating snapshots (minute/daily/weekly) with crash recovery.
// Exactly three files beside the exe: save_minute.json, save_daily.json,
// save_weekly.json. All writes atomic (tmp+rename); recovery picks the newest
// valid file in order minute→daily→weekly and skips wrong schema versions.
package sim

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// Save file names.
const (
	fileMinute = "save_minute.json"
	fileDaily  = "save_daily.json"
	fileWeekly = "save_weekly.json"
)

// Persist manages the rotating snapshot tiers.
type Persist struct {
	dir                   string
	lastMinute            time.Time
	lastDay               string
	lastWeek              string
	dailyDone, weeklyDone bool
}

// NewPersist seeds the tier timers from now.
func NewPersist(dir string, now time.Time) *Persist {
	day, week := rolloverDates(now)
	return &Persist{dir: dir, lastMinute: now, lastDay: day, lastWeek: week}
}

// Tick writes a tier snapshot when its interval elapsed.
func (p *Persist) Tick(w *World, now time.Time) error {
	if now.Sub(p.lastMinute) >= time.Duration(contract.SaveMinuteSec*float64(time.Second)) {
		p.lastMinute = now
		if err := p.write(w, fileMinute); err != nil {
			return err
		}
	}
	day, week := rolloverDates(now)
	if day != p.lastDay {
		p.lastDay = day
		p.dailyDone = false
	}
	if !p.dailyDone {
		if err := p.write(w, fileDaily); err != nil {
			return err
		}
		p.dailyDone = true
	}
	if week != p.lastWeek {
		p.lastWeek = week
		p.weeklyDone = false
	}
	if !p.weeklyDone {
		if err := p.write(w, fileWeekly); err != nil {
			return err
		}
		p.weeklyDone = true
	}
	return nil
}

// SnapshotNow writes an immediate minute-tier snapshot (big events, exit).
func (p *Persist) SnapshotNow(w *World) error {
	p.lastMinute = time.Now()
	return p.write(w, fileMinute)
}

// write atomically saves the world snapshot to dir/name.
func (p *Persist) write(w *World, name string) error {
	s := w.Snapshot()
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	tmp := filepath.Join(p.dir, name+".tmp")
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(p.dir, name))
}

// LoadLatest recovers the newest valid snapshot: minute→daily→weekly by
// modification time; wrong schema versions and corrupt files are skipped.
func LoadLatest(dir string) (contract.Save, error) {
	var out contract.Save
	type cand struct {
		path string
		mod  time.Time
	}
	var valid []cand
	for _, name := range []string{fileMinute, fileDaily, fileWeekly} {
		path := filepath.Join(dir, name)
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var s contract.Save
		if json.Unmarshal(b, &s) != nil || s.SchemaVersion > contract.SaveSchemaVersion {
			continue
		}
		if fi, err := os.Stat(path); err == nil {
			valid = append(valid, cand{path, fi.ModTime()})
		}
		_ = s
	}
	if len(valid) == 0 {
		return out, os.ErrNotExist
	}
	newest := valid[0]
	for _, c := range valid[1:] {
		if c.mod.After(newest.mod) {
			newest = c
		}
	}
	b, err := os.ReadFile(newest.path)
	if err != nil {
		return out, err
	}
	err = json.Unmarshal(b, &out)
	return out, err
}

// Snapshot exports the full live world state (contract.Save, schema v2).
func (w *World) Snapshot() contract.Save {
	s := contract.Save{
		SchemaVersion: contract.SaveSchemaVersion,
		Clock:         w.Clock,
		Day:           w.Day,
		RockSeed:      w.rockSeed,
		Care:          w.Care,
		WaterID:       w.WaterID,
		PlantIDs:      append([]string(nil), w.PlantIDs...),
		CoralIDs:      append([]string(nil), w.CoralIDs...),
		Creatures:     []contract.SavedCreature{}, // v0.3.8: crustaceans retired; field kept for save-file stability
		Fish:          make([]contract.SavedFish, 0, len(w.fishes)),
	}
	for _, f := range w.fishes {
		if f.Dying {
			continue // v0.3.8: dissolving fish are not tank history
		}
		if f.Sp.Role == contract.RoleTitan {
			continue // v1.1: the pod is a visit, never tank history
		}
		s.Fish = append(s.Fish, contract.SavedFish{
			SpeciesID: f.Sp.ID, Seed: f.Seed, AgeDays: f.AgeDays,
			Pos: f.Pos, Vel: f.Vel, Satiety: f.Satiety, Energy: f.Energy,
		})
	}
	for _, fd := range w.foods {
		s.Foods = append(s.Foods, contract.SavedFood{Pos: fd.Pos, Age: fd.Age})
	}
	for _, e := range w.eggs {
		s.Eggs = append(s.Eggs, contract.SavedEgg{SpeciesID: e.SpeciesID, Pos: e.Pos, Progress: e.Progress})
	}
	return s
}

// Restore rebuilds the world state from a snapshot. v1 snapshots migrate
// silently (Day estimated from Clock, RockSeed randomized, corals empty;
// saved crustaceans are parsed but ignored since v0.3.8) — nothing is ever
// deleted (D4). v0.3.1: positions written on a
// smaller canvas are rescaled into this one — the tank grew around them.
func (w *World) Restore(s contract.Save) error {
	sx := clampF(w.W/contract.DensityRefW, 1, 4)
	sy := clampF(w.H/contract.DensityRefH, 1, 4)
	rescale := func(p contract.Vec2) contract.Vec2 {
		return v2(clampF(p.X*sx, 8, w.W-8), clampF(p.Y*sy, 8, w.H-8))
	}
	w.fishes = w.fishes[:0]
	chosenSeen := false
	sharkSeen := 0
	for _, sf := range s.Fish {
		sp := w.storeSpecies(sf.SpeciesID)
		if sp == nil {
			continue
		}
		if sp.Role == contract.RoleTitan {
			// v1.1: the pod is seeded by ensurePod, never by a save — a hand-edited
			// entry can neither summon nor multiply the deep
			continue
		}
		if sp.Role == contract.RoleShark {
			// v1.1: residents persist, but never beyond the pair cap
			if sharkSeen >= contract.SharkMax {
				continue
			}
			sharkSeen++
		}
		// F23: the eternal one is ONE — legacy/edited saves carrying several
		// chosen entries keep only the first; the rest dissolve (logged).
		if sp.Role == contract.RoleChosen {
			if chosenSeen {
				w.logf("life", "a second eternal one dissolves — there is only one")
				continue
			}
			chosenSeen = true
		}
		age := sf.AgeDays
		if sp.Role == contract.RoleChosen {
			// F23: she restores at her prime — an elder-aged save can never
			// freeze her faded, stage-slowed or cull-eligible
			age = minF(age, 8)
		}
		f := newFish(sp, sf.Seed, rescale(sf.Pos), age, w.nextID())
		f.Vel = sf.Vel
		if hyp2(sf.Vel) > 1 {
			// G63: the body heading is not persisted — rebuild it from the
			// saved velocity so no fish opens a session tail-first
			f.headingA = math.Atan2(f.Vel.Y, f.Vel.X)
		}
		f.Satiety = clampF(sf.Satiety, 0, 1)
		f.Energy = clampF(sf.Energy, 0, 1)
		f.ElderP = 0
		f.followSpine(0)
		w.placeOutsideZones(f) // G92: restored residents re-enter outside her circle
		w.fishes = append(w.fishes, f)
	}
	w.foods = w.foods[:0]
	for _, sfd := range s.Foods {
		w.foods = append(w.foods, Food{Pos: rescale(sfd.Pos), Age: sfd.Age, Seed: w.rng.Float64() * 6.283})
	}
	w.eggs = w.eggs[:0]
	for _, se := range s.Eggs {
		w.eggs = append(w.eggs, Egg{SpeciesID: se.SpeciesID, Pos: rescale(se.Pos), Progress: se.Progress, Seed: w.rng.Int63()})
	}
	w.Clock = s.Clock
	w.Care = s.Care
	// v1 → v2 migration
	if s.SchemaVersion < 2 {
		w.Day = int(s.Clock / maxF(w.cfg.DaySeconds, 1))
		w.rockSeed = w.rng.Int63()
	} else {
		w.Day = s.Day
		w.rockSeed = s.RockSeed
		if w.rockSeed == 0 {
			w.rockSeed = w.rng.Int63()
		}
		w.corals = w.corals[:0]
		w.CoralIDs = append([]string(nil), s.CoralIDs...)
		// v0.3.8: s.Creatures is parsed but ignored — crustaceans are retired
	}
	// re-arm care thresholds below the loaded score
	for w.careThreshIdx < len(contract.CareThresholds) && w.Care >= contract.CareThresholds[w.careThreshIdx] {
		w.careThreshIdx++
	}
	if id := s.WaterID; id != "" {
		for _, wp := range w.waters {
			if wp.ID == id {
				w.ApplyWater(wp)
				w.WaterCur = w.WaterTgt
			}
		}
	}
	// re-place corals from the store defs, then re-assert the plant caps:
	// the render budget (N7) and the F14 fish-majority rule are checked
	// against the restored live school (the Chosen included)
	w.SetCorals(w.coralDefs)
	w.ensureChosen()         // FD9: the eternal one always endures
	w.enforcePlantMajority() // F14: legacy saves can carry unbounded plants
	w.logf("tank", "the tank remembers")
	return nil
}
