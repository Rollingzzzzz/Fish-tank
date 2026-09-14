// N12: compose.go — the ORIGINAL score, hand-written in the FEEL of a 1971
// wistful factory-tour ballad (no existing melody is reproduced). 90 BPM 4/4
// (2/3 s/beat — exactly 14700 samples), 24 bars = 64.000 s. Form: A (bars
// 1-4), A' (5-8), B (9-18 — the yearning flight stretched to ten bars with a
// borrowed-iv → dominant turnaround, music box doubling an octave up),
// A” (19-22, the soft return) and two C-major bars breathing the loop back
// into its start. Key C major with Imaj9 colors, a borrowed iv-minor (Fm6)
// turn and one chromatic-mediant lift (Emaj9 planing) at the top of the B
// section.
//
// v0.3.8.1 revision — goal: a closer RESEMBLANCE — the left
// hand rolls the signature 12/8 triplet broken-chord wave (twelve notes per
// bar, tails ringing into each other like a sustain pedal) and the melody
// rides the same triplet lilt with long phrase-end tones.
//
// v0.3.8.2 revision: (1) tempo 75 → 90 BPM, the reference's lilt
// instead of its lullaby drag; (2) the arpeggio wave now ascends THROUGH the
// warm tenth (low root → root → tenth → fifth crest); (3) a soft celesta
// halo doubles the big B-section notes two octaves up; (4) a dynamic arc —
// A' leans in a touch, B swells to its crest, the return recedes into the
// seam; (5) an authored seam ritardando (≤ 36 ms over the final bars) so
// the loop exhales into its own start. The melody remains an authored note
// list — the rng only humanizes onsets ±14 ms.
package music

import (
	"math"
	"math/rand"
)

// Composition constants (N12). tempoBPM mirrors contract.MusicBPM.
const (
	tempoBPM   = 90.0
	beatSec    = 60.0 / tempoBPM // 2/3 s; × Rate = exactly 14700 samples
	humanizeS  = 0.014           // ±14 ms onset humanization (rng's only use)
	boxGain    = 0.25            // music-box doubling level (B section)
	celGain    = 0.10            // celesta halo level (big B-section notes)
	bStartBeat = 32.0            // B section begins (beat)
	bEndBeat   = 72.0            // B section ends, turnaround included (beat)
)

// Voice tags for the exported Note list.
const (
	VoiceMelody  = "melody"
	VoiceBass    = "bass"
	VoiceBox     = "box"
	VoiceCelesta = "celesta"
)

// event is one authored melody note: MIDI pitch, position in beats, duration
// in beats (authoring intent; the piano rings per its own decay), velocity.
type event struct {
	midi float64
	beat float64
	dur  float64
	vel  float64
}

// melody is the authored single line (C major, treble, G4..G5). Phrases rise
// stepwise and settle on LONG held tones; pickups ride the triplet lilt
// (x.333 / x.667 positions); rests breathe between phrases. Contours are
// original — the resemblance lives in the rhythm feel and the accompaniment.
var melody = []event{
	// A — bars 1-4 over Cmaj9 | Am11 | Fmaj9 | G13: a patient stepwise rise
	// into a held C5, a falling answer, then two pickup notes lifting home.
	{67, 0, 1.5, .50}, {69, 1.667, .667, .46}, {71, 2.333, .667, .50},
	{72, 3, 2, .70},
	{74, 5.333, .667, .56}, {72, 6, 1.5, .60}, {71, 7.667, .333, .40},
	{69, 8, 1.5, .54}, {72, 9.667, .333, .42}, {71, 10, .667, .50},
	{69, 10.667, .667, .46}, {67, 11.333, .667, .44},
	{67, 12, 2, .56}, {71, 14.667, .667, .46}, {74, 15.333, .667, .52},
	// A' — bars 5-8 over Cmaj9 | Em7 | Fmaj9 | G13: the answer, one step
	// higher, ending on a long B4 that hands the phrase to the B section.
	{76, 16, 2, .68}, {74, 18.333, .667, .54}, {72, 19, 1, .56},
	{74, 20.333, .667, .54}, {76, 21, 2.5, .66},
	{79, 24, 1.5, .62}, {76, 25.667, .333, .48}, {74, 26, 1, .54},
	{72, 27, 1, .50},
	{71, 28, 2.5, .58}, {67, 30.667, .333, .36}, {69, 31.333, .667, .46},
	// B — bars 9-18 over Fmaj9 | G13 | Emaj9 | Am11 | Dm9 | Fm6 | Cmaj9 |
	// Am11 | Fm6 | G13: the flight. Emaj9 is the chromatic-mediant lift (a
	// long high E); the borrowed-iv bars ache twice; the turnaround link
	// sighs down and lifts the hand back toward home under the music box.
	{69, 32, .667, .42}, {72, 32.667, .667, .46}, {76, 33.333, 1.667, .74},
	{74, 35, 1, .58},
	{72, 36, .667, .50}, {71, 36.667, .667, .48}, {69, 37.333, .667, .46},
	{71, 38, 2, .60},
	{72, 40, .667, .48}, {74, 40.667, .667, .54}, {76, 41.333, 2.667, .84},
	{78, 44.5, .5, .82}, {76, 45, .667, .58}, {74, 45.667, .667, .54},
	{72, 46.333, .667, .52}, {71, 47, 1, .50},
	{69, 48, 1.5, .54}, {72, 49.667, .333, .46}, {74, 50, 1, .58},
	{77, 51, 1, .60},
	{76, 52, 1.5, .60}, {74, 53.667, .333, .48}, {72, 54, .667, .52},
	{71, 54.667, .667, .48}, {69, 55.333, .667, .46},
	{72, 56, 1.5, .60}, {76, 57.667, .333, .48}, {74, 58, .667, .54},
	{72, 58.667, .667, .52}, {74, 59.333, .667, .54},
	{71, 60, 2, .58}, {69, 62.333, .667, .40}, {71, 63, 1, .50},
	// turnaround — bars 17-18 over Fm6 | G13: the ache sighs down, then the
	// dominant lifts the line back toward home.
	{76, 64, 1.5, .60}, {74, 65.667, .667, .50}, {72, 66.333, .667, .52},
	{69, 67, .667, .48}, {71, 68, 1.5, .54}, {67, 69.667, .333, .40},
	{69, 70, .667, .48}, {71, 70.667, .667, .50}, {74, 71.333, .667, .56},
	// A'' — bars 19-22 over Cmaj9 | Am11 | Fmaj9 | Fm6: the soft return
	// (the B4 over Fmaj9 shines as a quiet Lydian shimmer), receding.
	{76, 72, 2, .64}, {74, 74.333, .667, .50}, {72, 75, .667, .52},
	{74, 75.667, .667, .52},
	{72, 76, 1.5, .56}, {69, 77.667, .333, .44}, {71, 78, 2, .54},
	{72, 80, 1, .54}, {74, 81.333, .667, .50}, {72, 82, .667, .52},
	{69, 82.667, .667, .46}, {67, 83.333, .667, .44},
	{67, 84, 2, .40},
	// the last whisper — two notes dissolving over the final C bars.
	{74, 86.333, .667, .34}, {72, 87, .667, .36},
}

