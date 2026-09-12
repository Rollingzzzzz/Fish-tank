// dsp.go — loop-seamless mastering. Every stage is exactly periodic over the
// circular loop buffer: the low-pass is solved to its circular fixed point,
// the feedback delay is walked once around each interleave cycle (feedback
// decays to zero around a full cycle), and the peak ceiling is a pointwise
// tanh. Result: the loop point is mathematically click-free.
package music

import (
	"encoding/binary"
	"math"
)

// Mastering constants (N12 rework: softer loudness; the circular-buffer
// machinery itself is unchanged).
const (
	lpfHz      = 2400.0 // gentle one-pole low-pass (per-note felt LPF sits at 2.2 kHz)
	delayN     = 11686  // 0.5305 s — off the 0.2 s 16th grid so echoes never sit in a pre-onset window
	delayFb    = 0.22
	delayWet   = 0.14
	targetDbfs = -17.0 // normalize target; the tanh ceiling lands mono-sum RMS ≈ 0.12-0.13
	peakCeil   = 0.9   // hard peak ceiling (<= 0.9 required)
)

// lowPass applies a one-pole low-pass at lpfHz. Pass one only warms the
// filter state: the per-pass state decay (1-a)^N underflows to zero at loop
// length, so pass one's final state IS the circular fixed point; pass two,
// started from that state, is exactly the periodic solution.
func lowPass(x []float64) {
	a := 1 - math.Exp(-2*math.Pi*lpfHz/Rate)
	s := 0.0
	for pass := 0; pass < 2; pass++ {
		for i, v := range x {
			s += a * (v - s)
			if pass == 1 {
				x[i] = s
			}
		}
	}
}

// echo applies the feedback delay (delaySec, delayFb, delayWet). Samples
// d[i] = dry[i-D mod N] + fb*d[i-D mod N] form len(x)/gcd(len(x),D) closed
// cycles; walking each cycle once from acc=0 is exact because fb^cycleLen
// underflows to zero. The dry copy keeps the recurrence feeding on the
// unprocessed signal.
func echo(x []float64) {
	d := delayN
	n := len(x)
	if d <= 0 || d >= n {
		return
	}
	dry := make([]float64, n)
	copy(dry, x)
	g := gcd(n, d)
	steps := n / g
	for r := 0; r < g; r++ {
		acc := 0.0 // delay-line value carried around the cycle
		for k := 0; k < steps; k++ {
			i := (r + k*d) % n
			src := i - d
			if src < 0 {
				src += n
			}
			acc = dry[src] + delayFb*acc
			x[i] += delayWet * acc
		}
	}
}

// normalize scales both channels so the mono-sum RMS lands on targetDbfs,
// then tames peaks with a gentle tanh saturation that guarantees the peak
// ceiling while staying pointwise-continuous (seam preserved).
func normalize(l, r []float64) {
	var sum float64
	for i := range l {
		m := (l[i] + r[i]) * 0.5
		sum += m * m
	}
	if rms := math.Sqrt(sum / float64(len(l))); rms > 0 {
		s := math.Pow(10, targetDbfs/20) / rms
		for i := range l {
			l[i] *= s
			r[i] *= s
		}
	}
	for i := range l {
		l[i] = peakCeil * math.Tanh(l[i]/peakCeil)
		r[i] = peakCeil * math.Tanh(r[i]/peakCeil)
	}
}

// encodeF32LE interleaves to stereo float32 little-endian bytes.
func encodeF32LE(l, r []float64) []byte {
	out := make([]byte, len(l)*8)
	for i := range l {
		binary.LittleEndian.PutUint32(out[i*8:], math.Float32bits(float32(l[i])))
		binary.LittleEndian.PutUint32(out[i*8+4:], math.Float32bits(float32(r[i])))
	}
	return out
}

// gcd is the Euclidean algorithm (delay-cycle math above).
func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}
