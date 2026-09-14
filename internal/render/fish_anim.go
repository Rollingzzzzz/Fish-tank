// v1.2 split (line ceiling): the fish animation state the game hands the
// renderer, and the per-stage alpha law.
package render

import ()

type FishAnim struct {
	Time       float64 // global seconds
	Speed01    float64 // 0..1 normalized speed, drives tail beat
	ElderP     float64 // F7: 0..1 elder fade (desaturation + alpha + dim glow)
	Attached   bool    // F18: suctioned to the glass (flat sucker pose)
	AttachSide int     // F18: -1 left wall, +1 right wall (0 when free)
	Hide01     float64 // F25: binary (v0.3.8): 0 visible .. 1 inside a crag
	Z          float64 // v1.1 depth lane 0 far .. 1 near (0 treated as mid)
	Gaze01     float64 // G93: 0 side view .. 1 full face-on stare
	PortalFade float64 // v1.1 G73: 0 solid .. 1 fully inside a wormhole
}

func stageAlpha(stage string) uint8 {
	switch stage {
	case "fry":
		return 190
	case "juvenile":
		return 228
	default:
		return 255
	}
}
