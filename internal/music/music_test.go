// N6/FD10 + N12 acceptance: exact render size, determinism, loudness/peak,
// loop-seam smoothness, non-silence, deterministic authored event count,
// 1/12-beat tempo grid (16ths + triplet eighths), and hammer-onset sharpness
// (RMS after >= 4x before — the proof the piano has a felt hammer, not a
// sine blip). Render is expensive (~1.4M samples), so tests share one cached
// render except the determinism check.
package music

import (
	"bytes"
	"encoding/binary"
	"math"
	"sort"
	"sync"
	"testing"
)

var (
	cacheMu   sync.Mutex
	cacheBuf  []byte
	cacheEvts []Note
)

// loop returns a shared Render result plus its authored onset list (the
// render is deterministic, so all tests may inspect the same bytes).
func loop(t *testing.T) ([]byte, []Note) {
	t.Helper()
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if cacheBuf == nil {
		cacheBuf, cacheEvts = RenderEvents()
	}
	return cacheBuf, cacheEvts
}

// decode splits interleaved stereo F32 LE bytes into L/R and mono-sum.
func decode(t *testing.T, data []byte) (l, r, mono []float64) {
	t.Helper()
	if len(data) != loopSamples*8 {
		t.Fatalf("render length = %d bytes, want %d", len(data), loopSamples*8)
	}
	l = make([]float64, loopSamples)
	r = make([]float64, loopSamples)
	mono = make([]float64, loopSamples)
	for i := 0; i < loopSamples; i++ {
		l[i] = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[i*8:])))
		r[i] = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[i*8+4:])))
		mono[i] = (l[i] + r[i]) * 0.5
	}
	return l, r, mono
}

func TestRenderLength(t *testing.T) {
	if got := len(Render()); got != 22050*64*8 {
		t.Fatalf("Render() = %d bytes, want %d (exactly 64.000 s)", got, 22050*64*8)
	}
}

func TestRenderDeterministic(t *testing.T) {
	a := Render()
	b, ev := RenderEvents()
	if !bytes.Equal(a, b) {
		t.Fatal("two renders are not byte-identical (D5 violated)")
	}
	if len(ev) == 0 {
		t.Fatal("render produced no onset events")
	}
}

func TestRenderLoudnessAndPeak(t *testing.T) {
	data, _ := loop(t)
	l, r, mono := decode(t, data)
	var sum float64
	for _, v := range mono {
		sum += v * v
	}
	rms := math.Sqrt(sum / float64(len(mono)))
	db := 20 * math.Log10(rms)
	// N12 comfort target: -17.5 dBFS (RMS ≈ 0.134), under the old -15.5.
	if db < -19 || db > -16 {
		t.Fatalf("mono RMS = %.2f dBFS (=%.3f), want [-19,-16]", db, rms)
	}
	for _, ch := range [][]float64{l, r} {
		for _, v := range ch {
			if a := math.Abs(v); a > 0.9 {
				t.Fatalf("peak %f exceeds 0.9", a)
			}
		}
	}
}

func TestLoopSeamSmooth(t *testing.T) {
	data, _ := loop(t)
	_, _, mono := decode(t, data)
	deltas := make([]float64, loopSamples-1)
	for i := range deltas {
		deltas[i] = math.Abs(mono[i+1] - mono[i])
	}
	sort.Float64s(deltas)
	median := deltas[len(deltas)/2]
	seam := math.Abs(mono[0] - mono[loopSamples-1])
	if seam > 6*median {
		t.Fatalf("seam step |x[0]-x[N-1]| = %g exceeds 6x median delta %g (click)", seam, median)
	}
}

func TestRenderNotSilence(t *testing.T) {
	data, _ := loop(t)
	_, _, mono := decode(t, data)
	var sum float64
	for _, v := range mono {
		sum += math.Abs(v)
	}
	if mean := sum / float64(len(mono)); mean <= 0.01 {
		t.Fatalf("mean |x| = %g, want > 0.01 (not silence)", mean)
	}
}

