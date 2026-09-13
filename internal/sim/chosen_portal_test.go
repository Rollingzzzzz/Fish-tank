// v1.1 G73: the Chosen's wormhole — rare by contract, far-reaching, fully
// staged. She must never be in two places at once, never blink without a
// door, and the pass itself is short theater (open, enter, exit).
package sim

import (
	"testing"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

func TestChosenPortalRareAndSound(t *testing.T) {
	cfg := contract.Config{MaxFish: 10, DaySeconds: 60}
	w := NewWorld(1720, 720, cfg, []*contract.Species{
		chosenTestSpecies(), cappedNormalSpecies("pt-neon", false),
	}, nil, nil)
	w.SeedRng(31)
	w.Update(0.05, Input{}) // she arrives

	var ch *Fish
	for _, f := range w.fishes {
		if f.Sp.Role == contract.RoleChosen {
			ch = f
		}
	}
	if ch == nil {
		t.Fatal("no Chosen in the tank")
	}

	const dt = 0.05
	events, active, maxFade := 0, 0.0, 0.0
	prevPos := ch.Pos
	jumps := []float64{}
	for i := 0; i < 60*400; i++ { // 400 s
		prevPhase := ch.portalPh
		w.Update(dt, Input{})
		if prevPhase == 0 && ch.portalPh == 1 {
			events++
			active = 0
		}
		if ch.portalPh != 0 {
			active += dt
			if active > 3.5 {
				t.Fatalf("a wormhole pass dragged on %.1f s", active)
			}
		}
		if d := hyp2(sub(ch.Pos, prevPos)); d > w.W*0.3 {
			jumps = append(jumps, d)
		}
		prevPos = ch.Pos
		if fd := ch.PortalFade01(); fd > maxFade {
			maxFade = fd
		}
		if ch.portalPh == 0 && ch.PortalFade01() != 0 {
			t.Fatal("she fades while no door is open")
		}
	}
	t.Logf("portal: %d events in 400 s, %d jumps, max fade %.2f", events, len(jumps), maxFade)
	if events < 1 {
		t.Fatal("she never opened a door in 400 s — the superpower is dead")
	}
	if events > 4 {
		t.Fatalf("%d passes in 400 s — no longer rare", events)
	}
	if len(jumps) < events {
		t.Fatalf("only %d big jumps for %d passes — the wormhole does not travel", len(jumps), events)
	}
	for _, j := range jumps {
		if j > w.W*0.95 {
			t.Fatalf("a %.0f px jump — that is out of the tank, not through a door", j)
		}
	}
	if maxFade < 0.99 {
		t.Fatalf("max fade %.2f — she only half steps through", maxFade)
	}
}
