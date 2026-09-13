// v1.1: hammerhead pair tests (G50) — the resident cap of two, pace
// supremacy of the Chosen at every hour (even a food-frenzy double stays
// below her), menu/egg exclusion, ambient bookkeeping and persistence with
// the pair cap enforced on restore.
package sim

import (
	"fmt"
	"testing"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

func sharkTestSpecies() *contract.Species {
	sp := testSpecies(0)
	c := *sp
	c.ID = "test-shark"
	c.Role = contract.RoleShark
	c.Size = 1.8
	c.Behavior.Speed = 1.0
	return &c
}

func sharkWorld(t *testing.T) *World {
	t.Helper()
	cfg := contract.Config{MaxFish: 20, DaySeconds: 60}
	w := NewWorld(800, 600, cfg, []*contract.Species{sharkTestSpecies(), chosenTestSpecies()}, nil, nil)
	w.SeedRng(11)
	w.fishes = w.fishes[:0]
	return w
}

// G50: the pair stays whole at exactly two, never more.
func TestSharkPairCap(t *testing.T) {
	w := sharkWorld(t)
	w.Update(0.05, Input{}) // first frame spawns the pair
	for i := 0; i < 100; i++ {
		w.Update(0.05, Input{})
	}
	if n := w.sharkCount(); n != contract.SharkMax {
		t.Fatalf("shark count %d, want %d", n, contract.SharkMax)
	}
	// a hand-edited or stale state cannot grow the pair
	for i := 0; i < 50; i++ {
		w.ensureSharks()
	}
	if n := w.sharkCount(); n != contract.SharkMax {
		t.Fatalf("ensureSharks overshot: %d", n)
	}
}

// G50: active roamer — the pair sweeps real distance, and never outswims
// the Chosen at any hour, even mid-frenzy at double pace.
func TestSharkRoamsBelowHer(t *testing.T) {
	w := sharkWorld(t)
	w.Update(0.05, Input{})
	ch := newFish(chosenTestSpecies(), 9, v2(400, 300), 8, 99)
	w.fishes = append(w.fishes, ch)
	var sh *Fish
	for _, f := range w.fishes {
		if f.Sp.Role == contract.RoleShark {
			sh = f
			break
		}
	}
	if sh == nil {
		t.Fatal("no shark in the tank after the first frame")
	}
	for _, night := range []float64{0, 0.5, 1} {
		if s := sh.maxSpeed(night) * 2; s >= ch.maxSpeed(night) {
			t.Fatalf("night=%.1f: a frenzying shark (%.0f) could outrun her (%.0f)",
				night, s, ch.maxSpeed(night))
		}
		if sh.maxSpeed(night) >= ch.maxSpeed(night) {
			t.Fatalf("night=%.1f: the shark's cruise is not below her", night)
		}
	}
	// roaming: 30 s of sim swims the shark a real path (the pair may circle
	// each other — mutual body avoidance — so path length, not displacement)
	w.SeedRng(3)
	path := 0.0
	prev := sh.Pos
	for i := 0; i < 600; i++ {
		w.Update(0.05, Input{})
		path += hyp2(sub(sh.Pos, prev))
		prev = v2(sh.Pos.X, sh.Pos.Y)
	}
	if path < 600 {
		t.Fatalf("the shark swam only %.0f px in 30 s — not the active roamer it should be", path)
	}
}

// G50 + ambient: off the menu, outside the bookkeeping, persisted as a pair.
func TestSharkExclusionsAndPersistence(t *testing.T) {
	w := sharkWorld(t)
	w.Update(0.05, Input{}) // pair arrives
	// the hunter answers no egg
	w.SpawnEgg("test-shark")
	if len(w.eggs) != 0 {
		t.Fatal("SpawnEgg created a shark egg")
	}
	// courtship never pairs the hunter
	if w.startCourtship(true) {
		t.Fatal("sharks started a courtship")
	}
	// residents persist — the snapshot carries the pair
	s := w.Snapshot()
	sharks := 0
	for _, sf := range s.Fish {
		if sf.SpeciesID == "test-shark" {
			sharks++
		}
	}
	if sharks != contract.SharkMax {
		t.Fatalf("snapshot carries %d sharks, want %d", sharks, contract.SharkMax)
	}
	// restore caps the pair even from a hand-edited save carrying five
	w2 := sharkWorld(t)
	five := contract.Save{SchemaVersion: 2}
	for i := 0; i < 5; i++ {
		five.Fish = append(five.Fish, contract.SavedFish{SpeciesID: "test-shark", Pos: v2(100+float64(i)*40, 300), Seed: int64(i + 1)})
	}
	if err := w2.Restore(five); err != nil {
		t.Fatal(err)
	}
	if n := w2.sharkCount(); n != contract.SharkMax {
		t.Fatalf("restored %d sharks, want the cap %d", n, contract.SharkMax)
	}
}

// G55: her circle is absolute for EVERY body, head to tail. A giant parked
// straight across the circle resolves out within seconds — and never leans
// on it again. The shark pair and the school live under the same law.
func TestBigBodiesNeverEnterTheCircle(t *testing.T) {
	cfg := contract.Config{MaxFish: 20, DaySeconds: 60}
	w := NewWorld(800, 600, cfg, []*contract.Species{
		titanTestSpecies(), sharkTestSpecies(), cappedNormalSpecies("test-neon", false),
	}, nil, nil)
	w.SeedRng(23)
	w.fishes = w.fishes[:0]
	w.Update(0.05, Input{}) // the shark pair arrives
	w.spawnPod()
	w.titanPhase = 2 // roaming
	nest := contract.Zone{Center: v2(w.W*0.5, w.H*contract.FloorLineFrac), Radius: contract.ZoneRadius, Owner: "chosen"}
	w.SetZones([]contract.Zone{nest})

	// worst-case start: the giant's body laid straight across her circle
	g := w.titanGiant()
	if g == nil {
		t.Fatal("no giant for the intrusion test")
	}
	g.Pos = v2(nest.Center.X+100, nest.Center.Y)
	for i := range g.Spine {
		g.Spine[i] = v2(g.Pos.X-float64(i)*g.segLen, g.Pos.Y)
	}

	const dt = 0.05
	worst := 1e18
	for i := 0; i < 30*20; i++ { // 30 s
		w.Update(dt, Input{})
		if i < 3*20 {
			continue // the first seconds resolve the parked intrusion
		}
		for _, f := range w.fishes {
			if f.Sp.Role == contract.RoleChosen {
				continue // she lives in her circle — everyone else stays out
			}
			for _, p := range f.Spine {
				if d := hyp2(sub(p, nest.Center)); d < worst {
					worst = d
				}
			}
		}
	}
	if worst < nest.Radius {
		t.Fatalf("a body crossed into her circle by %.0f px (closest approach %.0f < radius %.0f)",
			nest.Radius-worst, worst, nest.Radius)
	}
}

// G58 fix: the curl is carved FORWARD — through the whole turn the scalare
// keeps half its cruise speed and never glides tail-first.
func TestTitanCurlForwardOnly(t *testing.T) {
	w := titanWorld(t, 3)
	w.spawnPod()
	g := w.titanGiant()
	g.Pos = v2(w.W*0.86, 270) // just short of the curl trigger zone
	g.cruise = 1
	g.headingA = 0
	cruise := g.maxSpeed(0)
	const dt = 0.05
	minSp, curlSeen := 1e18, false
	for i := 0; i < 60*45; i++ { // 45 s: at least one edge curl happens
		g.Satiety = 1 // fed: no hunger lunges, no hunt -- pure sweep test
		w.Update(dt, Input{})
		if i%400 == 0 {
			fmt.Printf("dbg t=%.0f x=%.0f heading=%.2f cruise=%.0f turning=%.1f\n",
				float64(i)*dt, g.Pos.X, g.headingA, g.cruise, g.turning)
		}
		if g.turning > 0 {
			curlSeen = true
			if s := hyp2(g.Vel); s < minSp {
				minSp = s
			}
		}
	}
	if !curlSeen {
		t.Fatal("no curl within 45 s of sweeping")
	}
	if minSp < cruise*0.35 {
		t.Fatalf("the curl stalled to %.1f px/s (cruise %.1f) — a scalare must carve forward", minSp, cruise)
	}
}
