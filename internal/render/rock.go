// N2/N3/F25: volcanic rock decor — deterministic layout from Save.RockSeed.
// v0.3.6: two BIG jagged basalt crags flank the tank, each pierced by THREE
// door mouths (low / mid / high) — fish swim in one and out another (the sim
// transit state carries them through; Hide01 hides them, binary since v0.3.8).
// Two-pass render: back silhouette behind the fish, front faces over the low
// mouths. All meshes are baked once in NewRockLayout; per-frame work is
// replay only (C3). One mesh never contains overlapping triangles (v2.10
// fill rule) — every crag part lives in its own mesh.
package render

import (
	"math"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/hajimehoshi/ebiten/v2"
)

// basalt palette: dark violet rock, deep-indigo cave depth, ember cracks.
// F18: the near-black cave interiors (#03020a→#0b0618) read as flat black
// rectangles — regraded to deep indigo with a rim-light arc so every mouth
// reads as DEPTH; mounds lifted slightly off pure black.
var (
	cRockLo  = hexRGBA("#1b1335") // crag base
	cRockHi  = hexRGBA("#2b2050") // crag crown
	cColumn  = hexRGBA("#120b26") // basalt column striations
	cFrontLo = hexRGBA("#0e0920") // front face (darker than back)
	cFrontHi = hexRGBA("#1a1230")
	cDepthHi = hexRGBA("#161238") // cave interior, floor side (deep indigo)
	cDepthLo = hexRGBA("#0a0a24") // cave interior, crown side
	cEmber   = "#ff7a45"          // lava cracks
)

// cragMouth is one enterable door: a dark arch with rim light, jaw strips
// and (floor mouths) a covering lip so fish inside read as swallowed.
type cragMouth struct {
	hole     contract.Hole
	collar   mesh // light face ring around floating doors (back pass)
	interior mesh // deep-indigo arch cavity (back pass)
	rimArc   mesh // F18: soft rim-light arc over the mouth (back, additive)
	jaws     mesh // arch outline lower halves (front pass, floor doors)
	sill     mesh // bottom chord of floating mouths / lip of floor mouths
}

// CragInfo is the public geometry of one volcanic crag (tests + layout).
type CragInfo struct {
	X     float64 // center x
	BaseW float64 // base width along the floor
	PeakH float64 // skyline peak height above the floor
}

type skyPt struct{ x, y float64 }

// crag is one baked formation.
type crag struct {
	info    CragInfo
	sky     mesh // jagged skyline silhouette fan (back pass)
	columns mesh // basalt column striations (back pass)
	cracks  mesh // ember polylines (back pass, additive)
	mouths  []cragMouth
	env     []skyPt // skyline control points (x-monotone) for placement
}

// RockLayout describes the volcanic decor derived from a seed.
type RockLayout struct {
	Seed int64
	W, H int
	Nest contract.Vec2 // oyster nest mouth plane (Chosen's home, F24)

	Crags [2]CragInfo // public crag geometry (left, right)

	floorY float64
	pearls int
	frame  int // DrawBack frame counter (crack pulse clock)
	cg     [2]crag
	holes  []contract.Hole
}

