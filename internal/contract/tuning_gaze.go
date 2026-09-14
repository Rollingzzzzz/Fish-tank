// v1.2 G93: glass gaze tuning (split from tuning.go for the line ceiling).
package contract

// G93 glass gaze: a school fish now and then swims up to the glass, turns
// front-on and STARES at the viewer — the aquarist's favourite.
const (
	GazeCDMin       = 50.0  // s between gazes per fish at mid curiosity
	GazeCDMax       = 110.0 // s
	GazeMaxGazers   = 2     // the whole tank stares this rarely — two at once, tops
	GazeStareMin    = 3.0   // s of staring
	GazeStareMax    = 6.0
	GazeBlendInRate = 1.4  // /s — the face-on crossfade takes ~0.7 s
	GazeBlendOut    = 3.0  // /s — interrupted gazes fold away fast
	GazeSpeedMax    = 12.0 // px/s while staring (a hover, not a glide)
	GazeNearZ       = 0.90 // the stare happens at the near lane
	GazeArrivePx    = 34.0 // reach the spot, then turn
)
