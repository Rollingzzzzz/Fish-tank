// v0.2 acceptance tests: F3 food rush, F6 breeding soaks, F11 population
// floor + maintenance auto-feed.
package sim

import (
	"testing"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// soakTicks runs the world for the given number of virtual seconds at 30 Hz.
func soakTicks(w *World, seconds float64) {
	steps := int(seconds * 30)
	for i := 0; i < steps; i++ {
		w.Update(1/30.0, Input{})
	}
}

func TestFoodRushCovers300pxFast(t *testing.T) {
	// F3: a hungry fish reaches food 300 px away in <= 2.5 virtual seconds.
	w := testWorld(t, testSpecies(0), 1)
	f := w.fishes[0]
	f.Satiety = 0.1
	f.Pos = v2(100, 300)
	w.miteT = 1e9 // G89: the surface mite release is N7's own drama — this
	// contract judges the flake sprint alone, un-distracted by live prey
	f.Vel = v2(0, 0)
	w.foods = append(w.foods, Food{Pos: v2(400, 300), Age: 1})
	eaten := false
	for i := 0; i < int(2.5*60) && !eaten; i++ {
		w.Update(1/60.0, Input{})
		eaten = len(w.foods) == 0 || f.Satiety > 0.99
	}
	if !eaten {
		t.Fatalf("food rush too slow: flake survived 2.5 s at %d flakes", len(w.foods))
	}
}

func TestBreedingSoakWithAutoCare(t *testing.T) {
	// F6: AutoCare + AutoFeed on, 5 virtual days, 4 fish → recurring
	// courtships, ≥2 hatchings, population grows.
	sp := testSpecies(0)
	cfg := contract.Config{MaxFish: 30, DaySeconds: 60, AutoCare: true, AutoFeed: true}
	w := NewWorld(800, 600, cfg, []*contract.Species{sp}, nil, nil)
	w.SeedRng(4242)
	w.fishes = w.fishes[:0]
	for i := 0; i < 4; i++ {
		p := v2(200+w.rng.Float64()*400, 200+w.rng.Float64()*200)
		w.fishes = append(w.fishes, newFish(sp, w.rng.Int63(), p, 6, w.nextID()))
	}
	soakTicks(w, 5*60)
	if got := len(w.aliveFishes()); got < 6 {
		t.Fatalf("population did not grow with care: %d (start 4)", got)
	}
}

func TestNoBreedingWithoutCare(t *testing.T) {
	// F6 counter-soak: everything off, no mouse, 3 days → zero eggs.
	sp := testSpecies(0)
	cfg := contract.Config{MaxFish: 30, DaySeconds: 60}
	w := NewWorld(800, 600, cfg, []*contract.Species{sp}, nil, nil)
	w.SeedRng(99)
	w.miteT = 1e9                  // wild mite meals grant free care — disarm for a pure soak
	contract.ExtrasEnabled = false // v1.1: critter meals grant care too
	defer func() { contract.ExtrasEnabled = true }()
	w.fishes = w.fishes[:0]
	for i := 0; i < contract.MinPopulation; i++ {
		p := v2(200+w.rng.Float64()*400, 200+w.rng.Float64()*200)
		w.fishes = append(w.fishes, newFish(sp, w.rng.Int63(), p, 6, w.nextID()))
	}
	soakTicks(w, 3*60)
	if len(w.eggs) != 0 {
		t.Fatalf("eggs appeared without any care: %d", len(w.eggs))
	}
	if w.courtCD != 0 && len(w.courtships) != 0 {
		t.Fatal("courtship ran without care")
	}
}

func TestPopulationFloorSelfHeals(t *testing.T) {
	// FD11: below MinPopulation the tank courts even without care.
	sp := testSpecies(0)
	w := testWorld(t, sp, contract.MinPopulation-1)
	for _, f := range w.fishes {
		f.AgeDays = 6 // adults
	}
	soakTicks(w, 40)
	if len(w.eggs) == 0 && len(w.courtships) == 0 && len(w.aliveFishes()) < contract.MinPopulation {
		t.Fatal("tank below the floor never tried to breed")
	}
}

func TestMaintenanceAutoFeedHoldsBand(t *testing.T) {
	// F11: maintenance feeder keeps average satiety in the comfort band
	// without ever carpeting the floor.
	sp := testSpecies(0)
	cfg := contract.Config{MaxFish: 30, DaySeconds: 60, AutoFeed: true}
	w := NewWorld(800, 600, cfg, []*contract.Species{sp}, nil, nil)
	w.SeedRng(777)
	w.fishes = w.fishes[:0]
	for i := 0; i < 8; i++ {
		p := v2(150+w.rng.Float64()*500, 150+w.rng.Float64()*300)
		f := newFish(sp, w.rng.Int63(), p, 6, w.nextID())
		f.Satiety = 0.4
		w.fishes = append(w.fishes, f)
	}
	maxFoods, minAvg := 0, 1.0
	for i := 0; i < int(2*60*30); i++ {
		w.Update(1/30.0, Input{})
		if len(w.foods) > maxFoods {
			maxFoods = len(w.foods)
		}
		sum, n := 0.0, 0
		for _, f := range w.aliveFishes() {
			sum += f.Satiety
			n++
		}
		if n > 0 {
			if avg := sum / float64(n); avg < minAvg {
				minAvg = avg
			}
		}
	}
	// F15 note: lounging fish pause eating, so the feeder can now touch its
	// structural peak of 10 (its own gate fires only below len(foods) < 8 and
	// drops 3 flakes). Anything past that peak means the gate itself broke.
	if maxFoods > 10 {
		t.Fatalf("feeder carpeted the floor: %d flakes at once", maxFoods)
	}
	if minAvg < 0.35 {
		t.Fatalf("school starved despite auto-feed: avg dipped to %.2f", minAvg)
	}
}

func TestBehaviorPackSoak(t *testing.T) {
	// F4: a 10-virtual-day soak must log startles, chases and zoomies.
	sp := testSpecies(0)
	sp.Behavior.Skittish = 0.8
	sp.Behavior.Schooling = 0.6
	w := testWorld(t, sp, 8)
	soakTicks(w, 10*60)
	if w.startles < 5 {
		t.Fatalf("too few startles: %d", w.startles)
	}
	if w.chases < 3 {
		t.Fatalf("too few chases: %d", w.chases)
	}
	if w.zoomies < 3 {
		t.Fatalf("too few zoomies: %d", w.zoomies)
	}
	if w.nudges < 1 {
		t.Fatalf("no nudges: %d", w.nudges)
	}
}

func TestChosenAuraExclusion(t *testing.T) {
	// N3: nothing but the Chosen may stay inside the aura.
	sp := testSpecies(0)
	w := testWorld(t, sp, 6)
	chosen := *sp
	chosen.Role = contract.RoleChosen
	home := v2(400, 300)
	w.SetZones([]contract.Zone{{Center: home, Radius: contract.ZoneRadius, Owner: "chosen"}})
	w.fishes = append(w.fishes, newFish(&chosen, 11, home, 8, w.nextID()))
	for _, f := range w.fishes {
		if f.Sp.Role != contract.RoleChosen {
			f.Pos = v2(400+w.rng.Float64()*40-20, 300+w.rng.Float64()*40-20)
		}
	}
	soakTicks(w, 20)
	chosenOK := false
	for _, f := range w.fishes {
		if f.Sp.Role == contract.RoleChosen {
			chosenOK = !f.Dying
			continue
		}
		if d := hyp2(sub(f.Pos, home)); d < contract.ZoneRadius-4 {
			t.Fatalf("%s breached the aura: %.0f px from center", f.Sp.Name, d)
		}
	}
	if !chosenOK {
		t.Fatal("the Chosen did not survive the soak")
	}
}

func TestChosenAlwaysRestored(t *testing.T) {
	// FD9: NewWorld spawns exactly one Chosen; Restore re-spawns if missing.
	sp := testSpecies(0)
	chosen := *sp
	chosen.Role = contract.RoleChosen
	w := NewWorld(800, 600, contract.Config{MaxFish: 20, DaySeconds: 60},
		[]*contract.Species{sp, &chosen}, nil, nil)
	count := 0
	for _, f := range w.fishes {
		if f.Sp.Role == contract.RoleChosen {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 chosen, got %d", count)
	}
	// wipe it (simulating a legacy save) and restore
	w.fishes = w.fishes[:0]
	for i := 0; i < 4; i++ {
		w.fishes = append(w.fishes, newFish(sp, w.rng.Int63(), v2(300, 300), 6, w.nextID()))
	}
	if err := w.Restore(w.Snapshot()); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range w.fishes {
		if f.Sp.Role == contract.RoleChosen {
			found = true
		}
	}
	if !found {
		t.Fatal("the Chosen was not restored")
	}
}

func TestMitesSpawnAndFrenzy(t *testing.T) {
	// N7: mites appear on their own and get devoured by hungry fish.
	sp := testSpecies(0)
	w := testWorld(t, sp, 3)
	w.miteT = 0.05
	for i := 0; i < 600 && len(w.mites) == 0; i++ {
		w.Update(1/60.0, Input{})
	}
	if len(w.mites) == 0 {
		t.Fatal("no mite spawned naturally")
	}
	m := w.mites[0]
	f := w.fishes[0]
	f.Satiety = 0.3
	f.Pos = v2(m.Pos.X+5, m.Pos.Y)
	w.Update(1/60.0, Input{})
	if len(w.mites) != 0 {
		t.Fatal("hungry fish did not devour the mite")
	}
	// G88: a mite is a SNACK — a third of a meal (the teem tripled the
	// cadence; the calories must not follow)
	if f.Satiety < 0.3+0.33 || f.Satiety > 0.3+0.35 {
		t.Fatalf("mite fed the fish %.3f — want a third of a meal", f.Satiety)
	}
}

func TestPlecoAttachesToGlass(t *testing.T) {
	// N5: a high-Attachment species visits the glass.
	sp := testSpecies(0)
	sp.Behavior.Attachment = 0.9
	w := testWorld(t, sp, 2)
	attached := false
	for i := 0; i < 60*120 && !attached; i++ { // 2 virtual minutes
		w.Update(1/60.0, Input{})
		for _, f := range w.fishes {
			if f.attachT > 0 {
				attached = true
			}
		}
	}
	if !attached {
		t.Fatal("pleco never attached to the glass")
	}
}

func TestStartledFishSheltersInCave(t *testing.T) {
	// N2: after a scare, fish head for the nearest cave mouth.
	sp := testSpecies(0)
	w := testWorld(t, sp, 6)
	caves := []contract.Zone{
		{Center: v2(150, 560), Radius: 90, Owner: "cave"},
		{Center: v2(650, 560), Radius: 90, Owner: "cave"},
	}
	w.SetZones(caves)
	for _, f := range w.fishes {
		f.flee(640, 100, contract.MaxForce) // one big scare from above
	}
	inCave := 0
	for i := 0; i < 60*12; i++ {
		w.Update(1/60.0, Input{})
		if i%120 == 0 && i > 0 {
			for _, f := range w.fishes {
				for _, z := range caves {
					if hyp2(sub(f.Pos, z.Center)) < z.Radius {
						inCave++
						break
					}
				}
			}
		}
	}
	if inCave == 0 {
		t.Fatal("no fish ever sheltered in a cave after the scare")
	}
}
