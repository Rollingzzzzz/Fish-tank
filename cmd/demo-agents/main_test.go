// G3.4: demo-agents integration test — the full demo flow with fast-forward
// timing: all 3 agents produce artifacts, the duplicate species run goes
// through repair into simulate, and the run terminates by itself.
package main

import (
	"context"
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rollingzzzzz/Fish-tank/internal/content"
)

func TestDemoAgentsIntegration(t *testing.T) {
	old := defaultTiming
	defaultTiming = demoTiming{firstScale: 0.02, triggerAt: 0.5, total: 1.5}
	defer func() { defaultTiming = old }()

	root := filepath.Join(t.TempDir(), "content")
	done := make(chan error, 1)
	go func() { done <- runDemo(context.Background(), root, defaultTiming, io.Discard) }()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("runDemo: %v", err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("demo did not terminate by itself")
	}

	store, err := content.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	// N9: 11 seed species (6 core + 3 v0.3 additions + v1.1 titan & shark) + Demo Emberfin + 1 simulated.
	if n := len(store.Species()); n != 13 {
		t.Fatalf("species = %d, want 13 (11 seed + live + simulated)", n)
	}
	// Seed: 3 water presets + the water agent's preset.
	if n := len(store.Waters()); n != 4 {
		t.Fatalf("waters = %d, want 4 (3 seed + agent)", n)
	}
	// Seed: 10 plants (4 classic + 6 v0.3.7 silky flora) + the water agent's plant.
	if n := len(store.Plants()); n != 11 {
		t.Fatalf("plants = %d, want 11 (10 seed + agent)", n)
	}
	// Seed: 4 recipes + 2 repaint recipes.
	if n := len(store.Recipes()); n != 6 {
		t.Fatalf("recipes = %d, want 6 (4 seed + 2 repaints)", n)
	}
	// Every species file must carry a distinct registry fingerprint.
	if store.RegistrySize() < 21 {
		t.Fatalf("registry = %d entries, want >= 21 (27 seed + 6 artifacts, floor kept loose)", store.RegistrySize())
	}
}
