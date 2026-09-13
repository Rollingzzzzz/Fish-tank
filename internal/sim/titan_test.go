// v1.1: titan pod tests (G39–G43) — visit cadence and pod-size bounds, the
// ponderous pace, the ≥6× lunge burst, the rare-predation gate (giant only,
// starved long enough, cooled down, above the population floor, Chosen
// immune) and the ambient exclusions: never counted, never bred, never
// menu-spawnable, never persisted, never restorable from a save.
package sim

import (
	"math"
	"testing"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

func mathAtan2(y, x float64) float64 { return math.Atan2(y, x) }

func titanTestSpecies() *contract.Species {
	sp := testSpecies(0)
	c := *sp
	c.ID = "test-titan"
	c.Role = contract.RoleTitan
	c.Size = 8.0
	c.Behavior.Speed = 0.3
	return &c
}

func titanWorld(t *testing.T, normals int) *World {
	t.Helper()
	cfg := contract.Config{MaxFish: 30, DaySeconds: 60}
	w := NewWorld(800, 600, cfg, []*contract.Species{titanTestSpecies(), cappedNormalSpecies("test-neon", false)}, nil, nil)
	w.SeedRng(7)
	w.fishes = w.fishes[:0]
	for i := 0; i < normals; i++ {
		p := v2(300+w.rng.Float64()*200, 250+w.rng.Float64()*100)
		w.fishes = append(w.fishes, newFish(cappedNormalSpecies("test-neon", false), w.rng.Int63(), p, 6, w.nextID()))
	}
	w.titanPhase = 2 // roaming
	return w
}

// G39/G59: the pod is a permanent resident -- exactly five members with
// one leader, present from the first frame, never leaving.
func TestTitanPodPermanent(t *testing.T) {
	w := titanWorld(t, 10)
	w.spawnPod()
	const dt = 0.05
	for i := 0; i < 240*20; i++ { // 240 s of sweeps
		w.Update(dt, Input{})
		n := w.titanCount()
		if n != contract.TitanPodMax {
			t.Fatalf("pod size %d, want %d at t=%.0fs", n, contract.TitanPodMax, float64(i)*dt)
		}
		giants := 0
		for _, f := range w.fishes {
			if f.Sp.Role == contract.RoleTitan && f.sizeMul >= 0.99 {
				giants++
			}
		}
		if giants != 1 {
			t.Fatalf("pod has %d giants, want exactly 1", giants)
		}
	}
}

// G39: the ponderous cruise — a giant drifts far below every normal's pace.
func TestTitanPonderousCruise(t *testing.T) {
	g := newFish(titanTestSpecies(), 1, v2(400, 300), 6, 1)
	g.sizeMul = 1
	cruise := g.maxSpeed(1)
	if cruise > 0.45*contract.BaseSpeed || cruise < 0.15*contract.BaseSpeed {
		t.Fatalf("giant cruise %.1f px/s outside the ponderous band", cruise)
	}
	n := newFish(testSpecies(0), 2, v2(410, 300), 6, 2)
	if cruise >= n.maxSpeed(1)*0.6 {
		t.Fatalf("giant cruise %.1f not far below a normal's %.1f", cruise, n.maxSpeed(1))
	}
}

// G41: a hungry giant lunges at flakes at ≥6× its cruise speed.
func TestTitanHungerLungeBurst(t *testing.T) {
	w := titanWorld(t, 3)
	w.spawnPod()
	w.titanPhase = 2
	g := w.titanGiant()
	g.Satiety = 0.2 // hungry
	g.lungeCD = 0
	cruise := g.maxSpeed(1)
	w.foods = append(w.foods, Food{Pos: add(g.Pos, v2(240, 0)), Age: 1})
	const dt = 0.05
	lunged, maxSeen := false, 0.0
	for i := 0; i < 40; i++ {
		g.advance(dt, 0, w)
		if g.lungeT > 0 {
			lunged = true
			// the start tick aims the dive; every mid-burst tick carries the
			// frozen multiplier and the ≥6× speed cap
			if g.lungeT < contract.TitanLungeSec && g.seekBonus < contract.TitanLungeMul {
				t.Fatalf("lunge seekBonus %v below the frozen multiplier", g.seekBonus)
			}
			if s := hyp2(g.Vel); s > maxSeen {
				maxSeen = s
			}
		}
	}
	if !lunged {
		t.Fatal("a starving giant never lunged at the flake")
	}
	if maxSeen < 6*cruise {
		t.Fatalf("lunge burst %.1f px/s below 6× cruise %.1f", maxSeen, 6*cruise)
	}
}

// G42: the rare hunt — every guard in one pass.
func TestPredationGate(t *testing.T) {
	// fed giant: never hunts
	w := titanWorld(t, 10)
	w.spawnPod()
	g := w.titanGiant()
	g.Satiety = 1
	w.tickPredation(1)
	if w.predTgtID != "" {
		t.Fatal("a fed giant started a hunt")
	}
	// starved giant + enough neighbors: the hunt picks a normal, never the Chosen
	g.Satiety = 0
	w.predHunger = contract.PredationSustain // already sustained
	ch := newFish(chosenTestSpecies(), 9, g.Pos, 8, 99)
	w.fishes = append(w.fishes, ch)
	w.tickPredation(0.01)
	if w.predTgtID == "" {
		t.Fatal("a starved giant with neighbors never hunted")
	}
	if tgt := w.fishByID(w.predTgtID); tgt == nil || tgt.Sp.Role != contract.RoleNormal {
		t.Fatal("the hunt targeted something other than a normal fish")
	}
	// the swallow: fast fade, no corpse path, cooldown armed
	tgt := w.fishByID(w.predTgtID)
	tgt.Pos = g.Pos
	w.tickPredation(0.01)
	if !tgt.Dying || tgt.DieReason != "swallowed" || !tgt.fadeFast {
		t.Fatal("the swallow did not register as a fast fade")
	}
	if w.predCD != contract.PredationCD {
		t.Fatalf("post-feeding cooldown %.0f, want %.0f", w.predCD, contract.PredationCD)
	}
	// population floor: too few neighbors — the deep waits
	w2 := titanWorld(t, 5) // below PredationPopFloor (MinPopulation+2 = 6)
	w2.spawnPod()
	g2 := w2.titanGiant()
	g2.Satiety = 0
	w2.predHunger = contract.PredationSustain
	w2.tickPredation(0.01)
	if w2.predTgtID != "" || w2.predCD == 0 {
		t.Fatal("the giant hunted below the population floor")
	}
}

// G43 + ambient: the pod is invisible to the tank's own bookkeeping.
func TestTitanAmbientExclusions(t *testing.T) {
	w := titanWorld(t, 8)
	before := len(w.aliveFishes())
	w.spawnPod()
	if got := len(w.aliveFishes()); got != before {
		t.Fatalf("aliveFishes counted the pod: %d → %d", before, got)
	}
	if w.titanCount() < contract.TitanPodMin {
		t.Fatalf("pod missing: %d titans", w.titanCount())
	}
	// menu/egg path: the deep wanderer answers no egg
	w.SpawnEgg("test-titan")
	if len(w.eggs) != 0 {
		t.Fatal("SpawnEgg created a titan egg — visits cannot be summoned")
	}
	// courtship: titans never pair (candidates exclude the role)
	w2 := titanWorld(t, 0)
	w2.spawnPod()
	if w2.startCourtship(true) {
		t.Fatal("titans started a courtship")
	}
	// snapshots: the visit is never tank history
	s := w.Snapshot()
	for _, sf := range s.Fish {
		if sf.SpeciesID == "test-titan" {
			t.Fatal("a titan leaked into the snapshot")
		}
	}
	// a hand-edited save cannot summon the deep
	w3 := titanWorld(t, 4)
	w3.Restore(contract.Save{SchemaVersion: 2, Fish: []contract.SavedFish{
		{SpeciesID: "test-titan", Pos: v2(400, 300), Seed: 1},
		{SpeciesID: "test-neon", Pos: v2(200, 300), Seed: 2},
	}})
	for _, f := range w3.fishes {
		if f.Sp.Role == contract.RoleTitan {
			t.Fatal("restore resurrected a titan")
		}
	}
	// NewWorld never seeds the pod — visits only
	w4 := NewWorld(800, 600, contract.Config{MaxFish: 10, DaySeconds: 60},
		[]*contract.Species{titanTestSpecies(), testSpecies(0.5)}, nil, nil)
	if w4.titanCount() != 0 {
		t.Fatal("NewWorld seeded the pod instead of scheduling a visit")
	}
}

// G49: a right-click scare bolts the pod away from the point — the giants
// startle like everyone else, at lunge speed, decaying over the window.
func TestTitanScareBolt(t *testing.T) {
	w := titanWorld(t, 3)
	w.spawnPod()
	g := w.titanGiant()
	g.Pos = v2(400, 300)
	sc := v2(620, 300)
	cruise := g.maxSpeed(1)
	g.flee(sc.X, sc.Y, contract.MaxForce) // what the right-click path applies
	startDist := hyp2(sub(g.Pos, sc))
	maxSeen := 0.0
	const dt = 0.05
	for i := 0; i < 40; i++ { // 2 s of bolt
		g.advance(dt, 0, w)
		if s := hyp2(g.Vel); s > maxSeen {
			maxSeen = s
		}
	}
	if maxSeen < 6*cruise {
		t.Fatalf("scare burst %.1f px/s below 6x cruise %.1f", maxSeen, 6*cruise)
	}
	if d := hyp2(sub(g.Pos, sc)); d <= startDist {
		t.Fatalf("the giant did not bolt away from the point (%.0f -> %.0f)", startDist, d)
	}
}

// The curvature clamp (G40 keystone): even after lunges and scares the
// giant's body stays an arc — each segment bends at most TitanSpineBend plus
// the swimming wave, and the silhouette keeps its length on screen.
func TestTitanSpineNeverFolds(t *testing.T) {
	w := titanWorld(t, 4)
	w.spawnPod()
	g := w.titanGiant()
	g.Pos = v2(600, 300)
	g.Satiety = 0.1 // lunges incoming
	const dt = 0.05
	for i := 0; i < 1200; i++ { // 60 s of hungry roaming
		w.Update(dt, Input{MouseActive: false})
		if i%400 == 399 {
			g.flee(g.Pos.X+200, g.Pos.Y, contract.MaxForce) // shock on top
		}
	}
	maxAng := 0.0
	dir := func(a, b contract.Vec2) float64 { return mathAtan2(b.Y-a.Y, b.X-a.X) }
	for i := 2; i < len(g.Spine); i++ {
		a1, a2 := dir(g.Spine[i-2], g.Spine[i-1]), dir(g.Spine[i-1], g.Spine[i])
		d := mathAbs(a2 - a1)
		if d > 3.14159 {
			d = 6.28318 - d
		}
		if d > maxAng {
			maxAng = d
		}
	}
	if maxAng > contract.TitanSpineBend+0.45 {
		t.Fatalf("spine bent %.2f rad between segments (cap %.2f + wave)", maxAng, contract.TitanSpineBend)
	}
	x0, x1, y0, y1 := 1e9, -1e9, 1e9, -1e9
	for _, p := range g.Spine {
		x0 = min(x0, p.X)
		x1 = max(x1, p.X)
		y0 = min(y0, p.Y)
		y1 = max(y1, p.Y)
	}
	if span := max(x1-x0, y1-y0); span < 0.7*g.bodyLen {
		t.Fatalf("folded silhouette: spine span %.0f < 0.7×bodyLen %.0f", span, g.bodyLen)
	}
}
