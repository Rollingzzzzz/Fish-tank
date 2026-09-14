// Package render draws everything the player sees: the GPU water shader,
// glow sprites, phosphor trails, plants and spine-based fish bodies.
//
// Purpose: pure drawing — no simulation logic (sim lives in internal/sim).
// Owns: background shader + fallback, glow cache, trail buffer, mesh helpers,
// plant and fish renderers, demo entry points under cmd/demo-*.
// Public API: Background/WaterState, DrawGlow, Trail, DrawPlant, DrawFish,
// DrawFishPreview, FishAnim.
// Invariants: never panics on bad input (D4); no per-frame allocation in hot
// paths (C3); every emissive element uses additive glow (art direction §3).
// Extension points: new element renderers = new files here, mesh helpers first.
package render
