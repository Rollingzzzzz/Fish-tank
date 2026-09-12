// G0.2: frozen shared types — exact copies of README §4 (D3: change README first).
package contract

// Vec2 is a 2D point/velocity in world pixels.
type Vec2 struct{ X, Y float64 }

// Palette holds the four neon colors of a species (hex "#rrggbb").
type Palette struct {
	Body   string `json:"body"`
	Belly  string `json:"belly"`
	Accent string `json:"accent"`
	Glow   string `json:"glow"`
}

// Pattern describes the body overlay of a species.
type Pattern struct {
	Type    string  `json:"type"`    // "stripe" | "spot" | "koi" | "vein" | "wave"
	Density float64 `json:"density"` // 0..1
	Size    float64 `json:"size"`    // 0..1
}

// Behavior drives the steering weights of a species.
type Behavior struct {
	Speed       float64 `json:"speed"`      // 0.5..1.6
	Schooling   float64 `json:"schooling"`  // 0..1
	Curiosity   float64 `json:"curiosity"`  // 0..1
	Skittish    float64 `json:"skittish"`   // 0..1
	Depth       float64 `json:"depth"`      // 0..1 (0 surface .. 1 floor)
	Attachment  float64 `json:"attachment"` // 0..1 glass-hugging affinity (v2; 0 = never)
	NightActive bool    `json:"nightActive"`
}

// Species role constants (v2).
const (
	RoleNormal = "normal"
	RoleChosen = "chosen" // the one immortal lilac fish (FD8/FD9)
)

// Live-treat kinds for the Treat Store (N8).
const (
	TreatBug     = "bug"
	TreatWorm    = "worm"
	TreatShrimp  = "shrimp"
	TreatChicken = "chicken"
)

// Species is one fish species definition == one file in content/species/.
type Species struct {
	ID        string   `json:"id"`   // slug: ^[a-z0-9-]{3,32}$, == filename stem
	Name      string   `json:"name"` // display name (English), unique case-insensitively
	Latin     string   `json:"latin"`
	Role      string   `json:"role"`  // v2: "normal" | "chosen" (FD9)
	Size      float64  `json:"size"`  // 0.6..1.4
	Width     float64  `json:"width"` // 0.7..1.3
	Fin       float64  `json:"fin"`   // 0.6..1.5
	Tail      float64  `json:"tail"`  // 0.7..1.4
	Palette   Palette  `json:"palette"`
	Pattern   Pattern  `json:"pattern"`
	Behavior  Behavior `json:"behavior"`
	Note      string   `json:"note"`      // one sentence
	Source    string   `json:"source"`    // "core" | "species-agent" | "water-agent" | "pattern-agent" | "pack" | "user"
	CreatedAt string   `json:"createdAt"` // RFC3339 or ""
}

// WaterPreset is one atmosphere package == one file in content/water/.
type WaterPreset struct {
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	TopColor     string      `json:"topColor"`
	BottomColor  string      `json:"bottomColor"`
	Accent       string      `json:"accent"`
	Rays         float64     `json:"rays"`         // 0..1
	Caustics     float64     `json:"caustics"`     // 0..1
	Bubbles      float64     `json:"bubbles"`      // 0..1
	PlantPalette []string    `json:"plantPalette"` // 2..4 hex
	Event        *WaterEvent `json:"event,omitempty"`
	Source       string      `json:"source"`
}

// WaterEvent is a temporary atmosphere effect attached to a WaterPreset.
type WaterEvent struct {
	Name     string  `json:"name"`
	Kind     string  `json:"kind"`        // "bubbleStorm" | "glowWave" | "current" | "calm"
	Duration float64 `json:"durationSec"` // 15..40
	Note     string  `json:"note"`
}

// PlantDesign is one plant species design == one file in content/plants/.
type PlantDesign struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Kind   string   `json:"kind"`   // v0.3.7 (F30): "ribbon" | "feather" | "silk" — silhouette archetype
	Fronds int      `json:"fronds"` // 3..8
	Height float64  `json:"height"` // 0.15..0.55 (fraction of tank height)
	Width  float64  `json:"width"`  // 0.5..1.5 multiplier
	Curve  float64  `json:"curve"`  // 0..1
	Sway   float64  `json:"sway"`   // 0..1
	Colors []string `json:"colors"` // 2..4 hex (base→tip gradient)
	Glow   float64  `json:"glow"`   // 0..1
	Source string   `json:"source"`
}

// PatternRecipe is an age-stage repaint recipe == one file in content/patterns/.
type PatternRecipe struct {
	ID       string  `json:"id"`
	Stage    string  `json:"stage"` // "fry" | "juvenile" | "adult" | "elder"
	Name     string  `json:"name"`
	SatMul   float64 `json:"satMul"`   // 0.3..1.4 saturation multiplier
	AlphaMul float64 `json:"alphaMul"` // 0.4..1.0
	GlowMul  float64 `json:"glowMul"`  // 0.5..2.0
	Pattern  Pattern `json:"pattern"`
	Note     string  `json:"note"`
}

