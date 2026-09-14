// v1.1 split (300-line ceiling): pod construction + leader lookup live
// here; steering and the convoy arc stay in titan.go.
package sim

import (
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// titanGiant returns the pod leader (sizeMul ~ 1) -- the only hunter.
func (w *World) titanGiant() *Fish {
	for _, f := range w.fishes {
		if f.Sp.Role == contract.RoleTitan && !f.Dying && f.sizeMul >= 0.99 {
			return f
		}
	}
	return nil
}

// spawnPod builds the resident pod: 1 leader + 4 escorts, already inside
// the tank in a loose diagonal, all sweeping the same direction.
func (w *World) spawnPod() {
	sp := w.titanSpecies()
	if sp == nil {
		return
	}
	dir := 1.0
	if w.rng.Float64() < 0.5 {
		dir = -1.0
	}
	leadX := w.W * (0.30 + w.rng.Float64()*0.40)
	y := w.H * (0.30 + w.rng.Float64()*0.10)
	for i := 0; i < contract.TitanPodMax; i++ {
		mul := 1.0
		if i > 0 {
			mul = 0.31 + w.rng.Float64()*0.25 // escorts: ~130-230 px bodies
		}
		// staggered parallel slots behind the leader (G59) -- a loose
		// procession whose bodies overlap a little, never a stack. G84:
		// members are BORN on their slot (slotBack behind, slotY across) --
		// no opening shuffle, the procession exists from frame one
		slotBack := float64(i) * 45
		slotY := float64(i%3-1) * 34
		f := newFish(sp, w.rng.Int63(), v2(leadX-slotBack*dir, y+slotY), 6, w.nextID())
		f.slotBack, f.slotY = slotBack, slotY
		f.sizeMul = mul
		f.cruise = dir
		f.headingA = 0
		if dir < 0 {
			f.headingA = 3.14159
		}
		f.Vel = v2(dir*15, 0) // born under way along the sweep line
		// G84: born SETTLED — a personal altitude inside the cruising band
		// and the next draw a while away. Unset, altY=0 pulled the whole
		// pod toward the surface for the first ~12 s (the "fast and weird
		// at opening, settling later" read)
		f.altY = clampF(y/w.H+(f.slotY/w.H)*0.5, 0.24, 0.40) // born at the altitude they hold, slots breathing
		f.altT = 8 + w.rng.Float64()*16
		f.bodyLen = f.targetLen()
		f.segLen = f.bodyLen / (contract.SpineSegments - 1)
		for j := range f.Spine {
			f.Spine[j] = v2(f.Pos.X-float64(j)*f.segLen*dir, f.Pos.Y)
		}
		// layout hygiene: the laid chain stays inside the canvas (G66) --
		// test tanks are small, and a clamp-folded spawn reads as a glitch
		for j := range f.Spine {
			f.Spine[j].X = clampF(f.Spine[j].X, 10, w.W-10)
			f.Spine[j].Y = clampF(f.Spine[j].Y, 10, w.H-10)
		}
		f.followSpine(0)       // G90: born settled onto her own laws
		w.placeOutsideZones(f) // G92: the procession arrives outside her circle
		w.fishes = append(w.fishes, f)
	}
	w.logf("nature", "the silver elders glide in -- five shadows, one drift")
}