// NewRockLayout derives the layout deterministically from the seed. The two
// crags sit in the side bands (≈0.0..0.31W and 0.69..1.0W) so the center
// stage stays clear for the Chosen's oyster; each owns three mouths and the
// rock field covers 50% of the floor line (v0.3.6 composition).
// v0.3.7 (F31): the crags grew outward and taller —
// bigger rocky homes for the doors to bite into.
func NewRockLayout(seed int64, w, h int) *RockLayout {
	rng := contract.RandSeed(seed)
	l := &RockLayout{Seed: seed, W: w, H: h, pearls: 3}
	floor := float64(h) * 0.925
	l.floorY = floor
	k := clampF(float64(h)/720, 1, 2.2)
	fw := float64(w)

	for c := 0; c < 2; c++ {
		side := float64(c)*2 - 1 // -1 left crag, +1 right crag
		cx := fw * (0.145 + 0.015*rng.Float64() + float64(c)*0.70)
		bw := fw * (0.26 + 0.03*rng.Float64()) // ≈ 27% of the floor each
		peak := float64(h) * (0.46 + 0.12*rng.Float64())
		l.Crags[c] = CragInfo{X: cx, BaseW: bw, PeakH: peak}
		cg := &l.cg[c]
		cg.info = l.Crags[c]

		// jagged skyline: x-monotone control points around a skewed bell
		// whose main spire leans toward the tank center; jittered heights
		// make it a volcanic ridge, not a dome. Interior points never dip
		// below ~34% of the peak so the crag stays ONE massif, not spires.
		const pts = 11
		hw := bw * 0.5
		lean := 0.10 * float64(-side)
		for i := 0; i <= pts; i++ {
			u := float64(i) / float64(pts)
			bell := sin(math.Pi * clampF(u+lean, 0.03, 0.97))
			jag := 0.74 + 0.42*rng.Float64()
			hu := peak * clampF(math.Pow(bell, 1.3)*jag, 0.05, 1.1)
			if i > 0 && i < pts {
				hu = math.Max(hu, peak*0.34) // keep the ridge connected
			}
			cg.env = append(cg.env, skyPt{x: cx - hw + bw*u, y: floor - hu})
		}
		ci := cg.sky.vert(v2(cx, floor), cRockLo)
		for _, p := range cg.env {
			cg.sky.vert(v2(p.x, p.y), lerpRGBA(cRockLo, cRockHi, clampF((floor-p.y)/maxF(peak, 1), 0, 1)))
		}
		for i := 1; i <= pts; i++ {
			cg.sky.tri(ci, uint16(i), uint16(i+1))
		}

		// basalt columns — vertical striations climbing the ridge
		for s := 0; s < 7; s++ {
			u := 0.14 + 0.72*float64(s)/6
			x := cx - hw + bw*u
			top := envAt(cg, x)
			bot := floor - (floor-top.y)*0.10
			strokeQuads(&cg.columns, []contract.Vec2{
				{X: x, Y: bot}, {X: x + (rng.Float64()-0.5)*7, Y: top.y + (floor-top.y)*0.12},
			}, 2.0, withA(cColumn, 120))
		}

		// ember cracks — more of them on the big ridge
		bakeCracks(&cg.cracks, cx, floor, hw, peak, rng, 4)

		// three doors per crag: low (floor level), mid face, high shoulder.
		// Offsets mirror per side so the through-paths differ left vs right.
		// Floating doors sit DEEP in the face (well under the local skyline)
		// so they read as mouths bitten into the rock, not stickers.
		offs := [3][2]float64{
			{side * 0.24, 0.0},   // low — at the floor, lip-covered
			{-side * 0.18, 0.34}, // mid — on the crag face
			{side * 0.04, 0.55},  // high — below the shoulder
		}
		for m := 0; m < 3; m++ {
			mx := cx + offs[m][0]*bw
			local := envAt(cg, mx)
			localH := floor - local.y // skyline height at this x
			mh := (50 + rng.Float64()*18) * k
			mw := maxF(fw*0.040, (76+rng.Float64()*24)*k) // mouth width, enterable
			// keep the arch fully inside the silhouette
			mh = math.Min(mh, localH*(1.0-offs[m][1])*0.8)
			if mh < 26*k {
				mh = 26 * k
			}
			var my float64 // bottom chord of the arch
			if m == 0 {
				my = floor
			} else {
				my = floor - localH*offs[m][1] - mh*0.5
			}
			cm := cragMouth{}
			cm.hole = contract.Hole{Center: v2(mx, my-mh*0.5), Radius: mw * 0.5, Rock: c}
			if m > 0 {
				// light collar so the floating door reads as a hole bitten
				// into the dark face (contrast ring behind the arch)
				halfDome(&cm.collar, mx, my+1, mw*0.62, mh*1.14, cRockHi, cColumn, 12)
			}
			halfDome(&cm.interior, mx, my+1, mw*0.5, mh, cDepthLo, cDepthHi, 16)
			bakeRimArc(&cm.rimArc, mx, my+1, mw*0.5, mh)
			if m == 0 {
				// floor mouths get the swallowing jaws + lip (front pass)
				bakeJaws(&cm.jaws, mx, my, mw*0.5, mh)
				bakeLip(&cm.sill, mx, my, mw*0.5, mh*0.38)
			} else {
				bakeSill(&cm.sill, mx, my, mw*0.5)
			}
			cg.mouths = append(cg.mouths, cm)
			l.holes = append(l.holes, cm.hole)
		}
	}

	// v0.3.8: the Chosen's oyster home is DEAD CENTER on the shared floor
	// line — the anchor is the mouth plane, one cup-height (oh*0.30) above
	// the base (the old window-bottom anchor made it a flat overlay).
	_, noh := l.oysterSize()
	l.Nest = v2(float64(w)*0.5, l.NestBaseY()-noh*0.30)
	return l
}

// Zones exports the sim zones: every door mouth is a shelter, the nest is the
// Chosen's exclusive aura (contract.ZoneRadius).
func (r *RockLayout) Zones() []contract.Zone {
	zs := []contract.Zone{}
	for _, h := range r.holes {
		zs = append(zs, contract.Zone{Center: h.Center, Radius: maxF(70, h.Radius*1.8), Owner: "cave"})
	}
	zs = append(zs, contract.Zone{Center: r.Nest, Radius: contract.ZoneRadius, Owner: "chosen"})
	return zs
}

// DrawBack paints the behind-fish pass: basalt crags, deep-indigo door
// interiors with rim-lit mouths and slowly pulsing lava cracks
// (alpha ≤ 35*(0.7+0.6*night)).
func (r *RockLayout) DrawBack(dst *ebiten.Image, night float64) {
	if dst == nil {
		return
	}
	r.frame++
	for c := range r.cg {
		r.cg[c].sky.draw(dst, false)
		r.cg[c].columns.draw(dst, false)
		for m := range r.cg[c].mouths {
			r.cg[c].mouths[m].collar.draw(dst, false)
			r.cg[c].mouths[m].interior.draw(dst, false)
			r.cg[c].mouths[m].rimArc.draw(dst, true) // F18: rim light, additive + soft
		}
		k := (0.7 + 0.6*night) * (0.72 + 0.28*sin(float64(r.frame)*0.025+float64(c)*2.4))
		r.cg[c].cracks.drawA(dst, k)
	}
}

// DrawFront paints the after-fish pass: jaw strips, mouth-floor lips and
// floating sills, so a fish entering a door is visibly swallowed.
func (r *RockLayout) DrawFront(dst *ebiten.Image, night float64) {
	if dst == nil {
		return
	}
	for c := range r.cg {
		for m := range r.cg[c].mouths {
			r.cg[c].mouths[m].jaws.draw(dst, false)
			r.cg[c].mouths[m].sill.draw(dst, false)
		}
	}
}
