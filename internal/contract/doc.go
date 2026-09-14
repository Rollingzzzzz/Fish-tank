// Package contract holds the FROZEN shared types, JSON schemas and tuning
// constants for NEON TANK (README §4).
//
// Purpose: single source of truth every other package compiles against.
// Owns: nothing else — no other files may live here without a README change (D3).
// Public API: Vec2, Palette, Pattern, Behavior, Species, WaterPreset, WaterEvent,
// PlantDesign, PatternRecipe, Config, SavedFish, SavedFood, SavedEgg, Save,
// plus Clamp / RandSeed / Slugify / Fingerprint helpers and tuning constants.
// Invariants: no imports outside the Go stdlib; field names and JSON tags are
// exact per README §4; changing anything here requires a README change first.
// Extension points: new goals add NEW types here only via a README change.
package contract
