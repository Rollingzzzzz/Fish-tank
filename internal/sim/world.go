// G2.4: world orchestration — populations, clock, water crossfade, care,
// repaint queue, snapshots; the single object the game loop ticks.
package sim

import (
	"time"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// SFX event names are gone (FD7: the tank has no sound effects — music only).

// WaterLive is the crossfaded water state consumed by the renderer.
type WaterLive struct {
	Top, Bottom, Accent [3]float64
	Rays, Caustics      float64
	Bubbles             float64
	Night, DayFactor    float64
	TOD                 float64 // F9: time of day 0..1
	Time                float64
}

// LogEntry is one tank-log line.
type LogEntry struct{ At, Who, Text string }

// Plant is a placed plant instance.
type Plant struct {
	Def *contract.PlantDesign
	X   float64
	Y   float64
	Sc  float64
}

// Input carries per-frame player input into the world.
type Input struct {
	MouseX, MouseY float64
	MouseActive    bool
	MouseSpeed     float64 // px/s, fast swipes scare fish
	FeedAt         []contract.Vec2
	ScareAt        *contract.Vec2 // N10: right-click scare point (nil = none)
	HeldTreat      *HeldTreat     // N11: treat held at the cursor (nil = none)
}

// Courtship pairs two courting fish.
type Courtship struct {
	a, b *Fish
	t    float64
}

// World is the whole simulation.
type World struct {
	W, H    float64
	cfg     contract.Config
	species []*contract.Species
	waters  []*contract.WaterPreset

	plants    []*Plant
	corals    []*Coral
	mites     []*Mite
	treats    []*Treat
	fishes    []*Fish
	foods     []Food
	eggs      []Egg
	particles []Particle
	bubbles   []Bubble

	zones     []contract.Zone
	holes     []contract.Hole // F25: enterable crag mouths (door-to-door transit)
	rockSeed  int64
	coralDefs []*contract.CoralDesign // remembered for Restore re-placement

	Clock float64 // seconds inside the day cycle (wraps per cycle)
	time  float64 // total sim seconds
	Day   int     // full tank days elapsed
	Care  float64

	density  float64 // v0.3.1: canvas area ÷ reference area (1..DensityMax)
	popCap   int     // v0.3.1: area-scaled live-fish ceiling
	rockBase float64 // v0.3.2: floor width occupied by the rock field (px)

	CoralIDs []string

	careThreshIdx int
	watchAcc      float64
	nextFeedIn    float64
	courtCD       float64 // recurring courtship cooldown (F6)

	// F4 behavior event timers + counters (soak-test evidence)
	evStartle, evChase, evZoom, evNudge float64
	startles, chases, zoomies, nudges   int
	lounges                             int     // F15 cave-lounge count (soak evidence)
	miteT                               float64 // N7 spawn cadence

	// v1.1 ambient features (ExtrasEnabled off for A/B evidence runs)
	titanPhase int     // 0 absent, 1 entering, 2 roaming, 3 exiting
	titanT     float64 // countdown of the current phase
	predCD     float64 // tank-wide predation cooldown
	predHunger float64 // sustained starving seconds on the current giant
	predTgtID  string  // pursued fish ("" = none)
	predT      float64 // pursuit seconds left before breaking off
	creatures  []*Creature
	creatureT  float64
	sand       []float64 // v1.1: floor disturbance offsets, 0 = level bed

	WaterCur WaterLive
	WaterTgt WaterLive
	WaterID  string
	PlantIDs []string

	Log        []LogEntry
	repaintQ   []contract.PatternRequest
	courtships []*Courtship

	rng   *contractRand
	idSeq int
	input Input
}

// NewWorld builds the tank: places plants, seeds an initial population and
// applies the starting water preset.
func NewWorld(w, h float64, cfg contract.Config, species []*contract.Species,
	plants []*contract.PlantDesign, waters []*contract.WaterPreset) *World {
	world := &World{
		W: w, H: h, cfg: cfg,
		species:  species,
		waters:   waters,
		rockSeed: time.Now().UnixNano(),
		rng:      contractRandFrom(contract.RandSeed(time.Now().UnixNano())),
	}
	// v0.3.1: the screen IS the tank — a bigger canvas is a bigger aquarium:
	// population, garden and critter budgets scale with area (capped).
	world.density = clampF(w*h/(contract.DensityRefW*contract.DensityRefH), 1, contract.DensityMax)
	world.popCap = min(int(float64(cfg.MaxFish)*world.density), contract.PopCapMax)
	// initial population: up to 4 species (N9 variety) — the school fills up
	// to the slider value (v1: a fresh tank opens with MaxFish live fish, the
	// Chosen included), the area-scaled popCap remains the breeding ceiling
	target := min(int(cfg.MaxFish), world.popCap) - 1 // the Chosen takes the last seat
	per := max(1, target/4)
	count, kinds := 0, 0
	for _, sp := range species {
		if sp.Role == contract.RoleChosen || sp.Role == contract.RoleTitan || sp.Role == contract.RoleShark {
			continue // the Chosen alone (FD9); titans visit, sharks arrive via ensureSharks
		}
		if kinds >= 4 {
			break
		}
		kinds++
		capK := per
		if kinds == 4 {
			capK = target - count // the last school absorbs the remainder
		}
		for k := 0; k < capK; k++ {
			p := v2(w*(0.15+world.rng.Float64()*0.7), h*(0.25+world.rng.Float64()*0.55))
			world.fishes = append(world.fishes, newFish(sp, world.rng.Int63(), p, 6+world.rng.Float64()*3, world.nextID()))
			count++
		}
	}
	world.ensureChosen()
	// plants along the floor — double-capped (N7 render budget, F14 majority):
	// agents keep designing plants forever, but live fish must always outnumber
	// them, so the school is seeded first and the cap derives from it
	placeN := max(0, min(len(plants), world.effectivePlantCap()))
	for i := 0; i < placeN; i++ {
		pd := plants[i]
		var x float64
		if placeN > 1 {
			x = w * (0.10 + 0.80*float64(i)/float64(placeN-1))
		} else {
			x = w * 0.5
		}
		world.plants = append(world.plants, &Plant{Def: pd, X: x, Y: h - 6, Sc: 0.85 + world.rng.Float64()*0.4})
		world.PlantIDs = append(world.PlantIDs, pd.ID)
	}
	// starting water
	if len(waters) > 0 {
		world.ApplyWater(waters[0])
		world.WaterCur = world.WaterTgt // no fade on boot
	}
	world.logf("tank", "the tank wakes up")
	// v1.1: the first titan visit comes early enough to be discovered, the
	// rest keep the full gap cadence
	world.titanT = 45 + world.rng.Float64()*45
	world.creatureT = 2 + world.rng.Float64()*4
	return world
}

// ensureChosen guarantees exactly one Chosen fish exists (FD9): immortal,
// outside the MaxFish economy, re-spawned whenever missing.
func (w *World) ensureChosen() {
	var sp *contract.Species
	for _, s := range w.species {
		if s.Role == contract.RoleChosen {
			sp = s
			break
		}
	}
	if sp == nil {
		return
	}
	for _, f := range w.fishes {
		if f.Sp.Role == contract.RoleChosen && !f.Dying {
			return // the eternal one endures
		}
	}
	// home is the aura center when the rock layout is installed
	home := v2(w.W*0.62, w.H*0.5)
	for _, z := range w.zones {
		if z.Owner == "chosen" {
			home = add(z.Center, v2(0, -60))
		}
	}
	w.fishes = append(w.fishes, newFish(sp, w.rng.Int63(), home, 8, w.nextID()))
	w.logf("life", "the eternal one glides into view")
}

// Update advances the whole simulation by dt seconds.
func (w *World) Update(dt float64, in Input) {
	dt = minF(dt, 0.05) // clamp hitches (D4)
	w.input = in
	w.time += dt
	w.Clock += dt

	// water crossfade
	k := 1 - expF(-dt*0.8)
	w.WaterCur = lerpWater(w.WaterCur, w.WaterTgt, k)

	// care + auto feed
	w.tickCare(dt)
	w.nextFeedIn -= dt
	if w.cfg.AutoFeed && w.nextFeedIn <= 0 {
		// F11 maintenance feeding: hold the school at a comfortable satiety
		// band, scale with population, never carpet the floor with flakes.
		sum, n := 0.0, 0
		for _, f := range w.fishes {
			if !f.Dying && f.Sp.Role != contract.RoleTitan && f.Sp.Role != contract.RoleShark {
				// v1.1: the pod's hunger is its own drama — giants never drag
				// the maintenance average down
				sum += f.Satiety
				n++
			}
		}
		avg := 1.0
		if n > 0 {
			avg = sum / float64(n)
		}
		if avg < 0.7 && len(w.foods) < 8 {
			w.DropFood(w.W*(0.15+w.rng.Float64()*0.7), 60)
			w.nextFeedIn = 2.4
		} else {
			w.nextFeedIn = 1.0
		}
	}
	for _, p := range in.FeedAt {
		w.DropFood(p.X, p.Y)
	}

	// food + eating
	w.tickFood(dt)
	w.tryEat()
	w.tickTreats(dt)

	// fish life + movement
	for _, f := range w.fishes {
		w.tickLife(f, dt)
	}
	w.sweepCorpses() // logs departures, then re-asserts the F14 plant majority
	w.ensureChosen() // F23: if the eternal one was ever swept, she returns
	w.ensureSharks() // v1.1: the resident pair stays whole
	for _, f := range w.fishes {
		// fast mouse swipes scatter nearby fish
		if in.MouseActive && in.MouseSpeed > 900 {
			d := hyp2(sub(f.Pos, v2(in.MouseX, in.MouseY)))
			if d < 110 {
				f.flee(in.MouseX, in.MouseY, contract.MaxForce*1.1)
			}
		}
		// N10: right-click scare — a sharp pulse from the point; the caller
		// owns the pointer's lifetime, one non-nil frame = one scare event
		if in.ScareAt != nil && hyp2(sub(f.Pos, *in.ScareAt)) < contract.ScareRadius {
			f.flee(in.ScareAt.X, in.ScareAt.Y, contract.MaxForce*(0.95+0.3*w.rng.Float64()))
		}
		f.advance(dt, 1-w.dayFactor(), w)
	}

	w.tickCourtships(dt)
	w.tickEggs(dt)
	w.tickParticles(dt)
	w.tickDecor(dt)
	w.tickBehaviors(dt)
	if contract.ExtrasEnabled { // v1.1 ambient features; TANK_EXTRAS=0 = A/B off
		w.tickTitans(dt)
		w.tickCreatures(dt)
	}

	// day counter (F8/F9): a full DaySeconds elapse = one tank day
	if d := int(w.Clock / maxF(w.cfg.DaySeconds, 1)); d > w.Day {
		w.Day = d
		w.logf("time", "day "+itoa(w.Day)+" begins")
	}
}
