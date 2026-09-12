// Package music synthesizes the ambient underwater piano loop (N6, FD10).
//
// Purpose: pure-Go procedural composition — dreamy piano arpeggios over a
// warm pad, low-passed with a gentle delay tail — rendered to a stereo F32LE
// byte stream ebiten/audio loops seamlessly. ORIGINAL composition; no
// copyrighted melody may be reproduced.
// Owns: additive piano/pad voices, chord sequencer, low-pass + delay, mixer.
// Public API: Render, Rate.
// Invariants: deterministic (fixed seed); loop length is an integer sample
// count with phase-matched ends; stdlib only (C2).
// Extension points: chord cycle / voice parameters are named constants.
package music
