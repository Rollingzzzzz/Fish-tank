// G3.4: Agent base + Hub — goroutines, schedules, statuses, the event bus
// (buffered, non-blocking, drop-oldest) and the live/simulate mode policy
// including the 429 quota guard with silent 10-minute probes.
package agents

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/Rollingzzzzz/Fish-tank/internal/content"
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/Rollingzzzzz/Fish-tank/internal/llm"
)

// eventBuffer is the UI event channel capacity (C3: bounded or it doesn't merge).
const eventBuffer = 256

var agentIDs = []string{contract.AgentSpecies, contract.AgentWater, contract.AgentPattern}

// Hub owns the three agents, the event bus and the global failure policy.
type Hub struct {
	cfg    contract.Config
	store  *content.Store
	hooks  contract.AgentHooks
	events chan contract.Event
	newRNG func() *rand.Rand

	mu      sync.Mutex
	agents  map[string]*Agent
	simPerm bool // no key / network dead / key rejected: simulate until recreated
	simTemp bool // quota (429): temporary simulate for all agents
}

// NewHub wires the hub. An empty API key starts everyone in permanent
// simulate mode (D7: identical panel flow, tagged SIMULATED).
func NewHub(cfg contract.Config, store *content.Store, hooks contract.AgentHooks) *Hub {
	cfg.AgentFreq = contract.Clamp(cfg.AgentFreq, 0.5, 2)
	h := &Hub{
		cfg:    cfg,
		store:  store,
		hooks:  hooks,
		events: make(chan contract.Event, eventBuffer),
		agents: make(map[string]*Agent, 3),
		newRNG: func() *rand.Rand { return contract.RandSeed(time.Now().UnixNano()) },
	}
	for _, id := range agentIDs {
		a := &Agent{id: id, hub: h, trigger: make(chan struct{}, 1)}
		a.firstRun = dur(contract.AgentFirstRun[id])
		a.schedule = dur(contract.AgentSchedules[id] * cfg.AgentFreq)
		if a.schedule < time.Second {
			a.schedule = time.Second
		}
		h.agents[id] = a
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		h.simPerm = true
		log.Printf("agents: no API key configured - permanent simulate mode")
	}
	return h
}

func dur(seconds float64) time.Duration { return time.Duration(seconds * float64(time.Second)) }

// Events returns the agent-to-UI event stream.
func (h *Hub) Events() <-chan contract.Event { return h.events }

// Start spawns the three agent loops and the quota probe loop.
func (h *Hub) Start(ctx context.Context) {
	for _, id := range agentIDs {
		go h.agents[id].loop(ctx)
	}
	go h.quotaWatch(ctx)
}

// Trigger requests an immediate manual run ("Research New Species" button).
// The signal is dropped if one is already pending.
func (h *Hub) Trigger(id string) {
	h.mu.Lock()
	a := h.agents[id]
	h.mu.Unlock()
	if a == nil {
		return
	}
	select {
	case a.trigger <- struct{}{}:
	default:
	}
}

// SetSchedule overrides the first-run delay and repeat schedule of one agent.
// Production code relies on contract tuning only; this exists for cmd/demo
// fast-forward pacing (C2: demos are the acceptance tools).
func (h *Hub) SetSchedule(id string, first, schedule time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if a := h.agents[id]; a != nil {
		a.firstRun, a.schedule = first, schedule
	}
}

// emit delivers an event without ever blocking; when the buffer is full the
// oldest event is dropped (spec: capacity 256, non-blocking sends).
func (h *Hub) emit(ev contract.Event) {
	ev.At = time.Now().UTC().Format(time.RFC3339)
	h.mu.Lock()
	defer h.mu.Unlock()
	select {
	case h.events <- ev:
	default:
		select { // full: drop oldest, then retry once
		case <-h.events:
		default:
		}
		select {
		case h.events <- ev:
		default:
		}
	}
}

func (h *Hub) logf(agent, format string, args ...any) {
	h.emit(contract.Event{Kind: contract.EventLog, Agent: agent, Text: fmt.Sprintf(format, args...)})
}

func (h *Hub) emitArtifact(agent, id, summary string) {
	h.emit(contract.Event{Kind: contract.EventArtifact, Agent: agent, ArtifactID: id, Text: summary})
}

// Agent is one scheduled creator (species, water or pattern).
type Agent struct {
	id       string
	hub      *Hub
	firstRun time.Duration
	schedule time.Duration
	trigger  chan struct{}
	status   contract.AgentStatus
}

// loop runs until ctx is cancelled: first run after contract.AgentFirstRun,
// then every schedule; manual triggers jump the queue. Panics are contained.
func (a *Agent) loop(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("agents: %s run panicked: %v", a.id, r)
			a.setStatus(contract.StatusError)
		}
	}()
	a.setStatus(contract.StatusIdle)
	timer := time.NewTimer(a.firstRun)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		case <-a.trigger:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
		}
		a.runOnce(ctx)
		timer.Reset(a.schedule)
	}
}

func (a *Agent) runOnce(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("agents: %s run panicked: %v", a.id, r)
			a.setStatus(contract.StatusError)
		}
	}()
	switch a.id {
	case contract.AgentSpecies:
		a.runSpecies(ctx)
	case contract.AgentWater:
		a.runWater(ctx)
	case contract.AgentPattern:
		a.runPattern(ctx)
	}
}

func (a *Agent) setStatus(s contract.AgentStatus) {
	a.status = s
	a.hub.emit(contract.Event{Kind: contract.EventStatus, Agent: a.id, Status: s})
}

// stream calls the LLM, routing reasoning deltas to EventThought and content
// deltas to EventChunk (never mixed, C6).
func (a *Agent) stream(ctx context.Context, msgs []llm.Message) (string, error) {
	req := llm.Request{
		Endpoint: a.hub.cfg.Endpoint, APIKey: a.hub.cfg.APIKey,
		ProxyURL: a.hub.cfg.ProxyURL, Model: a.hub.cfg.Model, Messages: msgs,
	}
	return llm.Stream(ctx, req,
		func(delta string) {
			a.hub.emit(contract.Event{Kind: contract.EventChunk, Agent: a.id, Text: delta})
		},
		func(thought string) {
			a.hub.emit(contract.Event{Kind: contract.EventThought, Agent: a.id, Text: thought})
		})
}

// existingNames prefers the world hook (live fish) over the raw store list.
func (a *Agent) existingNames(kind string) []string {
	if h := a.hub.hooks; h.SpeciesNames != nil {
		if names := h.SpeciesNames(); len(names) > 0 {
			return names
		}
	}
	return a.hub.store.Names(kind)
}

// recentFingerprints prefers the hook, then the store registry.
func (a *Agent) recentFingerprints(n int) []string {
	if h := a.hub.hooks; h.RecentFingerprints != nil {
		if fps := h.RecentFingerprints(n); fps != nil {
			return fps
		}
	}
	return a.hub.store.RecentFingerprints(n)
}

// extractJSON slices the first '{' to the last '}' out of raw LLM output.
func extractJSON(raw string) (string, bool) {
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start < 0 || end <= start {
		return "", false
	}
	return raw[start : end+1], true
}
