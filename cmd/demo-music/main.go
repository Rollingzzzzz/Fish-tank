// N12: command demo-music renders the internal/music 64 s loop for human
// review: -wav <path> writes a 16-bit PCM stereo WAV (22050 Hz, standard
// 44-byte header); -stats prints the acceptance numbers: length, loudness,
// authored event counts, seam delta, 1/12-beat-grid deviation and
// hammer-onset sharpness (min/median ratio of RMS after vs before each
// melody onset).
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"math"
	"os"
	"sort"

	"github.com/Rollingzzzzz/Fish-tank/internal/music"
)

func main() {
	wav := flag.String("wav", "", "write a 16-bit PCM stereo WAV to this path")
	stats := flag.Bool("stats", false, "print full acceptance statistics")
	flag.Parse()

	data, notes := music.RenderEvents()
	perCh := len(data) / 8
	rmsDb, peak, mono := analyze(data)
	fmt.Printf("length : %d samples/channel (%.3f s @ %d Hz), %d bytes\n",
		perCh, float64(perCh)/music.Rate, music.Rate, len(data))
	fmt.Printf("loudness: %.2f dBFS mono-sum RMS (%.4f linear), peak %.4f\n",
		rmsDb, math.Pow(10, rmsDb/20), peak)

	counts := map[string]int{}
	for _, n := range notes {
		counts[n.Voice]++
	}
	fmt.Printf("events : %d onsets (%d melody, %d bass, %d music-box)\n",
		len(notes), counts[music.VoiceMelody], counts[music.VoiceBass], counts[music.VoiceBox])

	printSeam(mono, perCh)
	printGrid(notes)
	printSharpness(mono, perCh, notes)

	if *wav != "" {
		if err := writeWAV(*wav, data); err != nil {
			fmt.Fprintln(os.Stderr, "wav:", err)
			os.Exit(1)
		}
		fmt.Printf("wrote %s (%d bytes PCM, 16-bit stereo)\n", *wav, len(data)/2)
	} else if !*stats {
		fmt.Println("tip: -wav loop.wav to listen, -stats for the full numbers")
	}
}

// analyze decodes the interleaved F32 stream and returns mono-sum RMS in
// dBFS, absolute peak, and the mono samples.
func analyze(data []byte) (rmsDb float64, peak float64, mono []float64) {
	n := len(data) / 8
	mono = make([]float64, n)
	var sum float64
	for i := 0; i < n; i++ {
		l := float64(math.Float32frombits(binary.LittleEndian.Uint32(data[i*8:])))
		r := float64(math.Float32frombits(binary.LittleEndian.Uint32(data[i*8+4:])))
		mono[i] = (l + r) * 0.5
		sum += mono[i] * mono[i]
		if a := math.Abs(l); a > peak {
			peak = a
		}
		if a := math.Abs(r); a > peak {
			peak = a
		}
	}
	if rms := math.Sqrt(sum / float64(n)); rms > 0 {
		rmsDb = 20 * math.Log10(rms)
	}
	return rmsDb, peak, mono
}

// printSeam reports the loop-seam step against the median sample delta.
func printSeam(mono []float64, n int) {
	deltas := make([]float64, n-1)
	for i := range deltas {
		deltas[i] = math.Abs(mono[i+1] - mono[i])
	}
	sort.Float64s(deltas)
	fmt.Printf("seam   : |x[0]-x[N-1]| = %.6f (median sample delta %.6f, %.1fx)\n",
		math.Abs(mono[0]-mono[n-1]), deltas[len(deltas)/2],
		math.Abs(mono[0]-mono[n-1])/deltas[len(deltas)/2])
}

// printGrid reports how far the worst melody onset strays from the 1/12-beat
// grid (66.7 ms — contains both straight 16ths and triplet eighths).
func printGrid(notes []music.Note) {
	twelfth := int(math.Round(60.0 / 90.0 / 12 * float64(music.Rate))) // 1225 samples
	worst := 0.0
	for _, nt := range notes {
		if nt.Voice != music.VoiceMelody {
			continue
		}
		d := math.Mod(float64(nt.Start), float64(twelfth))
		d = math.Min(d, float64(twelfth)-d)
		if d > worst {
			worst = d
		}
	}
	fmt.Printf("grid   : worst melody deviation %.1f ms off the 1/12-beat grid (±14 ms humanize)\n",
		worst*1000/music.Rate)
}

// printSharpness reports the hammer quality: per melody onset, the RMS in
// the 30 ms after versus the 30 ms before (circular windows).
func printSharpness(mono []float64, n int, notes []music.Note) {
	win := int(math.Round(0.030 * float64(music.Rate)))
	rms := func(start int) float64 {
		var sum float64
		for i := 0; i < win; i++ {
			v := mono[((start+i)%n+n)%n]
			sum += v * v
		}
		return math.Sqrt(sum / float64(win))
	}
	ratios := make([]float64, 0, len(notes))
	for _, nt := range notes {
		if nt.Voice != music.VoiceMelody {
			continue
		}
		ratios = append(ratios, rms(nt.Start)/rms(nt.Start-win))
	}
	sort.Float64s(ratios)
	fmt.Printf("onsets : sharpness min %.2fx, median %.2fx, max %.2fx over %d melody onsets (accents >= 4x, all >= 2.2x)\n",
		ratios[0], ratios[len(ratios)/2], ratios[len(ratios)-1], len(ratios))
}

// writeWAV converts F32 to clamped 16-bit PCM and writes a standard 44-byte
// RIFF/WAVE header followed by the samples.
func writeWAV(path string, data []byte) error {
	n := len(data) / 8
	pcm := make([]byte, n*4)
	for i := 0; i < n; i++ {
		for c := 0; c < 2; c++ {
			f := float64(math.Float32frombits(binary.LittleEndian.Uint32(data[i*8+c*4:])))
			if f > 1 {
				f = 1
			} else if f < -1 {
				f = -1
			}
			binary.LittleEndian.PutUint16(pcm[i*4+c*2:], uint16(int16(f*32767)))
		}
	}
	var hdr [44]byte
	copy(hdr[0:], "RIFF")
	binary.LittleEndian.PutUint32(hdr[4:], uint32(36+len(pcm)))
	copy(hdr[8:], "WAVE")
	copy(hdr[12:], "fmt ")
	binary.LittleEndian.PutUint32(hdr[16:], 16)           // fmt chunk size
	binary.LittleEndian.PutUint16(hdr[20:], 1)            // PCM
	binary.LittleEndian.PutUint16(hdr[22:], 2)            // stereo
	binary.LittleEndian.PutUint32(hdr[24:], music.Rate)   // sample rate
	binary.LittleEndian.PutUint32(hdr[28:], music.Rate*4) // byte rate
	binary.LittleEndian.PutUint16(hdr[32:], 4)            // block align
	binary.LittleEndian.PutUint16(hdr[34:], 16)           // bits
	copy(hdr[36:], "data")
	binary.LittleEndian.PutUint32(hdr[40:], uint32(len(pcm)))

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(hdr[:]); err != nil {
		return err
	}
	_, err = f.Write(pcm)
	return err
}
