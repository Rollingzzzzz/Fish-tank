// v1.1: floor critters — crabs and shrimp walking the tank bottom (G44–G45).
// They emerge from the sand at intervals, roam the floor band, and in the
// last stretch of their cycle they struggle in the open: hungry fish feel
// the pull from CreatureLureRadius away and the first to arrive eats. An
// uneaten critter burrows back into the floor — no corpse is ever left. Her
// nest circle is absolute: a critter is projected out of it every tick,
// exactly like the wild mites. Warm amber/rust matte palettes only — the
// neon stays the Chosen's alone, and the v0.3.8 "dark clutter" failure is
// guarded by a measured luminance gate (see render/creature.go).
package sim

import (
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// Creature is one floor critter. Ephemeral, never saved — the Save.Creatures
// field stays write-empty/read-ignored so old snapshots keep loading.
type Creature struct {
	Kind   string // "shrimp" | "crab"
	Seed   int64
	Pos    contract.Vec2
	Phase  float64
	Age    float64
	Life   float64 // walking seconds before the vulnerable turn
	Dir    float64 // ±1 walking direction
	Burrow float64 // 0 = walking .. 1 = fully sunk into the floor
	pauseT float64
}

// Vulnerable reports the end-of-life phase: struggling in the open, luring
// hungry fish, edible.
func (c *Creature) Vulnerable() bool {
	return c.Burrow <= 0 && c.Life-c.Age <= contract.CreatureVulnSec
}

// Critters returns the live floor critters (renderer/evidence accessors).
func (w *World) Critters() []*Creature { return w.creatures }

// creatureCap scales the floor-critter budget with the tank area (v0.3.1
// pattern): the sandline is covered, never carpeted.
func (w *World) creatureCap() int {
	return min(int(contract.CreatureCapBase*w.density+0.5), contract.CreatureCapMax)
}

// spawnCreature picks a kind and an emergence point along the sand — never
// inside her circle (spawn-time check; the per-tick projection is the law).
func (w *World) spawnCreature() {
	kind := "shrimp"
	if w.rng.Float64() < 0.5 {
		kind = "crab"
	}
	for attempt := 0; attempt < 20; attempt++ {
		p := v2(w.W*(0.05+w.rng.Float64()*0.90), w.H*0.925)
		clear := true
		for _, z := range w.zones {
			if z.Owner == "chosen" && hyp2(sub(p, z.Center)) < z.Radius+20 {
				clear = false
			}
		}
		if !clear {
			continue
		}
		c := &Creature{
			Kind: kind, Seed: w.rng.Int63(), Pos: p,
			Phase: w.rng.Float64() * 6.283,
			Life:  contract.CreatureLifeMin + w.rng.Float64()*(contract.CreatureLifeMax-contract.CreatureLifeMin),
			Dir:   1,
		}
		if w.rng.Float64() < 0.5 {
			c.Dir = -1
		}
		w.creatures = append(w.creatures, c)
		w.logf("nature", "a "+kind+" emerges from the sand")
		return
	}
}

// tickCreatures advances emergence, walking, the vulnerable turn, the lure
// consumption and the burrow exit.
func (w *World) tickCreatures(dt float64) {
	w.creatureT -= dt
	if w.creatureT <= 0 {
		w.creatureT = contract.CreatureSpawnMean * (0.5 + w.rng.Float64())
		if len(w.creatures) < w.creatureCap() {
			w.spawnCreature()
		}
	}
	kept := w.creatures[:0]
	for _, c := range w.creatures {
		if c.Burrow > 0 {
			c.Burrow += dt / contract.CreatureBurrowSec
			if c.Burrow < 1 {
				kept = append(kept, c)
			}
			continue
		}
		c.Age += dt
		if c.Age >= c.Life {
			w.logf("nature", "a "+c.Kind+" sinks back into the floor")
			c.Burrow = dt / contract.CreatureBurrowSec
			kept = append(kept, c)
			continue
		}
		w.walkCritter(c, dt)
		w.enforceZonesPos(&c.Pos) // her circle is absolute (N3, G46)
		// consumption (G45): only the struggling phase is edible — the walk
		// is hard shell; the first hungry fish in reach takes it via the
		// same pipeline mites use (eat + care + sparkle, one log line)
		eaten := false
		if c.Vulnerable() {
			for _, f := range w.fishes {
				if f.Dying || f.Satiety > 0.85 {
					continue
				}
				if hyp2(sub(c.Pos, f.Pos)) < f.bodyLen*0.2 {
					f.eat()
					w.addCare(contract.CareFeedScore)
					w.burst(c.Pos, "#ffd9a0", 8)
					w.logf("nature", f.Sp.Name+" takes the struggling "+c.Kind)
					eaten = true
					break
				}
			}
		}
		if !eaten {
			kept = append(kept, c)
		}
	}
	w.creatures = kept
}

// walkCritter moves one critter along the sand band: crabs strut sideways,
// shrimp row forward; both pause, turn, and slow to a helpless struggle in
// the vulnerable phase.
func (w *World) walkCritter(c *Creature, dt float64) {
	c.Phase += dt * 5
	if c.pauseT > 0 {
		c.pauseT -= dt
		return
	}
	if w.rng.Float64() < 0.25*dt {
		c.pauseT = 0.4 + w.rng.Float64()*0.8
	}
	if w.rng.Float64() < 0.06*dt {
		c.Dir = -c.Dir
	}
	speed := contract.CreatureWalkSpeed
	if c.Vulnerable() {
		speed *= 0.3 // the struggle barely moves
	}
	c.Pos.X += c.Dir * speed * dt
	if c.Pos.X < w.W*0.05 {
		c.Dir = 1
	}
	if c.Pos.X > w.W*0.95 {
		c.Dir = -1
	}
	c.Pos.Y += (w.H*0.925 - c.Pos.Y) * minF(1, dt*2)
}

// DebugStageCritters force-spawns n critters spread across the sand —
// evidence harness only (-probe): it skips the spawn cadence and the cap on
// purpose (n ≤ 8 keeps the composition sane).
func (w *World) DebugStageCritters(n int) {
	w.creatures = w.creatures[:0]
	for i := 0; i < n; i++ {
		x := w.W * (0.10 + 0.80*float64(i)/float64(max(n-1, 1)))
		p := v2(x, w.H*0.925)
		clear := true
		for _, z := range w.zones {
			if z.Owner == "chosen" && hyp2(sub(p, z.Center)) < z.Radius+30 {
				clear = false
			}
		}
		if !clear {
			continue
		}
		kind := "shrimp"
		if i%2 == 0 {
			kind = "crab"
		}
		w.creatures = append(w.creatures, &Creature{
			Kind: kind, Seed: int64(i + 1), Pos: p, Phase: float64(i) * 1.1,
			Life: contract.CreatureLifeMax, Dir: 1,
		})
	}
}
