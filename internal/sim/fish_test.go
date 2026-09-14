// G2.1/G2.2: steering, spine and feeding acceptance tests.
package sim

import (
	"math"
	"testing"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

func testSpecies(schooling float64) *contract.Species {
	return &contract.Species{
		ID: "test-neon", Name: "Test Neon", Size: 1, Width: 1, Fin: 1, Tail: 1,
		Palette:  contract.Palette{Body: "#00ffee", Belly: "#083344", Accent: "#ff3df5", Glow: "#a0f0ff"},
		Pattern:  contract.Pattern{Type: "spot", Density: 0.5, Size: 0.5},
		Behavior: contract.Behavior{Speed: 1, Schooling: schooling, Curiosity: 0, Skittish: 0, Depth: 0.5},
	}
}

func testWorld(t *testing.T, sp *contract.Species, n int) *World {
	t.Helper()
	cfg := contract.Config{MaxFish: 30, DaySeconds: 60}
	w := NewWorld(800, 600, cfg, []*contract.Species{sp}, nil, nil)
	w.SeedRng(1234)
	w.fishes = w.fishes[:0]
	for i := 0; i < n; i++ {
		p := v2(300+w.rng.Float64()*200, 250+w.rng.Float64()*100)
		w.fishes = append(w.fishes, newFish(sp, w.rng.Int63(), p, 6, w.nextID()))
	}
	return w
}

func avgNearestNeighbor(w *World) float64 {
	total, cnt := 0.0, 0
	for i, a := range w.fishes {
		best := math.MaxFloat64
		for j, b := range w.fishes {
			if i == j {
				continue
			}
			if d := hyp2(sub(a.Pos, b.Pos)); d < best {
				best = d
			}
		}
		if best < math.MaxFloat64 {
			total += best
			cnt++
		}
	}
	if cnt == 0 {
		return 0
	}
	return total / float64(cnt)
}

func TestSchoolCohesion(t *testing.T) {
	w := testWorld(t, testSpecies(1.0), 8)
	// scatter the school across the tank first — cohesion must pull it in
	for i, f := range w.fishes {
		f.Pos = v2(80+float64(i%4)*180, 80+float64(i/4)*260)
		f.Vel = v2(0, 0)
	}
	in := Input{}
	before := avgNearestNeighbor(w)
	for i := 0; i < 600; i++ {
		w.Update(1/60.0, in)
	}
	after := avgNearestNeighbor(w)
	// separation keeps spacing near ~26px; cohesion must beat the scatter
	if after > before*0.6 || after > 90 {
		t.Fatalf("school did not cohere: before=%.1f after=%.1f", before, after)
	}
}

func TestSpineSegmentLengthExact(t *testing.T) {
	w := testWorld(t, testSpecies(0.5), 3)
	for i := 0; i < 300; i++ {
		w.Update(1/60.0, Input{})
	}
	for _, f := range w.fishes {
		for k := 1; k < len(f.Spine); k++ {
			d := hyp2(sub(f.Spine[k], f.Spine[k-1]))
			if math.Abs(d-f.segLen) > 1e-6 {
				t.Fatalf("segment length drifted: got %.9f want %.9f", d, f.segLen)
			}
		}
	}
}

func TestMaxSpeedNeverExceeded(t *testing.T) {
	w := testWorld(t, testSpecies(0.8), 5)
	for i := 0; i < 400; i++ {
		w.Update(1/60.0, Input{})
		for _, f := range w.fishes {
			bound := f.maxSpeed(1-w.dayFactor()) * maxF(1, f.seekBonus)
			if s := hyp2(f.Vel); s > bound+1e-6 {
				t.Fatalf("speed %.2f exceeds max %.2f", s, bound)
			}
		}
	}
}

func TestEatConsumesFlake(t *testing.T) {
	w := testWorld(t, testSpecies(0), 1)
	f := w.fishes[0]
	f.Satiety = 0.2
	f.Pos = v2(400, 300)
	w.foods = append(w.foods, Food{Pos: v2(401, 301), Age: 1})
	n := len(w.foods)
	w.tryEat()
	if len(w.foods) != n-1 {
		t.Fatal("flake not consumed")
	}
	if f.Satiety != 1 {
		t.Fatalf("satiety not reset: %v", f.Satiety)
	}
}
