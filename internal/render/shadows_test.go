// v1.1 G74: floor-shadow law tests — the occlusion band, the depth-lane and
// climb response, the night fraction, the footprint shape and the dune-seat
// of the projected mesh.
package render

import (
	"math"
	"testing"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

func TestShadowAlphaLaw(t *testing.T) {
	far := ShadowAlpha(0, 200, 720, 0)
	near := ShadowAlpha(1, 200, 720, 0)
	if near <= far {
		t.Fatalf("near lane %.3f must out-shadow far lane %.3f", near, far)
	}
	// the frozen band is the grounded envelope: lift only ever thins it
	if g := ShadowAlpha(0, 0, 720, 0); math.Abs(g-contract.ShadowAlphaFar) > 1e-9 {
		t.Fatalf("grounded far alpha %.3f, want %.3f", g, contract.ShadowAlphaFar)
	}
	if g := ShadowAlpha(1, 0, 720, 0); math.Abs(g-contract.ShadowAlphaNear) > 1e-9 {
		t.Fatalf("grounded near alpha %.3f, want %.3f", g, contract.ShadowAlphaNear)
	}
	low := ShadowAlpha(0.5, 30, 720, 0)
	high := ShadowAlpha(0.5, 650, 720, 0)
	if high >= low {
		t.Fatalf("a fish near the ceiling casts the thicker patch: %.3f vs %.3f", high, low)
	}
	day := ShadowAlpha(0.6, 100, 720, 0)
	night := ShadowAlpha(0.6, 100, 720, 1)
	if math.Abs(night-day*contract.ShadowNightMul) > 1e-9 {
		t.Fatalf("night keeps %.3f of the day bite, want exactly ×%.2f", night/day, contract.ShadowNightMul)
	}
}

func TestShadowRadiiLaw(t *testing.T) {
	rxH, ryH := ShadowRadii(300, 1, 0.5)
	rxV, ryV := ShadowRadii(300, 0, 0.5)
	if rxH <= rxV {
		t.Fatalf("a flat swimmer must stretch longer: %.1f vs %.1f", rxH, rxV)
	}
	if ryH >= ryV {
		t.Fatalf("a flat swimmer must sit thinner: %.1f vs %.1f", ryH, ryV)
	}
	if rxH > 0.45*300+1e-9 {
		t.Fatalf("footprint too wide for the body: %.1f", rxH)
	}
	rxFar, _ := ShadowRadii(300, 1, 0)
	if rxFar >= rxH {
		t.Fatalf("the near lane must buy width: far %.1f vs near %.1f", rxFar, rxH)
	}
}

func TestShadowsSitOnTheDunes(t *testing.T) {
	m := buildShadowMesh([]ShadowCaster{{X: 400, Hx: 1, BodyLen: 200, Z: 0.6, Lift: 50}}, 720, 0)
	if len(m.indices) == 0 {
		t.Fatal("no shadow triangles")
	}
	sy := contract.SandSurfaceY(720, 400) + 3
	lo, hi := 1e18, -1e18
	for _, v := range m.verts {
		lo = math.Min(lo, float64(v.DstY))
		hi = math.Max(hi, float64(v.DstY))
	}
	if mid := (lo + hi) / 2; math.Abs(mid-sy) > 1 {
		t.Fatalf("shadow center %.1f off the dune surface %.1f", mid, sy)
	}
	if lo < sy-14 || hi > sy+16 {
		t.Fatalf("shadow bleeds off the bed band: [%.1f, %.1f] around %.1f", lo, hi, sy)
	}
	// a fully faded caster (the Chosen mid-portal) projects nothing
	m = buildShadowMesh([]ShadowCaster{{X: 400, Hx: 1, BodyLen: 200, Z: 0.6, Fade: 1}}, 720, 0)
	if len(m.indices) != 0 {
		t.Fatal("a faded-through caster still casts a shadow")
	}
}
