// G6.2: bootstrap — first run, single-instance lock, world restore.
package game

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/Rollingzzzzz/Fish-tank/internal/audio"
	"github.com/Rollingzzzzz/Fish-tank/internal/config"
	"github.com/Rollingzzzzz/Fish-tank/internal/content"
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/Rollingzzzzz/Fish-tank/internal/music"
	"github.com/Rollingzzzzz/Fish-tank/internal/sim"
)

const lockFile = "tank.lock"

// Bootstrap prepares everything and returns a ready Game.
// root is the directory that holds config.json, content/ and saves.
func Bootstrap(root string) (*Game, error) {
	// single-instance lock: PID-based liveness. A crashed instance's lock
	// is detected as stale immediately (no 90 s wait), so crash recovery
	// stays instant.
	lockPath := filepath.Join(root, lockFile)
	if b, err := os.ReadFile(lockPath); err == nil {
		if pid := parsePID(string(b)); pid > 0 && processAlive(pid) {
			return nil, fmt.Errorf("tank is already running (pid %d)", pid)
		}
		fmt.Println("stale lock removed (owner pid is gone)")
	}
	touchLock := func() {
		_ = os.WriteFile(lockPath, []byte(fmt.Sprintf("pid=%d", os.Getpid())), 0o644)
	}
	touchLock()

	// config
	cfgPath := filepath.Join(root, "config.json")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Println("config was corrupt — defaults restored:", err)
	}
	if err := config.Save(cfgPath, cfg); err != nil {
		return nil, fmt.Errorf("save config: %w", err)
	}

	// content (seeds core files on first run)
	contentDir := filepath.Join(root, "content")
	store, err := content.Load(contentDir)
	if err != nil {
		return nil, err
	}
	if err := store.EnsureSeed(); err != nil {
		return nil, err
	}

	// world
	world := sim.NewWorld(float64(ScreenW), float64(ScreenH), *cfg, store.Species(),
		store.Plants(), store.Waters())
	world.SetCorals(store.Corals()) // M0 stub in content; Lane B fills seeds
	if s, err := sim.LoadLatest(root); err == nil {
		if err := world.Restore(s); err != nil {
			fmt.Println("save restore failed — fresh world:", err)
		}
	} else {
		fmt.Println("no previous save — fresh world")
	}
	persist := sim.NewPersist(root, time.Now())

	// audio: the music loop is synthesized in-process (N6/FD7) — render it
	// asynchronously so boot stays instant; silence until it is ready.
	eng := audio.New()
	eng.SetOn(cfg.MusicOn)
	go func() {
		eng.SetMusic(music.Render())
		if cfg.MusicOn {
			_ = eng.PlayMusic()
		}
	}()

	g := New(cfg, cfgPath, root, store, world, persist, eng)
	g.lockPath, g.touchLock = lockPath, touchLock
	return g, nil
}

// findZCodeKeyDefault wraps the exported scanner for the game package.
func findZCodeKeyDefault() (string, bool) { return config.FindZCodeKeyDefault() }

// saveConfig persists the config beside the exe.
func saveConfig(path string, cfg *contract.Config) error { return config.Save(path, cfg) }

// maskForLog keeps secrets out of logs.
func maskForLog(key string) string { return config.MaskKey(key) }

// findKey wraps the ZCode scan with the default directory.
func findKey() (string, bool) { return findZCodeKeyDefault() }

// removeSave deletes a save file if it exists.
func removeSave(root, name string) {
	_ = os.Remove(filepath.Join(root, name))
}

// parsePID reads "pid=N" from lock content.
func parsePID(s string) int {
	var pid int
	_, err := fmt.Sscanf(s, "pid=%d", &pid)
	if err != nil {
		return 0
	}
	return pid
}

// processAlive reports whether a process with pid exists (Windows tasklist
// probe; falls back to true when tasklist is unavailable).
func processAlive(pid int) bool {
	out, err := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/NH").Output()
	if err != nil {
		return true // cannot tell — assume alive (conservative)
	}
	return len(out) > 0 && !bytes.Contains(bytes.ToLower(out), []byte("no tasks"))
}
