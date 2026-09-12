// G3.3 + G3.4: agent framework, the three GLM agents and offline simulate
// generators.
//
// Purpose: runs Species/Water/Pattern agents as goroutines on fixed schedules
// (contract.AgentSchedules x Config.AgentFreq), streams GLM output as UI
// events, gates every artifact through the content uniqueness registry (C6)
// and falls back to local simulate generators when the API is unavailable.
//
// Owns: internal/agents/*.go. Imports only llm, content, contract (C2).
//
// Public API: NewHub, (*Hub).Events/Start/Trigger; simulate generators
// SimulateSpecies/SimulateWater/SimulatePlant/SimulateRecipe (deterministic
// per *rand.Rand seed); RandomStage.
//
// Invariants: every artifact is validated by the content store and passes the
// C6 gate (fingerprint + case-insensitive name) before it is written; exactly
// one repair round-trip on invalid or duplicate LLM output, then the simulate
// generator takes over; events are sent non-blocking on a 256-slot channel
// (oldest dropped); an agent goroutine can never crash the app (D4).
//
// Failure policy: APIKey empty or llm.ErrNetwork -> permanent simulate until
// the Hub is recreated; llm.ErrQuota -> temporary simulate for ALL agents with
// a silent real-mode probe every 10 minutes, automatic switch-back + log;
// ErrAuth/ErrHTTP -> error status for that run only.
//
// Extensions: a new agent = new run file + an entry in agentIDs + prompts; new
// artifact kinds extend the simulate generators and the matching agent only.
package agents
