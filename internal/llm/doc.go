// G3.2: GLM SSE streaming client (thinking max) — stdlib only.
//
// Purpose: one function, Stream, that POSTs a chat-completions request with
// stream:true and thinking:{type:"enabled"} (frozen per C6) and parses the SSE
// response: content deltas go to onDelta, reasoning_content deltas ONLY to
// onThought. Typed, errors.Is-able errors classify every failure so agents can
// react (quota -> temporary simulate, network -> permanent simulate, ...).
//
// Owns: internal/llm/*. No other package may do HTTP for agent traffic.
//
// Public API: Stream, Request, Message, ErrAuth/ErrQuota/ErrNetwork/ErrHTTP,
// DefaultEndpoint, DefaultModel.
//
// Invariants: 90 second hard timeout via context; no external dependencies
// (net/http + encoding/json only); the API key is only sent in the
// Authorization header and never logged; a mid-stream failure still returns
// the content received so far.
//
// Extensions: new body fields = streamBody only; new SSE event types = the
// line parser in Stream only.
package llm
