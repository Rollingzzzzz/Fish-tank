// G3.4: live/simulate mode policy — the C6/D7 failure rules in one place:
// no key or ErrNetwork -> permanent simulate; ErrQuota (429) -> temporary
// simulate for all agents with a silent 10-minute real-mode probe and
// automatic switch-back; ErrAuth/ErrHTTP -> error status for that run only.
package agents

import (
	"context"
	"errors"
	"time"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
	"github.com/Rollingzzzzz/Fish-tank/internal/llm"
)

// quotaRetryInterval is how often a temporary-simulate hub probes real mode.
var quotaRetryInterval = 10 * time.Minute

// simulate reports whether every agent must use local generators right now.
func (h *Hub) simulate() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.simPerm || h.simTemp
}

// enterQuotaMode flips every agent to temporary simulate and starts probing.
func (h *Hub) enterQuotaMode() {
	h.mu.Lock()
	already := h.simTemp || h.simPerm
	h.simTemp = true
	h.mu.Unlock()
	if !already {
		h.logf("hub", "GLM quota exhausted (429) - all agents switch to temporary simulate mode")
	}
}

// enterPermanentSimulate disables live mode until the Hub is recreated.
func (h *Hub) enterPermanentSimulate(reason string) {
	h.mu.Lock()
	already := h.simPerm
	h.simPerm = true
	h.mu.Unlock()
	if !already {
		h.logf("hub", "live mode lost (%s) - permanent simulate mode", reason)
	}
}

// quotaWatch silently retries a trivial real request every 10 minutes while
// temporary simulate is active and flips everyone back on success.
func (h *Hub) quotaWatch(ctx context.Context) {
	t := time.NewTicker(quotaRetryInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		h.mu.Lock()
		active := h.simTemp && !h.simPerm
		h.mu.Unlock()
		if !active {
			continue
		}
		_, err := llm.Stream(ctx, h.probeRequest(), nil, nil)
		switch {
		case err == nil:
			h.mu.Lock()
			h.simTemp = false
			h.mu.Unlock()
			h.logf("hub", "GLM quota recovered - agents return to live mode")
		case errors.Is(err, llm.ErrAuth):
			h.enterPermanentSimulate("api key rejected")
		default:
			// still throttled or a network hiccup: keep waiting silently
		}
	}
}

func (h *Hub) probeRequest() llm.Request {
	return llm.Request{
		Endpoint: h.cfg.Endpoint, APIKey: h.cfg.APIKey, ProxyURL: h.cfg.ProxyURL,
		Model:    h.cfg.Model,
		Messages: []llm.Message{{Role: "user", Content: "Reply with the single word: pong"}},
	}
}

// handleLLMError applies the failure policy; it returns true when the caller
// should finish this run with the simulate generator instead.
func (a *Agent) handleLLMError(err error) bool {
	switch {
	case errors.Is(err, llm.ErrQuota):
		a.hub.enterQuotaMode()
		return true
	case errors.Is(err, llm.ErrNetwork):
		a.hub.enterPermanentSimulate("network unreachable")
		return true
	default: // ErrAuth / ErrHTTP: surface the error, retry next schedule
		a.setStatus(contract.StatusError)
		a.hub.logf(a.id, "request failed: %v", err)
		return false
	}
}
