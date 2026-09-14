// v0.3.5/v0.3.6: the Chosen's oyster acceptance — dead center, exactly three
// pearls, seated on the shared floor line (v0.3.8), mouth plane above the cup.
package render

import "testing"

func TestNestDeadCenterWithThreePearls(t *testing.T) {
	for _, seed := range []int64{1, 7, 2026} {
		r := NewRockLayout(seed, 1280, 720)
		if r.Nest.X < 639 || r.Nest.X > 641 {
			t.Fatalf("seed %d: nest X = %.1f, want 640 (dead center)", seed, r.Nest.X)
		}
		if r.pearls != 3 {
			t.Fatalf("seed %d: pearls = %d, want exactly 3", seed, r.pearls)
		}
	}
}

// v0.3.8: the shell sits ON the floor line the crags stand on (0.925H) —
// one ground plane across the stage — mouth exactly one cup-height above.
func TestNestFloorLineAndMouthPlane(t *testing.T) {
	for _, sz := range [][2]int{{1280, 720}, {1720, 720}, {3440, 1440}} {
		r := NewRockLayout(1, sz[0], sz[1])
		wantBase := float64(sz[1]) * 0.925
		if base := r.NestBaseY(); base != wantBase {
			t.Fatalf("%dx%d: base Y = %.1f, want the floor line %.1f", sz[0], sz[1], base, wantBase)
		}
		_, oh := r.oysterSize()
		want := wantBase - oh*0.30
		if d := r.Nest.Y - want; d < -1 || d > 1 {
			t.Fatalf("%dx%d: mouth Y = %.1f, want %.1f (one cup-height up)", sz[0], sz[1], r.Nest.Y, want)
		}
	}
}

// v0.3.8: the shell is SCENERY scale — at most 30% of the canvas width and
// never wider than the old 470 px palace (the view stays open above it).
func TestNestSceneryScale(t *testing.T) {
	for _, sz := range [][2]int{{1280, 720}, {1720, 720}, {3440, 1440}} {
		r := NewRockLayout(1, sz[0], sz[1])
		ow, _ := r.oysterSize()
		if ow > float64(sz[0])*0.30+1e-9 {
			t.Fatalf("%dx%d: oyster %.0f px wide exceeds 30%% of the canvas", sz[0], sz[1], ow)
		}
		if ow > 360+1e-9 {
			t.Fatalf("%dx%d: oyster %.0f px wide exceeds the 360 px scenery cap", sz[0], sz[1], ow)
		}
	}
}

// nestFloorPixelProof (F24, re-seated v0.3.8) runs inside the pixel harness
// (ReadPixels needs a live game loop on v2.10): the rendered shell actually
// touches the floor-line rows of a 1280×720 canvas — no floating gap — and
// its wings still flank the nest.
func nestFloorPixelProof(g *pixelHarness) {
	r := NewRockLayout(7, 1280, 720)
	dst := newBG(1280, 720, testBG)
	r.DrawNest(dst, 4.2, 0.8)
	buf := make([]byte, 4*1280*720)
	dst.ReadPixels(buf) // v2.10: whole-image read only
	isBG := func(x, y int) bool {
		i := 4 * (y*1280 + x)
		return buf[i] == testBG.R && buf[i+1] == testBG.G && buf[i+2] == testBG.B
	}
	baseY := r.NestBaseY() // 666 on 720
	touched := 0
	for x := 480; x < 800; x++ {
		for _, y := range []int{int(baseY) - 2, int(baseY) - 1, int(baseY)} {
			if !isBG(x, y) {
				touched++
				break
			}
		}
	}
	if touched < 40 {
		g.fail("v0.3.8 oyster: floor-line rows touched at only %d of 320 columns — the shell floats", touched)
	}
	// the cup must still spread wide: the lower left wing band is shell
	any := false
	for y := int(baseY) - 40; y < int(baseY)-20 && !any; y++ {
		for x := int(r.Nest.X) - 170; x < int(r.Nest.X)-120; x++ {
			if !isBG(x, y) {
				any = true
				break
			}
		}
	}
	if !any {
		g.fail("v0.3.8 oyster: left wing empty at the lower band — cup too narrow")
	}
}

// F27: the gape is a real cavity (dense segments — no flat wedge), the lid's
// lower lip interlocks with the mouth plane (arches over it at the center,
// meets the cup's shoulders at the ends), and the byssus silk bundle roots
// at the hinge and trails onto the floor.
func TestNestOneShellGeometry(t *testing.T) {
	r := NewRockLayout(7, 1280, 720)
	if gapeSegs < 28 {
		t.Fatalf("gape has %d segments — it will read as a wedge", gapeSegs)
	}
	_, oh := r.oysterSize()
	cx, baseY := r.Nest.X, r.NestBaseY()
	ow, _ := r.oysterSize()
	hw := ow * 0.5
	mouthY := r.Nest.Y
	lidH := oh * 0.52 * 0.55
	skew := hw * 0.05
	const n = 26
	// arch shape of the lower lip: over the mouth center, onto the shoulders
	midY := lidBotPt(cx, mouthY, hw, oh, skew, n/2, n).Y
	if midY >= mouthY {
		t.Fatalf("lid lip center y %.1f must arch above the mouth plane %.1f", midY, mouthY)
	}
	for _, i := range []int{0, n} {
		endY := lidBotPt(cx, mouthY, hw, oh, skew, i, n).Y
		if endY < mouthY {
			t.Fatalf("lid lip end y %.1f must reach the cup shoulder (>= mouth plane %.1f)", endY, mouthY)
		}
	}
	// strip is never inverted: the top arc stays above the bottom arc
	for i := 0; i <= n; i++ {
		top := lidTopPt(cx, mouthY, hw, lidH, skew, i, n).Y
		bot := lidBotPt(cx, mouthY, hw, oh, skew, i, n).Y
		if top > bot {
			t.Fatalf("lid strip inverted at %d: top %.1f below bottom %.1f", i, top, bot)
		}
	}
	// silk: >=6 strands, rooted at the hinge, trailing onto the floor
	strands := byssusStrands(cx, baseY, hw, oh)
	if len(strands) < 6 {
		t.Fatalf("byssus bundle too thin: %d strands", len(strands))
	}
	root := baseY - oh*0.075
	for si, pts := range strands {
		if len(pts) < 4 {
			t.Fatalf("strand %d: only %d points", si, len(pts))
		}
		if d := pts[0].Y - root; d < -oh*0.015 || d > oh*0.015 {
			t.Fatalf("strand %d not rooted at the hinge (y %.1f, want ~%.1f)", si, pts[0].Y, root)
		}
		if d := pts[len(pts)-1].Y - baseY; d < -3 {
			t.Fatalf("strand %d does not reach the floor (ends at y %.1f)", si, pts[len(pts)-1].Y)
		}
	}
}
