// G3.2: GLM SSE streaming client — POST + Server-Sent-Events parsing with
// typed errors (ErrAuth/ErrQuota/ErrNetwork/ErrHTTP). Stdlib only (D8).
package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// Typed error wrappers — callers classify with errors.Is.
var (
	ErrAuth    = errors.New("llm: authentication rejected")
	ErrQuota   = errors.New("llm: quota exhausted")
	ErrNetwork = errors.New("llm: network failure")
	ErrHTTP    = errors.New("llm: http error")
)

// Frozen defaults (C6).
const (
	DefaultEndpoint = "https://api.z.ai/api/coding/paas/v4/chat/completions"
	DefaultModel    = "glm-5.3-flash"
	temperature     = 0.9
	requestTimeout  = 90 * time.Second // hard ceiling per request, via context
)

// streamTimeout is a variable so tests can shrink the ceiling.
var streamTimeout = requestTimeout

// Message is one chat message of the request payload.
type Message struct{ Role, Content string }

// Request fully describes one streaming call. Endpoint/Model fall back to the
// C6 defaults when empty.
type Request struct {
	Endpoint string
	APIKey   string
	ProxyURL string
	Model    string
	Messages []Message
}

// streamBody is the JSON payload; thinking is always enabled (C6).
type streamBody struct {
	Model       string      `json:"model"`
	Messages    []Message   `json:"messages"`
	Stream      bool        `json:"stream"`
	Temperature float64     `json:"temperature"`
	MaxTokens   int         `json:"max_tokens"`
	Thinking    thinkingCfg `json:"thinking"`
}

type thinkingCfg struct {
	Type string `json:"type"`
}

// sseChunk is one parsed data: line.
type sseChunk struct {
	Choices []struct {
		Delta struct {
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

// Stream POSTs the request and streams the SSE response: content deltas are
// passed to onDelta, reasoning deltas ONLY to onThought (never mixed, C6).
// It returns the full visible content and a typed error. Both callbacks may be
// nil. The 90s ceiling is enforced through the request context.
func Stream(ctx context.Context, req Request, onDelta func(string), onThought func(string)) (string, error) {
	if req.Endpoint == "" {
		req.Endpoint = DefaultEndpoint
	}
	if req.Model == "" {
		req.Model = DefaultModel
	}
	cctx, cancel := context.WithTimeout(ctx, streamTimeout)
	defer cancel()

	payload, err := json.Marshal(streamBody{
		Model:       req.Model,
		Messages:    req.Messages,
		Stream:      true,
		Temperature: temperature,
		MaxTokens:   contract.MaxOutputTokens,
		Thinking:    thinkingCfg{Type: "enabled"},
	})
	if err != nil {
		return "", fmt.Errorf("%w: marshal body: %v", ErrHTTP, err)
	}

	httpReq, err := http.NewRequestWithContext(cctx, http.MethodPost, req.Endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("%w: bad request: %v", ErrHTTP, err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if req.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+req.APIKey)
	}
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := clientFor(req).Do(httpReq)
	if err != nil {
		if cctx.Err() != nil {
			return "", fmt.Errorf("%w: timeout after %s", ErrNetwork, streamTimeout)
		}
		return "", fmt.Errorf("%w: %v", ErrNetwork, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		snippet := readSnippet(resp.Body)
		switch resp.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			return "", fmt.Errorf("%w: HTTP %d: %s", ErrAuth, resp.StatusCode, snippet)
		case http.StatusTooManyRequests:
			return "", fmt.Errorf("%w: HTTP %d: %s", ErrQuota, resp.StatusCode, snippet)
		default:
			return "", fmt.Errorf("%w: HTTP %d: %s", ErrHTTP, resp.StatusCode, snippet)
		}
	}

	var full strings.Builder
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024) // SSE lines can be long
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, ":") ||
			strings.HasPrefix(line, "event:") || strings.HasPrefix(line, "id:") || strings.HasPrefix(line, "retry:") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			return full.String(), nil
		}
		var chunk sseChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			log.Printf("llm: skipping malformed SSE data: %v", err)
			continue
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		d := chunk.Choices[0].Delta
		if d.ReasoningContent != "" && onThought != nil {
			onThought(d.ReasoningContent)
		}
		if d.Content != "" {
			full.WriteString(d.Content)
			if onDelta != nil {
				onDelta(d.Content)
			}
		}
	}
	if err := sc.Err(); err != nil {
		if cctx.Err() != nil {
			return full.String(), fmt.Errorf("%w: stream timeout", ErrNetwork)
		}
		return full.String(), fmt.Errorf("%w: stream interrupted: %v", ErrNetwork, err)
	}
	return full.String(), nil
}

// clientFor builds the HTTP client, honoring an optional proxy prefix.
// An unparseable proxy URL is logged and ignored (D4: degrade, never crash).
func clientFor(req Request) *http.Client {
	t := &http.Transport{}
	if req.ProxyURL != "" {
		u, err := url.Parse(req.ProxyURL)
		if err != nil {
			log.Printf("llm: ignoring bad proxy URL: %v", err)
		} else {
			t.Proxy = http.ProxyURL(u)
		}
	}
	return &http.Client{Transport: t}
}

// readSnippet caps an error body to 512 bytes for safe logging.
func readSnippet(r io.Reader) string {
	b, err := io.ReadAll(io.LimitReader(r, 512))
	if err != nil {
		return ""
	}
	s := strings.TrimSpace(string(b))
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > 512 {
		s = s[:512]
	}
	return s
}