// CoralDesign is one coral species design == one file in content/corals/ (v2).
type CoralDesign struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Kind      string   `json:"kind"`   // "fan" | "branch" | "brain"
	Fronds    int      `json:"fronds"` // 3..9 branches/polyps
	Height    float64  `json:"height"` // 0.08..0.35 (fraction of tank height)
	Width     float64  `json:"width"`  // 0.5..1.6 multiplier
	Curve     float64  `json:"curve"`  // 0..1
	Sway      float64  `json:"sway"`   // 0..1
	Colors    []string `json:"colors"` // 2..4 hex (base→tip gradient)
	Glow      float64  `json:"glow"`   // 0..1
	Source    string   `json:"source"`
	CreatedAt string   `json:"createdAt"`
}

// Zone is a circular sim region (v2): caves shelter fish, the chosen aura
// excludes every living thing but its owner.
type Zone struct {
	Center Vec2    `json:"center"`
	Radius float64 `json:"radius"`
	Owner  string  `json:"owner"` // "cave" | "chosen" | ""
}

// Hole is one enterable mouth of a volcanic crag (v0.3.6, F25). Fish swim in
// one hole and out another of the same RockID — the crag is hollow inside.
type Hole struct {
	Center Vec2    `json:"center"`
	Radius float64 `json:"radius"` // mouth half-width; entry trigger = r*0.5
	Rock   int     `json:"rock"`   // which crag this mouth belongs to
}

// Config mirrors config.json beside the exe.
type Config struct {
	Endpoint   string  `json:"endpoint"` // default Coding Plan chat completions URL
	Model      string  `json:"model"`    // FROZEN: glm-5.3-flash (C6), read-only in Settings
	APIKey     string  `json:"apiKey"`   // "" → simulate mode
	ProxyURL   string  `json:"proxy,omitempty"`
	AutoFeed   bool    `json:"autoFeed"`
	AutoCare   bool    `json:"autoCare"`
	MusicOn    bool    `json:"musicOn"`    // v2 (was soundOn; FD7: no SFX, music only)
	AgentFreq  float64 `json:"agentFreq"`  // 0.5..2, default 1
	DaySeconds float64 `json:"daySeconds"` // 20..180, default 60
	MaxFish    int     `json:"maxFish"`    // 8..100 (v0.3.7 F29), default 24

	FullscreenOnStart bool `json:"fullscreenOnStart"` // v3 (F13): borderless fullscreen launch
}

// SavedFish is the full live state of one fish (crash-safe snapshot, G2.5).
type SavedFish struct {
	SpeciesID string  `json:"speciesId"`
	Seed      int64   `json:"seed"`
	AgeDays   float64 `json:"ageDays"`
	Pos       Vec2    `json:"pos"`
	Vel       Vec2    `json:"vel"`
	Satiety   float64 `json:"satiety"` // 1 = full belly
	Energy    float64 `json:"energy"`
}

// SavedFood is one food flake mid-fall.
type SavedFood struct {
	Pos Vec2    `json:"pos"`
	Age float64 `json:"age"`
}

// SavedEgg is one unhatched egg.
type SavedEgg struct {
	SpeciesID string  `json:"speciesId"`
	Pos       Vec2    `json:"pos"`
	Progress  float64 `json:"progress"` // 0..1 toward hatch
}

// SavedCreature is one crustacean (N4, v2). Retired in v0.3.8 — the struct
// and the Save.Creatures field stay so old snapshots keep loading cleanly;
// restored entries are ignored.
type SavedCreature struct {
	Kind   string  `json:"kind"` // "shrimp" | "crab"
	Seed   int64   `json:"seed"`
	Pos    Vec2    `json:"pos"`
	ShellT float64 `json:"shellT"` // >0 = seconds left hiding in shell
}

// Save is the full world snapshot; written to save_minute/daily/weekly.json.
type Save struct {
	SchemaVersion int             `json:"schemaVersion"` // > current → refuse load (log + fresh world)
	Fish          []SavedFish     `json:"fish"`
	Foods         []SavedFood     `json:"foods"`
	Eggs          []SavedEgg      `json:"eggs"`
	Clock         float64         `json:"clock"`    // day-cycle seconds
	Day           int             `json:"day"`      // v2: full tank days elapsed
	RockSeed      int64           `json:"rockSeed"` // v2: deterministic rock/cave/nest layout
	Care          float64         `json:"care"`     // care score
	WaterID       string          `json:"waterId"`
	PlantIDs      []string        `json:"plantIds"`
	CoralIDs      []string        `json:"coralIds"`  // v2
	Creatures     []SavedCreature `json:"creatures"` // v2
}
