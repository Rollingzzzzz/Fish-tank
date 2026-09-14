// G3.4: fake GLM SSE server for the demo — streams canned species/water/
// pattern answers in three content chunks plus one reasoning delta. The
// species endpoint deliberately repeats its first design so the second
// "Research New Species" run demonstrates the C6 repair-then-simulate path.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

type demoLLM struct {
	mu           sync.Mutex
	speciesCalls int
}

func newDemoLLM() *demoLLM { return &demoLLM{} }

// canned species: the design the demo repeats to trigger the duplicate gate
// (distinct from every seed species so run 1 passes, run 2 gets rejected).
const demoSpeciesJSON = `{"name":"Demo Emberfin","latin":"Demothus ember","size":0.95,"width":1.05,
 "fin":1.3,"tail":0.9,"palette":{"body":"#20e0ff","belly":"#a0f8ff","accent":"#ff2060","glow":"#60f0ff"},
 "pattern":{"type":"wave","density":0.35,"size":0.6},
 "behavior":{"speed":1.2,"schooling":0.7,"curiosity":0.8,"skittish":0.3,"depth":0.25,"nightActive":false},
 "note":"An electric wave-coat born inside the demo server."}`

const demoWaterBundleJSON = `{"water":{"name":"Demo Aurora Trench","topColor":"#0a2a1f",
 "bottomColor":"#02100a","accent":"#ffb02a","rays":0.6,"caustics":0.35,"bubbles":0.55,
 "plantPalette":["#7dff3d","#3dd6ff"],
 "event":{"name":"Aurora Pulse","kind":"current","durationSec":26,"note":"A green current sweeps the trench."}},
 "plant":{"name":"Demo Glowfern","fronds":6,"height":0.4,"width":0.8,"curve":0.75,"sway":0.45,
 "colors":["#5c0f7d","#a619c6","#e07bff"],"glow":0.7}}`

func demoRecipeJSON(stage string) string {
	titles := map[string]string{"fry": "Fry", "juvenile": "Juvenile", "adult": "Adult", "elder": "Elder"}
	return fmt.Sprintf(`{"stage":"%s","name":"Demo %s Coat","satMul":1.1,"alphaMul":0.8,"glowMul":1.2,
 "pattern":{"type":"wave","density":0.5,"size":0.45},"note":"A fresh demo coat for the new stage."}`,
		stage, titles[stage])
}

func (f *demoLLM) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sys, user := parseDemoMessages(r)
		var payload string
		switch {
		case strings.Contains(sys, "Species Bot"):
			payload = f.speciesPayload()
		case strings.Contains(sys, "Water Bot"):
			payload = demoWaterBundleJSON
		case strings.Contains(sys, "Pattern Bot"):
			payload = demoRecipeJSON(demoStage(user))
		default: // probe or unknown caller
			payload = "pong"
		}
		streamDemoAnswer(w, payload)
	})
}

// speciesPayload repeats the first design twice (initial run + repair) so the
// second manual run is rejected by the gate and simulate takes over.
func (f *demoLLM) speciesPayload() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.speciesCalls++
	if f.speciesCalls <= 3 {
		return demoSpeciesJSON
	}
	return strings.Replace(demoSpeciesJSON, "Demo Emberfin",
		fmt.Sprintf("Demo Emberfin %d", f.speciesCalls), 1)
}

func parseDemoMessages(r *http.Request) (sys, user string) {
	raw, _ := io.ReadAll(r.Body)
	var body struct {
		Messages []struct{ Role, Content string } `json:"messages"`
	}
	if json.Unmarshal(raw, &body) != nil {
		return "", ""
	}
	for _, m := range body.Messages {
		switch m.Role {
		case "system":
			sys = m.Content
		case "user":
			user = m.Content
		}
	}
	return sys, user
}

func demoStage(user string) string {
	for _, s := range []string{"fry", "juvenile", "elder"} {
		if strings.Contains(user, s) {
			return s
		}
	}
	return "adult"
}

// streamDemoAnswer emits one thought delta, three content chunks and [DONE].
func streamDemoAnswer(w http.ResponseWriter, payload string) {
	w.Header().Set("Content-Type", "text/event-stream")
	fmt.Fprint(w, demoSSE("reasoning_content", "mulling over neon palettes..."))
	third := len(payload) / 3
	if third == 0 {
		third = len(payload)
	}
	parts := []string{payload[:third], payload[third : 2*third], payload[2*third:]}
	for _, p := range parts {
		if p == "" {
			continue
		}
		fmt.Fprint(w, demoSSE("content", p))
	}
	fmt.Fprint(w, "data: [DONE]\n\n")
}

func demoSSE(field, text string) string {
	b, _ := json.Marshal(map[string]any{
		"choices": []any{map[string]any{"delta": map[string]any{field: text}}},
	})
	return "data: " + string(b) + "\n\n"
}
