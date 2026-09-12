// N12: voices.go — the felt piano, the music box and the warm pad. The piano
// has a REAL hammer: a bright partial stack that decays in ~85 ms over a
// two-stage singing body (fast initial settle into a 2-3.9 s pitch-scaled
// ring), two slightly detuned string unisons per partial (±0.15%),
// velocity-driven brightness and a gentle 2.2 kHz felt low-pass (the
// underwater piano). Every write goes through addAt, which wraps modulo the
// loop length, so tails ring across the loop point seamlessly.
package music

import "math"

// Partial tables: [ratio, gain] pairs.
var (
	// bodyPartials is the singing string stack; the 4.2x partial gives a
	// slight inharmonic edge.
	bodyPartials = [][2]float64{{1, 1}, {2, 0.42}, {3, 0.19}, {4.2, 0.07}}
	// hammerPartials sits above the fundamental — the felt thump, weighted
	// to the mid range so the felt low-pass keeps its punch.
	hammerPartials = [][2]float64{{1.5, 0.3}, {2, 0.55}, {3, 0.85}, {4.2, 0.55}}
	// boxPartials uses bell ratios (inharmonic) for the B-section music box.
	boxPartials = [][2]float64{{1, 0.9}, {2.76, 0.42}, {5.4, 0.22}, {8.9, 0.1}}
	// padPartials is retuned warmer (octave + gentle upper stack).
	padPartials = [][2]float64{{1, 1}, {2, 0.26}, {3, 0.09}, {4, 0.035}}
)

// Piano voice shaping.
const (
	pianoAttackN = Rate * 4 / 1000 // ~4 ms attack (hammer must be immediate)
	hammerAttack = Rate * 2 / 1000 // hammer rises even faster
	hammerTau    = 0.060           // hammer decays in ~60 ms
	hammerGain   = 1.8             // hammer strength relative to the body
	pianoMaxN    = 6 * Rate        // hard length cap; tail fade below
	pianoFadeN   = Rate / 4        // final 0.25 s linear fade
	sustainFrac  = 0.12            // two-stage body: fast settle -> long ring
	feltLPFHz    = 2200.0          // felt damping (underwater piano)
	pianoBodyAmp = 0.30            // note gain (pad stays well under)
)

// Pad voice shaping (bar-length chords with overlap release).
const (
	padDurSec     = 4.5 // one 3.2 s bar + release overlapping the next pad
	padAttackSec  = 1.4
	padReleaseSec = 1.3
	padLPFHz      = 640.0
	padGain       = 0.007 // per-voice: a true bed, well under the piano
)

// addAt adds one sample pair at pos, wrapping modulo the loop length.
func addAt(l, r []float64, pos int, gl, gr float64) {
	i := pos % len(l)
	if i < 0 {
		i += len(l)
	}
	l[i] += gl
	r[i] += gr
}

// detune returns a deterministic per-partial detune factor within ±frac,
// derived from the note frequency (no rng — stable across renders).
func detune(freq float64, k int, frac float64) float64 {
	u := math.Sin(freq*12.9898 + float64(k)*78.233)
	u = u * 43758.5453
	u -= math.Floor(u) // 0..1
	return 1 + frac*(2*u-1)
}

// osc is a rotation oscillator: cheaper than math.Sin per sample.
type osc struct{ s, c, ds, dc float64 }

func newOsc(freq float64) *osc {
	d := 2 * math.Pi * freq / Rate
	return &osc{s: 0, c: 1, ds: math.Sin(d), dc: math.Cos(d)}
}

func (o *osc) tick() float64 {
	s, c := o.s*o.dc+o.c*o.ds, o.c*o.dc-o.s*o.ds
	o.s, o.c = s, c
	return s
}

// bodyTaus returns the two-stage decay constants (fast settle, long ring),
// both pitch-scaled: low notes ring longer (2.0..3.9 s ring, N12 spec).
func bodyTaus(freq float64) (tau1, tau2 float64) {
	k := 261.63 / freq
	tau1 = math.Min(math.Max(0.18*math.Pow(k, 0.25), 0.13), 0.40)
	tau2 = math.Min(math.Max(2.8*math.Pow(k, 0.35), 2.0), 3.9)
	return tau1, tau2
}

