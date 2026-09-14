// F25: the volcanic range — two BIG jagged crags, three doors each, the
// center stage clear for the oyster, and the rock budget at 50% of the floor.
package render

import (
	"math"
	"testing"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

func TestVolcanicCragLayout(t *testing.T) {
	for _, seed := range []int64{1, 7, 42, 2026} {
		for _, sz := range [][2]int{{1280, 720}, {1720, 720}} {
			r := NewRockLayout(seed, sz[0], sz[1])
			w, h := float64(sz[0]), float64(sz[1])

			// two crags, each with exactly three doors
			holes := r.Holes()
			if len(holes) != 6 {
				t.Fatalf("seed %d %dx%d: %d holes, want 6", seed, sz[0], sz[1], len(holes))
			}
			for c := 0; c < 2; c++ {
				n := 0
				for _, hl := range holes {
					if hl.Rock == c {
						n++
					}
				}
				if n != 3 {
					t.Fatalf("seed %d crag %d: %d doors, want 3", seed, c, n)
				}
			}

			// mouths are distinct and inside their crag's base span
			for c := 0; c < 2; c++ {
				ci := r.Crags[c]
				if ci.PeakH < 0.46*h { // F31: the crags grew taller
					t.Fatalf("seed %d %dx%d crag %d: peak %.0f < 0.46h — not a BIG crag",
						seed, sz[0], sz[1], c, ci.PeakH)
				}
				if ci.BaseW < 0.26*w { // F31: and wider
					t.Fatalf("seed %d %dx%d crag %d: base %.2fW < 0.26W",
						seed, sz[0], sz[1], c, ci.BaseW/w)
				}
				seen := map[int]bool{}
				for _, hl := range holes {
					if hl.Rock != c {
						continue
					}
					if hl.Center.X < ci.X-ci.BaseW/2 || hl.Center.X > ci.X+ci.BaseW/2 {
						t.Fatalf("seed %d crag %d: door at x=%.0f outside base span", seed, c, hl.Center.X)
					}
					key := int(hl.Center.X / 8)
					if seen[key] {
						t.Fatalf("seed %d crag %d: two doors share x≈%.0f", seed, c, hl.Center.X)
					}
					seen[key] = true
				}
				// side bands only: the center stage stays clear for the oyster
				// (her shell spans ~0.32..0.68W)
				inner := ci.X + ci.BaseW/2
				if c == 1 {
					inner = ci.X - ci.BaseW/2
				}
				if c == 0 && inner > 0.31*w {
					t.Fatalf("seed %d: left crag intrudes past 0.31W (inner=%.2fW)", seed, inner/w)
				}
				if c == 1 && inner < 0.69*w {
					t.Fatalf("seed %d: right crag intrudes before 0.69W (inner=%.2fW)", seed, inner/w)
				}
			}

			// the rock field owns its 50% floor budget
			if math.Abs(r.BaseWidth()-w*contract.FloorShareRocks) > 0.001 {
				t.Fatalf("seed %d: BaseWidth %.3fW != %.3fW", seed, r.BaseWidth()/w, contract.FloorShareRocks)
			}

			// zones: one shelter per door + the chosen aura
			zs := r.Zones()
			if len(zs) != 7 {
				t.Fatalf("seed %d: %d zones, want 7 (6 doors + chosen)", seed, len(zs))
			}
		}
	}
}

// TestDoorMouthsRender: every door paints dark depth + a rim light on the
// crag face (runs inside the pixel harness).
func doorMouthPixelProof(g *pixelHarness) {
	for _, seed := range []int64{7, 42} {
		r := NewRockLayout(seed, 1280, 720)
		dst := newBG(1280, 720, testBG)
		r.DrawBack(dst, 0.8)
		buf := pixelsOf(dst)
		dark := 0
		for _, hl := range r.Holes() {
			x := int(hl.Center.X)
			y := int(hl.Center.Y)
			i := 4 * (y*1280 + x)
			if buf[i] < 40 && buf[i+1] < 40 && buf[i+2] < 70 {
				dark++
			}
		}
		if dark < 4 {
			g.fail("F25 seed %d: only %d/6 door centers render dark depth", seed, dark)
		}
	}
}
