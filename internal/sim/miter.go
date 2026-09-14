// v1.1 split (line ceiling): the water mites — spawn cadence, her
// circle's exclusion, the wiggle and the frenzy feeding.
package sim

import (
	"math"

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

// enforceZonesPos is the velocityless variant for creatures and mites —
// with a slightly wider margin, since a critter's whole shell must clear.
func (w *World) enforceZonesPos(pos *contract.Vec2) {
	for _, z := range w.zones {
		if z.Owner != "chosen" {
			continue
		}
		d := sub(*pos, z.Center)
		if l := hyp2(d); l < z.Radius+12 {
			*pos = add(z.Center, mulS(norm2(d), z.Radius+12))
		}
	}
}

// zoneFix caps how far any correction may displace a fish in one frame
// (G65): the pin laws (G62/G63) discard the lateral part of avoidance, so
// fish now reach her rim still driving forward — an uncapped snap read as
// a teleport. Bounded, it reads as a quick slide back out.
const zoneFix = 12.0

// enforceFishZones is the body-aware projection (v1.1): the keep-clear
// margin grows with the fish's own body, so no part of any fish — head,
// belly or tail — crosses into her circle. A head found inside glides back
// out at a bounded pace (a snap would read as a teleport, G65); a sagging
// body drains out at the same capped pace.
func (w *World) enforceFishZones(f *Fish, dt float64) {
	for _, z := range w.zones {
		if z.Owner != "chosen" || f.Sp.Role == contract.RoleChosen {
			continue
		}
		need := z.Radius + 16 + f.bodyLen*0.5
		dmin := hyp2(sub(f.Pos, z.Center))
		for i := 1; i < len(f.Spine); i++ {
			if d := hyp2(sub(f.Spine[i], z.Center)); d < dmin {
				dmin = d
			}
		}
		if dmin >= need {
			continue
		}
		out := norm2(sub(f.Pos, z.Center))
		hd := hyp2(sub(f.Pos, z.Center))
		if hd < z.Radius {
			// the head crossed her line: land it ON the rim — a solid-wall
			// projection bounded by the frame's own travel. The old snap all
			// the way out to `need` (up to ~70 px for a shark) was the
			// teleport the live review caught (G65).
			f.Pos = add(z.Center, mulS(out, z.Radius+2))
		} else {
			// the body sags inside: drain outward, capped per frame (a deep
			// sag empties briskly, a graze whispers out — neither teleports)
			push := need - dmin
			if push > 12 {
				push = 12
			}
			f.Pos = add(f.Pos, mulS(out, push))
		}
		if vn := f.Vel.X*out.X + f.Vel.Y*out.Y; vn < 0 {
			f.Vel.X -= vn * out.X
			f.Vel.Y -= vn * out.Y
		}
		// G65: the pin laws ride headingA — if it still aims into her
		// circle, the next frame's rebuild would drive the fish straight
		// back in. Slide the heading toward the outward hemisphere — G84:
		// at a BOUNDED rate (1.2 rad/s). The old instant projection was a
		// heading teleport the big bodies wore as a visible twitch at her
		// rim; the position drain still holds the line while the nose
		// swings around like a nose, not a door.
		hx, hy := cos(f.headingA), sin(f.headingA)
		if vn := hx*out.X + hy*out.Y; vn < 0 {
			hx -= vn * out.X
			hy -= vn * out.Y
			if l := hyp2(v2(hx, hy)); l > 1e-3 {
				tgt := math.Atan2(hy/l, hx/l)
				da := math.Mod(tgt-f.headingA+3.14159, 6.28318) - 3.14159
				f.headingA += clampF(da, -1.2*dt, 1.2*dt)
			}
		}
	}
}

// dragSpineOut runs after followSpine re-lays the chain: any spine point
// that ended up inside her circle is pulled back over the line. followSpine
// re-normalizes segment lengths next frame, so this reads as one smooth
// pull — the tail can never lie across the nest boundary.
func (w *World) dragSpineOut(f *Fish) {
	for _, z := range w.zones {
		if z.Owner != "chosen" || f.Sp.Role == contract.RoleChosen {
			continue
		}
		for i := 1; i < len(f.Spine); i++ {
			p := f.Spine[i]
			d := hyp2(sub(p, z.Center))
			if d < z.Radius+8 && d > 0.5 {
				f.Spine[i] = add(z.Center, mulS(norm2(sub(p, z.Center)), z.Radius+8))
			}
		}
	}
}

// tickDecor advances the mites and the floor sand bed.
func (w *World) tickDecor(dt float64) {
	w.tickMites(dt)
	w.tickSand(dt)
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
			// owner's rule: no mites in the lilac one's circle — her home
			// stays undisturbed (draws that land inside her aura are pushed
			// out to the rim)
			for _, z := range w.zones {
				if z.Owner != "chosen" {
					continue
				}
				d := sub(anchor, z.Center)
				keep := z.Radius + 70
				if l := hyp2(d); l < keep {
					if l < 1 {
						d = v2(1, 0)
						l = 1
					}
					anchor = add(z.Center, mulS(d, keep/l))
				}
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
