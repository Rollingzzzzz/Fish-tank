// N8: treat motion and consumption tests. Worlds/fish come from the shared
// helpers in fish_test.go (same package); ticks call tickTreats directly so
// fish stay pinned where the test placed them.
package sim

import (
	"strings"
	"testing"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

func TestTreatWormNibbledAndDevoured(t *testing.T) {
	w := testWorld(t, testSpecies(0.5), 1)
	f := w.fishes[0]
	f.Pos = v2(400, 300)
	f.Vel = v2(0, 0)
	f.Satiety = 0.5 // hungry, will bite
	f.bodyLen = 200 // generous mouth reach; geometry stays pinned
	w.DropTreat(contract.TreatWorm, v2(401, 299))
	if got := len(w.Treats()); got != 1 {
		t.Fatalf("treats = %d, want 1", got)
	}
	// dt steps; satiety decays here exactly as advance() would, so the
	// >0.98 full-belly gate re-opens between nibbles as in the real loop.
	const dt = 1 / 60.0
	for i := 0; i < 60; i++ {
		f.Satiety = maxF(0, f.Satiety-dt/40)
		w.tickTreats(dt)
	}
	tr := w.Treats()[0]
	if tr.BitesLeft >= 4 {
		t.Fatal("worm was never bitten in 1 s next to a hungry fish")
	}
	if f.Satiety < 0.9 {
		t.Fatalf("fish satiety did not rise: %v", f.Satiety)
	}
	if w.Care < contract.CareFeedScore-1e-9 {
		t.Fatalf("care did not accrue per bite: %v", w.Care)
	}
	// two more seconds: satiety-gated nibbles finish the worm off
	for i := 0; i < 120; i++ {
		f.Satiety = maxF(0, f.Satiety-dt/40)
		w.tickTreats(dt)
	}
	if len(w.Treats()) != 0 {
		t.Fatalf("worm should be devoured after 3 s, %d left", len(w.Treats()))
	}
	if w.Care < 4*contract.CareFeedScore-1e-9 {
		t.Fatalf("expected 4 bites of care, got %v", w.Care)
	}
	if n := len(w.Log); n == 0 || !strings.Contains(w.Log[n-1].Text, "devours the worm") {
		t.Fatalf("missing devour log line, last=%+v", w.Log[n-1])
	}
}

func TestTreatBugReachesSurfaceBand(t *testing.T) {
	w := testWorld(t, testSpecies(0.5), 1)
	w.fishes = nil // nothing may eat the bug mid-flight
	w.DropTreat(contract.TreatBug, v2(400, 500))
	for i := 0; i < 480; i++ { // 8 s is plenty for the 80 px/s ascent
		w.tickTreats(1 / 60.0)
	}
	tr := w.Treats()[0]
	if tr.Pos.Y < bugBandLo-2 || tr.Pos.Y > bugBandHi+2 {
		t.Fatalf("bug settled at y=%.1f, want inside [%d,%d]", tr.Pos.Y, int(bugBandLo), int(bugBandHi))
	}
	if s := hyp2(tr.Vel); s > bugMaxSpeed+1e-9 {
		t.Fatalf("bug speed %f exceeds cap %f", s, bugMaxSpeed)
	}
}

func TestTreatTTLDissolves(t *testing.T) {
	w := testWorld(t, testSpecies(0.5), 1)
	w.fishes = nil
	w.DropTreat(contract.TreatChicken, v2(200, 200))
	w.Treats()[0].Age = treatTTLSec - 0.1
	for i := 0; i < 12; i++ { // 0.2 s later the TTL has passed
		w.tickTreats(1 / 60.0)
	}
	if len(w.Treats()) != 0 {
		t.Fatalf("expired treat not removed, %d left", len(w.Treats()))
	}
	if n := len(w.Log); n == 0 || !strings.Contains(w.Log[n-1].Text, "slips away") {
		t.Fatalf("missing dissolution log line, last=%+v", w.Log[n-1])
	}
}

func TestTreatWormWiggleAdvancesPhase(t *testing.T) {
	w := testWorld(t, testSpecies(0.5), 1)
	w.fishes = nil
	w.DropTreat(contract.TreatWorm, v2(100, 550))
	tr := w.Treats()[0]
	p0 := tr.Phase
	for i := 0; i < 60; i++ { // 1 s
		w.tickTreats(1 / 60.0)
	}
	if d := tr.Phase - p0; d < 5.9 {
		t.Fatalf("worm phase advanced by %f in 1 s, want ~6", d)
	}
	if tr.Pos.Y < 550+12-1.5 { // sinking ~12 px/s
		t.Fatalf("worm sank only to y=%.1f after 1 s", tr.Pos.Y)
	}
}

// N8 acceptance: one worm dropped mid-tank among hungry fish triggers a real
// scramble — several DISTINCT fish bite (the >0.98 full-belly gate rotates
// the crowd through the treat) and the worm is devoured, not TTL-expired.
// The world is fully seeded (SeedRng) and free of competing food: mites are
// disarmed and no feeder runs, so every recorded bite is a worm bite.
func TestTreatScrambleManyFishDevourWorm(t *testing.T) {
	w := testWorld(t, testSpecies(0), 4) // SeedRng(1234) inside
	w.miteT = 1e9                        // no mite may steal a bite
	for i, f := range w.fishes {
		f.Satiety = 0.2 // starving — everyone wants the worm
		f.Pos = v2(250+float64(i%2)*300, 250+float64(i/2)*100)
		f.Vel = v2(0, 0)
	}
	w.DropTreat(contract.TreatWorm, v2(400, 300))
	if got := len(w.Treats()); got != 1 {
		t.Fatalf("treats = %d, want 1", got)
	}
	const dt = 1 / 30.0
	for i := 0; i < int(10/dt); i++ { // 10 s scramble soak
		w.Update(dt, Input{})
	}
	if len(w.Treats()) != 0 {
		t.Fatalf("worm not consumed in 10 s: %d left, bites left %d",
			len(w.Treats()), w.Treats()[0].BitesLeft)
	}
	distinct, total := 0, 0
	for _, f := range w.fishes {
		if f.bites > 0 {
			distinct++
		}
		total += f.bites
	}
	if distinct < 2 {
		t.Fatalf("no scramble: only %d distinct fish bit the worm", distinct)
	}
	if total != defaultBites(contract.TreatWorm) {
		t.Fatalf("bite accounting off: %d total bites, want %d", total,
			defaultBites(contract.TreatWorm))
	}
	if w.Care < float64(defaultBites(contract.TreatWorm))*contract.CareFeedScore-1e-9 {
		t.Fatalf("care did not accrue per bite: %v", w.Care)
	}
	devoured := false
	for _, e := range w.Log { // creatures may log after the feast — scan, not tail
		if strings.Contains(e.Text, "devours the worm") {
			devoured = true
		}
	}
	if !devoured {
		t.Fatal("missing devour log line — the worm expired instead")
	}
}