// piano renders one felt-piano note: detuned string body (±0.15% unisons),
// velocity-weighted partial brightness, hammer transient on top, one-pole
// felt low-pass, pitch-based stereo pan.
func piano(l, r []float64, pos int, freq, vel float64) {
	tau1, tau2 := bodyTaus(freq)
	pan := 0.5 + 0.22*math.Sin(freq*0.7)
	gl, gr := math.Sqrt(1-pan), math.Sqrt(pan)

	// string body: two detuned unisons per partial
	amp := pianoBodyAmp * vel
	tilt := 0.55 + 0.45*vel // velocity -> brightness (upper partials)
	nb := len(bodyPartials)
	voices := make([][]*osc, nb)
	gains := make([]float64, nb)
	for i, p := range bodyPartials {
		f := freq * p[0]
		voices[i] = []*osc{
			newOsc(f * detune(freq, i, 0.0015)),
			newOsc(f * detune(freq, i+10, 0.0015)),
		}
		gains[i] = p[1] * math.Pow(tilt, float64(i))
	}

	// hammer: single bright stack, fast decay
	hAmp := hammerGain * math.Pow(vel, 0.8) * pianoBodyAmp
	hammers := make([]*osc, len(hammerPartials))
	hGains := make([]float64, len(hammerPartials))
	for i, p := range hammerPartials {
		hammers[i] = newOsc(freq * p[0])
		hGains[i] = p[1] * math.Pow(tilt, float64(i))
	}

	felt := 1 - math.Exp(-2*math.Pi*feltLPFHz/Rate) // one-pole coefficient
	var sf float64
	for i := 0; i < pianoMaxN; i++ {
		t := float64(i) / Rate
		env := (1-sustainFrac)*math.Exp(-t/tau1) + sustainFrac*math.Exp(-t/tau2)
		if i < pianoAttackN {
			env *= 0.5 - 0.5*math.Cos(math.Pi*float64(i)/pianoAttackN)
		} else if i >= pianoMaxN-pianoFadeN {
			env *= float64(pianoMaxN-i) / pianoFadeN
		}
		if i > pianoAttackN && env*amp < 1e-4 { // body silent, hammer long gone
			break
		}
		s := 0.0
		for k := 0; k < nb; k++ {
			s += gains[k] * 0.5 * (voices[k][0].tick() + voices[k][1].tick())
		}
		s *= env * amp
		if hEnv := math.Exp(-t / hammerTau); hEnv > 1e-3 { // hammer transient
			if i < hammerAttack {
				hEnv *= 0.5 - 0.5*math.Cos(math.Pi*float64(i)/hammerAttack)
			}
			h := 0.0
			for k := range hammers {
				h += hGains[k] * hammers[k].tick()
			}
			s += h * hEnv * hAmp
		}
		sf += felt * (s - sf) // felt damping
		addAt(l, r, pos+i, sf*gl, sf*gr)
	}
}

// musicbox renders one bright bell double (B section only): inharmonic
// partials, short decay, quick attack — a music box an octave above.
func musicbox(l, r []float64, pos int, freq, vel float64) {
	tau := math.Min(math.Max(0.5*math.Pow(261.63/freq, 0.2), 0.35), 0.8)
	n := int(1.6 * Rate)
	atk := Rate * 2 / 1000
	amp := 0.30 * vel
	pan := 0.56 + 0.1*math.Sin(freq*0.31)
	gl, gr := math.Sqrt(1-pan), math.Sqrt(pan)
	voices := make([]*osc, len(boxPartials))
	for i, p := range boxPartials {
		voices[i] = newOsc(freq * p[0])
	}
	for i := 0; i < n; i++ {
		env := math.Exp(-float64(i) / Rate / tau)
		if i < atk {
			env *= 0.5 - 0.5*math.Cos(math.Pi*float64(i)/float64(atk))
		}
		if env*amp < 1e-4 {
			break
		}
		s := 0.0
		for k, p := range boxPartials {
			s += p[1] * voices[k].tick()
		}
		addAt(l, r, pos+i, s*env*amp*gl, s*env*amp*gr)
	}
}

// pad renders one sustained chord tone: 1.4 s cosine attack, 1.3 s cosine
// release, one-pole low-passed and centered. The last bar's release runs
// past the loop end and wraps via addAt, crossfading the seam.
func pad(l, r []float64, pos int, freq, gain float64) {
	n := int(padDurSec * Rate)
	atk := int(padAttackSec * Rate)
	rel := int(padReleaseSec * Rate)
	a := 1 - math.Exp(-2*math.Pi*padLPFHz/Rate)
	voices := make([]*osc, len(padPartials))
	for i, p := range padPartials {
		voices[i] = newOsc(freq * p[0] * detune(freq, i, 0.0015))
	}
	var sl, sr float64
	for i := 0; i < n; i++ {
		env := 1.0
		if i < atk {
			env = 0.5 - 0.5*math.Cos(math.Pi*float64(i)/float64(atk))
		} else if i >= n-rel {
			env = 0.5 + 0.5*math.Cos(math.Pi*float64(i-(n-rel))/float64(rel))
		}
		s := 0.0
		for k, p := range padPartials {
			s += p[1] * voices[k].tick()
		}
		sl += a * (s*env*gain - sl)
		sr += a * (s*env*gain - sr)
		addAt(l, r, pos+i, sl, sr)
	}
}
