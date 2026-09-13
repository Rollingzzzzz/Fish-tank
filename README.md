# 🐠 NEON TANK — Agent-Powered Living Aquarium

> A Windows-native (`.exe`) neon aquarium where fish, plants, water moods and patterns are
> *created live by parallel GLM agents* — and the creation process is part of the show.
> Fully playable offline (procedural simulation mode) when no API key is configured.

**Repo:** https://github.com/Rollingzzzzz/Fish-tank
**Stack:** Go + Ebiten (GPU/OpenGL) · GLM `glm-5.3-flash` via Coding Plan endpoint · zero other deps

---

## Quick start (play in 60 seconds)

**Just play — no tools needed:** open
[github.com/Rollingzzzzz/Fish-tank/releases/latest](https://github.com/Rollingzzzzz/Fish-tank/releases/latest)
→ download `NEON-TANK-v1.zip` (or the bare `tank.exe`) → unzip anywhere → double-click
`tank.exe`. The tank is portable: saves, config and agent content all live next to the exe —
delete the folder to uninstall, nothing is written outside it. *(The exe is unsigned: Windows
SmartScreen → "More info" → "Run anyway".)*

**Or build from source** — Go 1.27+, no other dependencies:

```bash
git clone https://github.com/Rollingzzzzz/Fish-tank.git
cd Fish-tank
go build -ldflags "-H windowsgui" -o tank.exe ./cmd/tank
./tank.exe
```

**While playing:** left click = drop food · right screen edge = open/close the menu · TREATS
(top-right) = the live-treat store · fast mouse swipe = scatter fish · right-click = scare ·
`F` = debug overlay · `F11` = fullscreen · the **Setup** tab holds music, auto-feed, auto-care,
day length and the fish slider (up to 100). Without an API key the tank is fully playable in
procedural simulate mode; paste a GLM key in Setup to let live agents design species and plants
on screen. The full control table and options live in §9.

---

## 0. How to read this document

This README is the **single source of truth**. Every actionable line is a **Goal** with a unique ID
(`G0.1 … G7.3`). Goals are written so that **multiple agents can implement them in parallel** with
minimal error probability:

- Each goal lists the **exact files it owns** — an agent must never touch anything else.
- Each goal lists its **dependencies** — do not start before they are done.
- Each goal has **measurable acceptance criteria** — a goal is done only when every criterion is true.

---

## 1. Prime Directives (never violate)

| # | Directive |
|---|---|
| **D1** | If code and this README conflict, **the code is wrong**. Fix the code, not the README. |
| **D2** | An agent edits **only** the files in its goal's `Owns` field (plus new files inside its own package directory). |
| **D3** | `internal/contract` is **frozen** after G0.2. Changing a shared type requires changing this README first. |
| **D4** | **Never crash.** Bad JSON, missing files, no network, failed shader compile → graceful degradation (log + fallback), always. |
| **D5** | **Determinism:** all randomness goes through seeded RNG (`contract.RandSeed`); reloading a save reproduces the same world. |
| **D6** | **Language:** everything in English — identifiers, comments, UI strings, JSON field names, agent-generated content names. No i18n layer in v0.1. **No emoji in UI** — icons are drawn shapes or plain text (Ebiten cannot render color emoji). |
| **D7** | **Offline-first:** with an empty API key the app is 100% playable via simulate mode (panel flow identical, tagged `SIMULATED`). |
| **D8** | **Dependencies:** `ebiten` (incl. `ebiten/audio`) + Go stdlib only. Any other module requires a README change first. |
| **D9** | **Never commit secrets.** `config.json`, `save_*.json`, `tank.lock`, `*.zip` are git-ignored. |

---

## 2. Conventions (binding — same authority as the directives)

### C1 — Location: everything lives under `D:\`
- Dev repo: `D:\projeler\fishTank\`. Portable install: `D:\FishTank\` (copy of `tank.exe` + its folders).
- The exe is **self-contained**: everything it creates (`content/`, `config.json`, `save_*.json`,
  `tank.lock`, exported packs) lives **beside the exe**. App root is resolved via `os.Executable()`.
- **No absolute `C:\` paths, no registry, no user-profile writes.** Uninstall = delete the folder.
  Copy the folder to any `D:\` machine → it works.
- Only allowed external read: `%USERPROFILE%\.zcode\v2\*` (key import, G3.5, read-only, on request).

### C2 — One job, one Go program
- Each functional unit is its **own Go package with a single responsibility**. Sideways imports are
  forbidden; imports only point downward:
  ```
  game → {ui, sim, agents, render, audio, content, config}
  agents → {llm, content, contract}     ui → {render, content, contract}
  sim / render / audio → {contract}     everyone → {contract}
  ```
  Any import outside this graph is an architecture violation — reject the merge.
- Every major package ships a **standalone demo binary** that proves it works without the full game:
  ```
  cmd/demo-shader   → animated water shader          cmd/demo-fish   → gallery of species/patterns
  cmd/demo-plants   → plant designs, day/night       cmd/demo-audio  → ambient + event SFX
  cmd/demo-agents   → 3 agents vs fake LLM server    cmd/demo-ui     → menu, tabs, widgets
  cmd/demo-world    → full sim, no UI
  ```
  Each demo starts in ≤2 s and is the **acceptance tool** for its lane. Demos accept `--png <out>`
  to dump a frame — a developer self-review tool (the player is never asked to judge art; see §3
  Art direction).

### C3 — Memory never overflows (both kinds)

**App memory (runtime):**
- Hard caps, enforced in code: fish ≤ `MaxFish`, food ≤ 64, sparkles ≤ 256, bubbles ≤ 96,
  tank-log entries ≤ 200, content files ≤ 512/folder, registry entries ≤ 10 000 (oldest dropped),
  UI log buffer ≤ 500 lines. Pools preallocated.
- **Zero steady-state allocation** in hot paths (update + draw). Debug overlay (`F`) shows heap:
  24 h soak target: heap drift < 10 %. Bounded or it doesn't merge.

**Agent context (glm-5.3-flash) memory:**
- **File ceiling ~300 lines.** A file that wants to grow gets split by responsibility first.
- **Package summary law:** every package starts with `doc.go` — a compact 5–15 line block:
  *purpose / owns / public API / invariants / extension points*. Every file opens with a 1–3 line
  header comment: what it does + which goal created it (`// G2.1: ...`).
- `docs/NOTES.md`: running decision log. One short block per merged goal
  (`### G2.1 — <title>` / what / why / gotchas / follow-ups). Append-only, never delete.
- **Continuation test:** a fresh glm-5.3-flash agent must be able to resume any task by reading only
  (1) its goal row in this README, (2) the package's `doc.go`, (3) the last 10 `NOTES.md` blocks —
  never the whole codebase. If that's not enough, the notes are insufficient: fix the notes.

### C4 — Extension-first (glm-5.3-flash friendly)
- All GLM prompts live as **named constants in one file** (`internal/agents/prompts.go`) —
  tuning AI behavior never touches logic.
- Content is **data, not code**: new fish/plant/water/pattern = new JSON file. Agents and humans
  extend the app the same way; a new feature kind = new goal row + new package file, never a
  refactor of an old one.
- Options-struct style public APIs so additions never break callers.
- Every tunable number lives in `contract/tuning.go` or `Config`. No magic numbers in logic files.

### C5 — Work hygiene
- One goal = one branch = one commit, message `Gx.y: summary (lane X)`; the same commit appends its
  `NOTES.md` block. Done = `go build ./... && go vet ./... && go test ./...` green **+ the goal's
  demo binary runs + every Accept bullet true**.

### C6 — Frozen decisions (closed topics — implement, never re-litigate)
- **Model:** `glm-5.3-flash` at the Coding Plan endpoint (`https://api.z.ai/api/coding/paas/v4/chat/completions`).
  CLOSED. Settings renders the model read-only; no UI to change it (the `Config.model` field exists
  but its default is fixed and documented as locked).
- **Thinking max:** every GLM request includes `"thinking": {"type": "enabled"}` — maximum reasoning
  the endpoint allows. `reasoning_content` deltas stream to the agent card as a separate, dimmed
  *thinking* line (part of the show) and are **never** mixed into the JSON buffer.
- **Pattern uniqueness gate:** every accepted artifact carries a **fingerprint** = SHA-256 (first
  16 hex) over: pattern type + density/size quantized to 0.05 buckets + palette hues quantized to
  10° buckets. `content/registry.json` is append-only (hash, kind, id, timestamp), capped at
  10 000 entries (oldest dropped). A fingerprint already registered ⇒ artifact **rejected** → one
  automatic repair round-trip ("these fingerprints exist; produce something sufficiently different")
  ⇒ still duplicate ⇒ the simulate generator produces a guaranteed-different artifact (it checks
  the registry too). Species names must be unique case-insensitively. Pack import merges registries.

---

## 3. Architecture & parallel lanes

```
cmd/tank/main.go                 → boots game.Game
internal/
  contract/    ← G0.2  FROZEN shared types, JSON schemas, tuning constants
  config/      ← G6.2  config.json + ZCode key import + single-instance lock
  content/     ← G3.1  scan/validate/atomic-write content/ + seed content + zip packs + registry
  llm/         ← G3.2  GLM SSE streaming client (thinking max)
  agents/      ← G3.3–G3.6  agent framework, Species/Water/Pattern agents, simulate generators
  sim/         ← G2.x  world simulation: fish, food, eggs, plants, particles, aging, death, care
  render/      ← G1.x  Kage shaders, glow sprites, trail buffer, fish/plant/scene rendering
  audio/       ← G1.5  sound engine: ambient + event SFX (offline-synthesized assets)
  ui/          ← G5.x  widget kit + right slide-in menu (tabs)
  game/        ← G6.x  Ebiten loop wiring everything, input, persistence
content/                              ← runtime content (exe-beside folder)
  species/*.json water/*.json plants/*.json patterns/*.json registry.json
config.json  save_minute.json  save_daily.json  save_weekly.json  tank.lock
```

**Parallel map** (conflict-free by file ownership):

```
G0.1 ─ G0.2 ─┬─ Lane A (render+sim+audio): G1.1 → G1.2 ┬ G1.3 ┬ G1.4 ┬ G1.5 → G2.1 → G2.2 → G2.3 → G2.4 → G2.5
             ├─ Lane B (content+ai):      G3.1 → G3.2 ┬ G3.3 ┬ G3.6 → G3.4 → G3.5
             └─ Lane C (ui):              G5.1 → G5.2 → G5.3
                        all lanes ──→ G6.1 → G6.2 → G7.1 → G7.2 → G7.3
```

Three agents can work **simultaneously** after G0.2: one per lane. Merge order: A, B, C → G6.
Every lane goal also ships/updates its package's `doc.go`; a lane goal is only *done* when its
`cmd/demo-*` binary (per C2) runs cleanly.

### Art direction — glowy, slightly alien, fun to watch

Reference feel: the [“Wavy Plant” Rive loop](https://www.instagram.com/reel/DMEskyfRFkM/) —
"so smooth and full of life". These rules are **acceptance criteria inside G1.1/G1.3/G1.4**:

- **Glow is the default.** Every emissive element renders with an additive bloom halo; night
  intensifies glow; the dark deep-water backdrop makes neons pop.
- **Slightly alien forms.** Spiral-frond plants, bubble-tipped tendrils, drifting jelly orbs,
  exotic fin proportions (agents invent species anyway); drifting sparkle particles everywhere.
- **Fun motion.** Everything eased — no linear movement; soft pulsing; ripples through schools;
  light bursts when food drops; fish show cursor curiosity. Plants sway with layered phase-offset
  sines. Multi-stop gradients only — no flat fills.
- Visual sign-off is **not** part of the loop: demos expose `--png <out>` and G7.2's polish
  pass self-reviews frames against these rules (frames committed under `docs/art/`).

---

## 4. Frozen contracts (created by G0.2, exact — no deviation allowed)

```go
package contract // internal/contract/contract.go

type Vec2 struct{ X, Y float64 }

type Palette struct {
    Body   string `json:"body"`   // "#rrggbb"
    Belly  string `json:"belly"`  // "#rrggbb"
    Accent string `json:"accent"` // "#rrggbb"
    Glow   string `json:"glow"`   // "#rrggbb"
}

type Pattern struct {
    Type    string  `json:"type"`    // "stripe" | "spot" | "koi" | "vein" | "wave"
    Density float64 `json:"density"` // 0..1
    Size    float64 `json:"size"`    // 0..1
}

type Behavior struct {
    Speed       float64 `json:"speed"`     // 0.5..1.6
    Schooling   float64 `json:"schooling"` // 0..1
    Curiosity   float64 `json:"curiosity"` // 0..1
    Skittish    float64 `json:"skittish"`  // 0..1
    Depth       float64 `json:"depth"`     // 0..1  (0 surface .. 1 floor)
    NightActive bool    `json:"nightActive"`
}

type Species struct {
    ID        string   `json:"id"`        // slug: ^[a-z0-9-]{3,32}$, == filename stem
    Name      string   `json:"name"`      // display name (English), unique case-insensitively
    Latin     string   `json:"latin"`
    Size      float64  `json:"size"`      // 0.6..1.4
    Width     float64  `json:"width"`     // 0.7..1.3
    Fin       float64  `json:"fin"`       // 0.6..1.5
    Tail      float64  `json:"tail"`      // 0.7..1.4
    Palette   Palette  `json:"palette"`
    Pattern   Pattern  `json:"pattern"`
    Behavior  Behavior `json:"behavior"`
    Note      string   `json:"note"`      // one sentence
    Source    string   `json:"source"`    // "core" | "species-agent" | "water-agent" | "pattern-agent" | "pack" | "user"
    CreatedAt string   `json:"createdAt"` // RFC3339 or ""
}

type WaterPreset struct {
    ID           string      `json:"id"`
    Name         string      `json:"name"`
    TopColor     string      `json:"topColor"`
    BottomColor  string      `json:"bottomColor"`
    Accent       string      `json:"accent"`
    Rays         float64     `json:"rays"`       // 0..1
    Caustics     float64     `json:"caustics"`   // 0..1
    Bubbles      float64     `json:"bubbles"`    // 0..1
    PlantPalette []string    `json:"plantPalette"` // 2..4 hex
    Event        *WaterEvent `json:"event,omitempty"`
    Source       string      `json:"source"`
}

type WaterEvent struct {
    Name     string  `json:"name"`
    Kind     string  `json:"kind"`     // "bubbleStorm" | "glowWave" | "current" | "calm"
    Duration float64 `json:"durationSec"` // 15..40
    Note     string  `json:"note"`
}

type PlantDesign struct {
    ID     string   `json:"id"`
    Name   string   `json:"name"`
    Fronds int      `json:"fronds"`  // 3..8
    Height float64  `json:"height"`  // 0.15..0.55 (fraction of tank height)
    Width  float64  `json:"width"`   // 0.5..1.5 multiplier
    Curve  float64  `json:"curve"`   // 0..1
    Sway   float64  `json:"sway"`    // 0..1
    Colors []string `json:"colors"`  // 2..4 hex (base→tip gradient)
    Glow   float64  `json:"glow"`    // 0..1
    Source string   `json:"source"`
}

type PatternRecipe struct { // used by the Pattern agent for age-stage repaints
    ID       string  `json:"id"`
    Stage    string  `json:"stage"`     // "fry" | "juvenile" | "adult" | "elder"
    Name     string  `json:"name"`
    SatMul   float64 `json:"satMul"`    // 0.3..1.4 saturation multiplier
    AlphaMul float64 `json:"alphaMul"`  // 0.4..1.0
    GlowMul  float64 `json:"glowMul"`   // 0.5..2.0
    Pattern  Pattern `json:"pattern"`
    Note     string  `json:"note"`
}

type Config struct {
    Endpoint   string  `json:"endpoint"`  // default: https://api.z.ai/api/coding/paas/v4/chat/completions
    Model      string  `json:"model"`     // FROZEN: glm-5.3-flash (C6) — read-only in Settings
    APIKey     string  `json:"apiKey"`    // "" → simulate mode
    ProxyURL   string  `json:"proxy,omitempty"`
    AutoFeed   bool    `json:"autoFeed"`
    AutoCare   bool    `json:"autoCare"`
    SoundOn    bool    `json:"soundOn"`
    AgentFreq  float64 `json:"agentFreq"`  // 0.5..2, default 1
    DaySeconds float64 `json:"daySeconds"` // 20..180, default 60
    MaxFish    int     `json:"maxFish"`    // 8..100 (v0.3.7 F29), default 60 (v1: the fresh tank opens with 60 live fish)
}

type SavedFish struct {
    SpeciesID string  `json:"speciesId"`
    Seed      int64   `json:"seed"`
    AgeDays   float64 `json:"ageDays"`
    Pos       Vec2    `json:"pos"`    // live position
    Vel       Vec2    `json:"vel"`    // live velocity
    Satiety   float64 `json:"satiety"` // 1 = full belly
    Energy    float64 `json:"energy"`
}

type SavedFood struct {
    Pos Vec2    `json:"pos"`
    Age float64 `json:"age"`
}

type SavedEgg struct {
    SpeciesID string `json:"speciesId"`
    Pos       Vec2   `json:"pos"`
    Progress  float64 `json:"progress"` // 0..1 toward hatch
}

type Save struct {
    SchemaVersion int         `json:"schemaVersion"` // == SaveSchemaVersion; > current → refuse load (log + fresh world)
    Fish          []SavedFish `json:"fish"`
    Foods         []SavedFood `json:"foods"`
    Eggs          []SavedEgg  `json:"eggs"`
    Clock         float64     `json:"clock"`    // day-cycle seconds
    Care          float64     `json:"care"`
    WaterID       string      `json:"waterId"`
    PlantIDs      []string    `json:"plantIds"`
}
```

**G0.3 (coordinator amendment) — agent→UI event bus and world hooks** (`contract/events.go`):

```go
type EventKind string // "thought" | "chunk" | "status" | "artifact" | "log"
type AgentStatus string // "idle" | "thinking" | "writing" | "done" | "error" | "simulated"

type Event struct {
    Kind       EventKind
    Agent      string      // contract.AgentSpecies | AgentWater | AgentPattern
    Status     AgentStatus // valid when Kind == "status"
    Text       string      // streaming text / log line / artifact summary
    ArtifactID string      // content id, when Kind == "artifact"
    At         string      // RFC3339
}

type PatternRequest struct { // one fish waiting for an age repaint
    FishID  string
    Species Species
    Stage   string
    OldPal  Palette
    OldPat  Pattern
}

type AgentHooks struct { // wired by game (G6.1); nil-check before use
    SpeciesNames       func() []string        // existing names (prompt context)
    RecentFingerprints func(n int) []string   // last n registry hashes (prompt context)
    NextRepaints       func(max int) []PatternRequest
    ApplyRecipe        func(fishID string, recipe PatternRecipe)
    SpawnEgg           func(speciesID string)
}
```

**Frozen tuning constants** (in `contract/tuning.go`, agents must use these, not magic numbers):

```go
SpineSegments  = 14
StageDays      = map[string]float64{"fry": 0, "juvenile": 2, "adult": 5, "elder": 12}
StageSpeedMul  = map[string]float64{"fry": 0.8, "juvenile": 0.95, "adult": 1.0, "elder": 0.6}
BaseSpeed      = 72.0  // px/s × Behavior.Speed × stage/night multipliers
MaxForce       = 240.0 // px/s²
SchoolRadius   = 110.0
FoodSense      = 150.0
EggHatchSec    = 6.0
CareWatchTick  = 10.0                       // seconds of mouse activity → +1 care
CareFeedScore  = 2.0                        // per flake eaten
CareThresholds = []float64{10, 25, 50, 80}  // → breeding event (courtship ring → 1-2 eggs)
AgentSchedules = map[string]float64{"species": 140, "water": 110, "pattern": 80} // seconds × AgentFreq
AgentFirstRun  = map[string]float64{"species": 3, "water": 6, "pattern": 9}      // seconds after boot
SaveMinuteSec  = 60.0                       // rolling snapshot cadence
MinFishCount   = 2                          // death never drops the tank below this
DeathAfterElderDays = 4.0                  // elder + this many days → fade out over 60 s
SaveSchemaVersion = 1                      // bump on any Save shape change; see Save.SchemaVersion
MaxOutputTokens  = 1600                    // GLM max_tokens cap: bounds every agent response
```

---

## 5. Goal board

### G0 — Foundation (SERIAL — blocks every lane)

**G0.1 Module bootstrap**
- Owns: `go.mod`, `cmd/tank/main.go`, `.gitignore`, `docs/NOTES.md`
- Depends: —
- Goal: init module `github.com/Rollingzzzzz/Fish-tank`; add ebiten (latest stable, version pinned in NOTES); `main.go` opens a 1280×720 window titled `NEON TANK` with a solid dark background; `.gitignore` covers `config.json`, `save_*.json`, `tank.lock`, `*.zip`, `*.exe`; `docs/NOTES.md` created with the block template from C3.
- Accept: `go build ./...` succeeds; window opens and closes cleanly; `git status` shows no secret-ish files tracked; `docs/NOTES.md` template exists.

**G0.2 Frozen contracts**
- Owns: `internal/contract/*` (and nothing else)
- Depends: G0.1
- Goal: create every type and constant from §4 **verbatim** (field names, JSON tags, ranges as doc comments) + `func Clamp(v, lo, hi float64) float64`, `func RandSeed(seed int64) *rand.Rand`, `func Slugify(s string) string`, `func Fingerprint(kind, canonical string) string` (SHA-256, first 16 hex). Package has a `doc.go` per C3.
- Accept: `go vet ./...` clean; package has zero imports outside stdlib; every range from §4 appears as a documented constant or comment.

### Lane A — GPU rendering, simulation, audio

**G1.1 Background shader (GPU)**
- Owns: `internal/render/background.go`, `internal/render/shaders/*.kage`, `cmd/demo-shader/`
- Depends: G0.2
- Goal: fullscreen Kage fragment shader: animated vertical water gradient (`BottomColor→TopColor`), moving caustics (∝ `Caustics`), volumetric god rays from top (∝ `Rays`), drifting bioluminescent plankton glints, night darkening + accent glow boost, vignette. Uniforms: `time, dayFactor, topColor, bottomColor, accent, rays, caustics`. **Fallback:** if shader fails to compile, draw a plain 2-color gradient rect — never crash (D4). Art direction §3 applies (eased drift, multi-stop gradient feel).
- Accept: `cmd/demo-shader` shows visibly animated water; toggling `dayFactor 0↔1` darkens the scene and boosts glints; 60 FPS at 1920×1080 (debug overlay shows FPS); `--png` frame committed under `docs/art/`.

**G1.2 Glow sprites + phosphor trail buffer**
- Owns: `internal/render/glow.go`, `internal/render/trail.go`
- Depends: G0.2
- Goal: cache tinted radial-gradient glow sprites per color (additive draw helper `DrawGlow(dst, pos, radius, color, alpha)`); offscreen trail buffer that fades ~10%/frame instead of clearing, blitted additively onto the scene → phosphor motion trails.
- Accept: demo path shows an orbiting dot leaving a fading neon trail; glow color cache reused (no per-frame allocation in hot path).

**G1.3 Plant renderer**
- Owns: `internal/render/plant.go`, `cmd/demo-plants/`
- Depends: G0.2, G1.2
- Goal: render `PlantDesign` as ribbon fronds (bezier spine per frond, base→tip gradient through `Colors`, width taper, sway = layered phase-offset sines ∝ `Sway`, bend ∝ `Curve`), neon rim stroke, glow edge intensified at night ∝ `Glow`. Art direction §3: alien silhouettes welcome (spiral fronds, bubble-tipped tendrils via `Curve`/`Sway` extremes).
- Accept: `cmd/demo-plants` renders 5 seed designs, visually distinct; sway is continuous and eased; night mode visibly increases glow; `--png` frame committed.

**G1.4 Fish renderer (spine body)**
- Owns: `internal/render/fish.go`, `cmd/demo-fish/`
- Depends: G0.2, G1.2
- Goal: `DrawFish(dst, glowDst, spine []contract.Vec2, spec *Species, pal *contract.Palette, stage string, night float64, anim Phase)` — tapered body polygon from spine widths (`Width`, `Size`), multi-stop gradient fill `Body→Belly`, dorsal/pectoral/tail fins ∝ `Fin`/`Tail` with speed-driven eased swish, pattern overlay clipped to body per `Pattern.Type` (all 5 types), eye with glint, accent rim + glow sprite on head/tail (night ∝ bioluminescence). Stage look: `fry` translucent/smaller, `elder` desaturated via `PatternRecipe` multipliers.
- Accept: `cmd/demo-fish` renders one fish per pattern type — 5 clearly different fish; fry/elder variants visibly differ from adult; `--png` frame committed.

**G1.5 Audio engine + offline sound synthesis**
- Owns: `internal/audio/*`, `scripts/gen-audio.go`, `assets/audio/*.wav` (generated, committed), `cmd/demo-audio/`
- Depends: G0.2
- Goal: `scripts/gen-audio.go` synthesizes every sound offline via additive synthesis (ambient underwater loop, food plop, egg crack, agent ping) → `assets/audio/`; runtime mixer plays the ambient loop + event SFX; global mute/volume via `Config.soundOn`. `ebiten/audio` only (D8).
- Accept: `cmd/demo-audio` plays ambient and triggers each SFX; `go run ./scripts/gen-audio` output is deterministic (byte-identical regeneration); muting silences everything.

**G2.1 Fish entity + steering**
- Owns: `internal/sim/fish.go`, `cmd/demo-world/` (shared)
- Depends: G0.2
- Goal: spine chain (14 segments, follow + sinusoidal wave ∝ speed), steering: wander, boids school/separation/alignment/cohesion (weights ∝ `Behavior.Schooling`), food seek (∝ hunger), cursor curiosity (∝ `Curiosity`), flee impulse (fast mouse swipe / ∝ `Skittish`), rest at plants when energy low, wall margin, depth-band preference, night multiplier for `NightActive` species. **Rich interactions:** play-chase (a larger fish briefly follows a smaller one; both dart apart), night gliding (schools drift slowly, undulation dominates). All constants from `contract/tuning.go`.
- Accept: unit tests: (a) school cohesion decreases average pairwise distance over 600 ticks; (b) segment lengths stay within 1e-6 of `segLen`; (c) `MaxSpeed` never exceeded.

**G2.2 Food, eggs, particles**
- Owns: `internal/sim/food.go`, `internal/sim/eggs.go`, `internal/sim/particles.go`
- Depends: G0.2, G2.1
- Goal: flakes (sink 22 px/s, wobble, floor-rest, 25 s ttl), eating when flake within mouth radius (hunger→0, sparkle burst, tiny growth), eggs (pulse, glow, hatch after `EggHatchSec` into 1–2 fry, capped by `MaxFish`), bubbles ∝ `Bubbles`, click ripple with light burst (art direction §3).
- Accept: unit test: eating consumes flake and zeroes hunger; egg hatches exactly once at `EggHatchSec`; spawns respect `MaxFish`.

**G2.3 Aging, day/night, care, natural death**
- Owns: `internal/sim/life.go`
- Depends: G2.1
- Goal: `AgeDays += dt / DaySeconds`; stage from `StageDays`; on stage transition → push repaint request `{fishID, stage}` to the shared repaint queue (thread-safe). Day/night clock (cycle = `2 × DaySeconds`, smooth 20% transitions). Care score: `+1` per 10 s of active mouse, `+2` per flake eaten; crossing `CareThresholds` → **courtship**: two conspecifics circle each other ~4 s, then 1–2 eggs spawn between them. **Natural death:** `AgeDays > elder + DeathAfterElderDays` → fish fades out over 60 s (alpha→0) and is removed; farewell note to the tank log; death is skipped if it would drop the tank below `MinFishCount`.
- Accept: unit tests: stage transitions fire exactly once per boundary; care thresholds trigger courtship then the configured egg counts; death fires once, respects `MinFishCount`; clock wraps correctly.

**G2.4 World orchestration + save/load**
- Owns: `internal/sim/world.go`
- Depends: G2.1–G2.3
- Goal: `World` owns fishes/food/eggs/plants/particles; `Update(dt)`; `ApplyWater(preset)` crossfades uniforms & bubble rate; `Snapshot() Save` / `Restore(Save)` full live state (positions, velocities, satiety, energy, foods, eggs, clock, care, water, plants).
- Accept: round-trip test: snapshot → mutate → restore returns identical census, positions (±1e-6), satiety, foods, eggs, care, water; dt clamp (≤50 ms) prevents tunneling on hitch.

**G2.5 Save tiers & crash recovery**
- Owns: `internal/sim/persist.go`
- Depends: G2.4
- Goal: exactly **three rotating snapshot files** beside the exe, same schema: `save_minute.json` (every `SaveMinuteSec` **+ immediately** on big events: egg arrival, hatch, death, water preset change, app exit), `save_daily.json` (on calendar-day change), `save_weekly.json` (every 7 days). All writes atomic (tmp+rename). **Recovery:** newest valid file by mtime, order minute→daily→weekly; skip files with `schemaVersion > SaveSchemaVersion`; all invalid → fresh world (never crash, log everything).
- Accept: tests: corrupt `save_minute.json` → recovery falls back to daily; simulated kill -9 mid-run → restart loses at most 60 s of state; calendar rollover triggers the daily snapshot exactly once; high-version file is skipped, not fatal.

### Lane B — Content, LLM, agents

**G3.1 Content store**
- Owns: `internal/content/store.go`, `internal/content/seed.go`, `content/**/*.json` (seed files)
- Depends: G0.2
- Goal: scan `content/{species,water,plants,patterns}/*.json`; validate against contract; **clamp** every float to its range (never reject for range, reject only for unparseable JSON / bad hex / bad ID slug — log line per skipped file); atomic write (`tmp`+rename); `EnsureSeed()` writes embedded core content when missing: **4 species, 3 water presets, 4 plant designs, 4 pattern recipes** (one per stage), all `source:"core"`, all styled per §3 art direction. Pack export/import: zip the `content/` tree (`*.zip`) / extract+validate into it, merging `registry.json` (dedupe by hash).
- Accept: unit tests with temp dir: invalid JSON skipped + logged; clamping works; `EnsureSeed` idempotent; export→import round-trip preserves file set; registry merge dedupes.

**G3.2 GLM streaming client**
- Owns: `internal/llm/glm.go`
- Depends: G0.2
- Goal: `Stream(ctx, req Request, onDelta func(string), onThought func(string)) (full string, err error)` — POST `{Endpoint}` with `Authorization: Bearer`, body `{model, messages, stream:true, temperature:0.9, max_tokens: MaxOutputTokens, thinking:{type:"enabled"}}` (model + thinking frozen per C6). Parse SSE lines (`data:` prefix, `[DONE]`); 90 s timeout; optional `ProxyURL` prefix. `reasoning_content` deltas route **only** to `onThought`. Typed errors: `ErrAuth` (401/403), `ErrQuota` (429), `ErrNetwork` (transport), `ErrHTTP` (other).
- Accept: `httptest` fixture streams 3 content chunks + 1 `reasoning_content` delta → content assembled in order, thought routed separately; `[DONE]` terminates; each error kind maps correctly.

**G3.3 Simulate generators (offline mode)**
- Owns: `internal/agents/simulate.go`
- Depends: G0.2
- Goal: procedural generators producing contract-valid `Species` / `WaterPreset` / `PlantDesign` / `PatternRecipe` + a short English description — neon palettes from HSL random walks (art direction §3: multi-stop gradient feel), English name syllable combiner, guaranteed-unique IDs and names. Deterministic per seed; registry-aware (never emits a registered fingerprint).
- Accept: property test: 200 generated artifacts all pass content-store validation and the uniqueness gate.

**G3.6 Uniqueness registry & acceptance gate**
- Owns: `internal/content/registry.go`, `internal/content/fingerprint.go`
- Depends: G3.1
- Goal: fingerprint per C6; registry read/write with 10k cap (append-only, oldest dropped); `Gate(a Artifact) (ok bool, conflicts []string)` consulted by every agent before writing; case-insensitive name uniqueness; registry merge on pack import.
- Accept: unit tests: 2° hue shift → duplicate rejected; 20° shift → accepted; entry 10 001 drops the oldest; merge dedupes identical hashes.

**G3.4 Agent framework + 3 agents**
- Owns: `internal/agents/agent.go`, `internal/agents/species_agent.go`, `internal/agents/water_agent.go`, `internal/agents/pattern_agent.go`, `internal/agents/prompts.go`, `cmd/demo-agents/`
- Depends: G3.1, G3.2, G3.3, G3.6
- Goal: `Agent` base (goroutine, schedule ∝ `AgentSchedules`×`AgentFreq`, first run at `AgentFirstRun`, statuses: `idle|thinking|writing|done|error|simulated`) emitting UI events on a channel: `{kind: thought|chunk|status|artifact|log, agentID, text, artifactID}`.
  - **Species agent:** on run (or the **"Research New Species"** button): streams its plan (thinking line + text), then emits a species JSON → uniqueness gate → validated → written to `content/species/` → emits egg-spawn request. Prompt contract: *"Return ONLY an AgentOutput JSON matching the schema; English name; neon palette; concept sufficiently different from the existing list"* (existing names + recent fingerprints passed in context).
  - **Water agent:** water preset + 1 new plant design per run → written to `content/water/` + `content/plants/` → world applies preset with crossfade + event.
  - **Pattern agent:** consumes ≤2 repaint-queue items per run (age-appropriate `PatternRecipe` → applies to fish, emits before/after preview event); if queue empty → random mutation of one fish. Also writes the recipe to `content/patterns/`.
  - **Repair rule:** unparsable LLM output → one repair round-trip ("Return only valid JSON"); still bad → `error` status + log. **Uniqueness rule:** duplicate fingerprint → one repair round-trip listing the conflicts; still duplicate → simulate generator takes over. **Simulate fallback:** `APIKey==""` OR `ErrNetwork` → permanent simulate mode (until settings change) with `SIMULATED` tag, identical panel flow. **Quota guard:** `ErrQuota` → temporary simulate mode for all agents; silent real-mode retry every 10 min; automatic switch back + log entry.
- Accept: `cmd/demo-agents` (fake LLM server): all 3 agents run concurrently; artifacts land in temp `content/`; event sequence thought→chunk→status→artifact observed; duplicate-fingerprint response triggers repair then gate rejection; server returning 429 flips agents to temporary simulate and back after the retry window.

**G3.5 ZCode key import**
- Owns: `internal/config/zcode.go`
- Depends: G0.2
- Goal: scan `%USERPROFILE%/.zcode/v2/` (`config.json`, `credentials.json`) for the GLM key (document the exact JSON path found during implementation); return `(key, found)`; **never log the full key** (mask: only last 4 chars).
- Accept: unit test with fixture file in temp dir; no real key string appears in repo or logs.

### Lane C — UI

**G5.1 Widget kit**
- Owns: `internal/ui/widget.go`, `internal/ui/textinput.go`, `internal/ui/scrolltext.go`, `internal/ui/theme.go`, `cmd/demo-ui/`
- Depends: G0.2, G1.2
- Goal: immediate-mode, hit-tested widgets on Ebiten: `Panel`, `Button`, `Toggle`, `Slider`, `TextInput` (masked option for key), `ScrollText` (streaming, word-wrapped, auto-scroll, dimmed thought lines), `TabBar`. Neon-glass theme (dark glass, cyan/magenta accents, Latin font embedded via `go:embed`). No emoji (D6) — icons are drawn shapes.
- Accept: `cmd/demo-ui` renders all widgets; text input works; ScrollText streams without flicker; `--png` frame committed.

**G5.2 Slide-in menu (immersion rule)**
- Owns: `internal/ui/menu.go`
- Depends: G5.1
- Goal: right-edge handle (≤8 px visible when closed); click → 0.25 s eased slide to a 380 px panel; tabs: `Agents / Species / Plants / Log / Settings`. **When closed, NOTHING else is drawn — the tank is pixel-pure.** Agent activity shows as a small pulsing dot on the handle. Menu never resizes/shifts the tank.
- Accept: closed state renders only the handle; open/close animation smooth and eased; no state leaks between tabs.

**G5.3 Tab screens**
- Owns: `internal/ui/tab_agents.go`, `internal/ui/tab_catalog.go`, `internal/ui/tab_plants.go`, `internal/ui/tab_log.go`, `internal/ui/tab_settings.go`
- Depends: G5.2, G3.4 (events), G3.1 (catalogs)
- Goal: **Agents:** 3 live cards (name, status badge, dimmed thinking line, streaming text, artifact preview: species card with palette swatches + mini fish render; water swatches; pattern before/after) + **Run All** + per-agent ▶ + **"Research New Species"**. **Species/Plants:** catalog grids from the content store with mini previews (render package), click to apply/select. **Log:** shared tank log. **Settings:** form bound to `Config` (endpoint, model read-only per C6, key, proxy, autoFeed, autoCare, soundOn, agent frequency, day length, max fish), **"Import key from ZCode"**, **"Export Pack" / "Import Pack"**, **"Reset Tank"**.
- Accept: every button performs its real action through its owning package; settings round-trip to `config.json`; model field is read-only; no tab crashes with an empty content folder.

### G6 — Wiring (after lanes merge)

**G6.1 Game loop**
- Owns: `internal/game/game.go`, `internal/game/input.go`, `cmd/tank/main.go` (final)
- Depends: G2.5, G3.4, G5.3
- Goal: `game.Game` owns World + Menu + AgentHub; input: left click = drop food, mouse move = watch/care activity, fast swipe = flee impulse; `F` toggles the debug overlay (FPS + heap); `F11` toggles fullscreen; fixed-clamp timestep; clean shutdown hook (immediate minute-snapshot on exit).
- Accept: full loop runs ≥10 min without leak (allocation stable in profile); input behaves per spec; window size/position restored between runs.

**G6.2 First-run, single-instance, persistence**
- Owns: `internal/game/bootstrap.go`, `internal/config/config.go`
- Depends: G6.1, G3.5
- Goal: first run: create `config.json` defaults → `EnsureSeed()` → if no key and ZCode key found → show **"Import key from ZCode"** prompt. **Single-instance lock:** `tank.lock` beside the exe (PID inside); second launch shows "Tank is already running" and exits; a stale lock (PID no longer alive) is broken with a warning. Persistence handled by G2.5 tiers; window size/position stored in `config.json`.
- Accept: restart restores the world via the tier system; deleting `content/` re-seeds; no-key first run goes straight to playable simulate mode; double-launch is prevented; stale lock self-heals.

### G7 — Ship

**G7.1 Build & convention check**
- Owns: `README.md` badges/usage section, `scripts/check.go`
- Depends: G6.2
- Goal: `go build -ldflags "-H windowsgui" -o tank.exe ./cmd/tank`; `go vet ./... && go test ./...` green; `scripts/check.go` verifies conventions automatically: every package has `doc.go`, no file exceeds ~300 lines, import graph matches C2; document build steps + controls in README.
- Accept: `tank.exe` runs double-clicked (no console window), offline, as a polished neon tank; `go run ./scripts/check` passes.

**G7.2 Visual polish pass + QA sweep**
- Owns: fixes across owned packages (only regressions), `docs/art/*.png`
- Depends: G7.1
- Goal: capture `--png` frames of every demo; self-review against the §3 Art direction rules (glow default, alien forms, eased fun motion, multi-stop gradients); tune shaders/palette/motion until every rule is visibly satisfied — commit the final frames under `docs/art/`. Then execute the full §8 checklist; fix and re-run until all pass. No external sign-off is required.
- Accept: `docs/art/` contains one frame per demo; every §8 item checked.

**G7.3 Release v0.1**
- Owns: GitHub Release artifacts, release notes in `docs/`
- Depends: G7.2
- Goal: tag `v0.1`; build `tank.exe` (windowsgui); bundle seed `content/` into `sample-pack.zip`; create a GitHub Release on this repo attaching **tank.exe + sample-pack.zip**; notes = highlights from `docs/NOTES.md` + SmartScreen guidance (*"More info → Run anyway"*; portable, keep the folder under `D:\FishTank\`; delete folder = uninstall).
- Accept: release page offers both artifacts; on a clean machine: download exe + pack → `D:\FishTank\` → Settings → Import Pack → offline play works; delete folder = uninstall (C1).

---

## 6. Parallel agent protocol

1. Read this README **fully** before any code. `internal/contract` is law (D3); C6 topics are closed.
2. Claim **one goal**; touch **only** its `Owns` files. Never edit another lane's package.
3. Missing an upstream goal? Implement against the contract with a stub in *your* package, marked `// TODO(Gx.y)` — do not borrow others' unfinished work.
4. Done means: `go build ./... && go vet ./... && go test ./...` all green **plus every Accept bullet true**, including the lane's `cmd/demo-*` binary running.
5. **Acceptance must be machine-checkable:** every Accept bullet maps to a command, a test, or a committed artifact (e.g., a `--png` frame). "Looks fine to me" is not acceptance — that is the low-babysitting guarantee.
6. Commit message format: `G2.1: fish entity + steering (lane A)` — the same commit appends the goal's block to `docs/NOTES.md` (C3/C5).
7. New dependency, contract change, or tuning change ⇒ README change request first (D3/D8).
8. Everything English (D6). Never print API keys (D9).
9. Before handing off, re-run the **continuation test** (C3): could a fresh glm-5.3-flash agent resume from README goal + `doc.go` + last NOTES blocks alone? If not, improve the notes — that is part of *done*.

---

## 7. Global definition of done

- [ ] All goals G0.1→G7.3 accepted
- [ ] `tank.exe` (~20–25 MB) runs offline on a clean Windows 10/11 machine, installed under `D:\` (C1)
- [ ] 3 agents stream concurrently; with no key the panel runs in `SIMULATED` mode, identical flow
- [ ] Agent thinking (`reasoning_content`) visibly streams dimmed; never enters JSON
- [ ] Uniqueness gate proven by tests: no registered fingerprint can be re-accepted
- [ ] `content/` beside the exe is the single source of fish/water/plant/pattern content; agents write there; packs export/import as zip with registry merge
- [ ] Closed menu = pixel-pure tank; right handle opens everything
- [ ] Restart restores the saved world; care-driven courtship/breeding works; aging repaints fire at stage transitions; natural death respects `MinFishCount`
- [ ] 3-tier save: process killed mid-run → restart loses ≤ 60 s; disk always holds exactly 3 snapshots
- [ ] Quota exhaustion (429) → temporary simulate mode → silent auto-return
- [ ] Audio: ambient + event SFX offline; mute toggle works
- [ ] Conventions verified by `scripts/check.go`: `doc.go` everywhere, ≤300-line files, C2 import graph
- [ ] Every package provable in isolation via its `cmd/demo-*` binary; art frames under `docs/art/`
- [ ] 24 h soak: heap drift < 10 % (C3)
- [ ] Continuation test passes: a fresh glm-5.3-flash agent resumes work from README + `doc.go` + NOTES only (C3)
- [ ] `v0.1` GitHub Release published with `tank.exe` + `sample-pack.zip`

---

## 8. Manual QA checklist

- [ ] Boot with empty `content/` → seeds and plays; boot with no key → `SIMULATED` agents still produce species/water/plants as files
- [ ] **"Research New Species"** → thinking + streaming text → new file in `content/species/` → egg → hatch → fry swims
- [ ] Duplicate design attempt → gate rejects → repair → distinct result (registry grows)
- [ ] Click = food drop with light burst; fish race for it; eater sparkles; care score rises; threshold → courtship ring → eggs
- [ ] Fast mouse swipe near a school → scatter, then regroup; larger fish occasionally play-chases a smaller one
- [ ] Night: scene darkens, bioluminescence/trails intensify; `nightActive` fish speed up; schools glide
- [ ] Fish visibly age through 4 stages; Pattern agent repaints on transitions (before/after in panel); elder → fade-out death with farewell log note; tank never drops below 2 fish
- [ ] Water agent preset crossfades water + spawns plant design; plant catalog applies it
- [ ] Menu open/close: tank never shifts; closed = only handle; activity dot pulses on new agent output
- [ ] Settings: wrong key → `ErrAuth` shown, app keeps running; no key → simulate; model field read-only; pack export → zip appears; import → files + registry merge
- [ ] Kill network mid-run → agents degrade to simulate, no crash; 429 → temporary simulate → silent return
- [ ] Kill the process (kill -9 style) mid-run → restart → world restored, ≤60 s of state lost
- [ ] Sound: ambient loop plays; food/crack/ping SFX fire; mute toggle silences all
- [ ] Second exe launch → "Tank is already running"; stale lock self-heals
- [ ] Restart app → window size/position and full world restored

---

## 9. Build & run

```bash
go build -ldflags "-H windowsgui" -o tank.exe ./cmd/tank
./tank.exe          # double-click; everything lives next to the exe (D:\FishTank\)
```

**Controls**

| Input | Effect |
|---|---|
| Left click | drop food (manual feeding always works, even with auto-feed on) |
| Right edge click | open / close the side menu |
| Fast mouse swipe | scatter nearby fish |
| TREATS button (top-right) | open the treat store; click a live treat (bug / worm / shrimp / chicken), then click in the tank to drop it |
| F | debug overlay (FPS + heap) |
| F11 | fullscreen |

`-menu-smoke` runs the scripted UI acceptance (exit 0 = pass).

Menu tabs: **Agents** (live streaming cards + *Research New Species* + Run All), **Species** /
**Plants** catalogs (click to spawn egg / plant), **Log**, **Settings** (model read-only, key,
proxy, auto-feed, auto-care, sound, agent frequency, day length, max fish, *Import key from
ZCode*, *Export/Import Pack*, *Reset Tank*).

> **SmartScreen note:** the exe is unsigned; Windows may show *"Windows protected your PC"* —
> choose *More info → Run anyway*. The app is portable: keep the folder, delete the folder to
> uninstall. Nothing is written outside the folder.

> **Implementation note (G7.2):** the GPU water shader was shelved — ebiten v2.10.1 does not
> deliver more than one named uniform per draw on this setup. The same look is composited on
> the stable vertex/sprite path (see docs/NOTES-laneA.md); the GLSL version returns when
> upstream fixes named uniforms.

---

## 10. v0.2 Amendment — DEEP BLOOM (fix batch + wonder features)

Signed off: 2026-09-11. Same authority as the rest of this document. v0.1 goals above are
history; the board below is the live one. **G0 rule (traceability):** every requirement maps
1:1 to a goal ID in `docs/TRACEABILITY.md` (requirement → goal → evidence: commit / test / PNG).
A goal without acceptance evidence **blocks the release** — "planned but not built" is impossible
by construction.

### FD — new frozen decisions
- **FD7 No sound effects, ever.** plop/crack/ping are deleted from the game path. The only audio is
  the ambient music loop (N6): toggleable, default ON.
- **FD8 Lilac is reserved.** Hue band 270–300° belongs exclusively to the Chosen fish; the
  uniqueness gate rejects any other generated content in that band.
- **FD9 The Chosen is eternal.** Exactly one Chosen exists at all times; it never ages, never dies,
  ignores MaxFish, and is re-spawned on load if missing.
- **FD10 Music is original.** The piano loop is procedurally synthesized in-process (dreamy,
  piano-led). Mood may be inspired by external references; melodies must not be copied.
- **FD11 Population floor.** Fish count never drops below `MinPopulation`: deaths are spared at the
  floor and breeding urgency rises below it. The tank self-heals upward.

### Contract v2 (README §4 diffs — contract.go updated to match, `SaveSchemaVersion = 2`)
- `Behavior` += `Attachment float64` (0..1 glass affinity; 0 = never attaches).
- `Species` += `Role string` (`"normal"` | `"chosen"`).
- New `CoralDesign{ID,Name,Kind,Fronds,Height,Width,Curve,Sway,Colors,Glow,Source,CreatedAt}`
  (Kind: `fan|branch|brain`) == one file in `content/corals/`.
- New `Zone{Center Vec2, Radius float64, Owner string}` — sim zones (caves shelter, chosen aura).
- `Save` v2 += `Day int`, `RockSeed int64`, `Creatures []SavedCreature`, `CoralIDs []string`;
  new `SavedCreature{Kind,Seed,Pos,ShellT}`. v1 files migrate silently, never deleted.
- `Config`: `SoundOn` → `MusicOn` (`json:"musicOn"`, legacy `soundOn` migrated on load).
- C2 import-graph amendment: `audio → {contract, music}`, `music → {} (stdlib only)`.
- New tuning constants: `MinPopulation=4`, `PlantCoverageMax=0.35`, `MiteCap=6`,
  `MiteSpawnMeanSec=25`, `CareDecayPerSec=0.02`, `CourtshipCooldownSec=45`, `StartleMeanSec=50`,
  `ZoneRadius=140`, `MusicLoopSec=64`, `RockCaves=2`, `CreatureCount=4`, `TreatMax=6`.

### v0.2 goal board

| ID | Goal | Acceptance (evidence required) |
|----|------|--------------------------------|
| F1 | Smooth plants: 32 arc-length samples, joint discs in a separate mesh, rounded base; sway math untouched | 4-phase PNG strip, zero visible kinks at 2×; 12-plant frame ≤1.5 ms; designs keep identity |
| F2 | Glow budget: `render.GlowBudget` (0.55) at DrawGlow alpha + additive mesh draws; vein alpha 63→38, plankton ×0.7 | busy tank ≤2 % pixels >240 luminance; night still bioluminescent |
| F3 | Food urgency: seek weight 3.0→4.5 as hunger rises, ×1.35 speed, ×1.3 force cap, sense 150→190 | hungry fish covers 300 px ≤2.5 s (was >4 s); rush ≤1.5× cruise |
| F4 | Behavior pack: startle (uses `Skittish`), bully chase (no contact), zoomies, nudge — logged | 10-day soak ≥5 startles, ≥3 chases, ≥3 zoomies |
| F5 | Agent menu works: level-triggered mouse input; menu clicks consumed; double-feed removed; tabs at slide ≥0.6 | `tank.exe -menu-smoke` exits 0; first-click open always |
| F6 | Breeding works: `AutoCare` wired (was dead config), recurring care tiers (decay), courtship cooldown, elder-priority at MaxFish | soak (care+feed on, 5 d, 4 fish): ≥2 courtships, ≥2 hatchings, pop ≥6; counter-soak 0 eggs |
| F7 | Elder fade: `p=(Age−12)/4` → desat ×(1−0.6p), alpha 255→175, glow ×(1−0.7p), slower tail | monotonic fade unit test; PNG pair clearly paler; **release-blocking** |
| F8 | HUD clock: top-right `Day N · HH:MM` + phase glyph, above menu, non-interactive | matches TimeOfDay; 1280×720 + 2560×1440 screenshots |
| F9 | Day cycle: TimeOfDay + saved `Day`; dawn/noon/dusk/night LUT over water/rays/caustics/plankton/plant glow | PNGs at 0/.25/.5/.75 clearly distinct; C0-continuous; day increments on wrap |
| F10 | SFX removal: all `audio.Play`/`DrainSFX` deleted; toggle becomes "Music" | grep = zero SFX in game path |
| F11 | Population floor + maintenance auto-feed: MinPopulation spare+breeding boost; care→breeding/lifespan asymmetry; auto-feed holds avg satiety 0.55–0.85 at the current count; manual feed always works | 20-day mixed soak never < floor; autofeed: flakes <8, satiety in band; 2-fish uses less than 8-fish |
| N1 | Corals: `CoralDesign` end-to-end + `DrawCoral` (fan/branch/brain, sine-rib flutter) + water agent designs corals + Save.CoralIDs | demo-coral PNG; simulate run commits content/corals/*.json |
| N2 | Volcanic rocks + caves from `RockSeed` (2 caves + ledge), two-pass render, shelter zones, lava cracks in budget | fish-into-cave PNG sequence; startled fish shelters ≥50 %; seed roundtrip identical |
| N3 | Chosen One `chosen-lilastar`: reserved lilac, 1.3× size, ×1.3 speed, aurora shimmer, immortal; oyster nest + 3–4 pearls; aura r≈140 px hard-excludes everything else alive | 10-day soak: zero intrusions; present after every load; visibly grander |
| N4 | Crustaceans (3–5): wander, nibble flakes; bully-chases → shell retreat 3–6 s; rare strut; SavedCreature persist | ≥2 retreats per 5-day soak; persist across save/load |
| N5 | Glass pleco `glass-sucker` (Attachment 0.9): attach to glass, side profile + mouth disc, pulses, 30–90 s per spot | ≥1 attach/virtual day; smooth transitions; convincing PNG |
| N6 | Underwater piano loop: pure-Go synth, 64 s seamless, async render at boot, low-pass + delay tail, MusicOn toggle | seam <1e-3; RMS −20..−14 dBFS; zero SFX remain |
| N7 | Ecosystem: PlantCoverageMax cap (plants+corals); wild water mites (cap 6) spawn naturally → frenzy contest | coverage test under cap incl. agent additions; ≥2 frenzies/virtual day |
| N8 | Treat Store: tray next to HUD; grab-with-mouse treats (bug/worm/shrimp/chicken), drop → swarm; palette-true, rive-quality | PNG sheet; smoke grab→drop; nearest fish ≤1.5 s, ≥3-fish swarm |

### Waves
- **M0 (serial):** this amendment + TRACEABILITY, contract v2, F5, F10, lane stubs.
- **Wave 1:** Lane A: F1→F2→F3→F6→F7→F11 · Lane B: N1+N2 · Lane C: N6+N5-sim-hooks.
- **Wave 2:** Lane A: F4+F8+F9+N7 · Lane B: N3+N4 content/render · Lane C: N8.
- **Wave 3:** integration, soaks, `-menu-smoke`, PNG suite, TRACEABILITY evidence.
- **Wave 4:** tank.exe build, live verification, `v0.2` GitHub Release.

## 11. v0.3 Amendment — WIDE GLASS (visibility, scale, life)

User-reported fixes + debt closure, each with acceptance evidence (docs/TRACEABILITY.md v0.3 rows).

| ID | Goal | Acceptance (evidence) |
|----|------|----------------------|
| F12 | Menu visibility: 16 px handle + MENU chip, 480 px panel, 34 px scale-2 tabs, resting button fills ≥0.35, tray 80×30 scale-2 | `-shot` PNGs ×4 aspects; uiaudit 336/336 PASS; live: tab labels + toggles read on screen |
| F13 | Fullscreen-first + height-locked proportional scale (720-locked width from monitor aspect, clamped 1024–1920); persisted toggle | opens borderless-fullscreen; evidence matrix 16:9/16:10/3:2/21:9 — no bars, all controls on-frame; width-clamp unit tests |
| F14 | Live fish always outnumber plants: cap = min(12, alive−1) at all placement points + death-cull | TestFishAlwaysOutnumberPlantsSoak (10 tank-days, AddPlant spam); live: fish 17 > plants 5 |
| F15 | Caves are holes fish enjoy: indigo depth + rim-light; idle fed fish lounge 8–18 s (mean 45 s) | TestForcedLoungeHoldsCave + TestLoungeFrequencySoak PASS |
| F16 | Neon is the Chosen's alone: every additive fish path gated by role; vein renders matte for normals | TestGlowVisible + TestPixelAcceptance (normal = 0 glow px); f16-gallery.png |
| F17 | Plants lose glow, keep waves | f17 before/after pngstats: hot 3.65%→0.013% (~280×), sway untouched |
| F18 | No weird blackness + sucker realism: graded cave interiors, softer vignette; attached pose = flat, pale belly band, ~1 Hz mouth disc | darkscan: no region ≥300×80 at lum<14; f18-sucker-1/2.png mouth pump; live sucker PNG |
| F19 | Evidence harness: `-shot` scripted captures + manifest + `uiaudit`/`darkscan` gates wired into `scripts/check` | 4-aspect matrix uiaudit 336/336; check fails on regression |
| F20 | Docs + release v0.3 | README §11 + TRACEABILITY v0.3 + GitHub release |
| N9 | Species variety + Chosen supremacy: role-aware clamps (size ≤1.15, fin ≤1.35, tail ≤1.30), +3 seed species (ribbon-streamer, puff-orbit, dart-spindle), 4-species starting school | TestSeedSpeciesSupremacy + TestSimulateSpeciesUnderNormalCaps; n9-gallery.png (9 species, Chosen measured largest 297×180 vs 243×135) |
| N10 | Right-click scare: fish within 190 px bolt then forget (~2 s) | TestRightClickScareBoltsAndFades; -menu-smoke right-click step |
| N11 | Hook treats: DrawTreat implemented (was an invisible stub); grabbed treat writhes at the cursor; hungry fish crowd on a 34 px keep-back ring; drop → frenzy | TestHeldTreatPullsHungryFishToRing + TestTreatScrambleManyFishDevourWorm (≥2 biters, exact bite accounting); live: hold→crowd→drop→frenzy→treats 0 |
| F22 | v0.2 debt: creatures + mites rendered (were no-op stubs) | DrawCreature/DrawMite implemented; f22-decor.png; pixel tests |
| N12 | Music rework — "Pure Imagination" feel, original: 75 BPM AABA, authored 142 onsets, hammer piano, music-box B section, 3 s fade-in | music tests: onset sharpness ≥4.28×, seam delta 3e-4, −18 dBFS; n12-music.wav |

Controls added: right-click scares fish; app opens fullscreen (Setup ▸ Start fullscreen to disable; F11 toggles).

### v0.3.1 — the screen IS the tank

Big monitors are big aquariums now, not zoomed-in small ones:

- The logical canvas equals the monitor's **physical resolution** (device-independent × scale factor) — fish keep their natural size on any screen and edges stay **pixel-perfect** (the upscale blur is gone). Cursor input is scaled to match.
- **Density scales with tank area** (vs the 1280×720 reference, capped ×4): live-fish ceiling (≤56), plant budget (≤30, still fish-majority-gated), crustaceans (≤10), wild mites (≤16), coral placement pitch — and the starting school grows per species. Verify live: 3440×1440 fullscreen ran 47 fish + 23 plants + 6 corals + 10 crustaceans at a locked **60/60 TPS/FPS**.
- Volcanic caves and domes scale with tank height; old 720p saves are **position-rescaled** into the bigger tank on restore.

Evidence: `docs/art/v03/shot-219-big/` (3440×1440 shot set, uiaudit PASS), `live-fullscreen-v031.png`, sim tests `TestBigTankIsABiggerAquarium`, `TestPlantCapScalesButFishStillOutnumber`, `TestRestoreRescalesOldPositions`.

### v0.3.2 — floor composition 20/40/20/20

The tank bottom now follows a designed split, at every tank size:

| Share | Target | Content |
|-------|--------|---------|
| Plants | ~20% | base-width budget (`W×0.20 ÷ 26 px`), still fish-majority-gated |
| Rocks | ~40% | two enterable cave domes (15% each) + ledge (8%) + nest pedestal (2%), mouths scale with the tank |
| Corals & things | ~20% | base-width budget (`W×0.20 ÷ 42 px`), spacing-aware placement |
| Open water | ~20% | guaranteed leftover for fish and moving critters |

`World.FloorShares()` reports the live split; the `-shot` debug line prints it. Evidence: `TestFloorCompositionSplit` at 1280×720 / 1720×720 / 3440×1440 — measured 20/40/20/20-21%; visual QA on the 3440×1440 frame agrees.

### v0.3.3 — close button + HUD corners

- **Always-visible close button**: a red-framed **X** pinned top-right, drawn above even the open menu; clicking it (or pressing ESC) saves and exits cleanly — no more hunting for a window chrome that fullscreen hides.
- **Day/clock HUD moved top-left** (`DAY n - HH:MM`), so the sliding menu panel and the tray never tangle with it. Verified live at 3440×1440.
- Both controls are in the shared geometry + evidence manifests: `uiaudit` gates their visibility in every `-shot` frame.

### v0.3.5 — the Chosen's oyster

- **A giant camera-facing open oyster, dead center**: ~5× the Chosen's body length — deep nacred cup with shell rings and a lilac lip ridge, a back-tilted lid showing its mother-of-pearl, a breathing mantle fringe, and a living satin tongue that swells and dances toward her.
- **Exactly three pearls** bob inside; when the lilac one approaches the mouth, they blaze with cross-flares, the tongue reaches out, the lid opens wider and lilac motes rise from the shell — the touch scene, live-verified.
- **Lilastar is one of a kind**: removed from the species catalog and `SpawnEgg` refuses her outright — no second Chosen can ever be created.

Evidence: `docs/art/v03/v035-oyster.png`, `v035-live-touch.png` (Chosen in the mouth, pearls blazing, motes rising); `TestNestDeadCenterWithThreePearls`, `TestCatalogExcludesChosen`, `TestChosenCannotBeSpawned`, `TestChosenNestDistAtHome`.

### v0.3.6 — Volcano & the Eternal One

Four fixes, each live-verified (`docs/art/v036-1280/` evidence set, uiaudit 551/551):

- **The Eternal One is supreme** — she stays the single lilac (a corrupted save with two dissolves the extra), can never be culled or fade to elder, runs at adult pace forever, and is now *mathematically* the fastest fish at every hour: normal species are clamped to `speed ≤ 1.35` (validation, generator, seeds) while she keeps a frozen **×1.45** multiplier — worst case (midnight, vs the fastest night hunter) she still wins by ≥15%. Neon glow remains hers alone.
- **The oyster sits on the tank floor** — the cup is anchored to the *window bottom* (±4 px, tested at three canvas sizes) and re-sculpted into true anatomy: two fluted asymmetric valves joined by a hinge heel, a tilted lid, a curled satin tongue, and her exactly-three pearls blazing inside the gape — no floating gap, no fan shape.
- **A volcanic range with doors** — two big jagged basalt crags claim half the floor (rocks 50% share), each with **three mouth doors** (low/mid/high, dark collar + sill so they read as entrances). Cruising fish near a door sometimes slip in, travel hidden for 1.2–2.5 s, and **burst out of a different door** of the same crag with a bubble puff and a speed spike. She never uses the doors — the star is always on camera.
- **Click-toggle treat carry** — pressing a tray row *selects* the treat (the tray closes); it then follows the cursor with **no button held**. Fish smell it — the scent ring grows the longer you hold it (+20 px/s, up to +140 px, pulling distant fish in) — but nobody can bite until the **next left press drops it exactly there**, and the frenzy begins. While carrying, a left press is consumed by the drop (it never clicks through into the menu or the close button); right-click still scares.

Evidence: `TestChosenFastestAtAllHours`, `TestCullNeverTakesTheChosen`, `TestRestoreDedupesChosen`, `TestNestFlushBottomAndMouthPlane`, `TestVolcanicCragLayout`, `TestForcedTransitThroughCrag`, `TestTransitFrequencySoak`, `TestCarryToggleMenuClosed`, `TestHeldTreatScentGrowsWithCarryTime`; door-to-door sequence `docs/art/v036-1280/11-…`–`13-…` PNGs; live tour `docs/art/v036-live/` (60/60 TPS/FPS, carry cycle, drop frenzy).

### v0.3.7 — Silk & Sanctuary

Five fixes, each with test + screenshot evidence (`docs/art/v037-1280/`, uiaudit 674/674):

- **The oyster is ONE shell, hers alone** — the lid's scalloped lower lip now arches over the mouth and lands on the cup's rim shoulders (no floating chord, no see-through gap), the gape is a layered dark cavity instead of a flat wedge, and silky **byssus threads** flow from the hinge onto the floor, binding the valves the way a real oyster anchors itself. An ambient shadow pocket seats the shell into the scene; the free-floating halo is gone, the nacre and pearls stay luminous. In the sim the aura never pushes **her** away from her own nest again — every other fish and creature is still hard-projected out (`TestNestBelongsToTheChosen`).
- **The menu opens below TREATS and the close button** — the panel top anchors under the top chrome (`PanelTopY`), tab clicks follow, and clicks on TREATS/X reach those buttons even with the menu open (`TestMenuClearsTopChrome`).
- **Max fish up to 100** — the slider, the config clamp and the population ceiling all admit 100 (`PopCapMax 56→100`, `TestPopCapHonorsUserMax`). Measured on the reference machine: sim holds a locked **60 TPS at 100 fish**; frame rate at the cap is GPU-bound (~17 FPS fullscreen), smooth 60 up to ≈60 fish — the school now renders as **one batched draw call** after a CPU profile pinned 86% of frame time in per-call driver submission. Find your sweet spot on the slider.
- **Silky, feathery, strange flora** — two new matte plant archetypes: **feather** (sea-pen plumes with dense silky barbs) and **silk** (fine hair threads that sway like soft hair), shipping as six hand-authored seeds (`ghost-pen-feather`, `ember-plume-feather`, `twilight-plume-feather`, `moon-silk-grass`, `abyss-silkthread`, `pearl-veil-silk`). Legacy packs default to the classic ribbon silhouette (F17 stays: no plant glow).
- **Doors truly hide; grander crags** — transiting fish render *behind* the crag silhouettes, vanishing the frame they cross the mouth plane and re-emerging at the exit door (see the 11→12→13 sequence). The volcanic crags grew outward and taller (peaks ≥0.46H, bases ≥0.26W each) with ~15% larger mouths.

Evidence: `TestNestOneShellGeometry`, `TestNestBelongsToTheChosen`, `TestMenuClearsTopChrome`, `TestPopCapHonorsUserMax`, `TestPlantKindClamp`, `TestSilkySeedsShipInStore`, `TestVolcanicCragLayout`, `TestForcedTransitThroughCrag`; flora gallery `docs/art/v037-1280/14-flora.png`; crowd frame `15-crowd.png` (alive=100, TPS 60); smoke 6/6, uiaudit 674/674.

### v0.3.8 — Stagecraft (shipped as v1)

Five fixes, each with test + screenshot evidence (`docs/art/v038-live/`, uiaudit 123/123) — the tank stops being a pile of props and composes as one stage with real depth:

- **The nest becomes scenery, not a wall** — the oyster shrank to throne scale (≤30% of canvas width, was 470 px), left the window edge and now sits **on the shared floor line** (0.925H — the same ground plane the crags stand on, `TestNestFloorLineAndMouthPlane`, `TestNestSceneryScale`), it renders **behind the school** so fish glide in front of it (plants read behind the shell), and its near-black shadow pocket, gape palette and halo were lifted two steps — darkscan's biggest dark region on the opening frame collapsed from a 7,985 px² blob (the old shell) to ≤495 px² specks. The lila fish stays visible at home, in front of her own gape.
- **Doors swallow in binary — no more translucent ghosts** — the old 0.25 s alpha ramp rendered entering/exiting fish as faded silhouettes over open water. Now `Hide01` is strictly 0 or 1: the fish stays fully opaque on the crag face until it crosses the mouth plane, vanishes behind the dark hole, and pops out fully opaque just outside the far mouth (`TestForcedTransitThroughCrag` re-proves the door-to-door trip; the 11→12→13 PNG sequence shows opaque → gone → emerged).
- **The tank belongs to the school** — idle cruisers now sweep the whole tank on soft random waypoints (`roam.go`), lounging became a sanctuary instead of a dormitory (one fish per cave, ≤20% of the school at once, mean pick interval 45→120 s, dwell 8–18→5–10 s), tired-fish shelter needs 40% less energy drain, and species cohesion no longer counts fish parked at the caves — the school no longer drags itself to the towers. Measured with a new `-shot` probe (% of fish within 200 px of a door): **57–85% mid-run before → 6% after**; the crowd frame shows the school spread across the entire water column. A startled fish also keeps seeking cover until it actually *reaches* a cave (the old pull died with the 2 s impulse).
- **The crustaceans retired** — the two crabs and two shrimp (dark ~17 px floor specks that read as black U-shapes against the night-dimmed floor) are gone from sim, render and saves; old snapshots still load (the field is parsed and ignored). Mites stay.
- **The near-glass plane** — three to four large, cool-dimmed fronds (×0.38 shade with a blue bias) rooted at the very bottom edge at the flanks, drawn after the rock front: far crags → midground flora/school/nest → dark foreground silhouettes = a 2D canvas that reads 3D. Each frond is **baked once into its own offscreen image** (one `DrawImage` per frame with a whole-plant ±0.7° breathing) because the per-frond mesh renderers measured +45% draw calls and a visible TPS drop at the cap.
- **The score now sways like the reference** (v0.3.8.1 + .2 revision) — the harmony language was already right, but the reference intro's identity is its **texture**, so the left hand was rewritten from sparse half-note sweeps into the signature **rolling 12/8 triplet broken-chord wave**, and v0.3.8.2 pushed the resemblance all the way: **tempo 75 → 90 BPM** (24 bars, the loop still exactly 64.000 s), the arpeggio now ascends **through the warm tenth** (low root → root → tenth → fifth crest), a **soft celesta halo** doubles the big B-section notes two octaves up, a **dynamic arc** leans A' in, swells B to its crest and recedes the return into the seam, and an authored **seam ritardando** (≤ 36 ms) breathes the loop into its own downbeat. Still fully ORIGINAL (no existing melody reproduced). Melody 84 + left hand 282 + music box 42 + celesta 13 = **421 authored onsets**; the tempo-grid proof lives on a 1/12-beat grid (worst 9.3 ms; the tolerance covers the authored ritardando), and the hammer proof is tiered the way the music now is: phrase accents (vel ≥ 0.60) jump ≥ 4×, every melody onset ≥ 2.2× (measured min 2.40×, median 4.89× over the pedaled bed). Listening evidence: `docs/art/v038/n12-music-v3.wav` (−17.62 dBFS, peak 0.88, seam 0.1× median sample delta).

Perf note (measured, interleaved on the same save): at the 100-fish cap the v0.3.8 build holds **54–56 TPS vs v0.3.7's 57** on this machine (the historical 60 was a quieter moment at a 1280-wide canvas; the logical canvas here is 1720×720). A full tank also **stops breeding** now — the old hatch→cull→fade churn kept dozens of dissolving corpses on screen and cost real frames; dying fish no longer persist into saves.

Evidence: `TestNestFloorLineAndMouthPlane`, `TestNestSceneryScale`, `nestFloorPixelProof`, `TestNestDeadCenterWithThreePearls`, `TestNestOneShellGeometry`, `TestForcedTransitThroughCrag`, `TestTransitFrequencySoak`, `TestScareAbortsApproach`, `TestStartledFishSheltersInCave`, `TestMitesSpawnAndFrenzy`; opening frame `docs/art/v038-live/01-closed.png` (nest seated among plants, fish in front), door sequence `11-…`–`13-…`, crowd `15-crowd.png` (alive=100, TPS 56, cluster 6%); smoke 6/6, uiaudit 123/123, `scripts/check` all conventions hold.

---

## v1 known issues

- Fish tails over-curl on sharp turns — looks unrealistic. CLOSED in v1.1
  (G63): the spine curvature clamp now covers every species, and the chain
  hangs from the correct trailing axis (the v1.1 clamp had referenced the
  forward axis, which re-laid the whole body ahead of the head — eyes
  trailing, tail fins leading like clock hands).

## v1.1 — Titans & floor critters (ambient life)

Two ambient features, measured per docs/TRACEABILITY.md rows 56–64 (goals G39–G47).

**Titan pod (G39–G43, G48–G49).** Every 3–6 minutes a pack of 2–5 deep-water
giants (`titan-abyssdrifter`, one vast leader + escorts) slides in from a
side edge, roams the tank for 2–4 minutes at a ponderous cruise, then swims
off. They are SCALARE (angelfish) like the reference photo: a five-fish
group with one biggest leader, silver-pearl bodies with dark slate stripes,
tall dorsal and anal fins and trailing ventral streamers. The pod sweeps
the tank left-to-right and right-to-left, moving only toward its head; at
the glass each member curls 180° gracefully around its own axis — carved
forward (the speed never stalls mid-turn, so no tail-first drifting) with
the spine bend tightened during the curl so the tail stays clear of the
body. The pod holds staggered slots behind the leader and drifts the far
depth lane, so the small school reads in front of them. A hungry scalare still darts at ≥6× its cruise
speed, and only the guarded rare hunt of G42 remains above that. A hungry giant lunges at
flakes at ≥6× its cruise speed; a right-click scare bolts the whole pod away
from the point at the same burst, decaying over the startle window; and only
after starving 20 s straight — with a 300 s tank-wide cooldown, a population
floor and the lilac fish forever immune — may the leader take one small fish
(fast fade, no corpse, one log line). The pod is not selectable anywhere:
catalog filter, SpawnEgg guard, and the `titan` role is core-seed-only in
content validation. It never counts toward popCap, breeding, aging or
snapshots.

**Hammerhead pair (G50).** A resident hunter in dark blue-black with a
white belly — always on the move, sweeping the whole tank on roam waypoints,
at most two of them, and never faster than the lilac fish at any hour (the
same-hour supremacy test covers even a food-frenzy double). The hammer
rostrum renders as a crossbar with tip eyes. The pair persists in snapshots
(cap enforced on restore), sits outside popCap, breeding and aging, and is
off the menu and eggs like everything else that is not catalog stock.

**Her circle is absolute (G55).** "No creature enters the lilac fish's
circle" now means the whole body: the projection is spine-aware, so even a
640 px giant cannot lie across the nest — the head clears instantly and the
rest of the body drains out within frames, while body-aware early repulsion
turns long fish away half a body-length before the line. Critters and mites
keep a wider shell margin.

**Depth lanes & crosswise turns (G51).** The water has near and far lanes:
small fish drift slowly between them, so they sometimes cross the big bodies
in front and sometimes slip behind them — painter's-order rendering plus a
small lane-based size and alpha cue sells the depth. Nobody swims through a
big body: small fish steer around titan and shark silhouettes. And the
giants no longer cruise straight forever — every ~25 s a giant re-aims its
heading by up to ≈180°, bending into a wide turn (the curvature clamp draws
the arc).

**Staging & realism (G52).** Giants never linger off the frame — anything
roaming past the margin steers back into view within seconds (a scare bolt
cannot strand one there; the two forces never cancel). The pod favors the
upper 80% of the water column and hugs the sand line: headings stay near
horizontal, steep pitches are rare, and crosswise turns bend into the
posture clamp instead. Growth weights the bodies: a bigger pod member
cruises heavier but lunges farther — one tail whip covers real distance —
and every tail beat slows with size.

**Vivid school (G56).** The seed school wears vivid palettes only: the old
slate-gray glass-sucker is now rose with a cream belly and gold accents, and
the depth-lane dimming floor was raised so even far-water fish stay lively.

**Sand bed (G54, G57, G60).** A solid yellowish-white sand layer fills the
floor from the water line down to the bottom edge, full width — the nest
pedestal sits on it, so nothing floats.
Fish skimming the floor scatter the grains (they lift and push sideways,
stronger with speed), and the bed eases back to perfectly flat on its own
within about half a minute. Purely visual state; never persisted.

**Floor critters (G44–G46).** Crabs and shrimp emerge from the sand
(cap 8 × area density, ≤14), walk the floor band, and in the last 25 s of
their 90–150 s cycle they struggle in the open: hungry fish feel the pull
from 260 px and the first to arrive eats one. An uneaten critter burrows back
— no corpse. The lilac fish's circle stays absolute: spawn check + per-tick
projection, zero intrusions measured. The shells are vivid coral-orange, and
visibility is measured, not assumed: 1.81–1.89× floor luminance by day and
1.76–1.90× by night — the v0.3.8 "dark clutter" failure class is absent.

**Performance (G47).** Interleaved A/B on the same save (`TANK_EXTRAS=0` vs
default): capture TPS OFF 55–60 vs ON 58–60, exit rates OFF 54/55 vs ON
57/57 — parity within machine noise on this GPU-bound box. Tuning constants
live in `internal/contract/tuning.go` (no config keys, no menu entries).
Evidence frames: `go run ./cmd/tank -probe <dir>` (writes PNG + TPS lines,
no manifest).
