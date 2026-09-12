// v0.2: decor + zones — placed corals (N1), sim zones (N2 caves / N3 aura),
// wild water mites (N7), rock seed and day counter. v0.3.8: the crustaceans
// (N4) were retired — their dark floor silhouettes read as clutter.
package sim

import (
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// absF is the package-local float absolute value.
func absF(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// Coral is a placed coral instance (mirrors Plant).
type Coral struct {
	Def *contract.CoralDesign
	X   float64
	Y   float64
	Sc  float64
}

// Mite is a wild water mite (N7) — ephemeral, never saved.
type Mite struct {
	Pos    contract.Vec2
	Phase  float64
	Anchor contract.Vec2
	TTL    float64
}

// spacingOK enforces the N7 ecosystem balance: a new anchor must keep a
// minimum distance from every existing plant/coral so the tank is never
// carpeted by greenery.
func (w *World) spacingOK(x float64) bool {
	return w.spacingOKGap(x, 70)
}

// spacingOKGap is spacingOK with a tunable gap (corals may sit closer).
func (w *World) spacingOKGap(x float64, gap float64) bool {
	for _, pl := range w.plants {
		if absF(pl.X-x) < gap {
			return false
		}
	}
	for _, c := range w.corals {
		if absF(c.X-x) < gap {
			return false
		}
	}
	return true
}

// SetCorals (re)places coral designs along the floor with spacing rules.
// The defs are remembered so Restore can re-place them after a load.
func (w *World) SetCorals(defs []*contract.CoralDesign) {
	w.coralDefs = defs
	w.corals = w.corals[:0]
	w.CoralIDs = w.CoralIDs[:0]
	cap := w.coralCap()
	for _, d := range defs {
		if d == nil || len(w.corals) >= cap {
			continue
		}
		// random probe first, then a dense scan — dense plant fields leave
		// narrow windows, so corals accept a slightly smaller gap (N7 keeps
		// its balance without starving the reef)
		placed := false
		for attempt := 0; attempt < 30 && !placed; attempt++ {
			x := w.W * (0.06 + w.rng.Float64()*0.88)
			if w.spacingOKGap(x, contract.CoralBasePx) {
				placed = true
				w.placeCoral(d, x)
			}
		}
		for x := w.W * 0.06; x < w.W*0.94 && !placed; x += 4 {
			if w.spacingOKGap(x, 44*clampF(w.W/contract.DensityRefW, 1, 2)) { // scaled pitch
				placed = true
				w.placeCoral(d, x)
			}
		}
	}
}

func (w *World) placeCoral(d *contract.CoralDesign, x float64) {
	w.corals = append(w.corals, &Coral{Def: d, X: x, Y: w.H - 4, Sc: 0.8 + w.rng.Float64()*0.4})
	w.CoralIDs = append(w.CoralIDs, d.ID)
}

// Corals returns placed corals.
func (w *World) Corals() []*Coral { return w.corals }

// Mites returns live water mites.
func (w *World) Mites() []*Mite { return w.mites }

// SetZones installs the sim zones (called by the game after building the
// rock layout). Owner "chosen" hard-excludes every living thing but the
// Chosen; owner "cave" is a shelter.
func (w *World) SetZones(zs []contract.Zone) { w.zones = zs }

// Zones returns the active sim zones.
func (w *World) Zones() []contract.Zone { return w.zones }

// RockSeed returns the deterministic decor seed (the game builds the render
// layout from it).
func (w *World) RockSeed() int64 { return w.rockSeed }

// enforceZones keeps non-chosen living things out of the Chosen's aura (N3).
// The projection is positional, not force-based — boids jitter cannot creep in.
func (w *World) enforceZones(pos *contract.Vec2, vel *contract.Vec2, isChosen bool) {
	for _, z := range w.zones {
		if z.Owner != "chosen" || isChosen {
			continue
		}
		d := sub(*pos, z.Center)
		l := hyp2(d)
		if l < z.Radius+4 {
			outward := norm2(d)
			*pos = add(z.Center, mulS(outward, z.Radius+4))
			if vel != nil {
				vn := vel.X*outward.X + vel.Y*outward.Y
				if vn < 0 { // kill inward velocity
					vel.X -= vn * outward.X
					vel.Y -= vn * outward.Y
				}
			}
		}
	}
}

// enforceZonesPos is the velocityless variant for creatures and mites.
func (w *World) enforceZonesPos(pos *contract.Vec2) { w.enforceZones(pos, nil, false) }

// tickDecor advances the mites.
func (w *World) tickDecor(dt float64) {
	w.tickMites(dt)
}

func (w *World) tickMites(dt float64) {
	// spawn: rare, natural, capped (N7)
	w.miteT -= dt
	if w.miteT <= 0 {
		w.miteT = contract.MiteSpawnMeanSec * (0.5 + w.rng.Float64())
		if len(w.mites) < w.miteCap() {
			var anchor contract.Vec2
			if len(w.plants) > 0 && w.rng.Float64() < 0.7 {
				pl := w.plants[w.rng.Intn(len(w.plants))]
				anchor = v2(pl.X+(w.rng.Float64()*30-15), pl.Y-20-w.rng.Float64()*40)
			} else {
				side := w.rng.Float64()
				anchor = v2(w.W*w.rng.Float64(), w.H*(0.15+0.7*side))
			}
			w.mites = append(w.mites, &Mite{Pos: anchor, Anchor: anchor, Phase: w.rng.Float64() * 6.283, TTL: 25})
			w.logf("nature", "a water mite appears — the school stirs")
		}
	}
	kept := w.mites[:0]
	for _, m := range w.mites {
		m.TTL -= dt
		m.Phase += dt * 6
		m.Pos.X = m.Anchor.X + cos(m.Phase*0.6)*6
		m.Pos.Y = m.Anchor.Y + sin(m.Phase)*3
		// frenzy: the first hungry fish to reach it devours it
		eaten := false
		for _, f := range w.fishes {
			if f.Dying || f.Satiety > 0.98 {
				continue
			}
			if hyp2(sub(m.Pos, f.Pos)) < f.bodyLen*0.2 {
				f.eat()
				w.addCare(contract.CareFeedScore)
				w.burst(m.Pos, "#d8ffe8", 8)
				w.logf("nature", f.Sp.Name+" snaps up the mite")
				eaten = true
				break
			}
		}
		if !eaten && m.TTL > 0 {
			w.enforceZonesPos(&m.Pos)
			kept = append(kept, m)
		}
	}
	w.mites = kept
}

// miteCap scales the wild-mite budget with the tank area (v0.3.1).
func (w *World) miteCap() int {
	return min(int(float64(contract.MiteCap*w.density)+0.5), contract.MiteCapMax)
}

// coralCap gives the reef ~20% of the floor line (v0.3.2 composition).
func (w *World) coralCap() int {
	return clampI(int(w.W*contract.FloorShareCorals/contract.CoralBasePx+0.5), 4, 24)
}
