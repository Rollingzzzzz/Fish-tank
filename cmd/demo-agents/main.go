// G3.4: cmd/demo-agents — proves the agent lane in isolation (C2): the three
// agents run concurrently against a fake GLM SSE server, streaming
// thought/chunk/status/artifact/log events to stdout. The demo exits by
// itself after ~20 s; nothing is drawn, this is the Lane B acceptance tool.
package main

import (
	"context"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"time"

	"github.com/Rollingzzzzz/Fish-tank/internal/agents"
	"github.com/Rollingzzzzz/Fish-tank/internal/content"
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// demoTiming paces the show; the integration test shrinks everything.
type demoTiming struct {
	firstScale float64 // scales contract.AgentFirstRun
	triggerAt  float64 // seconds until the manual "Research New Species" run
	total      float64 // demo lifetime in seconds
}

var defaultTiming = demoTiming{firstScale: 1, triggerAt: 13, total: 19}

func main() {
	root, err := os.MkdirTemp("", "neon-tank-demo-agents")
	if err != nil {
		fmt.Println("demo: create temp dir:", err)
		os.Exit(1)
	}
	defer os.RemoveAll(root)

	fmt.Println("NEON TANK demo-agents: 3 agents vs a fake GLM server (exits by itself)")
	fmt.Println("content root:", root)
	if err := runDemo(context.Background(), root, defaultTiming, os.Stdout); err != nil {
		fmt.Println("demo error:", err)
		os.Exit(1)
	}
}

// runDemo wires store + fake LLM + hub, prints the event stream and returns
// when the demo context expires.
func runDemo(parent context.Context, root string, t demoTiming, out io.Writer) error {
	store, err := content.Load(root)
	if err != nil {
		return err
	}
	if err := store.EnsureSeed(); err != nil {
		return err
	}

	fake := newDemoLLM()
	srv := httptest.NewServer(fake.handler())
	defer srv.Close()

	start := time.Now()
	ctx, cancel := context.WithTimeout(parent, time.Duration(t.total*float64(time.Second)))
	defer cancel()

	hooks := contract.AgentHooks{
		SpeciesNames:       func() []string { return store.Names(content.KindSpecies) },
		RecentFingerprints: func(n int) []string { return store.RecentFingerprints(n) },
		NextRepaints:       demoRepaints,
		ApplyRecipe: func(fishID string, r contract.PatternRecipe) {
			fmt.Fprintf(out, "[%5.1fs] world    applies recipe %s to fish %s\n", since(start), r.Name, fishID)
		},
		SpawnEgg: func(speciesID string) {
			fmt.Fprintf(out, "[%5.1fs] world    spawns an egg of %s\n", since(start), speciesID)
		},
	}

	hub := agents.NewHub(contract.Config{Endpoint: srv.URL, APIKey: "demo-key"}, store, hooks)
	for _, id := range []string{contract.AgentSpecies, contract.AgentWater, contract.AgentPattern} {
		first := time.Duration(contract.AgentFirstRun[id] * t.firstScale * float64(time.Second))
		hub.SetSchedule(id, first, time.Duration(contract.AgentSchedules[id]*float64(time.Second)))
	}
	go printEvents(ctx, hub.Events(), out, start)
	hub.Start(ctx)

	time.AfterFunc(time.Duration(t.triggerAt*float64(time.Second)), func() {
		fmt.Fprintf(out, "[%5.1fs] user     presses \"Research New Species\"\n", since(start))
		hub.Trigger(contract.AgentSpecies)
	})

	<-ctx.Done()
	time.Sleep(100 * time.Millisecond) // let the printer flush the tail
	printInventory(out, store, since(start))
	return nil
}

func since(start time.Time) float64 { return time.Since(start).Seconds() }

// printEvents renders the agent-to-UI bus as timestamped stdout lines.
func printEvents(ctx context.Context, ch <-chan contract.Event, out io.Writer, start time.Time) {
	for {
		select {
		case <-ctx.Done():
			return
		case ev := <-ch:
			text := ev.Text
			if ev.Kind == contract.EventStatus {
				text = string(ev.Status)
			}
			if ev.Kind == contract.EventArtifact {
				text = fmt.Sprintf("%s [%s]", ev.Text, ev.ArtifactID)
			}
			fmt.Fprintf(out, "[%5.1fs] %-7s %-8s %s\n", since(start), ev.Agent, ev.Kind, text)
		}
	}
}

// printInventory lists what landed in the content tree.
func printInventory(out io.Writer, store *content.Store, at float64) {
	fmt.Fprintf(out, "[%5.1fs] inventory: %d species, %d waters, %d plants, %d recipes, %d registry entries\n",
		at, len(store.Species()), len(store.Waters()), len(store.Plants()),
		len(store.Recipes()), store.RegistrySize())
}

// demoRepaints is the world-side repaint queue (consumed by the pattern agent).
var repaintQueue = []contract.PatternRequest{
	{FishID: "fish-demo-1", Stage: "juvenile", Species: contract.Species{Name: "Neon Emberfin"}},
	{FishID: "fish-demo-2", Stage: "elder", Species: contract.Species{Name: "Cyan Veilglow"}},
}

func demoRepaints(max int) []contract.PatternRequest {
	n := max
	if n > len(repaintQueue) {
		n = len(repaintQueue)
	}
	out := repaintQueue[:n]
	repaintQueue = repaintQueue[n:]
	return out
}