// TestAuthoredEventCount: the score is authored, so the onset list length is
// frozen: 84 melody + 282 left-hand triplet arpeggio + 42 music-box doubles
// + 13 celesta halos = 421 onsets (v0.3.8.2: 90 BPM, 24 bars, the halo).
func TestAuthoredEventCount(t *testing.T) {
	_, ev := loop(t)
	counts := map[string]int{}
	for _, n := range ev {
		counts[n.Voice]++
	}
	want := map[string]int{VoiceMelody: 84, VoiceBass: 282, VoiceBox: 42, VoiceCelesta: 13}
	if len(ev) != 421 {
		t.Fatalf("total onsets = %d, want 421", len(ev))
	}
	for voice, n := range want {
		if counts[voice] != n {
			t.Fatalf("%s onsets = %d, want %d", voice, counts[voice], n)
		}
	}
}

// TestTempoGrid12ths: every authored onset sits on a 1/12-beat boundary
// within tolerance. The 1/12 grid contains BOTH the straight 16ths and the
// triplet eighths (1/4 = 3/12, 1/3 = 4/12). The tolerance covers the ±14 ms
// humanization PLUS the authored seam ritardando (≤ 36 ms from beat 84) —
// the one place the score deliberately leaves the grid.
func TestTempoGrid12ths(t *testing.T) {
	_, ev := loop(t)
	twelfth := int(math.Round(beatSec / 12 * Rate)) // 1225 samples at 90 BPM
	tol := int(math.Round(0.050 * Rate))            // 50 ms (humanize + ritardando)
	worst := 0.0
	for _, n := range ev {
		if n.Voice != VoiceMelody {
			continue // bass/box share the melody grid by construction
		}
		d := math.Mod(float64(n.Start), float64(twelfth))
		d = math.Min(d, float64(twelfth)-d)
		if d > float64(tol) {
			t.Fatalf("melody onset at %d samples is %.1f ms off the 1/12-beat grid",
				n.Start, d*1000/Rate)
		}
		if d > worst {
			worst = d
		}
	}
	t.Logf("worst grid deviation: %.1f ms (tolerance 50 ms)", worst*1000/Rate)
}

// TestOnsetSharpness: melody onsets must JUMP — this is the felt-hammer
// proof: a pure-sine voice with a slow attack cannot pass it. The v0.3.8
// score plays a sustain-pedal texture (a rolling 12/8 arpeggio under
// everything, tails ringing into every note), which raises the RMS floor in
// front of each onset — so the proof is tiered the way the music is: phrase
// ACCENTS (vel >= 0.60) must still jump >= 4x, and EVERY melody onset must
// jump >= 2.2x.
func TestOnsetSharpness(t *testing.T) {
	data, ev := loop(t)
	_, _, mono := decode(t, data)
	win := int(math.Round(0.030 * Rate))
	rms := func(start, n int) float64 {
		var sum float64
		for i := 0; i < n; i++ {
			v := mono[((start+i)%loopSamples+loopSamples)%loopSamples]
			sum += v * v
		}
		return math.Sqrt(sum / float64(n))
	}
	ratios := make([]float64, 0, 64)
	for _, n := range ev {
		if n.Voice != VoiceMelody {
			continue
		}
		before := rms(n.Start-win, win)
		after := rms(n.Start, win)
		ratio := after / before
		min := 2.2 // the every-note attack floor
		if n.Vel >= 0.60 {
			min = 4.0 // phrase accents carry the full hammer proof
		}
		if ratio < min {
			t.Fatalf("onset MIDI %.0f at sample %d (vel %.2f): RMS %.4f -> %.4f, ratio %.2f < %.1f",
				n.MIDI, n.Start, n.Vel, before, after, ratio, min)
		}
		ratios = append(ratios, ratio)
	}
	sort.Float64s(ratios)
	t.Logf("onset sharpness: min %.2f, median %.2f, max %.2f over %d melody onsets",
		ratios[0], ratios[len(ratios)/2], ratios[len(ratios)-1], len(ratios))
}
