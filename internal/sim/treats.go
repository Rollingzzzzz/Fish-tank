// N8: live treats dropped from the Treat Store.
// LANE C OWNS the movement/consumption bodies (tickTreats + DropTreat spawn
// nuances); the types and accessors below are the frozen M0 API the game
// loop and the steering code already use.
package sim

import (
	"math"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// Treat tuning.
const (
	treatTTLSec = 30.0 // live food dissolves after half a minute
	treatBiteCd = 0.35 // seconds between nibbles on one treat
	bugRise     = 80.0 // px/s buoyant ascent
	bugSkitter  = 60.0 // px/s horizontal impulse magnitude
	bugMaxSpeed = 90.0 // px/s speed cap
	bugBandLo   = 18.0 // surface band the bug settles into
	bugBandHi   = 30.0
	shrimpPanic = 150.0 // px/s panic dash away from a fish
)

// Treat is one live-food item dropped by the player.
type Treat struct {
	Kind      string // contract.TreatBug/Worm/Shrimp/Chicken
	Pos       contract.Vec2
	Vel       contract.Vec2
	Phase     float64 // animation phase for the renderer
	Age       float64
	BitesLeft int // worm/chicken take several nibbles

	// unexported per-kind motion state (Lane C)
	timer float64 // seconds until the next skitter/panic/shed event
	dartT float64 // remaining shrimp panic-dash time
	cool  float64 // bite cooldown
	dart  contract.Vec2
}

// DropTreat spawns a live treat at pos (mouse release point in tank space).
// Entry velocity is refined per kind: bugs pop upward toward the surface,
// shrimp tumble into a tumble-sink, worms and chicken just settle.
func (w *World) DropTreat(kind string, pos contract.Vec2) {
	if len(w.treats) >= contract.TreatMax {
		w.treats[0].Age = 1e9 // oldest dissolves to make room
	}
	w.treats = append(w.treats, &Treat{
		Kind: kind, Pos: pos, Phase: w.rng.Float64() * 6.283,
		Vel:       entryVel(kind, w.rng),
		BitesLeft: defaultBites(kind),
		timer:     0.3,
	})
	w.logf("treat", "a live "+kind+" enters the tank")
}

// entryVel gives each kind its opening push.
func entryVel(kind string, r *contractRand) contract.Vec2 {
	switch kind {
	case contract.TreatBug:
		return v2(r.Float64()*20-10, -bugRise)
	case contract.TreatShrimp:
		return v2(r.Float64()*30-15, 24)
	default:
		return v2(0, 8)
	}
}

// Treats returns live treats (steering reads this for the rush behavior).
func (w *World) Treats() []*Treat { return w.treats }

// heldTreatTarget returns the point on the keep-back ring (N11) around the
// treat held at the cursor where this fish should hover — the nearest ring
// point, so the school crowds and orbits just out of reach. ok=false when no
// treat is held, the belly is full (Satiety ≥ 0.95) or it wriggles too far
// away. The held treat is NOT in w.treats: no bite can ever land on it.
// F26: the scent radius grows with carry time, capped — the smell spreads
// through the tank the longer the bait hangs at the cursor.
func (w *World) heldTreatTarget(f *Fish) (*contract.Vec2, bool) {
	ht := w.input.HeldTreat
	if ht == nil || f.Dying || f.Satiety >= 0.95 {
		return nil, false
	}
	radius := contract.HeldTreatRadius +
		math.Min(contract.HeldScentGrowCap, ht.Age*contract.HeldScentGrowPx)
	d := sub(f.Pos, ht.Pos)
	if hyp2(d) >= radius {
		return nil, false
	}
	p := add(ht.Pos, mulS(norm2(d), contract.HeldTreatRing))
	return &p, true
}

// defaultBites maps kind → nibbles needed to consume it.
func defaultBites(kind string) int {
	switch kind {
	case contract.TreatWorm:
		return 4
	case contract.TreatChicken:
		return 4
	case contract.TreatShrimp:
		return 1
	default:
		return 1
	}
}

// tickTreats advances treat motion and consumption.
func (w *World) tickTreats(dt float64) {
	for _, tr := range w.treats {
		tr.Age += dt
		if tr.cool > 0 {
			tr.cool -= dt
		}
		switch tr.Kind {
		case contract.TreatBug:
			w.tickBug(tr, dt)
		case contract.TreatWorm:
			w.tickWorm(tr, dt)
		case contract.TreatShrimp:
			w.tickShrimp(tr, dt)
		default:
			w.tickChicken(tr, dt)
		}
		tr.Pos.X = clampF(tr.Pos.X, 10, w.W-10)
		tr.Pos.Y = clampF(tr.Pos.Y, 14, w.H-8)
	}
	w.treatBites()
	w.compactTreats()
}

// tickBug: buoyant rise into the surface band, then nervous skittering.
func (w *World) tickBug(tr *Treat, dt float64) {
	tr.Phase += 10 * dt // fast wobble for the renderer
	tr.timer -= dt
	if tr.timer <= 0 {
		tr.timer = 0.22 + w.rng.Float64()*0.16
		if tr.Pos.Y >= bugBandLo && tr.Pos.Y <= bugBandHi {
			tr.Vel.X = (w.rng.Float64()*2 - 1) * bugSkitter
		}
	}
	switch {
	case tr.Pos.Y > bugBandHi:
		tr.Vel.Y = -bugRise // buoyant: make for the surface
	case tr.Pos.Y < bugBandLo:
		tr.Vel.Y = 18 // stay inside the band
	default:
		tr.Vel.Y *= maxF(0, 1-8*dt)
	}
	if s := hyp2(tr.Vel); s > bugMaxSpeed {
		tr.Vel = mulS(tr.Vel, bugMaxSpeed/s)
	}
	tr.Pos.X += tr.Vel.X * dt
	tr.Pos.Y += tr.Vel.Y * dt
}

// tickWorm: slow sink with a sinusoidal wiggle (renderer reads Phase).
func (w *World) tickWorm(tr *Treat, dt float64) {
	tr.Phase += 6 * dt
	tr.Pos.Y += 12 * dt
	tr.Pos.X += sin(w.time*6+tr.Phase) * 14 * dt
}

// tickShrimp: sink, occasionally panic-dashing away from the nearest fish.
func (w *World) tickShrimp(tr *Treat, dt float64) {
	tr.Phase += 4 * dt
	tr.Pos.Y += 20 * dt
	tr.timer -= dt
	if tr.timer <= 0 {
		tr.timer = 0.8 + w.rng.Float64()*0.6
		if f := w.nearestTreatFish(tr.Pos, 160); f != nil {
			d := sub(tr.Pos, f.Pos)
			tr.dart = mulS(d, 1/maxF(hyp2(d), 1))
			tr.dartT = 0.35
		}
	}
	if tr.dartT > 0 {
		tr.dartT -= dt
		tr.Pos.X += tr.dart.X * shrimpPanic * dt
		tr.Pos.Y += tr.dart.Y * shrimpPanic * dt
	} else {
		tr.Pos.X += sin(w.time*3+tr.Phase) * 8 * dt // idle drift
	}
}

// tickChicken: near-static slow sink that sheds warm crumbs.
func (w *World) tickChicken(tr *Treat, dt float64) {
	tr.Phase += 2 * dt
	tr.Pos.Y += 8 * dt
	tr.Pos.X += sin(w.time*1.2+tr.Phase) * 3 * dt
	tr.timer -= dt
	if tr.timer <= 0 {
		tr.timer = 1.8 + w.rng.Float64()*0.5
		w.burst(tr.Pos, "#ffd9a0", 2)
	}
}

// nearestTreatFish finds the closest fish within radius (dying fish ignored).
func (w *World) nearestTreatFish(p contract.Vec2, radius float64) *Fish {
	var best *Fish
	bestD := radius
	for _, f := range w.fishes {
		if f.Dying {
			continue
		}
		if d := hyp2(sub(f.Pos, p)); d < bestD {
			bestD = d
			best = f
		}
	}
	return best
}

// treatBites lets hungry fish nibble treats inside their mouth reach (the
// same reach flakes use).
func (w *World) treatBites() {
	for _, tr := range w.treats {
		if tr.cool > 0 {
			continue
		}
		for _, f := range w.fishes {
			if f.Dying || f.Satiety > 0.98 {
				continue
			}
			if hyp2(sub(tr.Pos, f.Pos)) < f.bodyLen*0.16*1.2 {
				tr.BitesLeft--
				tr.cool = treatBiteCd
				f.eat()
				w.addCare(contract.CareFeedScore)
				w.burst(tr.Pos, "#ffe9a0", 4)
				break
			}
		}
	}
}

// compactTreats drops devoured or expired treats and logs the outcome.
func (w *World) compactTreats() {
	kept := w.treats[:0]
	for _, tr := range w.treats {
		switch {
		case tr.BitesLeft <= 0:
			w.logf("treat", "the school devours the "+tr.Kind)
		case tr.Age > treatTTLSec:
			w.logf("treat", "the "+tr.Kind+" slips away")
		default:
			kept = append(kept, tr)
		}
	}
	w.treats = kept
}
