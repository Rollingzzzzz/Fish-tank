// v1.1 split (line ceiling): the water mites — spawn cadence, her
// circle's exclusion, the wiggle and the frenzy feeding.
package sim

import (
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

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

// tickDecor advances the mites and the floor sand bed.
func (w *World) tickDecor(dt float64) {
	w.tickMites(dt)
	w.tickSand(dt)
}

// miteKeepOut projects a mite position out of the lilac one's circle with a
// +70 px buffer — the owner's rule (her home stays undisturbed) holds at the
// release AND through the whole descent.
func (w *World) miteKeepOut(p contract.Vec2) contract.Vec2 {
	for _, z := range w.zones {
		if z.Owner != "chosen" {
			continue
		}
		d := sub(p, z.Center)
		keep := z.Radius + 70
		if l := hyp2(d); l < keep {
			if l < 1 {
				d = v2(1, 0)
				l = 1
			}
			p = add(z.Center, mulS(d, keep/l))
		}
	}
	return p
}

func (w *World) tickMites(dt float64) {
	// spawn: rare, natural, capped (N7)
	w.miteT -= dt
	if w.miteT <= 0 {
		w.miteT = contract.MiteSpawnMeanSec * (0.5 + w.rng.Float64())
		if len(w.mites) < w.miteCap() {
			// G89: released at the surface, the same lane the auto-feeder
			// uses — never conjured mid-water beside a plant
			anchor := w.miteKeepOut(v2(w.W*(0.08+w.rng.Float64()*0.84), contract.MiteDropY))
			w.mites = append(w.mites, &Mite{Pos: anchor, Anchor: anchor, Phase: w.rng.Float64() * 6.283, TTL: 25})
			w.logf("nature", "a water mite drifts down from the surface — the tank stirs")
		}
	}
	kept := w.mites[:0]
	for _, m := range w.mites {
		m.TTL -= dt
		m.Phase += dt * 6
		// G89: the release settles — a slow sink from the surface that runs
		// out of push at the lingering depth (mites never carpet the floor)
		if m.Anchor.Y < w.H*contract.MiteSinkMaxFrac {
			m.Anchor.Y += contract.MiteSinkSpeed * dt
			// the drift may carry a mite over her circle — the keep-out that
			// guards the release guards the whole descent
			m.Anchor = w.miteKeepOut(m.Anchor)
		}
		m.Pos.X = m.Anchor.X + cos(m.Phase*0.6)*6
		m.Pos.Y = m.Anchor.Y + sin(m.Phase)*3
		// frenzy: the first hungry fish to reach it devours it
		eaten := false
		for _, f := range w.fishes {
			if f.Dying || f.Satiety > 0.98 {
				continue
			}
			if hyp2(sub(m.Pos, f.Pos)) < f.bodyLen*0.2 {
				f.snack(0.34) // a third of a meal — the teem adds motion, not calories
				w.addCare(contract.CareFeedScore / 3)
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
