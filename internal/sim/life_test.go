// G2.3: aging, care thresholds, courtship breeding and natural death tests.
package sim

import (
	"testing"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

func TestStageTransitionsFireOnce(t *testing.T) {
	w := testWorld(t, testSpecies(0), 2)
	f := w.fishes[0]
	// resync to a consistent fry state just below the juvenile boundary
	f.AgeDays = 1.98
	f.Stage = StageFor(f.AgeDays)
	f.prevStage = f.Stage
	q0 := len(w.repaintQ)
	w.tickLife(f, 0)
	if len(w.repaintQ) != q0 {
		t.Fatal("repaint queued without a stage change")
	}
	// cross the boundary: 2.01 days → juvenile, exactly one repaint request
	f.AgeDays = 2.01
	w.tickLife(f, 0)
	if f.Stage != "juvenile" {
		t.Fatalf("stage not advanced: %s", f.Stage)
	}
	if len(w.repaintQ) != q0+1 {
		t.Fatalf("repaint request not queued exactly once: %d", len(w.repaintQ)-q0)
	}
	w.tickLife(f, 0)
	if len(w.repaintQ) != q0+1 {
		t.Fatal("duplicate repaint request on second tick")
	}
}

func TestCareThresholdTriggersCourtshipAndEggs(t *testing.T) {
	w := testWorld(t, testSpecies(0), 4)
	for _, f := range w.fishes {
		f.AgeDays = 6 // adults
	}
	w.Care = contract.CareThresholds[0] - 0.01
	w.careThreshIdx = 0
	w.addCare(0.02)
	w.tickCare(0.01) // crosses the threshold → courtship starts
	paired := 0
	for _, f := range w.fishes {
		if f.CourtID != "" {
			paired++
		}
	}
	if paired != 2 {
		t.Fatalf("courtship did not pair 2 fish: %d", paired)
	}
	// advance past CourtshipSec
	for i := 0; i < int(contract.CourtshipSec*60)+30; i++ {
		w.Update(1/60.0, Input{})
	}
	if len(w.eggs) == 0 {
		t.Fatal("no eggs laid after courtship")
	}
}

func TestNaturalDeathRespectsMinPopulation(t *testing.T) {
	// FD11: deaths never take the tank below MinPopulation.
	w := testWorld(t, testSpecies(0), contract.MinPopulation+1)
	w.miteT = 1e9 // isolate the death logic — the teeming mites would feed and breed the school mid-fade
	// fish 0 is ancient; the others stay young
	f0 := w.fishes[0]
	f0.AgeDays = contract.StageDays["elder"] + contract.DeathAfterElderDays + 1
	w.tickLife(f0, 0.01)
	if !f0.Dying {
		t.Fatal("elder death not triggered")
	}
	// fade fully out
	for i := 0; i < int(contract.DeathFadeSec*60)+10; i++ {
		w.Update(1/60.0, Input{})
	}
	if len(w.fishes) != contract.MinPopulation {
		t.Fatalf("dying fish not removed: %d remain", len(w.fishes))
	}
	// now at exactly the floor: death must be refused
	f1 := w.fishes[0]
	f1.AgeDays = contract.StageDays["elder"] + contract.DeathAfterElderDays + 1
	w.tickLife(f1, 0.01)
	if f1.Dying {
		t.Fatal("death ignored the MinPopulation guarantee")
	}
}

func TestElderFadeProgressMonotonic(t *testing.T) {
	// F7: ElderP grows monotonically through elder age and drives the fade.
	w := testWorld(t, testSpecies(0), 2)
	f := w.fishes[0]
	f.AgeDays = contract.StageDays["elder"]
	prev := -1.0
	for d := 0.0; d <= contract.DeathAfterElderDays+0.5; d += 0.25 {
		f.AgeDays = contract.StageDays["elder"] + d
		w.tickLife(f, 0)
		if f.ElderP < prev-1e-9 {
			t.Fatalf("elder fade not monotonic: %v after %v", f.ElderP, prev)
		}
		if f.ElderP < 0 || f.ElderP > 1 {
			t.Fatalf("elder fade out of range: %v", f.ElderP)
		}
		prev = f.ElderP
	}
	if prev < 1 {
		t.Fatalf("elder fade never completes: %v", prev)
	}
}

func TestChosenNeverAgesOrDies(t *testing.T) {
	// FD9: the eternal one skips aging, fade and death entirely.
	w := testWorld(t, testSpecies(0), 2)
	sp := testSpecies(0)
	sp.Role = contract.RoleChosen
	f := newFish(sp, 7, v2(100, 100), 0, 1)
	w.fishes = append(w.fishes, f)
	f.AgeDays = 100 // far past any death threshold
	w.tickLife(f, 0.016)
	if f.Dying {
		t.Fatal("the Chosen must never die")
	}
	if f.ElderP != 0 {
		t.Fatal("the Chosen must never fade")
	}
}

func TestDayClockWraps(t *testing.T) {
	w := testWorld(t, testSpecies(0), 1)
	w.Clock = 0
	seen := map[int]bool{}
	for i := 0; i < 60*200; i++ {
		w.Clock += 1
		seen[int(w.dayFactor()*4)] = true
	}
	if !seen[0] || !seen[4] {
		t.Fatalf("day/night cycle never covered both extremes: %v", seen)
	}
}
