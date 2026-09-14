// v1.1 G68: the sand bed's shared topography — one deterministic relief
// curve used by BOTH the simulation (where flakes come to rest) and the
// renderer (where the bed is drawn), so food sits exactly on the dunes
// the viewer sees. Gentle swells and a shallow channel: real seabeds are
// never a ruler line.
package contract

import "math"

// SandRelief is the bed's height offset at world x (px, + = deeper).
func SandRelief(x float64) float64 {
	return math.Sin(x*0.0043+1.7)*6 +
		math.Sin(x*0.011+0.4)*3.5 +
		math.Sin(x*0.021+2.9)*2
}

// SandSurfaceY is the y of the sand surface at x in a tank of height h.
func SandSurfaceY(h, x float64) float64 {
	return h*FloorLineFrac + SandRelief(x)
}
