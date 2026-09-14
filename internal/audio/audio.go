// Package audio plays the ambient music loop for NEON TANK (FD7: no SFX).
//
// Purpose: tiny player on ebiten/audio; receives the in-process synthesized
// loop from internal/music and exposes SetMusic/SetOn/PlayMusic.
// Owns: engine + player lifecycle.
// Public API: New, (*Engine).SetMusic/SetOn/On/PlayMusic.
// Invariants: missing music degrades to silence (D4); ebiten/audio + stdlib
// only (C2).
// Extension points: volume/tone shaping constants in engine.go.
package audio
