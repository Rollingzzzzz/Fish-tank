// v1.1 split (line ceiling): the feeding economy — a full meal and the
// teeming mites' snack scale.
package sim

// eat is a full meal: the belly resets (flakes, treats, critters).
func (f *Fish) eat() {
	f.Satiety = 1
	f.EatFlash = 0.6
	f.Energy = minF(1, f.Energy+0.08)
	f.bites++
}

// snack is a partial meal — the teeming mites stir the school without
// feeding it: at triple cadence each mite carries a third of a meal, so
// the total nutrition per minute is exactly what one full mite-meal in
// three used to give (G88, owner: activity, not calories — the ecosystem
// balance must not move).
func (f *Fish) snack(k float64) {
	f.Satiety = minF(1, f.Satiety+k)
	f.EatFlash = 0.6
	f.Energy = minF(1, f.Energy+0.08*k)
	f.bites++
}
