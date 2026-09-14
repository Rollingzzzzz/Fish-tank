// G3.4: agent flow tests against the fake GLM server — live species flow,
// duplicate -> repair -> simulate takeover, invalid JSON -> simulate, water
// and pattern flows, quota guard switch-back and permanent simulate modes.
package agents

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// startServer mounts the fake LLM and points the hub at it.
func startServer(t *testing.T, f *fakeLLM) string {
	t.Helper()
	srv := httptestServer(t, f)
	return srv
}

func TestSpeciesAgentLiveFlow(t *testing.T) {
	f := &fakeLLM{mode: "fresh"}
	endpoint := startServer(t, f)

	eggs := make(chan string, 8)
	hooks := contract.AgentHooks{SpawnEgg: func(id string) { eggs <- id }}
	h, store := newTestHub(t, endpoint, "test-key", hooks)
	defer cancelAfter(t, h, 1500)()
	h.Trigger(contract.AgentSpecies)

	ev, ok := waitForEvent(t, h, contract.EventArtifact, 5*time.Second)
	if !ok {
		t.Fatal("no species artifact event")
	}
	if !strings.HasPrefix(ev.Text, "New species:") {
		t.Fatalf("artifact text = %q", ev.Text)
	}
	if ev.ArtifactID == "" || store.SpeciesByID(ev.ArtifactID) == nil {
		t.Fatalf("artifact %q not in store", ev.ArtifactID)
	}
	select {
	case id := <-eggs:
		if id != ev.ArtifactID {
			t.Fatalf("egg species %q != artifact %q", id, ev.ArtifactID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("SpawnEgg hook not called")
	}
	if f.callCount() == 0 {
		t.Fatal("fake server never called")
	}
}

func TestSpeciesAgentDuplicateRepairThenSimulate(t *testing.T) {
	f := &fakeLLM{mode: "dup"}
	endpoint := startServer(t, f)

	h, store := newTestHub(t, endpoint, "test-key", contract.AgentHooks{})
	defer cancelAfter(t, h, 1500)()
	// Pre-register the exact design the server will answer with.
	dup := seedDupSpecies(t, store)

	h.Trigger(contract.AgentSpecies)
	ev, ok := waitForEvent(t, h, contract.EventArtifact, 5*time.Second)
	if !ok {
		t.Fatal("no artifact after repair + simulate takeover")
	}
	if ev.ArtifactID == dup.ID {
		t.Fatal("simulate generator must not reproduce the duplicate")
	}
	if n := len(store.Species()); n != 2 {
		t.Fatalf("species files = %d, want 2 (duplicate + simulated)", n)
	}
	// The repair round-trip must have been logged with the conflict hashes.
	if ev, ok := waitForEvent(t, h, contract.EventLog, 2*time.Second); ok {
		if !strings.Contains(ev.Text, "repair") && !strings.Contains(ev.Text, "takes over") {
			t.Fatalf("expected repair/simulate log, got %q", ev.Text)
		}
	}
}

func TestSpeciesAgentInvalidJSONThenSimulate(t *testing.T) {
	f := &fakeLLM{mode: "bad"}
	endpoint := startServer(t, f)
	h, store := newTestHub(t, endpoint, "test-key", contract.AgentHooks{})
	defer cancelAfter(t, h, 1500)()
	h.Trigger(contract.AgentSpecies)

	ev, ok := waitForEvent(t, h, contract.EventArtifact, 5*time.Second)
	if !ok {
		t.Fatal("bad JSON twice must still produce a simulated artifact")
	}
	if ev.ArtifactID == "" || store.SpeciesByID(ev.ArtifactID) == nil {
		t.Fatalf("simulated species %q missing", ev.ArtifactID)
	}
}

func TestWaterAgentWritesBothArtifacts(t *testing.T) {
	f := &fakeLLM{mode: "fresh"}
	endpoint := startServer(t, f)
	var eggCalls sync.Map // must stay empty: water agent never spawns
	hooks := contract.AgentHooks{SpawnEgg: func(id string) { eggCalls.Store(id, true) }}
	h, store := newTestHub(t, endpoint, "test-key", hooks)
	defer cancelAfter(t, h, 1500)()
	h.Trigger(contract.AgentWater)

	if _, ok := waitForEvent(t, h, contract.EventArtifact, 5*time.Second); !ok {
		t.Fatal("no water artifact event")
	}
	if _, ok := waitForEvent(t, h, contract.EventArtifact, 5*time.Second); !ok {
		t.Fatal("no plant artifact event")
	}
	if len(store.Waters()) != 1 || len(store.Plants()) != 1 {
		t.Fatalf("waters=%d plants=%d, want 1+1", len(store.Waters()), len(store.Plants()))
	}
	if _, called := eggCalls.Load("anything"); called {
		t.Fatal("water agent must not spawn eggs")
	}
}

func TestPatternAgentRepaints(t *testing.T) {
	f := &fakeLLM{mode: "fresh"}
	endpoint := startServer(t, f)

	applied := make(chan contract.PatternRecipe, 8)
	reqs := []contract.PatternRequest{
		{FishID: "fish-1", Stage: "juvenile"},
		{FishID: "fish-2", Stage: "elder"},
	}
	hooks := contract.AgentHooks{
		NextRepaints: func(max int) []contract.PatternRequest { return reqs },
		ApplyRecipe:  func(fishID string, r contract.PatternRecipe) { applied <- r },
	}
	h, store := newTestHub(t, endpoint, "test-key", hooks)
	defer cancelAfter(t, h, 1500)()
	h.Trigger(contract.AgentPattern)

	for i := 0; i < 2; i++ {
		select {
		case r := <-applied:
			if r.Stage != "juvenile" && r.Stage != "elder" {
				t.Fatalf("applied recipe stage %q", r.Stage)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("recipe %d not applied", i)
		}
	}
	if n := len(store.Recipes()); n != 2 {
		t.Fatalf("recipe files = %d, want 2", n)
	}
}

func TestQuotaGuardTemporarySimulateAndBack(t *testing.T) {
	f := &fakeLLM{mode: "quota"}
	endpoint := startServer(t, f)

	old := quotaRetryInterval
	quotaRetryInterval = 40 * time.Millisecond
	defer func() { quotaRetryInterval = old }()

	h, store := newTestHub(t, endpoint, "test-key", contract.AgentHooks{})
	defer cancelAfter(t, h, 3000)()
	h.Trigger(contract.AgentSpecies)

	// The 429 flips agents to temporary simulate; the run still yields an
	// artifact via the local generator.
	ev, ok := waitForEvent(t, h, contract.EventArtifact, 5*time.Second)
	if !ok {
		t.Fatal("quota run must still produce a simulated artifact")
	}
	if store.SpeciesByID(ev.ArtifactID) == nil {
		t.Fatalf("simulated species %q missing", ev.ArtifactID)
	}
	// The probe (3rd call: 429 twice, then pong) restores live mode + logs it.
	deadline := time.Now().Add(3 * time.Second)
	for {
		if e, ok := waitForEvent(t, h, contract.EventLog, 500*time.Millisecond); ok {
			if strings.Contains(e.Text, "quota recovered") {
				break
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("quota recovery log not observed")
		}
	}
	if f.callCount() < 3 {
		t.Fatalf("probe call count = %d, want >= 3", f.callCount())
	}
}

func TestPermanentSimulateOnNetworkError(t *testing.T) {
	h, store := newTestHub(t, "http://127.0.0.1:1/nope", "test-key", contract.AgentHooks{})
	defer cancelAfter(t, h, 1500)()
	h.Trigger(contract.AgentSpecies)
	ev, ok := waitForEvent(t, h, contract.EventArtifact, 5*time.Second)
	if !ok {
		t.Fatal("network failure must degrade to a simulated artifact")
	}
	if store.SpeciesByID(ev.ArtifactID) == nil {
		t.Fatal("simulated species missing")
	}
	if !h.simulate() {
		t.Fatal("ErrNetwork must enable permanent simulate")
	}
}

func TestNoKeyStartsPermanentSimulate(t *testing.T) {
	f := &fakeLLM{mode: "fresh"}
	endpoint := startServer(t, f)
	h, store := newTestHub(t, endpoint, "", contract.AgentHooks{})
	defer cancelAfter(t, h, 1500)()
	h.Trigger(contract.AgentSpecies)

	if _, ok := waitForEvent(t, h, contract.EventArtifact, 5*time.Second); !ok {
		t.Fatal("no-key hub must still produce artifacts (simulate)")
	}
	if len(store.Species()) != 1 {
		t.Fatalf("species = %d, want 1", len(store.Species()))
	}
	if f.callCount() != 0 {
		t.Fatalf("no-key hub must not call the LLM, called %d times", f.callCount())
	}
}
