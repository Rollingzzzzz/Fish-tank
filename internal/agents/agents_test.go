// G3.3 + G3.4: shared test fixtures — a scriptable fake GLM SSE server, a
// fast hub constructor and event-drain helpers — plus the simulate property
// test (200 artifacts) and the event-buffer bound test.
package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Rollingzzzzz/Fish-tank/internal/content"
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// ---- fake GLM server -------------------------------------------------------

type fakeLLM struct {
	mu    sync.Mutex
	mode  string // "fresh" | "dup" | "bad" | "quota"
	calls int
}

func (f *fakeLLM) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func (f *fakeLLM) setMode(mode string) {
	f.mu.Lock()
	f.mode = mode
	f.mu.Unlock()
}

const dupSpeciesJSON = `{"name":"Dup Fish","latin":"Duplius fish","size":1,"width":1,"fin":1,"tail":1,
 "palette":{"body":"#8040ff","belly":"#c090ff","accent":"#ff40c0","glow":"#c060ff"},
 "pattern":{"type":"stripe","density":0.5,"size":0.5},
 "behavior":{"speed":1,"schooling":0.5,"curiosity":0.5,"skittish":0.5,"depth":0.5,"nightActive":true},
 "note":"Always the same fish."}`

// httptestServer mounts the fake LLM and returns its URL.
func httptestServer(t *testing.T, f *fakeLLM) string {
	t.Helper()
	srv := httptest.NewServer(f.handler())
	t.Cleanup(srv.Close)
	return srv.URL
}

// seedDupSpecies registers the exact design the fake server answers with in
// "dup" mode, so the C6 gate rejects the agent's first attempt.
func seedDupSpecies(t *testing.T, store *content.Store) *contract.Species {
	t.Helper()
	var sp contract.Species
	if err := json.Unmarshal([]byte(dupSpeciesJSON), &sp); err != nil {
		t.Fatalf("dup fixture: %v", err)
	}
	sp.ID = "dup-fish"
	if err := store.WriteSpecies(&sp); err != nil {
		t.Fatalf("seed dup species: %v", err)
	}
	return &sp
}

func (f *fakeLLM) speciesPayload() string {
	f.mu.Lock()
	mode, call := f.mode, f.calls
	f.mu.Unlock()
	switch mode {
	case "dup":
		return dupSpeciesJSON
	case "bad":
		return "I would love to help, but here is prose without any JSON at all."
	default: // fresh: unique name and rotating hue per call
		hue := 30 * (call % 11)
		return fmt.Sprintf(`{"name":"Fresh Fish %d","latin":"Freshus fish","size":1,"width":1,"fin":1,"tail":1,
 "palette":{"body":"%s","belly":"%s","accent":"%s","glow":"%s"},
 "pattern":{"type":"spot","density":%.2f,"size":0.5},
 "behavior":{"speed":1,"schooling":0.5,"curiosity":0.5,"skittish":0.5,"depth":0.5,"nightActive":0},
 "note":"Fish number %d from the fake server."}`,
			call, hslHex(float64(hue), 1, 0.5), hslHex(float64(hue)+30, 0.8, 0.7),
			hslHex(float64(hue)+170, 1, 0.55), hslHex(float64(hue)+190, 1, 0.6),
			0.3+0.05*float64(call%10), call)
	}
}

const waterBundleJSON = `{"water":{"name":"Test Aurora Trench","topColor":"#0b1e3a","bottomColor":"#03060f",
 "accent":"#7b4dff","rays":0.3,"caustics":0.5,"bubbles":0.4,"plantPalette":["#19c6a6","#7b4dff"],
 "event":{"name":"Test Pulse","kind":"glowWave","durationSec":24,"note":"Waves roll by."}},
 "plant":{"name":"Test Glowfern","fronds":5,"height":0.3,"width":1.0,"curve":0.6,"sway":0.5,
 "colors":["#0f7d5c","#19c6a6","#7bffe0"],"glow":0.6}}`

var stageTitles = map[string]string{"fry": "Fry", "juvenile": "Juvenile", "adult": "Adult", "elder": "Elder"}

func recipeJSON(stage string) string {
	return fmt.Sprintf(`{"stage":"%s","name":"Test %s Coat","satMul":1.1,"alphaMul":0.8,"glowMul":1.2,
 "pattern":{"type":"wave","density":0.5,"size":0.45},"note":"A test coat."}`, stage, stageTitles[stage])
}

