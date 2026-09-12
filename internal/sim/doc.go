// Package sim runs the living aquarium: fish steering + schooling, food,
// eggs, particles, aging, courtship breeding, natural death, care score,
// the day/night clock and the 3-tier crash-safe persistence.
//
// Purpose: world state + update logic, no drawing (render reads it).
// Owns: Fish/World/Food/Egg/Particle/LifeCycle/Persist.
// Public API: NewWorld, (*World).Update/Snapshot/Restore/ApplyWater/
// SpawnEgg/DropFood/NextRepaints/ApplyRecipe + accessors for render.
// Invariants: deterministic via seeded RNG (D5); bounded populations (C3);
// stdlib + contract only.
// Extension points: new behaviors = new force terms in fish.go; new events =
// new queues in world.go.
package sim