// bar is one bar of harmony: low warm pad chord plus the four arpeggio
// tones the left hand rolls in triplet eighths.
type bar struct {
	pad   [3]float64 // low pad bed: root, fifth, tenth
	root  float64    // mid root (the arpeggio's home tone)
	fifth float64    // fifth above it (0 = the lone-root air bar)
}

// progression is the 24-bar harmonic plan (MIDI numbers).
var progression = []bar{
	{[3]float64{36, 43, 52}, 48, 55}, // 1  Cmaj9
	{[3]float64{45, 52, 60}, 45, 52}, // 2  Am11
	{[3]float64{41, 48, 57}, 41, 48}, // 3  Fmaj9
	{[3]float64{43, 50, 59}, 43, 50}, // 4  G13
	{[3]float64{36, 43, 52}, 48, 55}, // 5  Cmaj9
	{[3]float64{40, 47, 55}, 40, 47}, // 6  Em7
	{[3]float64{41, 48, 57}, 41, 48}, // 7  Fmaj9
	{[3]float64{43, 50, 59}, 43, 50}, // 8  G13
	{[3]float64{41, 48, 57}, 41, 48}, // 9  Fmaj9 (B begins)
	{[3]float64{43, 50, 59}, 43, 50}, // 10 G13
	{[3]float64{40, 47, 56}, 40, 47}, // 11 Emaj9 — chromatic-mediant lift
	{[3]float64{45, 52, 60}, 45, 52}, // 12 Am11
	{[3]float64{38, 45, 53}, 38, 45}, // 13 Dm9
	{[3]float64{41, 48, 56}, 41, 48}, // 14 Fm6 — borrowed iv
	{[3]float64{36, 43, 52}, 48, 55}, // 15 Cmaj9
	{[3]float64{45, 52, 60}, 45, 52}, // 16 Am11 — the ache returns
	{[3]float64{41, 48, 56}, 41, 48}, // 17 Fm6 — borrowed iv again
	{[3]float64{43, 50, 59}, 43, 50}, // 18 G13 (turnaround into the return)
	{[3]float64{36, 43, 52}, 48, 55}, // 19 Cmaj9 (A'' begins)
	{[3]float64{45, 52, 60}, 45, 52}, // 20 Am11
	{[3]float64{41, 48, 57}, 41, 48}, // 21 Fmaj9 — the quiet Lydian bar
	{[3]float64{41, 48, 56}, 41, 48}, // 22 Fm6 — the wistful turn
	{[3]float64{36, 43, 52}, 48, 55}, // 23 Cmaj9 — soft
	{[3]float64{36, 43, 52}, 48, 0},  // 24 Cmaj9 — half bar, then air:
	// the accompaniment exhales so the loop resolves back into its start
}

// midiHz converts a MIDI note number to frequency in Hz.
func midiHz(m float64) float64 { return 440 * math.Pow(2, (m-69)/12) }

