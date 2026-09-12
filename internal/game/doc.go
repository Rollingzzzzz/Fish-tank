// Package game wires the NEON TANK world, renderer, UI menu, agent hub and
// audio into the ebiten loop, plus bootstrap (config, lock, restore).
//
// Purpose: composition root only — all behavior lives in the packages it
// imports.
// Owns: Game, input mapping, bootstrap/first-run, single-instance lock.
// Public API: Bootstrap, (*Game) ebiten Game methods, Trigger.
// Invariants: never panic on missing files (D4); save on exit (C3).
// Extension points: new menu actions get a case in runAction.
package game
