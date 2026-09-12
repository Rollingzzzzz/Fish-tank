// F25: door-to-door transit acceptance — a fish enters one volcanic door,
// hides inside the crag, and bursts out of a DIFFERENT door of the same crag.
package sim

import (
	"strings"
	"testing"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

func transitWorld(t *testing.T, n int) *World {
	t.Helper()
	w := testWorld(t, testSpecies(0), n)
	w.miteT = 1e9
	holes := []contract.Hole{
		{Center: v2(200, 600), Radius: 40, Rock: 0},
		{Center: v2(260, 480), Radius: 36, Rock: 0},
		{Center: v2(320, 380), Radius: 36, Rock: 0},
		{Center: v2(1000, 600), Radius: 40, Rock: 1},
		{Center: v2(1060, 480), Radius: 36, Rock: 1},
		{Center: v2(1120, 380), Radius: 36, Rock: 1},
	}
	w.SetHoles(holes)
	zs := make([]contract.Zone, len(holes))
	for i, hl := range holes {
		zs[i] = contract.Zone{Center: hl.Center, Radius: maxF(70, hl.Radius*1.8), Owner: "cave"}
	}
	w.SetZones(zs) // tired fish shelter at the doors — visitors for the soak
	return w
}

func TestForcedTransitThroughCrag(t *testing.T) {
	w := transitWorld(t, 3)
	f := w.fishes[0]
	f.Pos = v2(150, 590) // parked by the low-left door
	f.Vel = v2(0, 0)
	if !w.DebugTransit() {
		t.Fatal("DebugTransit refused to start")
	}
	exit := f.outHole.Center
	if exit == f.inHole.Center {
		t.Fatal("exit door must differ from the entrance")
	}

	hiddenBy := -1.0
	visibleAfter := -1.0
	exitSpeed := 0.0
	var emergePos contract.Vec2
	cruise := f.maxSpeed(1 - w.dayFactor())
	wasInside := false
	for i := 1; i <= 60*8; i++ { // up to 8 s @60 Hz
		w.Update(1/60.0, Input{})
		if f.transitPh == 1 {
			wasInside = true
			if hiddenBy < 0 {
				hiddenBy = float64(i) / 60
			}
		}
		if f.transitPh == 2 && exitSpeed == 0 {
			exitSpeed = hyp2(f.Vel)
		}
		if wasInside && !f.transiting && visibleAfter < 0 {
			visibleAfter = float64(i) / 60
			emergePos = f.Pos
		}
	}
	if !wasInside {
		t.Fatal("fish never entered the crag")
	}
	if hiddenBy > 2.0 {
		t.Fatalf("fish not hidden within 2 s (took %.2f s)", hiddenBy)
	}
	if visibleAfter < 0 {
		t.Fatal("fish never re-emerged fully visible")
	}
	// it came out the OTHER side: at emergence it is near the exit door and
	// clearly not near the entrance
	if d := hyp2(sub(emergePos, exit)); d > 90 {
		t.Fatalf("emerged %.0f px from the exit door", d)
	}
	if dIn := hyp2(sub(emergePos, f.inHole.Center)); dIn < hyp2(sub(emergePos, exit)) {
		t.Fatalf("emerged near the entrance (%.0f px) — did not cross the crag", dIn)
	}
	if exitSpeed < 1.5*cruise {
		t.Fatalf("exit burst %.1f < 1.5× cruise %.1f", exitSpeed, cruise)
	}
	// the burst decays back to a normal cruise
	for i := 0; i < 60*3; i++ {
		w.Update(1/60.0, Input{})
	}
	if s := hyp2(f.Vel); s > 1.35*cruise {
		t.Fatalf("still bursting 3 s after exit: %.1f > 1.35×%.1f", s, cruise)
	}
}

func TestTransitChosenNeverVanishes(t *testing.T) {
	sp := testSpecies(0)
	ch := *sp
	ch.ID = "test-chosen"
	ch.Role = contract.RoleChosen
	w := testWorld(t, sp, 2)
	w.SetZones(nil)
	w.SetHoles([]contract.Hole{
		{Center: v2(300, 600), Radius: 40, Rock: 0},
		{Center: v2(360, 480), Radius: 36, Rock: 0},
	})
	she := newFish(&ch, 99, v2(300, 590), 8, w.nextID())
	w.fishes = append(w.fishes, she)
	//DebugTransit must pick a normal fish, never her
	if !w.DebugTransit() {
		t.Fatal("no fish available")
	}
	for _, f := range w.fishes {
		if f.Sp.Role == contract.RoleChosen && f.transiting {
			t.Fatal("the Chosen must never transit — she is always on camera")
		}
	}
	// and the scheduler never picks her either
	for i := 0; i < 60*60; i++ {
		w.Update(1/60.0, Input{})
		if she.transiting {
			t.Fatal("the scheduler sent the Chosen into a door")
		}
	}
}

func TestTransitFrequencySoak(t *testing.T) {
	w := transitWorld(t, 5)
	for i := 0; i < 60*60*10; i++ { // 10 sim-minutes @60 Hz
		w.Update(1/60.0, Input{})
	}
	hit := false
	for _, l := range w.Log {
		if strings.Contains(l.Text, "bursts out of another door") {
			hit = true
			break
		}
	}
	if !hit {
		t.Fatal("no door-to-door transit occurred in 10 minutes with 5 fish")
	}
}

func TestScareAbortsApproach(t *testing.T) {
	w := transitWorld(t, 1)
	f := w.fishes[0]
	f.Pos = v2(150, 590)
	w.DebugTransit()
	if !f.transiting || f.transitPh != 0 {
		t.Fatal("setup: transit not approaching")
	}
	f.flee(140, 580, 300)
	if f.transiting {
		t.Fatal("a scared fish must abandon the approach")
	}
}