// arcMult is the dynamic arc (v0.3.8.2): A' leans in a touch, B swells to
// its crest across the flight, the turnaround holds the light, and the
// return recedes bar by bar into the seam.
func arcMult(beat float64) float64 {
	switch {
	case beat < 16:
		return 1.00
	case beat < 32:
		return 1.03
	case beat < 64:
		return 0.97 + 0.09*(beat-32)/32 // swell toward the lift
	case beat < 72:
		return 1.06
	case beat < 88:
		return 0.95 - 0.10*(beat-72)/16 // recede into the seam
	default:
		return 0.82
	}
}

// ritardS is the authored seam ritardando (v0.3.8.2): from beat 84 on, every
// onset lands a little later — at most 36 ms — so the loop breathes out
// before wrapping into its own downbeat.
func ritardS(beat float64) float64 {
	if beat < 84 {
		return 0
	}
	return math.Min(0.036, (beat-84)*0.004)
}

// Note is one scheduled percussive onset in the rendered loop
// (post-humanization); Start is the wrapped sample index within the 64 s
// buffer. Pads are a bed and are deliberately not listed.
type Note struct {
	MIDI  float64
	Start int
	Vel   float64
	Voice string
}

// compose writes the whole loop into the circular L/R buffers and returns
// every onset it scheduled: the rolling triplet left hand first, then the
// melody with its B-section music-box doubles and celesta halos. Onset
// humanization is drawn once per BEAT position (memoized), so the left hand
// and a melody note sharing a downbeat strike together like a rolled chord
// instead of hammering into each other's pre-onset windows.
func compose(l, r []float64, rng *rand.Rand) []Note {
	ev := make([]Note, 0, 2*len(melody)+14*len(progression))
	jit := make(map[float64]float64, 160)
	jitter := func(beat float64) float64 {
		if j, ok := jit[beat]; ok {
			return j
		}
		j := (rng.Float64()*2 - 1) * humanizeS
		jit[beat] = j
		return j
	}
	beatN := int(math.Round(beatSec * Rate))
	for b, br := range progression {
		base := b * 4 * beatN
		for i, m := range br.pad {
			if m == 0 {
				continue // silent-bar marker: air before the loop restarts
			}
			pad(l, r, base, midiHz(m), padGain*(1-0.06*float64(i)))
		}
		// left hand: the signature 12/8 wave — the arpeggio ascends through
		// the warm tenth to the fifth crest and rocks back, in triplet
		// eighths, well under the melody in level. The air bar plays only
		// the first half, softer, so the loop can breathe.
		tones := [4]float64{br.pad[0], br.root, br.fifth, br.pad[2]}
		wave := [12]int{0, 1, 3, 2, 3, 1, 0, 1, 3, 2, 3, 1}
		n := 12
		soft := 1.0
		if br.fifth == 0 { // bar 24: six notes, then silence
			n, soft = 6, 0.7
		}
		for k := 0; k < n; k++ {
			beat := float64(b*4) + float64(k)/3
			vel := 0.175 - 0.025*float64(k%3) + 0.025*float64(k%2)
			if k%3 == 0 {
				vel = 0.20 // each triplet group leans on its first tone
			}
			if k == 0 {
				vel = 0.23 // the bar's low root grounds the chord
			}
			ev = appendOnset(l, r, jitter, ev, beat, tones[wave[k]], vel*soft*arcMult(beat), VoiceBass)
		}
	}
	for _, e := range melody {
		vel := e.vel * arcMult(e.beat)
		pos := onset(jitter, e.beat)
		piano(l, r, pos, midiHz(e.midi), vel)
		ev = append(ev, Note{MIDI: e.midi, Start: pos, Vel: vel, Voice: VoiceMelody})
		if e.beat >= bStartBeat && e.beat < bEndBeat {
			musicbox(l, r, pos, midiHz(e.midi+12), vel*boxGain)
			ev = append(ev, Note{MIDI: e.midi + 12, Start: pos,
				Vel: vel * boxGain, Voice: VoiceBox})
			if e.vel >= 0.55 { // celesta halo over the big B-section notes
				musicbox(l, r, pos, midiHz(e.midi+24), vel*celGain)
				ev = append(ev, Note{MIDI: e.midi + 24, Start: pos,
					Vel: vel * celGain, Voice: VoiceCelesta})
			}
		}
	}
	return ev
}

// onset computes one humanized sample position (±humanizeS plus the authored
// seam ritardando) for a beat. Negative positions (a jittered beat-0
// downbeat) clamp to 0 rather than wrapping — the loop must START on its
// downbeat, not ring one at its end.
func onset(jitter func(float64) float64, beat float64) int {
	t := (beat+jitter(beat))*beatSec*Rate + ritardS(beat)*Rate
	pos := int(math.Round(t))
	if pos < 0 {
		pos = 0
	}
	return pos % loopSamples
}

// appendOnset schedules one piano onset and appends its Note.
func appendOnset(l, r []float64, jitter func(float64) float64, ev []Note, beat, midi, vel float64, voice string) []Note {
	pos := onset(jitter, beat)
	piano(l, r, pos, midiHz(midi), vel)
	return append(ev, Note{MIDI: midi, Start: pos, Vel: vel, Voice: voice})
}
