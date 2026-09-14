// G3.2: Stream tests — httptest SSE fixture: chunk assembly, thought routing,
// [DONE] handling, timeout and every typed error mapping.
package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// sseData renders one SSE data line with the given delta fields.
func sseData(content, reasoning string) string {
	b, _ := json.Marshal(map[string]any{
		"choices": []any{map[string]any{
			"delta": map[string]any{
				"content":           content,
				"reasoning_content": reasoning,
			},
		}},
	})
	return "data: " + string(b) + "\n\n"
}

// streamingServer returns a server emitting the given SSE body with a small
// delay between writes, asserting the request looks like a GLM stream call.
func streamingServer(t *testing.T, body string, slow bool) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("Authorization header = %q", got)
		}
		raw, _ := io.ReadAll(r.Body)
		var req map[string]any
		if err := json.Unmarshal(raw, &req); err != nil {
			t.Errorf("request body not JSON: %v", err)
		}
		if req["stream"] != true {
			t.Error("stream must be true")
		}
		if req["max_tokens"] != float64(1600) {
			t.Errorf("max_tokens = %v, want 1600", req["max_tokens"])
		}
		th, ok := req["thinking"].(map[string]any)
		if !ok || th["type"] != "enabled" {
			t.Errorf("thinking must be enabled, got %v", req["thinking"])
		}
		if req["model"] != DefaultModel {
			t.Errorf("model = %v, want %s", req["model"], DefaultModel)
		}
		emitSSE(t, w, body, slow)
	}))
}

// emitSSE writes the body in \n\n-delimited frames, flushed per frame.
func emitSSE(t *testing.T, w http.ResponseWriter, body string, slow bool) {
	t.Helper()
	w.Header().Set("Content-Type", "text/event-stream")
	flusher := w.(http.Flusher)
	for _, line := range strings.SplitAfter(body, "\n\n") {
		fmt.Fprint(w, line)
		if slow {
			time.Sleep(20 * time.Millisecond)
		}
		flusher.Flush()
	}
}

func TestStreamAssemblesChunksAndRoutesThoughts(t *testing.T) {
	body := sseData("", "thinking hard") +
		sseData(`{"na`, "") +
		sseData("me\":", "") +
		sseData(`"Ember"}`, "") +
		"data: [DONE]\n\n"
	srv := streamingServer(t, body, false)
	defer srv.Close()

	var thoughts, chunks []string
	full, err := Stream(context.Background(), Request{
		Endpoint: srv.URL, APIKey: "test-key",
		Messages: []Message{{Role: "user", Content: "hello"}},
	}, func(d string) { chunks = append(chunks, d) }, func(s string) { thoughts = append(thoughts, s) })
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	if full != `{"name":"Ember"}` {
		t.Fatalf("full = %q", full)
	}
	if strings.Join(chunks, "") != full {
		t.Fatalf("chunks = %v", chunks)
	}
	if len(thoughts) != 1 || thoughts[0] != "thinking hard" {
		t.Fatalf("thoughts = %v, want exactly the reasoning delta", thoughts)
	}
}

func TestStreamEndsWithoutDONE(t *testing.T) {
	// Server closes the stream without [DONE]: content so far is returned.
	body := sseData("abc", "") + sseData("def", "")
	srv := plainSSEServer(t, body, false)
	defer srv.Close()
	full, err := Stream(context.Background(), Request{Endpoint: srv.URL}, nil, nil)
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	if full != "abcdef" {
		t.Fatalf("full = %q", full)
	}
}

// plainSSEServer streams body without asserting auth/body shape.
func plainSSEServer(t *testing.T, body string, slow bool) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		emitSSE(t, w, body, slow)
	}))
}

func errorServer(status int, payload string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		fmt.Fprint(w, payload)
	}))
}

func TestStreamErrorMapping(t *testing.T) {
	cases := []struct {
		status int
		want   error
	}{
		{http.StatusUnauthorized, ErrAuth},
		{http.StatusForbidden, ErrAuth},
		{http.StatusTooManyRequests, ErrQuota},
		{http.StatusInternalServerError, ErrHTTP},
		{http.StatusBadGateway, ErrHTTP},
	}
	for _, c := range cases {
		srv := errorServer(c.status, `{"error":"nope details here"}`)
		_, err := Stream(context.Background(), Request{Endpoint: srv.URL}, nil, nil)
		srv.Close()
		if err == nil {
			t.Fatalf("HTTP %d: expected error", c.status)
		}
		if !strings.Contains(err.Error(), "nope details here") {
			t.Fatalf("HTTP %d: error must include body snippet, got %v", c.status, err)
		}
	}
}

func TestStreamNetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close() // nothing listens anymore -> transport error
	_, err := Stream(context.Background(), Request{Endpoint: url}, nil, nil)
	if err == nil || !strings.Contains(err.Error(), ErrNetwork.Error()) {
		t.Fatalf("want ErrNetwork, got %v", err)
	}
}

func TestStreamTimeout(t *testing.T) {
	old := streamTimeout
	streamTimeout = 50 * time.Millisecond
	defer func() { streamTimeout = old }()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, sseData("a", ""))
		w.(http.Flusher).Flush()
		time.Sleep(2 * time.Second) // far beyond the shrunken ceiling
	}))
	defer srv.Close()

	start := time.Now()
	_, err := Stream(context.Background(), Request{Endpoint: srv.URL}, nil, nil)
	if err == nil || !strings.Contains(err.Error(), ErrNetwork.Error()) {
		t.Fatalf("want ErrNetwork on timeout, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("timeout took %s, ceiling not enforced", elapsed)
	}
}

func TestStreamProxyPrefix(t *testing.T) {
	// Proxy equal to the server itself must not break the request.
	srv := plainSSEServer(t, "data: [DONE]\n\n", false)
	defer srv.Close()
	_, err := Stream(context.Background(), Request{Endpoint: srv.URL, ProxyURL: srv.URL}, nil, nil)
	if err != nil {
		t.Fatalf("proxy prefix should pass through: %v", err)
	}
}
