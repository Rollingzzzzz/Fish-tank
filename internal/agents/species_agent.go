// G3.4: Species agent — invents one fish per run: prompt -> streamed plan ->
// JSON extraction -> validation -> C6 gate -> one repair round-trip if needed
// -> write + egg-spawn hook + artifact event. Simulate mode mirrors the flow
// with the local generator.
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

// flexBool tolerates both JSON booleans and 0|1 integers for nightActive.
type flexBool bool

func (b *flexBool) UnmarshalJSON(p []byte) error {
	switch string(p) {
	case "true", `"true"`, "1":
		*b = true
	case "false", `"false"`, "0", "null":
		*b = false
	default:
		return fmt.Errorf("invalid boolean %s", p)
	}
	return nil
}

// speciesDraft mirrors the prompt schema; only nightActive needs flexing.
type speciesDraft struct {
	Name     string           `json:"name"`
	Latin    string           `json:"latin"`
	Size     float64          `json:"size"`
	Width    float64          `json:"width"`
	Fin      float64          `json:"fin"`
	Tail     float64          `json:"tail"`
	Palette  contract.Palette `json:"palette"`
	Pattern  contract.Pattern `json:"pattern"`
	Behavior struct {
		Speed       float64  `json:"speed"`
		Schooling   float64  `json:"schooling"`
		Curiosity   float64  `json:"curiosity"`
		Skittish    float64  `json:"skittish"`
		Depth       float64  `json:"depth"`
		NightActive flexBool `json:"nightActive"`
	} `json:"behavior"`
	Note string `json:"note"`
}

func (d speciesDraft) toSpecies() contract.Species {
	return contract.Species{
		Name:  strings.TrimSpace(d.Name),
		Latin: strings.TrimSpace(d.Latin),
		Role:  contract.RoleNormal, // agents only invent normal fish (FD9)
		Size:  d.Size, Width: d.Width, Fin: d.Fin, Tail: d.Tail,
		Palette: d.Palette,
		Pattern: d.Pattern,
		Behavior: contract.Behavior{
			Speed:       d.Behavior.Speed,
			Schooling:   d.Behavior.Schooling,
			Curiosity:   d.Behavior.Curiosity,
			Skittish:    d.Behavior.Skittish,
			Depth:       d.Behavior.Depth,
			NightActive: bool(d.Behavior.NightActive),
		},
		Note:   d.Note,
		Source: "species-agent",
	}
}

// runSpecies is the full panel flow for one species invention.
func (a *Agent) runSpecies(ctx context.Context) {
	if a.hub.simulate() {
		a.simulateSpeciesRun()
		return
	}
	a.setStatus(contract.StatusThinking)
	msgs := []llm.Message{
		{Role: "system", Content: SpeciesSystemPrompt},
		{Role: "user", Content: SpeciesUserPrompt(
			a.existingNames(content.KindSpecies), a.recentFingerprints(8))},
	}
	raw, err := a.stream(ctx, msgs)
	if err != nil {
		if a.handleLLMError(err) {
			a.simulateSpeciesRun()
		}
		return
	}
	// Attempt 1, then exactly one repair round-trip, then simulate takeover.
	msgs = append(msgs, llm.Message{Role: "assistant", Content: raw})
	attempt := raw
	for pass := 0; pass < 2; pass++ {
		sp, conflicts, perr := a.completeSpecies(attempt)
		if perr == nil && conflicts == nil {
			a.commitSpecies(sp)
			return
		}
		if pass == 0 {
			a.hub.logf(a.id, "output rejected (%s) - repair round-trip", failureSummary(perr, conflicts))
			msgs = append(msgs, llm.Message{Role: "user",
				Content: RepairPrompt("species", failureSummary(perr, conflicts), conflicts)})
			a.setStatus(contract.StatusThinking)
			if attempt, err = a.stream(ctx, msgs); err != nil {
				if a.handleLLMError(err) {
					a.simulateSpeciesRun()
				}
				return
			}
			continue
		}
		a.hub.logf(a.id, "repair failed (%s) - simulate generator takes over", failureSummary(perr, conflicts))
		a.simulateSpeciesRun()
		return
	}
}

// completeSpecies parses, ids, validates and gates the raw answer.
// perr != nil means invalid output; conflicts != nil means gate rejection.
func (a *Agent) completeSpecies(raw string) (sp contract.Species, conflicts []string, err error) {
	blob, ok := extractJSON(raw)
	if !ok {
		return sp, nil, errors.New("no JSON object in answer")
	}
	var d speciesDraft
	if e := json.Unmarshal([]byte(blob), &d); e != nil {
		return sp, nil, e
	}
	sp = d.toSpecies()
	if strings.TrimSpace(sp.Name) == "" {
		return sp, nil, errors.New("empty name")
	}
	sp.ID = uniqueID(a.hub.store, content.KindSpecies, slugWithHeadroom(sp.Name))
	if e := a.hub.store.ValidateSpecies(&sp); e != nil {
		return contract.Species{}, nil, e
	}
	// Species gate path: C6 uniqueness plus the FD8 lilac reservation, so a
	// lilac candidate feeds the repair loop instead of reaching the store.
	if ok, conf := a.hub.store.GateSpecies(&sp); !ok {
		return sp, conf, nil
	}
	return sp, nil, nil
}

// commitSpecies writes the artifact, asks the world for an egg and reports.
func (a *Agent) commitSpecies(sp contract.Species) {
	a.setStatus(contract.StatusWriting)
	if err := a.hub.store.WriteSpecies(&sp); err != nil {
		a.setStatus(contract.StatusError)
		a.hub.logf(a.id, "species write failed: %v", err)
		return
	}
	if h := a.hub.hooks; h.SpawnEgg != nil {
		h.SpawnEgg(sp.ID)
	}
	a.hub.emitArtifact(a.id, sp.ID,
		fmt.Sprintf("New species: %s (%s)", sp.Name, sp.Pattern.Type))
	a.setStatus(contract.StatusDone)
}

func (a *Agent) simulateSpeciesRun() {
	a.setStatus(contract.StatusSimulated)
	a.commitSpecies(SimulateSpecies(a.hub.newRNG(), a.hub.store))
}

// failureSummary renders parse/validation error or gate conflicts for logs.
func failureSummary(err error, conflicts []string) string {
	if err != nil {
		return err.Error()
	}
	if len(conflicts) > 0 {
		return "duplicate of " + strings.Join(conflicts, ",")
	}
	return "unknown"
}
