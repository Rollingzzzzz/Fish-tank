# NOTES — decision log (append-only)

One block per merged goal, newest at the bottom. Never delete entries (C3).

<!-- block template:
### Gx.y — <title>
- What:
- Why:
- Gotchas:
- Follow-ups:
-->

### G0.1 — Module bootstrap
- What: go.mod (module github.com/Rollingzzzzz/Fish-tank, Go 1.27), ebiten v2.10.1 pinned,
  temporary cmd/tank/main.go opening a 1280x720 "NEON TANK" window (resizable), docs/NOTES.md.
- Why: serial foundation required before any lane starts (README G0.1).
- Gotchas: main.go is temporary — replaced by game.Game wiring in G6.1.
- Follow-ups: none.

### G6.1-G6.2 — game loop + bootstrap (integration)
- What: game.Game wiring (world/menu/hub/audio/render), input mapping
  (click=feed, swipe=scatter, F/F11), PID-liveness single-instance lock,
  config save/load with normalize, save-restore on boot, snapshot on exit.
- Why: composition root; menu actions and agent events meet here.
- Gotchas: hub is recreated (context cancel) after API-key changes; agent
  artifact events drive SpawnEgg/ApplyWater; menu click can consume the feed
  click (feed suppressed while menu open).

### G7.1-G7.3 — build, polish, release
- What: scripts/check (doc.go, 300-line ceiling, C2 import graph), README
  controls, final art frames in docs/art, v0.1 tag + GitHub Release with
  tank.exe + sample-pack.zip.
- Gotchas: Kage GPU water shelved due to ebiten v2.10.1 multi-uniform
  delivery bug (single vec4 OK) — CPU water implements the same look; see
  NOTES-laneA. ReadPixels returns stale data for DrawRectShader output —
  OS screenshots are the source of truth for shader-era frames.
- Follow-ups: revisit GLSL water when upstream fixes named uniforms.

## M0 — v0.2 Deep Bloom foundation (integration, 2026-09-11)
- README §10 amendment: FD7 no-SFX-forever, FD8 lilac reserved (hue 270-300), FD9 chosen eternal,
  FD10 music original, FD11 population floor (MinPopulation=4). Contract v2: Behavior.Attachment,
  Species.Role, CoralDesign, Zone, Save.Day/RockSeed/Creatures/CoralIDs, Config.MusicOn (legacy
  soundOn migrated), SaveSchemaVersion=2. Tuning: v0.2 block in tuning.go.
- F5 root cause: game fed EDGE-triggered IsMouseButtonJustPressed into the LEVEL-triggered UI —
  a physical click disarms the handle on frame 2, menu could never open. Fixed: pass
  IsMouseButtonPressed (level) + JustReleased (edge). Menu.Consumes(mx,my) added; feed clicks
  suppressed over UI; double FeedAt (game.go + input.go) collapsed to one. Tabs clickable from
  slide 0.6.
- F10: DrainSFX/sfxQ/Play removed everywhere; engine rewritten (New + SetMusic/PlayMusic only);
  demo-audio + scripts/gen-audio deleted; MusicOn toggle in Settings; bootstrap synthesizes music
  async via internal/music (M0 stub returns silence; Lane C fills).
- Lane stubs (frozen APIs): render/rock.go (RockLayout+Zones+3 draws), render/coral.go
  (CoralSpot+DrawCorals), render/treat.go (DrawTreat), render/decor.go (DrawCreature/DrawMite),
  sim/treats.go (Treat/DropTreat/tickTreats), sim/decor.go (Coral/Creature/Mite/SetZones/RockSeed/
  Day), ui/treatstore.go (TreatTray), internal/music (Render). game.go draw order wired for all
  new layers; HUD chip (ui.Chip) + world.TimeOfDay() landed early (F8 base).
- v1 saves migrate silently (Day estimated, RockSeed randomized); LoadLatest accepts v1+v2.
- Lessons: keep lane stub APIs compiling BEFORE launching parallel lanes; subagents replace stub
  bodies only, never signatures.

## Wave 1+2 — Lane A fixes & features + Lane C merge (integration, 2026-09-11)
- F1: plants arc-length resampled (cumulative-chord LUT, 32 samples, smoothstep width ramp).
  Joint discs replaced by resampling — per-disc meshes explode draw calls and same-mesh discs
  cancel under the v2.10 fill rule; even arc-length kills the visible kinks equally well.
- F2: render.GlowBudget=0.55 applied in DrawGlow + mesh.draw/drawA (DrawTrianglesOptions has NO
  ColorScale in v2.10 — budget must scale vertex RGB in place). Vein alphas 210/160→140/105,
  plankton dimmed, game-layer glows trimmed (food .7→.45, egg .3→.2, flash .4→.28, particle .6→.45).
- F3: food seek ramps weight 3→4.5 with hunger, rush ×2.0 speed, seekBonus caps (MaxForce*2 during
  chase), perception = FoodSenseFloor(190) + 60*curiosity + 190*hunger (starving fish smell far).
  Live treats outrank flakes; mites outrank treats.
