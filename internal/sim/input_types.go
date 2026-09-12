// N10/N11: player-input value types (split from world.go, line ceiling).
// These are the frozen v0.3 surfaces the game loop fills each frame.
package sim

import "github.com/Rollingzzzzz/Fish-tank/internal/contract"

// HeldTreat describes the live treat the player is holding at the cursor
// (N11): fish see it wriggle and swarm just out of reach until it is dropped.
// Age is seconds since the grab (F26) — the scent cloud spreads as it is
// carried around, so distant fish catch the smell too.
type HeldTreat struct {
	Kind string
	Pos  contract.Vec2
	Age  float64
}
