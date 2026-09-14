// v1.1 G78: bubble anatomy tests — the hollow read. A bubble is air: the
// interior window, rim and glint must keep their contrast ordering down to
// the radius floors, and the mesh must be built as window+ring+glint (not a
// filled orb — that collapsed into a solid dot at screen scale).
package render

import "testing"

func TestBubbleAnatomyLaw(t *testing.T) {
	rim, win, glint := bubbleParts(2.5)
	if win <= 0 || rim <= 0 || glint <= 0 {
		t.Fatalf("degenerate anatomy: rim %.2f win %.2f glint %.2f", rim, win, glint)
	}
	if outer := win + rim; outer <= win {
		t.Fatalf("rim adds nothing: outer %.2f vs window %.2f", outer, win)
	}
	// the floors: a tiny bubble still holds rim+window structure (≥1.1 outer)
	rim2, win2, glint2 := bubbleParts(0.5)
	if outer := win2 + rim2; outer < 1.1 {
		t.Fatalf("tiny bubble collapses to nothing: outer %.2f", outer)
	}
	if glint2 > win2+rim2 {
		t.Fatalf("glint wider than the bubble: %.2f vs %.2f", glint2, win2+rim2)
	}
}

func TestBubbleMeshIsAHollowRing(t *testing.T) {
	m := buildBubbleMesh([]BubbleView{{X: 100, Y: 100, R: 2.5, Seed: 0.3}})
	// window fan (7 tris) + 8 ring quads (16) + glint fan (5) = 28 tris
	if want := 3 * (7 + 16 + 5); len(m.indices) != want {
		t.Fatalf("index count %d, want the hollow anatomy %d (a filled orb is not a bubble)", len(m.indices), want)
	}
	// batching: two bubbles share the one mesh, linearly
	m = buildBubbleMesh([]BubbleView{{X: 10, Y: 10, R: 2, Seed: 0}, {X: 30, Y: 30, R: 3, Seed: 1}})
	if len(m.indices) != 2*3*(7+16+5) {
		t.Fatalf("batch broke: %d indices for two bubbles", len(m.indices))
	}
}