- F4: behaviors.go — startle (Skittish knob finally live), bully chase (break-off before contact),
  zoomies, nudges; deterministic timers + counters for soak evidence; nobody bullies the Chosen.
- F6: AutoCare now accrues care (was loaded and never read!); recurring courtship via cooldown
  (CourtshipCooldownSec scaled 1.6−0.8·care01) gated at Care ≥ tier 2; cullOldestElder un-stalls
  eggs at MaxFish. Deviation from plan: care decay replaced by cooldown+gate (simpler, deterministic).
- F7: Fish.ElderP drives render fade — desat(color, 1−0.6p), alpha ×(1−0.31p), glow ×(1−0.7p),
  tail beat ×(1−0.25p). desat = luminance lerp in fish_helpers.
- F11: MinPopulation=4 floor (deaths spare + self-heal courtship below floor, cooldown 12 s);
  elder lifespan +ElderLifespanBonus(3d)·care01; maintenance auto-feed: feeds only when avg
  satiety < 0.7 and flakes < 8 — band held, floor never carpeted. Manual feed always works.
- F9: WaterState.TOD; dayTint 4-key smooth LUT (dawn amber / noon cyan / dusk violet-gold / night
  blue-black) tints water gradient + accent pools + rays + caustics; C0-continuity + distinctness
  unit-tested; Day counter + HUD chip (ui.Chip) + world.TimeOfDay().
- N3: ensureChosen on NewWorld+Restore (Role=chosen, immortal, ×1.3 speed, skips aging/breeding/
  death/MaxFish); aura zone hard projection (enforceZones) + steering repulsion; nest pearls render
  via Lane B rock.go.
- N4: 4 crustaceans (shrimp/crab), roam + floor bob, shell retreat when fast fish within 30 px,
  crab strut event; SavedCreature persisted; Restore re-seeds for v1 saves.
- N5: pleco glass-attach (Attachment>0.5, ~20 s mean cycle, 30–90 s hold, suck pulse, flee
  releases); wall forces bypassed because position is pinned.
- N7: water mites spawn near plants/glass (MiteSpawnMeanSec, cap 6), frenzy priority in food seek,
  devoured on contact; plant/coral spacing rule (70 px) via spacingOK in AddPlant/SetCorals.
- F5/N8 acceptance: internal/game/smoke.go — tank.exe -menu-smoke drives handle/tabs/tray/treat
  drop with assertions, exit 0 = pass. game.Update refactored: processInput (shared by live+smoke).
- Lane C merged as-is: music synth (voices/compose/dsp, −15.8 dBFS RMS, peak 0.53, seamless seam),
  treats motion+bites, tray UI. demo-agents test updated: 8 species now (6 seed + live + simulated).
- Transient parallel-lane breakage observed and absorbed (lane stubs kept integration cheap).

## Wave 3 — Lane B merge + integration polish (2026-09-11)
- Lane B landed: corals dir end-to-end (store refactor scanTyped[T] to hold the 300-line ceiling),
  GateCoral/GateSpecies + FD8 lilac reservation ([270,300] hue, core/chosen exempt), rock bodies
  (baked mound/arch/jaw/crack meshes, frame-counter crack pulse), coral fan/branch/brain with rib
  flutter, oyster nest + pearls, demo-coral, water-agent coral production + SimulateCoral,
  glass-sucker + chosen-lilastar seeds.
- Integration polish: brain-coral crown dimmed 0.42+0.34·elev (was a loud orange loaf), pearl glow
  raised (mesmerizing), HUD text ASCII-only (5x7 font has no U+00B7 middle dot — it garbled the
  chip).
- Live-run diagnosis: the 3-fish tank state is a legacy v0.1 save (old MinFishCount=2 era); FD11
  self-heal engaged (2 same-species adults + floor < MinPopulation → recurring courtship).

## Wave 3b — live verification findings + fixes (integration, 2026-09-11)
- Perf: a legacy save had accumulated 35 plant instances (NewWorld seeded EVERY store design;
  agents add designs forever). NewWorld caps placement at 12, Restore trims overflowing legacy
  saves. FPS 11 → 28 (capture overhead included; TPS 60).
- Corals were 0 live: SetCorals ran before Restore, and Restore wiped them; coralDefs now
  remembered and re-placed after Restore. The 70px spacing rule made corals impossible on a full
  floor (12 plants × 93px pitch) — corals probe at 70 then dense-scan at 44.
- FD11 live proof: a 3-fish legacy tank self-healed to 4 — startCourtship(relaxed) lets elders
  breed below the floor (the only same-species pair were elders).
- F5 live: real clicks land fine; the menu handle also got a same-frame press+release fallback
  (Menu.pd) for instant synthetic clicks. menu-smoke: SMOKE OK.
- HUD: bitmap font has no U+00B7 middle dot — ASCII-only HUD text.
- Water agent visibly cycled presets live (abyss-bloom → mint green → violet) and produced 19
  coral + many plant designs across sessions — the agentic loop is thriving.
- Debug overlay now shows plants/corals/creatures/mites/treats counts (F key).
