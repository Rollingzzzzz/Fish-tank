// G0.3: agent→UI event bus types and world hooks (coordinator amendment, README §4).
// These decouple Lane B (agents) from Lane C (ui): both code against contract only.
package contract

// EventKind classifies agent→UI events.
type EventKind string

const (
	EventThought  EventKind = "thought"  // reasoning_content deltas (dimmed line)
	EventChunk    EventKind = "chunk"    // visible text deltas
	EventStatus   EventKind = "status"   // status change
	EventArtifact EventKind = "artifact" // artifact applied; Text = summary, ArtifactID = content id
	EventLog      EventKind = "log"      // shared tank log line
)

// AgentStatus is the lifecycle status of an agent run.
type AgentStatus string

const (
	StatusIdle      AgentStatus = "idle"
	StatusThinking  AgentStatus = "thinking"
	StatusWriting   AgentStatus = "writing"
	StatusDone      AgentStatus = "done"
	StatusError     AgentStatus = "error"
	StatusSimulated AgentStatus = "simulated"
)

// Event is one message on the agent→UI bus (buffered channel in the agent hub).
type Event struct {
	Kind       EventKind
	Agent      string      // AgentSpecies | AgentWater | AgentPattern
	Status     AgentStatus // valid when Kind == EventStatus
	Text       string      // streaming text / log line / artifact summary
	ArtifactID string      // content id, when Kind == EventArtifact
	At         string      // RFC3339
}

// PatternRequest describes one fish waiting for an age-driven repaint.
type PatternRequest struct {
	FishID  string
	Species Species
	Stage   string  // target stage: "fry" | "juvenile" | "adult" | "elder"
	OldPal  Palette // current palette (before/after previews)
	OldPat  Pattern
}

// AgentHooks is the world-facing callback set wired by game (G6.1).
// The agents package must nil-check every hook before calling.
type AgentHooks struct {
	SpeciesNames       func() []string                      // existing names (prompt context)
	RecentFingerprints func(n int) []string                 // last n registry hashes (prompt context)
	NextRepaints       func(max int) []PatternRequest       // consume up to max repaint requests
	ApplyRecipe        func(fishID string, r PatternRecipe) // apply a repaint to a fish
	SpawnEgg           func(speciesID string)               // spawn an egg of speciesID
}
