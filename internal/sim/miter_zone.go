// v1.2 split (line ceiling): her circle's laws — zone installation, the
// body-aware aura projection, the rim drag and spawn placement. G92 lives
// here: the nest encounters read as deflections, never teleports.
package sim

import (
	"math"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// SetZones installs the sim zones (called by the game after building the
// rock layout). Owner "chosen" hard-excludes every living thing but the
// Chosen; owner "cave" is a shelter.
func (w *World) SetZones(zs []contract.Zone) {
	w.zones = zs
	// G92: zones may arrive after residents do (test worlds, future order
	// changes). Re-place everyone once — nobody lives inside her circle.
	for _, f := range w.fishes {
		w.placeOutsideZones(f)
	}
}

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
// belly or tail — crosses into her circle. G92: the correction is
// DEPTH-GRADED — a graze at the band's edge glides at ~2 px/frame, a deep
// strike-point evacuates at the G65 bounded pace (12 px/frame). The old
// flat 12 px/frame drain re-fired across the whole band every frame and
// the big bodies wore it as a crawl; zoneLookahead now keeps traffic out
// of the band in the first place, so what remains is gentle grazes and
// brisk bounces.
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
			// the body sags inside the band: drain outward, graded —
			// shallow sag whispers out, deep sag evacuates at the pace the
			// G65 contract names. Neither teleports.
			push := clampF((need-dmin)*0.3, 2, 12)
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
// that ended up inside her circle is pulled back onto the rim — EXACT and
// unbounded, because her line is the one absolute in the tank (G65,
// TestBigBodiesNeverEnterTheCircle). G90's glide laws govern OPEN WATER;
// here the guarantee wins. The G92 lookahead steers traffic around the
// circle, so in natural play this projection almost never fires — the old
// every-frame 12 px crawl across the 380 px band is gone with it.
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

// placeOutsideZones nudges a freshly placed resident clear of the Chosen's
// circle (G92): a fish born, restored or parked INSIDE the band used to be
// rim-snapped 20-70 px on its next frame — a placement problem, fixed at
// placement. One-time at birth; the circle is soft to the eye, the fix is
// invisible. The whole body shifts rigidly so the laid spine stays a rope.
func (w *World) placeOutsideZones(f *Fish) {
	if f.Sp.Role == contract.RoleChosen {
		return // her circle is her own
	}
	before := f.Pos
	w.enforceZonesPos(&f.Pos)
	if d := sub(f.Pos, before); d.X != 0 || d.Y != 0 {
		for j := range f.Spine {
			f.Spine[j].X += d.X
			f.Spine[j].Y += d.Y
		}
	}
}
