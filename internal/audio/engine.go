// N6/FD7 + N12: the sound engine — one seamless music loop, nothing else.
// Sound effects are gone forever: the tank plays only the ambient piano
// loop synthesized by internal/music. N12: PlayMusic fades in over ~3 s so
// the felt piano never startles the listener on launch.
package audio

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

// Playback-comfort constants (N12).
const (
	fadeInSec  = 3.0 // music ramps from silence to full volume over 3 s
	fadeStepMs = 50  // ramp resolution
)

// Engine holds the music player and the mute state.
type Engine struct {
	mu      sync.Mutex
	ctx     *audio.Context
	on      bool
	music   []byte // stereo F32 LE bytes (loop source)
	player  *audio.Player
	volume  float64
	started bool
}

// New builds an engine (no files needed — music is synthesized in-process).
func New() *Engine {
	return &Engine{ctx: audio.NewContext(22050), on: true, volume: 0.6}
}

// SetMusic installs the loop bytes (from music.Render). Safe to call once;
// later calls replace the source at the next PlayMusic.
func (e *Engine) SetMusic(stereoF32LE []byte) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.music = stereoF32LE
	if e.player != nil {
		e.player.Close()
		e.player = nil
		e.started = false
	}
}

// SetOn toggles the music.
func (e *Engine) SetOn(on bool) {
	e.mu.Lock()
	e.on = on
	p := e.player
	e.mu.Unlock()
	if p == nil {
		return
	}
	if on {
		p.Play()
	} else {
		p.Pause()
	}
}

// On reports the mute state.
func (e *Engine) On() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.on
}

// SetVolume sets the target volume (0..1); an in-progress fade-in ramps
// toward the new target from its current level.
func (e *Engine) SetVolume(v float64) {
	e.mu.Lock()
	e.volume = v
	e.mu.Unlock()
}

// PlayMusic starts the seamless loop with a ~3 s fade-in from silence (the
// ramping goroutine calls player.SetVolume repeatedly, which is safe). It is
// a no-op when muted, empty, or already playing.
func (e *Engine) PlayMusic() error {
	e.mu.Lock()
	if !e.on || len(e.music) == 0 || e.started {
		e.mu.Unlock()
		return nil
	}
	loop := audio.NewInfiniteLoopF32(bytes.NewReader(e.music), int64(len(e.music)))
	p, err := e.ctx.NewPlayerF32(loop)
	if err != nil {
		e.mu.Unlock()
		return fmt.Errorf("audio: music player: %w", err)
	}
	p.SetVolume(0) // silence at the seam of the fade-in
	e.player = p
	e.started = true
	p.Play()
	e.mu.Unlock()
	go e.fadeIn(p)
	return nil
}

// fadeIn ramps the player from silence to the engine's target volume over
// fadeInSec, stepping every fadeStepMs. It stops if the player it belongs to
// was replaced (SetMusic) meanwhile.
func (e *Engine) fadeIn(p *audio.Player) {
	steps := int(fadeInSec * 1000 / fadeStepMs)
	for i := 1; i <= steps; i++ {
		time.Sleep(time.Duration(fadeStepMs) * time.Millisecond)
		e.mu.Lock()
		target := e.volume
		current := e.player
		e.mu.Unlock()
		if current != p {
			return // replaced by SetMusic: nothing to ramp anymore
		}
		p.SetVolume(target * float64(i) / float64(steps))
	}
}
