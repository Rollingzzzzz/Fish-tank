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
	f.loungeNext = contract.LoungeMeanSec * (0.5 + f.rng.Float64()) // F15 stagger
	f.bodyLen = f.targetLen()
	f.segLen = f.bodyLen / (contract.SpineSegments - 1)
	f.Spine = make([]contract.Vec2, contract.SpineSegments)
	for i := range f.Spine {
		f.Spine[i] = v2(pos.X-float64(i)*f.segLen, pos.Y)
	}
	return f
}
