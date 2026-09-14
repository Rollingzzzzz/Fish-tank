// newFish split from fish.go (line ceiling) — birth-time state only.
package sim

import (
	"fmt"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// newFish builds a fish of the species at pos with the given seed.
func newFish(sp *contract.Species, seed int64, pos contract.Vec2, ageDays float64, idSeq int) *Fish {
	f := &Fish{
		ID:      fmt.Sprintf("f-%d", idSeq),
		Sp:      sp,
		Seed:    seed,
		rng:     contract.RandSeed(seed),
		Pos:     pos,
		AgeDays: ageDays,
		Stage:   StageFor(ageDays),
		Satiety: 0.6 + 0.4*contract.RandSeed(seed+1).Float64(),
		Energy:  1,
		Fade:    1,
	}
	f.Pal = sp.Palette
	f.Pat = sp.Pattern
	f.wanderA = f.rng.Float64() * 6.283
	f.zPhase = f.rng.Float64() * 6.283
	f.zSpeed = 0.05 + f.rng.Float64()*0.08
	f.z = 0.5
	// G93: the first urge to visit the glass arrives within a couple of
	// minutes; curious species come sooner. The roll draws from its OWN
	// seed stream — the gameplay rng stream must stay bit-identical to the
	// pre-gaze tank (an extra draw here re-rolled every later wander and
	// lounge draw and broke the tuned convoy trajectories).
	cur := 0.6 + 0.8*contract.Clamp(sp.Behavior.Curiosity, 0, 1)
	gseed := contract.RandSeed(seed + 3)
	f.gazeCD = (contract.GazeCDMin + gseed.Float64()*(contract.GazeCDMax-contract.GazeCDMin)) / cur
	f.loungeNext = contract.LoungeMeanSec * (0.5 + f.rng.Float64()) // F15 stagger
	f.bodyLen = f.targetLen()
	f.segLen = f.bodyLen / (contract.SpineSegments - 1)
	f.Spine = make([]contract.Vec2, contract.SpineSegments)
	for i := range f.Spine {
		f.Spine[i] = v2(pos.X-float64(i)*f.segLen, pos.Y)
	}
	// G90: born SETTLED — the chain is laid through the same pass every
	// later frame rides, so the first live frame eases the wave in instead
	// of snapping a flat-laid rope into a swimming body (the same pattern
	// the save-restore uses).
	f.followSpine(0)
	return f
}
