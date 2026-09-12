// G2.2: food flakes, eggs and particles (sparkles + bubbles).
package sim

import (
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// Food is one flake sinking toward the floor.
type Food struct {
	Pos  contract.Vec2
	Vel  contract.Vec2
	Age  float64
	Seed float64
	Held string // fish ID currently targeted (informational)
}

const (
	foodSink   = 22.0 // px/s
	foodTTLSec = 25.0
)

// tickFood advances flakes (they dissolve after their TTL).
func (w *World) tickFood(dt float64) {
	kept := w.foods[:0]
	for i := range w.foods {
		fd := &w.foods[i]
		fd.Age += dt
		if fd.Pos.Y < w.H-10 {
			fd.Vel.Y = minF(fd.Vel.Y+dt*30, foodSink)
			fd.Pos.X += (fd.Vel.X + sin(w.time*3+fd.Seed)*8) * dt
			fd.Pos.Y += fd.Vel.Y * dt
			fd.Vel.X *= 1 - 0.6*dt
		}
		if fd.Age > foodTTLSec {
			continue // dissolved
		}
		kept = append(kept, *fd)
	}
	w.foods = kept
}

// tryEat lets each fish consume the nearest flake inside its mouth radius.
func (w *World) tryEat() {
	for _, f := range w.fishes {
		if f.Dying || f.Satiety > 0.98 {
			continue
		}
		reach := f.bodyLen * 0.16 * 1.2
		for i := range w.foods {
			fd := &w.foods[i]
			if fd.Age < 0.15 {
				continue // just spawned, give everyone a chance
			}
			if hyp2(sub(fd.Pos, f.Pos)) < reach {
				f.eat()
				w.addCare(contract.CareFeedScore)
				w.burst(fd.Pos, f.Pal.Accent, 6)
				// remove by swap
				w.foods[i] = w.foods[len(w.foods)-1]
				w.foods = w.foods[:len(w.foods)-1]
				break
			}
		}
	}
}

// Egg waits EggHatchSec then becomes 1-2 fry.
type Egg struct {
	SpeciesID string
	Pos       contract.Vec2
	Progress  float64 // 0..1
	Seed      int64
}

// tickEggs ages eggs and hatches them.
func (w *World) tickEggs(dt float64) {
	kept := w.eggs[:0]
	for i := range w.eggs {
		e := &w.eggs[i]
		e.Progress += dt / contract.EggHatchSec
		if e.Progress < 1 {
			kept = append(kept, *e)
			continue
		}
		sp := w.storeSpecies(e.SpeciesID)
		if sp == nil || len(w.aliveFishes()) >= w.popCap {
			if sp != nil {
				w.cullOldestElder() // F6: full tank — the eldest makes room
			}
			kept = append(kept, *e) // waits until there is room
			continue
		}
		n := 1 + int(contract.RandSeed(e.Seed).Float64()*2) // 1-2 fry
		for k := 0; k < n && len(w.aliveFishes()) < w.popCap; k++ {
			f := newFish(sp, w.rng.Int63(), add(e.Pos, v2(w.rng.Float64()*14-7, w.rng.Float64()*10-5)), 0, w.nextID())
			w.fishes = append(w.fishes, f)
			w.burst(f.Pos, sp.Palette.Accent, 10)
		}
		w.logf("life", sp.Name+" hatched")
	}
	w.eggs = kept
}

// Particle is a short-lived sparkle.
type Particle struct {
	Pos, Vel contract.Vec2
	Life     float64
	Max      float64
	Color    string
	Size     float64
}

// Bubble rises to the surface.
type Bubble struct {
	Pos    contract.Vec2
	R      float64
	Speed  float64
	Wobble float64
	Seed   float64
}

const maxParticles = 256
const maxBubbles = 96

// burst spawns a sparkle explosion at p.
func (w *World) burst(p contract.Vec2, colorHex string, n int) {
	for i := 0; i < n && len(w.particles) < maxParticles; i++ {
		a := w.rng.Float64() * 6.283
		sp := 20 + w.rng.Float64()*60
		w.particles = append(w.particles, Particle{
			Pos: p, Vel: v2(cos(a)*sp, sin(a)*sp),
			Life: 0.5 + w.rng.Float64()*0.4, Max: 0.9,
			Color: colorHex, Size: 1 + w.rng.Float64()*2,
		})
	}
}

// tickParticles advances sparkles and bubbles; spawns bubbles per the water.
func (w *World) tickParticles(dt float64) {
	kept := w.particles[:0]
	for _, p := range w.particles {
		p.Life -= dt
		if p.Life <= 0 {
			continue
		}
		p.Pos.X += p.Vel.X * dt
		p.Pos.Y += p.Vel.Y * dt
		p.Vel.Y += 14 * dt
		kept = append(kept, p)
	}
	w.particles = kept

	// ambient bubbles ∝ water setting
	if len(w.bubbles) < maxBubbles && w.rng.Float64() < w.WaterCur.Bubbles*2.2*dt {
		w.bubbles = append(w.bubbles, Bubble{
			Pos:    v2(w.rng.Float64()*w.W, w.H-4),
			R:      1 + w.rng.Float64()*2.6,
			Speed:  26 + w.rng.Float64()*34,
			Wobble: w.rng.Float64() * 6.283,
			Seed:   w.rng.Float64(),
		})
	}
	keptB := w.bubbles[:0]
	for _, b := range w.bubbles {
		b.Pos.Y -= b.Speed * dt
		b.Pos.X += sin(w.time*2.2+b.Wobble) * 9 * dt
		if b.Pos.Y > -4 {
			keptB = append(keptB, b)
		}
	}
	w.bubbles = keptB
}
