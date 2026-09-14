# NOTES — Lane A (render + sim + audio)

### G1.1 — Background water
- What: CPU-composited water (gradient quads + accent glow pools + god-ray
  shafts + caustic patches + plankton motes + vignette) on DrawTriangles.
- Why: the Kage fragment shader version is SHELVED — ebiten v2.10.1 does not
  deliver more than one named uniform per draw on this setup (1 vec4 works,
  2+ arrive as zero). Uniforms were verified with a minimal red shader.
- Gotchas: v2.10 renamed the Fragment signature (dstPos/src0Pos + uv must be
  derived from dstPos.xy - imageDstOrigin() over imageDstSize()); uniforms
  must be EXPORTED (uppercase) globals; reversed smoothstep edges are
  undefined behavior and produced NaN regions on D3D. Revisit when upstream
  fixes named uniforms.
- Follow-ups: restore the GLSL water when upstream is fixed (file kept in
  git history).

### G1.2/G1.3/G1.4 — glow, trail, plants, fish
- What: cached radial glow sprites (premultiplied pixels!), ping-pong phosphor
  trail buffer, ribbon-plant renderer, spine fish renderer with 5 pattern types.
- Gotchas: ebiten v2.10 CULLS back-facing triangles — quad() emits both
  triangles with consistent winding (the earlier "wedge"/sawtooth artifacts
  were inconsistent winding + additive saturation, both fixed). Overlapping
  pieces inside one mesh cancel under the new fill rule — every overlapping
  piece draws in its own mesh.
- Follow-ups: bloom post-pass if we ever add a shader again.

### G1.5 — audio
- What: scripts/gen-audio synthesizes 4 WAVs (deterministic); engine loops
  ambient and plays one-shot SFX; Config.soundOn mute.
- Gotchas: ebiten audio context sample rate 22050; mono→stereo F32 conversion.

### G2.1–G2.5 — simulation
- What: spine fish + boids (gather radius 2.5× school radius so scattered
  schools reform), food/eggs/particles, aging/care/courtship/natural death,
  world orchestration, 3-tier rotating snapshots with crash recovery.
- Gotchas: followSpine re-normalizes each segment to exactly segLen after the
  wave displacement (acceptance requires 1e-6); death is skipped below
  MinFishCount; recovery order minute→daily→weekly skips future schema versions.
- Follow-ups: none.
