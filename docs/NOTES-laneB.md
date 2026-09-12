# NOTES — Lane B (content + LLM + agents)

One block per goal (C3 template). Append-only.

### G3.1 — Content store
- What: `internal/content` package: `store.go` (Load/scan/accessors), `validate.go`
  (strict slug/hex/enum checks + range clamps), `write.go` (clamped copy, atomic
  tmp+rename, registry append), `pack.go` (zip export/import with registry merge),
  `seed.go` + `seed/**` (15 embedded core artifacts: 4 species, 3 water, 4 plants,
  4 recipes, all `source:"core"`), `registry.go`, `fingerprint.go`, `gate.go`.
- Why: content is data, not code (C4); the exe reads/writes `content/` beside it (C1).
- Gotchas: clamping happens BEFORE validation on load and write, so out-of-range
  floats never reject a file; only unparseable JSON / bad hex / bad slug / bad enum
  skip a file (logged). `WaterPreset`/`PlantDesign`/`PatternRecipe` have no
  `CreatedAt` field — only `Species` gets the RFC3339 default. Empty color lists are
  padded with neutral gray so count-clamping never rejects.
- Decision: seed sources are **go:embed**ed JSON under `internal/content/seed/**`
  (chosen over inline Go literals: per-artifact JSON stays readable/diffable and
  `EnsureSeed` walks the embed FS generically). The runtime `content/` folder is
  generated at app start and not committed.
- Follow-ups: none.

### G3.2 — GLM streaming client
- What: `internal/llm/glm.go` — `Stream(ctx, Request, onDelta, onThought)`; SSE
  parser (data:/[DONE]), `reasoning_content` routed ONLY to onThought; typed
  errors ErrAuth (401/403), ErrQuota (429), ErrNetwork (transport/timeout),
  ErrHTTP (other, body snippet included). Stdlib only.
- Why: one place owns agent HTTP traffic (C2 import graph).
- Gotchas: `streamTimeout` is a package var (90s default) so tests can shrink it;
  proxy URL parse failure is logged and ignored, never fatal (D4); scanner buffer
  raised to 1 MiB for long SSE lines; mid-stream failures still return the partial
  content (with ErrNetwork) — agents treat that as network death per spec.
- Follow-ups: none.

### G3.3 — Simulate generators
- What: `internal/agents/simulate.go` (HSL->hex, neon palette random walk, English
  syllable name combiner, unique ID/name helpers) + `simulate_gen.go`
  (SimulateSpecies/Water/Plant/Recipe, all store- and registry-aware via
  `content.Store.Gate`/`GateRecipe` re-roll loops, 128 attempts).
- Why: D7 offline-first; the simulate path runs the identical panel flow.
- Gotchas: `SimulateRecipe` needed the store parameter (IDs + gate) like the other
  generators — signature is `(rng, stage, store)`. Determinism per `*rand.Rand`
  (D5); hue rotation `+37*attempt` on gate retries diversifies re-rolls.
- Accept: property test generates 200 artifacts (50 per kind) — all validate and
  pass the gate, registry grows accordingly.
- Follow-ups: none.

### G3.6 — Uniqueness registry & gate
- What: `internal/content/registry.go` (append-only `registry.json`, {hash,kind,id,at},
  dedupe by hash, cap 10000 oldest-dropped, atomic save), `fingerprint.go`
  (canonical = pattern type + density/size @0.05 + 4 palette hues @10 degrees, hue
  extraction implemented locally; recipes use stage + sat/alpha/glow multipliers),
  `gate.go` (Gate + GateRecipe + RecentFingerprints + case-insensitive name
  uniqueness with `name:<lower>` conflict tokens).
- Gotchas: Gate and the Write* registration MUST build the canonical identically —
  both go through `canonicalFor`/hueCanonical so they cannot drift. Re-gating an
  already-registered artifact is rejected even for the same ID (C6 semantics;
  test pins this). Registry `Append` dedupes too, so pack re-import is idempotent.
- Accept: 2-degree hue shift rejected, 20-degree accepted; entry 10001 drops the
  oldest; merge dedupes identical hashes.
- Follow-ups: none.

### G3.4 — Agent framework + 3 agents + demo
- What: `internal/agents/agent.go` (Hub: event bus 256 cap drop-oldest, schedules =
  `AgentSchedules x AgentFreq`, first run at `AgentFirstRun`, panic containment),
  `mode.go` (live/simulate policy + 429 quota guard + silent 10 min probe +
  automatic switch-back + EventLog), `prompts.go` (ALL prompt strings as named
  constants, C4), `species_agent.go` / `water_agent.go` / `pattern_agent.go`
  (stream -> JSON extract -> validate -> gate -> ONE repair round-trip -> simulate
  takeover -> write -> hooks -> EventArtifact), `cmd/demo-agents` (fake SSE server,
  3 concurrent agents, manual "Research New Species" trigger, self-exit ~19 s).
- Why: agents are the product's show; every rule (repair, uniqueness, quota,
  simulate fallback) is observable in the event stream.
- Gotchas: `nightActive` arrives as 0|1 from the LLM — `flexBool` accepts bool or
  number. Recipe stage is forced to the requested stage after parsing. Water agent
  emits TWO artifact events (water + plant) and never spawns eggs. Repair combines
  parse/validation/gate failures into one round-trip (spec allows one per failure
  kind; combined keeps a single round-trip). Water presets/plants carry no
  CreatedAt; sources are "species-agent"/"water-agent"/"pattern-agent".
- Demo gotcha: initial canned demo species/water accidentally equaled the seed
  fingerprints (copied palette) — the gate rightly rejected run 1; canned data now
  differs so the arc is: run 1 live artifact -> manual run 2 duplicate -> repair ->
  simulate takeover.
- Accept: agents_flow_test covers live flow, duplicate->repair->simulate, bad
  JSON->simulate, water+plant, pattern repaints (ApplyRecipe called), 429 ->
  temporary simulate -> probe -> recovery log, ErrNetwork/no-key -> permanent
  simulate with zero LLM calls; event drop-oldest bound test; demo integration
  test runs the whole demo fast-forward and checks the file inventory.
- Follow-ups: G6.1 wires Hub hooks to the real world (repaint queue, SpawnEgg).

### G3.5 — ZCode key import
- What: `internal/config/zcode.go` — `FindZCodeKey(dir)` scans `config.json` then
  `credentials.json`; JSON token-stream walk in document order (phase machine
  distinguishes keys from values); fields matched case-insensitively after
  stripping `_`/`-` against {apikey, authtoken, token, key, authorization};
  plausible = len>=20, no whitespace. `MaskKey` shows only the last 4 chars.
- Found JSON paths (documented per goal): `%USERPROFILE%\.zcode\v2\config.json`
  nested e.g. `client.apiKey`; `credentials.json` top-level `authtoken`.
- Gotchas: never logs the key (D9); corrupt/missing files yield "not found" (D4);
  `FindZCodeKeyDefault` resolves `%USERPROFILE%\.zcode\v2` (C1's one allowed
  external read). config.go (loading/lock) is G6.2's, intentionally absent.
- Follow-ups: G6.2 consumes FindZCodeKeyDefault for the first-run prompt.
