// G3.4: Pattern agent — consumes up to 2 repaint requests per run, designs a
// stage-appropriate PatternRecipe for each (gate + repair + write) and applies
// it via the world hook. Empty queue: one random-stage simulate recipe,
// written but not applied.
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

// runPattern handles the repaint queue or produces one idle recipe.
func (a *Agent) runPattern(ctx context.Context) {
	reqs := a.nextRepaints(2)
	if len(reqs) == 0 {
		rng := a.hub.newRNG()
		r := SimulateRecipe(rng, RandomStage(rng), a.hub.store)
		a.commitRecipe(r, "", true)
		return
	}
	for _, req := range reqs {
		a.repaintOne(ctx, req)
	}
}

func (a *Agent) nextRepaints(max int) []contract.PatternRequest {
	if h := a.hub.hooks; h.NextRepaints != nil {
		return h.NextRepaints(max)
	}
	return nil
}

// repaintOne is the full flow for a single waiting fish.
func (a *Agent) repaintOne(ctx context.Context, req contract.PatternRequest) {
	if a.hub.simulate() {
		a.simulateRecipeFor(req)
		return
	}
	a.setStatus(contract.StatusThinking)
	msgs := []llm.Message{
		{Role: "system", Content: PatternSystemPrompt},
		{Role: "user", Content: PatternUserPrompt(req)},
	}
	raw, err := a.stream(ctx, msgs)
	if err != nil {
		if a.handleLLMError(err) {
			a.simulateRecipeFor(req)
		}
		return
	}
	msgs = append(msgs, llm.Message{Role: "assistant", Content: raw})
	attempt := raw
	for pass := 0; pass < 2; pass++ {
		r, conflicts, perr := a.completeRecipe(attempt, req.Stage)
		if perr == nil && conflicts == nil {
			a.commitRecipe(r, req.FishID, false)
			return
		}
		if pass == 0 {
			a.hub.logf(a.id, "recipe rejected (%s) - repair round-trip", failureSummary(perr, conflicts))
			msgs = append(msgs, llm.Message{Role: "user",
				Content: RepairPrompt("pattern recipe", failureSummary(perr, conflicts), conflicts)})
			a.setStatus(contract.StatusThinking)
			if attempt, err = a.stream(ctx, msgs); err != nil {
				if a.handleLLMError(err) {
					a.simulateRecipeFor(req)
				}
				return
			}
			continue
		}
		a.hub.logf(a.id, "recipe repair failed (%s) - simulate takes over", failureSummary(perr, conflicts))
		a.simulateRecipeFor(req)
		return
	}
}

// completeRecipe parses, validates and gates one recipe; the stage is forced
// to the requested stage so the repaint always matches the fish's life stage.
func (a *Agent) completeRecipe(raw, stage string) (r contract.PatternRecipe, conflicts []string, err error) {
	blob, ok := extractJSON(raw)
	if !ok {
		return r, nil, errors.New("no JSON object in answer")
	}
	if e := json.Unmarshal([]byte(blob), &r); e != nil {
		return r, nil, e
	}
	r.Stage = stage
	if strings.TrimSpace(r.Name) == "" {
		return r, nil, errors.New("empty recipe name")
	}
	r.ID = uniqueID(a.hub.store, content.KindPattern, slugWithHeadroom(stage+"-"+r.Name))
	if e := a.hub.store.ValidateRecipe(&r); e != nil {
		return contract.PatternRecipe{}, nil, e
	}
	if ok, conf := a.hub.store.GateRecipe(&r); !ok {
		return r, conf, nil
	}
	return r, nil, nil
}

// commitRecipe writes the recipe file; applyID non-empty also repaints the
// live fish via the world hook. idle recipes are write-only per spec.
func (a *Agent) commitRecipe(r contract.PatternRecipe, fishID string, idle bool) {
	a.setStatus(contract.StatusWriting)
	if err := a.hub.store.WriteRecipe(&r); err != nil {
		a.setStatus(contract.StatusError)
		a.hub.logf(a.id, "recipe write failed: %v", err)
		return
	}
	if !idle {
		if h := a.hub.hooks; h.ApplyRecipe != nil {
			h.ApplyRecipe(fishID, r)
		}
	}
	summary := fmt.Sprintf("Repaint recipe: %s (%s)", r.Name, r.Stage)
	if fishID != "" {
		summary = fmt.Sprintf("Repainted %s: %s (%s)", fishID, r.Name, r.Stage)
	}
	a.hub.emitArtifact(a.id, r.ID, summary)
	a.setStatus(contract.StatusDone)
}

func (a *Agent) simulateRecipeFor(req contract.PatternRequest) {
	a.setStatus(contract.StatusSimulated)
	a.commitRecipe(SimulateRecipe(a.hub.newRNG(), req.Stage, a.hub.store), req.FishID, false)
}
