// G3.4: Water agent — one water preset, one plant and one coral per run,
// written to content/water/, content/plants/ and content/corals/, all gated
// and registered; no egg spawn. Simulate mode mirrors the flow with local
// generators.
package agents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Rollingzzzzz/Fish-tank/internal/content"
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/Rollingzzzzz/Fish-tank/internal/llm"
)

// waterBundle is the combined answer schema {"water":..., "plant":..., "coral":...}.
type waterBundle struct {
	Water contract.WaterPreset `json:"water"`
	Plant contract.PlantDesign `json:"plant"`
	Coral contract.CoralDesign `json:"coral"`
}

func (a *Agent) runWater(ctx context.Context) {
	if a.hub.simulate() {
		a.simulateWaterRun()
		return
	}
	a.setStatus(contract.StatusThinking)
	msgs := []llm.Message{
		{Role: "system", Content: WaterSystemPrompt},
		{Role: "user", Content: WaterUserPrompt(
			a.existingNames(content.KindWater), a.recentFingerprints(8))},
	}
	raw, err := a.stream(ctx, msgs)
	if err != nil {
		if a.handleLLMError(err) {
			a.simulateWaterRun()
		}
		return
	}
	msgs = append(msgs, llm.Message{Role: "assistant", Content: raw})
	attempt := raw
	for pass := 0; pass < 2; pass++ {
		w, p, c, unresolved := a.completeWater(attempt)
		if unresolved == nil {
			a.commitWater(w, p, c)
			return
		}
		if pass == 0 {
			a.hub.logf(a.id, "output rejected (%s) - repair round-trip", unresolved.Error())
			msgs = append(msgs, llm.Message{Role: "user",
				Content: RepairPrompt("water preset, plant and coral", unresolved.Error(), nil)})
			a.setStatus(contract.StatusThinking)
			if attempt, err = a.stream(ctx, msgs); err != nil {
				if a.handleLLMError(err) {
					a.simulateWaterRun()
				}
				return
			}
			continue
		}
		a.hub.logf(a.id, "repair failed (%s) - simulate generators take over", unresolved.Error())
		a.simulateWaterRun()
		return
	}
}

// completeWater parses, ids, validates and gates all three artifacts. It
// returns a non-nil error describing the first problem (validation or gate),
// so one repair round-trip covers every failure kind (C6 repair rule). A model
// answer without a coral section degrades gracefully (D4): only the coral is
// replaced by the local generator, the water and plant still commit.
func (a *Agent) completeWater(raw string) (contract.WaterPreset, contract.PlantDesign, contract.CoralDesign, error) {
	blob, ok := extractJSON(raw)
	if !ok {
		return contract.WaterPreset{}, contract.PlantDesign{}, contract.CoralDesign{}, errors.New("no JSON object in answer")
	}
	var b waterBundle
	if err := json.Unmarshal([]byte(blob), &b); err != nil {
		return contract.WaterPreset{}, contract.PlantDesign{}, contract.CoralDesign{}, err
	}
	w, p, c := b.Water, b.Plant, b.Coral
	w.Event = normalizeWaterEvent(w.Event)
	w.Source, p.Source, c.Source = "water-agent", "water-agent", "water-agent"
	if strings.TrimSpace(w.Name) == "" || strings.TrimSpace(p.Name) == "" {
		return w, p, c, errors.New("empty water or plant name")
	}
	if strings.TrimSpace(c.Name) == "" { // no coral section: simulate just it
		c = SimulateCoral(a.hub.newRNG(), a.hub.store)
	}
	w.ID = uniqueID(a.hub.store, content.KindWater, slugWithHeadroom(w.Name))
	p.ID = uniqueID(a.hub.store, content.KindPlant, slugWithHeadroom(p.Name))
	c.ID = uniqueID(a.hub.store, content.KindCoral, slugWithHeadroom(c.Name))
	if err := a.hub.store.ValidateWater(&w); err != nil {
		return w, p, c, err
	}
	if err := a.hub.store.ValidatePlant(&p); err != nil {
		return w, p, c, err
	}
	if err := a.hub.store.ValidateCoral(&c); err != nil {
		return w, p, c, err
	}
	if ok, conf := a.hub.store.Gate(content.KindWater, w.ID, w.Name,
		contract.Pattern{}, content.WaterPalette(&w)); !ok {
		return w, p, c, fmt.Errorf("water gate: %s", strings.Join(conf, ","))
	}
	if ok, conf := a.hub.store.Gate(content.KindPlant, p.ID, p.Name,
		contract.Pattern{}, content.PlantPalette(&p)); !ok {
		return w, p, c, fmt.Errorf("plant gate: %s", strings.Join(conf, ","))
	}
	if ok, conf := a.hub.store.GateCoral(&c); !ok {
		return w, p, c, fmt.Errorf("coral gate: %s", strings.Join(conf, ","))
	}
	return w, p, c, nil
}

// commitWater writes all three artifacts and emits one artifact event each.
func (a *Agent) commitWater(w contract.WaterPreset, p contract.PlantDesign, c contract.CoralDesign) {
	a.setStatus(contract.StatusWriting)
	if err := a.hub.store.WriteWater(&w); err != nil {
		a.setStatus(contract.StatusError)
		a.hub.logf(a.id, "water write failed: %v", err)
		return
	}
	a.hub.emitArtifact(a.id, w.ID, fmt.Sprintf("New water mood: %s", w.Name))
	if err := a.hub.store.WritePlant(&p); err != nil {
		a.setStatus(contract.StatusError)
		a.hub.logf(a.id, "plant write failed: %v", err)
		return
	}
	a.hub.emitArtifact(a.id, p.ID, fmt.Sprintf("New plant: %s", p.Name))
	if err := a.hub.store.WriteCoral(&c); err != nil {
		a.setStatus(contract.StatusError)
		a.hub.logf(a.id, "coral write failed: %v", err)
		return
	}
	a.hub.emitArtifact(a.id, c.ID, fmt.Sprintf("New coral: %s (%s)", c.Name, c.Kind))
	a.setStatus(contract.StatusDone)
}

func (a *Agent) simulateWaterRun() {
	a.setStatus(contract.StatusSimulated)
	rng := a.hub.newRNG()
	w, p, c := SimulateWater(rng, a.hub.store), SimulatePlant(rng, a.hub.store), SimulateCoral(rng, a.hub.store)
	w.Source, p.Source, c.Source = "water-agent", "water-agent", "water-agent"
	a.commitWater(w, p, c)
}

// normalizeWaterEvent drops an event shell without a valid kind (D4) so a
// half-filled event object cannot fail validation of the whole preset.
func normalizeWaterEvent(e *contract.WaterEvent) *contract.WaterEvent {
	if e == nil {
		return nil
	}
	switch e.Kind {
	case "bubbleStorm", "glowWave", "current", "calm":
		return e
	default:
		return nil
	}
}
