// v0.3 "Wide Glass" acceptance tests: F14 fish-majority plant cap, F15 cave
// lounging, N9 four-species opening school, N10 right-click scare, N11 the
// hook-treat pull. Shared helpers come from fish_test.go / v02_test.go.
package sim

import (
	"strings"
	"testing"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// testPlantDefs builds n distinct plant designs for cap/cull tests.
func testPlantDefs(n int) []*contract.PlantDesign {
	out := make([]*contract.PlantDesign, n)
	for i := 0; i < n; i++ {
		out[i] = &contract.PlantDesign{
			ID: "test-plant-" + itoa(i), Name: "Test Plant " + itoa(i),
			Fronds: 4, Height: 0.3, Width: 1, Curve: 0.5, Sway: 0.5,
			Colors: []string{"#134022", "#2e7d4f"}, Source: "test",
		}
	}
	return out
}

// slowSpecies is a half-speed test species (keeps scare-speed margins wide).
func slowSpecies() *contract.Species {
	sp := testSpecies(0)
	sp.Behavior.Speed = 0.5
	return sp
}

// ---- F14: live fish always outnumber plants ----

func TestInitialSchoolSeedsFourSpecies(t *testing.T) {
	// N9: the opening school spans up to 4 species × 4 fish, MaxFish-gated.
	sps := make([]*contract.Species, 6)
	for i := range sps {
		sps[i] = testSpecies(0)
		sps[i].ID = "sp-" + itoa(i)
		sps[i].Name = "Species " + itoa(i)
	}
	w := NewWorld(800, 600, contract.Config{MaxFish: 30, DaySeconds: 60}, sps, nil, nil)
	// v1: the school fills to MaxFish-1 (test worlds carry no embedded
	// Chosen — in the real tank she takes the last seat)
	if got := len(w.fishes); got != 29 {
		t.Fatalf("initial school = %d fish, want MaxFish-1 = 29", got)
	}
	// MaxFish gates the same seeding
	w2 := NewWorld(800, 600, contract.Config{MaxFish: 10, DaySeconds: 60}, sps, nil, nil)
	if got := len(w2.fishes); got != 9 {
		t.Fatalf("MaxFish gate broken: %d fish, want 9", got)
	}
}

func TestInitialPlacementRespectsFishMajority(t *testing.T) {
	// F14: NewWorld places at most alive-1 plants even when handed more.
	sp := testSpecies(0)
	w := NewWorld(800, 600, contract.Config{MaxFish: 30, DaySeconds: 60},
		[]*contract.Species{sp}, testPlantDefs(16), nil)
	// one normal species → the placement respects the effective cap
	if got, want := len(w.Plants()), w.effectivePlantCap(); got != want {
		t.Fatalf("initial plants = %d, want the effective cap %d (alive %d)", got, want, len(w.aliveFishes()))
	}
	if len(w.aliveFishes()) < len(w.Plants())+1 {
		t.Fatal("fish do not outnumber plants at boot")
	}
}

func TestAddPlantAboveEffectiveCapIsNoOp(t *testing.T) {
	// F14: 5 alive fish → cap 4; the 5th+ AddPlant silently does nothing.
	w := testWorld(t, testSpecies(0), 5)
	defs := testPlantDefs(8)
	for _, d := range defs {
		w.AddPlant(d)
	}
	if got := len(w.Plants()); got != 4 {
		t.Fatalf("plants = %d, want effective cap 4", got)
	}
	if len(w.aliveFishes()) < len(w.Plants())+1 {
		t.Fatal("AddPlant broke the fish majority")
	}
}

func TestDeathCullsOldestPlant(t *testing.T) {
	// F14: when corpses sweep out, the OLDEST plant goes first, one per breach.
	w := testWorld(t, testSpecies(0), 6)
	for i, d := range testPlantDefs(5) {
		w.plants = append(w.plants, &Plant{Def: d, X: 100 + float64(i)*30, Y: 594, Sc: 1})
		w.PlantIDs = append(w.PlantIDs, d.ID)
	}
	// four fish fade out at once: 2 survivors → the garden must thin to 1
	for i := 0; i < 4; i++ {
		f := w.fishes[i]
		f.Dying, f.Fade, f.DieReason = true, 0, "old age"
	}
	w.Update(1/60.0, Input{})
	if got := len(w.aliveFishes()); got != 2 {
		t.Fatalf("corpses not swept: %d alive", got)
	}
	if got := len(w.Plants()); got != 1 {
		t.Fatalf("plants = %d after the cull, want 1 (2 alive fish)", got)
	}
	if w.Plants()[0].Def.ID != "test-plant-4" {
		t.Fatalf("oldest plant survived: %s", w.Plants()[0].Def.ID)
	}
	thinned := 0
	for _, e := range w.Log {
		if e.Text == "the garden thins" {
			thinned++
		}
	}
	if thinned != 4 {
		t.Fatalf("garden log lines = %d, want one per breach (4)", thinned)
	}
}

func TestRestoreTrimsPlantsToLiveSchool(t *testing.T) {
	// F14: the Restore trim uses the effective cap of the restored school.
	sp := testSpecies(0)
	w := NewWorld(800, 600, contract.Config{MaxFish: 30, DaySeconds: 60},
		[]*contract.Species{sp}, testPlantDefs(4), nil)
	w.SeedRng(7)
	// simulate a legacy over-planted save: 30 plants over the 29-fish school
	for _, d := range testPlantDefs(30) {
		w.plants = append(w.plants, &Plant{Def: d, X: w.W * 0.5, Y: w.H - 6, Sc: 1})
		w.PlantIDs = append(w.PlantIDs, d.ID)
	}
	if err := w.Restore(w.Snapshot()); err != nil {
		t.Fatal(err)
	}
	if got, want := len(w.Plants()), w.effectivePlantCap(); got != want {
		t.Fatalf("restore trim left %d plants, want the effective cap %d", got, want)
	}
	if len(w.aliveFishes()) < len(w.Plants())+1 {
		t.Fatal("restore broke the fish majority")
	}
}

func TestFishAlwaysOutnumberPlantsSoak(t *testing.T) {
	// F14 soak: 10 tank-days of AddPlant spam, elder deaths and hatches —
	// the invariant alive ≥ plants+1 must hold after EVERY batch.
	// Phase 1 (care/feed off, 120 s): two elder cohorts fade out while the
	// plant spam has the garden at the cap — every corpse sweep must cull.
	// Phase 2 (care/feed on, 480 s): breeding hatches the school back and
	// the garden regrows to the new cap.
	sp := testSpecies(0.5)
	cfg := contract.Config{MaxFish: 30, DaySeconds: 60}
	defs := testPlantDefs(16)
	w := NewWorld(800, 600, cfg, []*contract.Species{sp}, defs, nil)
	w.SeedRng(2024)
	w.fishes = w.fishes[:0]
	for i := 0; i < 12; i++ {
		p := v2(150+w.rng.Float64()*500, 150+w.rng.Float64()*300)
		w.fishes = append(w.fishes, newFish(sp, w.rng.Int63(), p, 6, w.nextID()))
	}
	// eight elders on the brink: four past the line already, four crossing
	// it ~half a minute in — deaths arrive in two waves
	for i := 0; i < 4; i++ {
		w.fishes[i].AgeDays = contract.StageDays["elder"] + contract.DeathAfterElderDays + 0.2
	}
	for i := 4; i < 8; i++ {
		w.fishes[i].AgeDays = contract.StageDays["elder"] + contract.DeathAfterElderDays - 0.4
	}
	thinned, hatched, logMark := 0, 0, 0
	for batch := 0; batch < 600; batch++ { // 600 s = 10 tank-days
		if batch == 120 {
			w.cfg.AutoCare, w.cfg.AutoFeed = true, true // life returns
		}
		soakTicks(w, 1)
		w.AddPlant(defs[batch%len(defs)]) // the spam never stops
		if alive, plants := len(w.aliveFishes()), len(w.Plants()); alive < plants+1 {
			t.Fatalf("batch %d: %d alive fish do not outnumber %d plants", batch, alive, plants)
		}
		// the log is a bounded ring — count evidence incrementally
		for _, e := range w.Log[logMark:] {
			switch {
			case e.Text == "the garden thins":
				thinned++
			case e.Text == sp.Name+" hatched":
				hatched++
			}
		}
		logMark = len(w.Log)
	}
	if thinned == 0 {
		t.Fatal("elders died but the garden never thinned")
	}
	if hatched == 0 {
		t.Fatal("the school never bounced back after the deaths")
	}
}

// ---- F15: cave lounging ----

func TestForcedLoungeHoldsCave(t *testing.T) {
	// F15: a lounging fish stays inside its cave zone, drifts at a calm
	// speed, regenerates energy and beats its tail slowly.
	w := testWorld(t, testSpecies(0), 1)
	cave := contract.Zone{Center: v2(150, 560), Radius: 90, Owner: "cave"}
	w.SetZones([]contract.Zone{cave})
	f := w.fishes[0]
	f.Pos, f.Vel = cave.Center, v2(0, 0)
	f.Energy = 0.2
	f.Satiety = 0.9 // fed — comfortably above the lounge entry bar for 10 s
	phase0 := f.phase
	f.loungeT, f.loungeC = 12, cave.Center
	for i := 0; i < 600; i++ { // 10 s
		w.Update(1/60.0, Input{})
		if d := hyp2(sub(f.Pos, cave.Center)); d >= cave.Radius {
			t.Fatalf("lounging fish left the cave: %.0f px from center", d)
		}
		bound := 0.6 * f.maxSpeed(1-w.dayFactor())
		if s := hyp2(f.Vel); s > bound {
			t.Fatalf("lounging fish too fast: %.1f px/s > %.1f", s, bound)
		}
	}
	if !f.Lounging() {
		t.Fatal("lounge ended before the 10 s dwell window elapsed")
	}
	if f.Energy < 0.9 {
		t.Fatalf("the cave did not restore energy: %.2f", f.Energy)
	}
	if d := f.phase - phase0; d > 20 {
		t.Fatalf("lounging tail beat too fast: %.1f rad in 10 s", d)
	}
}

func TestLoungeFrequencySoak(t *testing.T) {
	// F15: with caves installed and bellies kept full, lounges actually
	// happen — the staggered countdown targets one pick per fish per
	// LoungeMeanSec on average. Seeded for determinism.
	w := testWorld(t, testSpecies(0), 5)
	w.cfg.AutoFeed = true // lounging needs fed fish (Satiety > 0.3)
	w.SetZones([]contract.Zone{
		{Center: v2(150, 560), Radius: 90, Owner: "cave"},
		{Center: v2(650, 560), Radius: 90, Owner: "cave"},
	})
	soakTicks(w, 600)
	if w.lounges < 1 {
		t.Fatalf("no fish ever lounged in 600 s (counter=%d)", w.lounges)
	}
	found := false
	for _, e := range w.Log {
		if strings.Contains(e.Text, "lounges in the cave") {
			found = true
		}
	}
	if !found {
		t.Fatal("lounge counter ran but the tank log missed the event")
	}
}

// ---- N10: right-click scare ----

func TestRightClickScareBoltsAndFades(t *testing.T) {
	// N10: a scare point scatters every fish inside ScareRadius — within
	// 0.3 s they bolt well past 2× cruise heading AWAY from the point, and
	// after 2.5 calm seconds every one of them is back under 1.25× cruise.
	w := testWorld(t, slowSpecies(), 6)
	w.miteT = 1e9 // no mite frenzy may inflate the speeds
	scare := v2(400, 300)
	angles := []float64{0, 1.05, 2.1, 3.15, 4.2, 5.25}
	for i, f := range w.fishes {
		f.Pos = add(scare, v2(cos(angles[i])*55, sin(angles[i])*55))
		f.Vel = v2(0, 0)
		f.Satiety = 1 // full — no food chase muddies the heading
	}
	in := Input{ScareAt: &scare}
	bestRatio := 0.0
	for i := 0; i < 18; i++ { // 0.3 s of sustained scare @60 Hz
		w.Update(1/60.0, in)
		sumSp, sumCr := 0.0, 0.0
		for _, f := range w.fishes {
			sumSp += hyp2(f.Vel)
			sumCr += f.maxSpeed(1 - w.dayFactor())
		}
		bestRatio = maxF(bestRatio, sumSp/sumCr)
	}
	if bestRatio <= 2 {
		t.Fatalf("mean speed only reached %.2f× cruise inside 0.3 s", bestRatio)
	}
	for _, f := range w.fishes { // headings point away from the scare
		if dot := f.Vel.X*(f.Pos.X-scare.X) + f.Vel.Y*(f.Pos.Y-scare.Y); dot <= 0 {
			t.Fatalf("fish fled toward the scare (dot=%.1f)", dot)
		}
	}
	for i := 0; i < 150; i++ { // 2.5 s of no-scare soak → they forget
		w.Update(1/60.0, Input{})
	}
	for _, f := range w.fishes {
		cruise := f.maxSpeed(1 - w.dayFactor())
		if s := hyp2(f.Vel); s > 1.25*cruise {
			t.Fatalf("fish still panicked after 2.5 s: %.1f > %.1f", s, 1.25*cruise)
		}
	}
}

// ---- N11: the hook-treat pull ----

func TestHeldTreatPullsHungryFishToRing(t *testing.T) {
	// N11: fish within HeldTreatRadius swarm the held treat and crowd on
	// the keep-back ring — close to the cursor, mouths open, but NO bite
	// ever lands (the held treat is not in the tank) and satiety only
	// decays naturally.
	w := testWorld(t, testSpecies(0), 4)
	w.miteT = 1e9
	cursor := v2(400, 300)
	spots := []contract.Vec2{v2(180, 300), v2(620, 280), v2(390, 520), v2(560, 470)}
	for i, f := range w.fishes {
		f.Satiety = 0.5 // hungry enough to lust for the hook
		f.Pos = spots[i]
		f.Vel = v2(0, 0)
	}
	sat0 := make([]float64, len(w.fishes))
	for i, f := range w.fishes {
		sat0[i] = f.Satiety
	}
	in := Input{HeldTreat: &HeldTreat{Kind: contract.TreatWorm, Pos: cursor}}
	for i := 0; i < 90; i++ { // 3 s @30 Hz
		w.Update(1/30.0, in)
	}
	for i, f := range w.fishes {
		if d := hyp2(sub(f.Pos, cursor)); d > 60 {
			t.Fatalf("fish %d not pulled in: %.0f px from the cursor", i, d)
		}
		if f.bites != 0 {
			t.Fatalf("a bite landed on the held treat (bites=%d)", f.bites)
		}
		want := sat0[i] - 3.0/40 // only the natural decay
		if absF(f.Satiety-want) > 0.01 {
			t.Fatalf("satiety changed beyond decay: %.3f want %.3f", f.Satiety, want)
		}
	}
	if len(w.Treats()) != 0 || len(w.Foods()) != 0 {
		t.Fatal("a real treat/food appeared while one was held")
	}
}

// ---- F26: the scent cloud spreads while the bait is carried ----

func TestHeldTreatScentGrowsWithCarryTime(t *testing.T) {
	// A fish parked BEYOND the base radius must catch the smell once the
	// carry age has widened the scent cloud (260 + 20/s, capped at +140).
	w := testWorld(t, testSpecies(0), 1)
	w.miteT = 1e9
	f := w.fishes[0]
	f.Satiety = 0.5
	f.Pos = v2(400 + contract.HeldTreatRadius + 40, 300) // 300 px out
	f.Vel = v2(0, 0)
	cursor := v2(400, 300)

	// fresh grab: out of smell range, the fish holds position (no pull)
	in := Input{HeldTreat: &HeldTreat{Kind: contract.TreatWorm, Pos: cursor}}
	for i := 0; i < 30; i++ {
		w.Update(1/30.0, in)
	}
	d0 := hyp2(sub(f.Pos, cursor))
	if d0 < contract.HeldTreatRadius {
		t.Fatalf("fish moved before the scent reached it (%.0f px)", d0)
	}

	// after 4 s of carrying the radius is 260+80=340: the fish smells it
	in.HeldTreat.Age = 4.0
	for i := 0; i < 120; i++ { // 4 s @30 Hz
		w.Update(1/30.0, in)
	}
	if d := hyp2(sub(f.Pos, cursor)); d > 60 {
		t.Fatalf("fish did not follow the grown scent: %.0f px from cursor", d)
	}
}
