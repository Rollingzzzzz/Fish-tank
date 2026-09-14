// v1.2 titan silk tuning (split from tuning.go for the line ceiling): the
// silver elders' slow, mesmerizing body undulation and their silk filament
// render — the beat only quickens when a strike is armed.
package contract

// G94 titan silk: the giants read HEAVY — their swim wave beats at about
// half the generic big-body rate while cruising, and the lunge burst
// (seekBonus ×10 → Speed01 ~1) naturally swings them into a fast flick.
const (
	TitanBeatCalm   = 1.6  // rad/s base beat drive at Speed01 0
	TitanBeatAttack = 9.0  // rad/s beat drive at Speed01 1 (the strike flick)
	TitanBeatMul    = 0.55 // × the generic big-body rate — the mesmerizing crawl
	TitanSilkCount  = 6    // white silk filaments: 2 head barbels + 4 body veils
	TitanSilkSegs   = 7    // segments per filament (the traveling wave)
)