// parseMessages extracts (system, lastUser) from the request body.
func parseMessages(r *http.Request) (sys, user string) {
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

func stageFrom(user string) string {
	for _, s := range []string{"fry", "juvenile", "elder"} {
		if strings.Contains(user, s) {
			return s
		}
	}
	return "adult"
}

// handler serves SSE responses routed by the system prompt marker.
func (f *fakeLLM) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.calls++
		mode := f.mode
		f.mu.Unlock()
		if mode == "quota" && f.callCount() <= 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprint(w, `{"error":"quota exceeded"}`)
			return
		}
		sys, user := parseMessages(r)
		var payload string
		switch {
		case strings.Contains(sys, "Species Bot"):
			payload = f.speciesPayload()
		case strings.Contains(sys, "Water Bot"):
			payload = waterBundleJSON
		case strings.Contains(sys, "Pattern Bot"):
			payload = recipeJSON(stageFrom(user))
		default: // probe or unknown caller
			payload = "pong"
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, sseLine("underwater thoughts", "reasoning_content"))
		third := len(payload) / 3
		for _, chunk := range []string{payload[:third], payload[third : 2*third], payload[2*third:]} {
			fmt.Fprint(w, sseLine(chunk, "content"))
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	})
}

func sseLine(text, field string) string {
	b, _ := json.Marshal(map[string]any{
		"choices": []any{map[string]any{"delta": map[string]any{field: text}}},
	})
	return "data: " + string(b) + "\n\n"
}

// ---- hub + event helpers ---------------------------------------------------

// newTestHub builds a hub whose agents run almost immediately and then sleep
// for a long time, so tests see exactly the runs they trigger.
func newTestHub(t *testing.T, endpoint, apiKey string, hooks contract.AgentHooks) (*Hub, *content.Store) {
	t.Helper()
	store, err := content.Load(t.TempDir())
	if err != nil {
		t.Fatalf("content.Load: %v", err)
	}
	h := NewHub(contract.Config{Endpoint: endpoint, APIKey: apiKey}, store, hooks)
	h.mu.Lock()
	for _, a := range h.agents {
		a.firstRun = 5 * time.Millisecond
		a.schedule = 10 * time.Second
	}
	h.mu.Unlock()
	return h, store
}

// waitForEvent drains events until one of the kind arrives or timeout hits.
func waitForEvent(t *testing.T, h *Hub, kind contract.EventKind, timeout time.Duration) (contract.Event, bool) {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case ev := <-h.Events():
			if ev.Kind == kind {
				return ev, true
			}
		case <-deadline:
			return contract.Event{}, false
		}
	}
}

// ---- tests -----------------------------------------------------------------

// G3.3 acceptance: 200 generated artifacts all pass store validation and the
// uniqueness gate (the generators consult Gate; Write registers, so every
// later artifact must differ from all earlier ones).
func TestSimulateGeneratorsProperty(t *testing.T) {
	st, err := content.Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	rng := contract.RandSeed(42)
	for i := 0; i < 50; i++ {
		sp := SimulateSpecies(rng, st)
		if err := st.ValidateSpecies(&sp); err != nil {
			t.Fatalf("species %d invalid: %v", i, err)
		}
		if err := st.WriteSpecies(&sp); err != nil {
			t.Fatalf("species %d write: %v", i, err)
		}
		w := SimulateWater(rng, st)
		if err := st.ValidateWater(&w); err != nil {
			t.Fatalf("water %d invalid: %v", i, err)
		}
		if err := st.WriteWater(&w); err != nil {
			t.Fatalf("water %d write: %v", i, err)
		}
		p := SimulatePlant(rng, st)
		if err := st.ValidatePlant(&p); err != nil {
			t.Fatalf("plant %d invalid: %v", i, err)
		}
		if err := st.WritePlant(&p); err != nil {
			t.Fatalf("plant %d write: %v", i, err)
		}
		r := SimulateRecipe(rng, RandomStage(rng), st)
		if err := st.ValidateRecipe(&r); err != nil {
			t.Fatalf("recipe %d invalid: %v", i, err)
		}
		if err := st.WriteRecipe(&r); err != nil {
			t.Fatalf("recipe %d write: %v", i, err)
		}
	}
	if n := len(st.Species()); n != 50 {
		t.Fatalf("species = %d, want 50 unique", n)
	}
	if n := len(st.Waters()); n != 50 {
		t.Fatalf("waters = %d, want 50 unique", n)
	}
}

// C3: the event channel is bounded; overflowing drops the oldest, never blocks.
func TestEventsDropOldest(t *testing.T) {
	h, _ := newTestHub(t, "http://unused.invalid", "key", contract.AgentHooks{})
	for i := 0; i < eventBuffer+44; i++ {
		h.emit(contract.Event{Kind: contract.EventLog, Text: fmt.Sprintf("e%d", i)})
	}
	if got := len(h.events); got != eventBuffer {
		t.Fatalf("buffer len = %d, want %d", got, eventBuffer)
	}
	first := <-h.Events()
	if first.Text != "e44" { // 44 oldest entries dropped
		t.Fatalf("oldest surviving event = %q, want e44", first.Text)
	}
}

// cancelAfter starts and later stops a hub; it fails the test on timeout.
func cancelAfter(t *testing.T, h *Hub, millis int) context.CancelFunc {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	h.Start(ctx)
	time.AfterFunc(time.Duration(millis)*time.Millisecond, cancel)
	return cancel
}
