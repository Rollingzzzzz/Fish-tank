// Package audio plays the ambient music loop for NEON TANK (FD7: no SFX).
//
// Purpose: tiny player on ebiten/audio; receives the in-process synthesized
// loop bytes from internal/music and exposes SetMusic/SetOn/PlayMusic
// (N12: PlayMusic fades in over ~3 s; SetVolume adjusts the ramp target).
// Owns: engine + player lifecycle.
// Public API: New, (*Engine).SetMusic/SetOn/On/PlayMusic/SetVolume.
// Invariants: missing music degrades to silence (D4); ebiten/audio + stdlib
// only (C2).
// Extension points: volume/shaping constants in engine.go.
package audio
