// F23: the eternal one — structurally fastest at every hour, prime forever,
// exactly one. These pin the guarantees the audits found missing: no
// night-active normal may ever outsprint her, and no restore/cull path can
// age, fade, duplicate or kill her.
package sim

import (
	"testing"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

func chosenTestSpecies() *contract.Species {
	sp := testSpecies(0)
	c := *sp
	c.ID = "test-chosen"
	c.Role = contract.RoleChosen
	c.Behavior.Speed = 1.6 // her seeded stat (chosen-lilastar)
	return &c
}

func cappedNormalSpecies(id string, nightActive bool) *contract.Species {
	sp := testSpecies(0)
	c := *sp
	c.ID = id
	c.Role = contract.RoleNormal
	c.Behavior.Speed = contract.NormalSpeedMax // the legal ceiling for normals
	c.Behavior.NightActive = nightActive
	return &c
}

func TestChosenFastestAtAllHours(t *testing.T) {
	ch := newFish(chosenTestSpecies(), 1, v2(100, 100), 8, 1)
	normals := []*contract.Species{
		cappedNormalSpecies("worst-night", true),
		cappedNormalSpecies("worst-day", false),
	}
	for _, night := range []float64{0, 0.5, 1} {
		cs := ch.maxSpeed(night)
		for _, ns := range normals {
			nf := newFish(ns, 2, v2(200, 100), 6, 2)
			for _, stage := range contract.StageOrder {
				nf.Stage = stage
				if got := nf.maxSpeed(night); cs <= got*1.15 {
					t.Fatalf("night=%.1f vs %s/%s: chosen %.1f not ≥15%% above %.1f",
						night, ns.ID, stage, cs, got)
				}
			}
		}
	}
}

func TestChosenElderKeepsPrimePaceAndNeverFades(t *testing.T) {
	prime := newFish(chosenTestSpecies(), 1, v2(100, 100), 8, 1)
	elder := newFish(chosenTestSpecies(), 1, v2(100, 100), 8, 2)
	elder.Stage = "elder" // a restored elder-aged save
	elder.AgeDays = 30
	if elder.maxSpeed(1) != prime.maxSpeed(1) {
		t.Fatalf("elder chosen %.1f != prime %.1f", elder.maxSpeed(1), prime.maxSpeed(1))
	}
	// elder fade never accumulates on her, however long she lives
	w := testWorld(t, chosenTestSpecies(), 1)
	she := w.fishes[0]
	she.Stage = "elder"
	she.AgeDays = 30
	for i := 0; i < 300; i++ {
		w.tickLife(she, 1/30.0)
	}
	if she.ElderP != 0 || she.Dying {
		t.Fatalf("the eternal one aged: ElderP=%.2f Dying=%v", she.ElderP, she.Dying)
	}
}

func TestRestoreDedupesChosen(t *testing.T) {
	sp := testSpecies(0)
	ch := chosenTestSpecies()
	w := NewWorld(800, 600, contract.Config{MaxFish: 20, DaySeconds: 60},
		[]*contract.Species{sp, ch}, nil, nil)
	s := contract.Save{SchemaVersion: 2}
	for i := 0; i < 2; i++ { // a legacy/edited save with TWO eternals
		s.Fish = append(s.Fish, contract.SavedFish{
			SpeciesID: ch.ID, AgeDays: 8, Pos: v2(300, 300), Satiety: 0.5, Energy: 1,
		})
	}
	for i := 0; i < 3; i++ {
		s.Fish = append(s.Fish, contract.SavedFish{
			SpeciesID: sp.ID, AgeDays: 6, Pos: v2(320, 300), Satiety: 0.5, Energy: 1,
		})
	}
	if err := w.Restore(s); err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, f := range w.fishes {
		if f.Sp.Role == contract.RoleChosen {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("dedupe: %d chosen after restore, want 1", n)
	}
}

func TestRestoredElderChosenStaysPrime(t *testing.T) {
	sp := testSpecies(0)
	ch := chosenTestSpecies()
	w := NewWorld(800, 600, contract.Config{MaxFish: 20, DaySeconds: 60},
		[]*contract.Species{sp, ch}, nil, nil)
	s := contract.Save{SchemaVersion: 2}
	s.Fish = append(s.Fish, contract.SavedFish{
		SpeciesID: ch.ID, AgeDays: 30, Pos: v2(300, 300), Satiety: 0.5, Energy: 1,
	})
	if err := w.Restore(s); err != nil {
		t.Fatal(err)
	}
	var she *Fish
	for _, f := range w.fishes {
		if f.Sp.Role == contract.RoleChosen {
			she = f
		}
	}
	if she == nil {
		t.Fatal("the Chosen was not restored")
	}
	if she.Stage != "adult" {
		t.Fatalf("restored stage %q, want adult (her prime)", she.Stage)
	}
	if she.ElderP != 0 {
		t.Fatalf("restored ElderP %.2f, want 0", she.ElderP)
	}
	// soak: hatching pressure and time never cull or fade her
	for i := 0; i < 600; i++ {
		w.Update(1/30.0, Input{})
	}
	for _, f := range w.fishes {
		if f.Sp.Role == contract.RoleChosen && (f.Dying || f.Fade < 1) {
			t.Fatal("the eternal one faded after an elder restore")
		}
	}
}

func TestCullNeverTakesTheChosen(t *testing.T) {
	w := testWorld(t, testSpecies(0), contract.MinPopulation+3)
	she := newFish(chosenTestSpecies(), 77, v2(400, 300), 30, 99)
	she.Stage = "elder" // worst case: an impossible aged Chosen in a full tank
	w.fishes = append(w.fishes, she)
	victim := newFish(testSpecies(0), 78, v2(420, 300), 20, 100) // elder past threshold
	w.fishes = append(w.fishes, victim)
	w.cullOldestElder()
	if she.Dying {
		t.Fatal("the cull took the Chosen")
	}
	if !victim.Dying {
		t.Fatal("the cull spared the oldest normal elder")
	}
}

// F27: the nest is hers alone. The aura's soft repulsion (steering) and the
// hard projection (advance) keep every OTHER fish out of the nest zone — and
// since v0.3.7 they no longer push HER away from her own home.
func TestNestBelongsToTheChosen(t *testing.T) {
	sp := testSpecies(0)
	ch := chosenTestSpecies()
	w := NewWorld(1280, 720, contract.Config{MaxFish: 30, DaySeconds: 60},
		[]*contract.Species{sp, ch}, nil, nil)
	w.miteT = 1e9
	nest := contract.Zone{Center: v2(640, 700), Radius: contract.ZoneRadius, Owner: "chosen"}
	w.SetZones([]contract.Zone{nest})

	// herd every normal fish into the aura and soak: the projection must
	// keep them all outside the aura itself for the whole run (they may park
	// on its boundary ring — inside means inside the nest)
	for _, f := range w.fishes {
		if f.Sp.Role != contract.RoleChosen {
			f.Pos = v2(640+20, 690)
			f.Vel = v2(0, 0)
		}
	}
	steps := 60 * 20 // 20 virtual seconds
	for i := 0; i < steps; i++ {
		w.Update(1/60.0, Input{})
		for _, f := range w.fishes {
			if f.Sp.Role == contract.RoleChosen {
				continue
			}
			dx, dy := f.Pos.X-nest.Center.X, f.Pos.Y-nest.Center.Y
			if dx*dx+dy*dy < nest.Radius*nest.Radius {
				t.Fatalf("a %s slipped inside the nest (%.0f px from center) — the nest is hers alone",
					f.Sp.ID, sqrt(dx*dx+dy*dy))
			}
		}
	}

	// she feels NO outward pull from her own nest: steer() called in place,
	// at rest, just inside the aura — the only radial force allowed is ~0
	// (wander noise is bounded, the old repulsion dominated it 5:1)
	var she *Fish
	for _, f := range w.fishes {
		if f.Sp.Role == contract.RoleChosen {
			she = f
		}
	}
	if she == nil {
		t.Fatal("no chosen in the world")
	}
	she.Pos = v2(nest.Center.X+nest.Radius*0.9, nest.Center.Y)
	she.Vel = v2(0, 0)
	var outward float64
	for i := 0; i < 61; i++ { // one virtual second of steering samples
		acc := she.steer(1/60.0, she.maxSpeed(0), 0, w)
		outward += acc.X // +X points out of the nest from this spot
	}
	if outward > 4000 {
		t.Fatalf("the chosen is repelled by her own nest (outward accel sum %.0f over 1 s)", outward)
	}

	// a NORMAL in the same spot feels the full repulsion — the guard lives
	norm := w.fishes[0]
	if norm.Sp.Role == contract.RoleChosen {
		t.Fatal("expected a normal fish first in the world")
	}
	norm.Pos = she.Pos
	norm.Vel = v2(0, 0)
	norm.rng = contract.RandSeed(42) // pin the wander draw — deterministic sum
	outward = 0
	for i := 0; i < 61; i++ {
		acc := norm.steer(1/60.0, norm.maxSpeed(0), 0, w)
		outward += acc.X
	}
	if outward <= 3000 {
		t.Fatalf("the aura no longer repels normals (outward accel sum %.0f)", outward)
	}
}
