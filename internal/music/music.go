// music.go — render entry points. Render synthesizes the loop; RenderEvents
// also returns the authored score's onset list (deterministic per D5).
package music

import "math/rand"

// Rate is the sample rate the engine context runs at.
const Rate = 22050

// loopSeconds mirrors contract.MusicLoopSec (this package stays stdlib-only).
const loopSeconds = 64.0

// loopSamples is the exact per-channel sample count of one loop.
const loopSamples = Rate * int(loopSeconds)

// seedFixed pins the composition RNG so Render is byte-identical across
// runs and machines (D5 determinism). The rng only humanizes onsets ±14 ms.
const seedFixed = 0x4E454F4E // "NEON"

// Render synthesizes one seamless 64 s loop as interleaved stereo F32 LE
// bytes at Rate: the ORIGINAL N12 piano piece — a slow, wistful 75 BPM AABA
// ballad (C major, Imaj9 colors, borrowed-iv turn, one chromatic-mediant
// lift) with a felt hammer piano, music-box doubling in the B section and a
// warm pad bed. Notes are written onto a circular stereo buffer so decay
// tails wrap the loop point; the mastering stages (low-pass, feedback delay,
// tanh peak ceiling) are exactly periodic over that buffer, so the loop seam
// is mathematically click-free. The score is authored in compose.go and
// reproduces no existing melody.
func Render() []byte {
	pcm, _ := RenderEvents()
	return pcm
}

// RenderEvents is Render plus the list of percussive onsets (melody, bass,
// music box) the score scheduled, with their post-humanization sample
// positions — evidence input for tests and the demo-music stats dump.
func RenderEvents() ([]byte, []Note) {
	l := make([]float64, loopSamples)
	r := make([]float64, loopSamples)
	ev := compose(l, r, rand.New(rand.NewSource(seedFixed)))
	lowPass(l)
	lowPass(r)
	echo(l)
	echo(r)
	normalize(l, r)
	return encodeF32LE(l, r), ev
}
