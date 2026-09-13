// v1.1: -probe — the evidence pass for the ambient features (G39–G47).
// It reuses the -shot capture plumbing with a manifest-free script: forces a
// titan visit, stages floor critters, flips the clock to night, and freezes
// one PNG per probe point together with a TPS/census info line. The uiaudit
// gate never audits probe output (no manifest.json is written).
package game

// StartProbe arms the -probe evidence run (main.go flag `-probe dir`).
func (g *Game) StartProbe(dir string) {
	g.shot = &shotRun{dir: dir, probe: true, caps: map[int]string{},
		pending: map[string][]byte{}, pendInfo: map[string]string{}}
	g.shot.buildProbe()
}

// buildProbe assembles the probe timeline (60 tps):
//  1. stage six floor critters immediately — capture at 8 s (day light)
//  2. force the pod's arrival at 10 s — five visit captures (40/65/90/115/140 s)
//     across the roaming dwell; the area metric takes the best pose (the
//     giant's share depends on swim direction, so one frame proves the
//     "fills a tenth of the tank" reach). Each capture re-pins the clock to
//     mid-day first — the 60 s day/night cycle would otherwise drift the
//     late frames into night.
//  3. pin the clock to mid-night at 144 s — capture at 148 s (dimmed scene);
//     the trailing idle step keeps the driver alive until the last capture
func (s *shotRun) buildProbe() {
	s.steps = []smokeStep{
		{until: 8 * 60, act: func(g *Game) { g.world.DebugStageCritters(6) }},
	}
	s.caps[8*60+2] = "10-critters.png"
	s.steps = append(s.steps,
		smokeStep{until: 10 * 60, act: func(g *Game) { g.world.DebugForceTitanVisit() }})
	dayCaps := []struct {
		act  int
		name string
	}{{38, "11-titan.png"}, {63, "13-titan.png"}, {88, "14-titan.png"}, {113, "15-titan.png"}, {138, "16-titan.png"}}
	for i, c := range dayCaps {
		pin := 8 * 60 // first pin rides the visit-act step
		if i > 0 {
			pin = (c.act - 2) * 60
		}
		s.steps = append(s.steps, smokeStep{until: pin, act: func(g *Game) { g.world.DebugSetDay() }})
		s.caps[c.act*60] = c.name
	}
	s.steps = append(s.steps,
		smokeStep{until: 144 * 60, act: func(g *Game) { g.world.DebugSetNight() }},
		smokeStep{until: 149 * 60})
	s.caps[148*60] = "12-night.png"
}
