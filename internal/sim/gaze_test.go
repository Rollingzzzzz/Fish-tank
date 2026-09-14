// v1.2 G93: the glass gaze — the behavior contract. A school fish now and
// then swims to a front spot, turns face-on and stares: the tank looks
// back. The tests pin the SMART envelope — gazes actually happen, the
// whole tank never stares at once, a stare is a hover at the near lane,
// and real life (a startle) interrupts it instantly.
package sim

import (
	"testing"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

func gazeWorld(t *testing.T) *World {
	t.Helper()
	cfg := contract.Config{MaxFish: 12, DaySeconds: 60}
	w := NewWorld(1720, 720, cfg, []*contract.Species{
		cappedNormalSpecies("gz-neon", false), cappedNormalSpecies("gz-dart", true),
	}, nil, nil)
	w.SeedRng(31)
	w.SetZones([]contract.Zone{{Owner: "chosen",
		Center: v2(w.W*0.30, w.H*0.78), Radius: contract.ZoneRadius}})
	w.Update(1/60.0, Input{})
	return w
}

func TestGlassGaze(t *testing.T) {
	w := gazeWorld(t)
	const dt = 1 / 60.0
	gazes := map[string]int{}
	maxConcurrent := 0
	stareFrames := 0
	settled := map[string]int{} // frames since the blend first went full
	for i := 0; i < 60*900; i++ { // 15 min of tank time
		w.Update(dt, Input{})
		conc := 0
		for _, f := range w.fishes {
			if f.Sp.Role != contract.RoleNormal {
				continue
			}
			if f.gazePhase > 0 {
				conc++
				if f.gazePhase >= 2 {
					gazes[f.ID]++
					if f.gazeBlend > 0.9 {
						stareFrames++
						settled[f.ID]++
						// the arrival speed GLIDES down to the hover ceiling
						// (G88); the hover contract applies once settled
						if settled[f.ID] > 45 {
							if sp := hyp2(f.Vel); sp > contract.GazeSpeedMax+4 {
								t.Fatalf("a staring fish glides at %.1f px/s — the stare is a hover", sp)
							}
							if f.z < 0.70 {
								t.Fatalf("a staring fish sat at depth %.2f — the stare is at the glass", f.z)
							}
						}
					} else {
						settled[f.ID] = 0
					}
				}
			}
		}
		if conc > maxConcurrent {
			maxConcurrent = conc
		}
		if conc > contract.GazeMaxGazers {
			t.Fatalf("%d fish stared at once — the cap is %d", conc, contract.GazeMaxGazers)
		}
	}
	total := 0
	for _, n := range gazes {
		total += n
	}
	if total < 12 {
		t.Fatalf("the glass was ignored: %d stare frames across the whole tank in 15 min", total)
	}
	if maxConcurrent == 0 {
		t.Fatal("nobody ever gazed")
	}
	t.Logf("gaze ok: %d gaze-frames on %d fish, max concurrent %d, %d full-stare frames",
		total, len(gazes), maxConcurrent, stareFrames)
}

func TestGazeInterruptedByStartle(t *testing.T) {
	w := gazeWorld(t)
	const dt = 1 / 60.0
	var starer *Fish
	for _, f := range w.fishes {
		if f.Sp.Role == contract.RoleNormal {
			starer = f
			break
		}
	}
	w.DebugForceGaze(starer)
	blendFull := false
	for i := 0; i < 60*30 && !blendFull; i++ {
		w.Update(dt, Input{})
		blendFull = starer.gazePhase == 2 && starer.gazeBlend > 0.9
	}
	if !blendFull {
		t.Fatal("the forced gaze never reached the full face-on pose")
	}
	// a startle bolt right at her nose — real life interrupts the stare
	in := Input{MouseActive: true, MouseSpeed: 1500, MouseX: starer.Pos.X, MouseY: starer.Pos.Y}
	cleared := false
	for i := 0; i < 40 && !cleared; i++ {
		w.Update(dt, in)
		cleared = starer.gazeBlend < 0.3
	}
	if !cleared {
		t.Fatalf("a startled fish kept staring (blend %.2f) — the stare must fold away fast", starer.gazeBlend)
	}
	if starer.gazeCD <= 0 {
		t.Fatal("after an interrupted gaze the cooldown never re-rolls")
	}
}
