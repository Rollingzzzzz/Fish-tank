// G6.2: config.json management — defaults, load/save beside the exe.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// Default returns the canonical configuration (C6: model is frozen).
// v1 defaults: the tank wakes up self-sufficient — auto feed + auto care on,
// music off by request, and a lively 60-fish school (slider: 8..100).
func Default() *contract.Config {
	return &contract.Config{
		Endpoint:          "https://api.z.ai/api/coding/paas/v4/chat/completions",
		Model:             "glm-5.3-flash",
		APIKey:            "",
		ProxyURL:          "",
		AutoFeed:          true,
		AutoCare:          true,
		MusicOn:           false,
		FullscreenOnStart: true, // F13: the tank opens immersive, edge to edge
		AgentFreq:         1,
		DaySeconds:        60,
		MaxFish:           60,
	}
}

// Load reads config.json; missing file → defaults; corrupt file → defaults
// with a .bad backup of the corrupt one (D4: never crash).
func Load(path string) (*contract.Config, error) {
	cfg := Default()
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := json.Unmarshal(b, cfg); err != nil {
		backup := path + ".bad"
		_ = os.Rename(path, backup)
		return Default(), err
	}
	migrateLegacy(cfg, b)
	normalize(cfg)
	return cfg, nil
}

// migrateLegacy carries pre-v2 keys: "soundOn" → MusicOn (FD7 renamed the
// toggle; the user's choice must survive the upgrade).
func migrateLegacy(cfg *contract.Config, raw []byte) {
	var legacy map[string]any
	if json.Unmarshal(raw, &legacy) != nil {
		return
	}
	if _, hasMusic := legacy["musicOn"]; !hasMusic {
		if v, ok := legacy["soundOn"].(bool); ok {
			cfg.MusicOn = v
		}
	}
}

// Save atomically writes the config.
func Save(path string, cfg *contract.Config) error {
	normalize(cfg)
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// normalize clamps every knob to its documented range.
func normalize(c *contract.Config) {
	if c.Endpoint == "" {
		c.Endpoint = Default().Endpoint
	}
	if c.Model == "" {
		c.Model = Default().Model
	}
	c.AgentFreq = contract.Clamp(c.AgentFreq, 0.5, 2)
	c.DaySeconds = contract.Clamp(c.DaySeconds, 20, 180)
	c.MaxFish = int(contract.Clamp(float64(c.MaxFish), 8, 100)) // v0.3.7 F29: the player may stock up to 100
}

// DirOf returns the directory part of a path (app root helper).
func DirOf(path string) string { return filepath.Dir(path) }
