// G5.1: ScrollText buffer tests — cap enforcement, streaming chunk merge,
// wrap cache behavior. Pure (no ebiten, no Draw).
package ui

import "testing"

func TestScrollTextCap(t *testing.T) {
	s := NewScrollText(500)
	for i := 0; i < 600; i++ {
		s.AppendLine(KindLog, "line")
	}
	if s.Len() != 500 {
		t.Fatalf("cap 500 enforced, got %d lines", s.Len())
	}
	// Cap is clamped to the C3 hard ceiling.
	if got := NewScrollText(10000); got.max != 500 {
		t.Fatalf("max must clamp to 500, got %d", got.max)
	}
}

func TestScrollTextChunkMerge(t *testing.T) {
	s := NewScrollText(10)
	s.AppendChunk(KindText, "Hel")
	s.AppendChunk(KindText, "lo ")
	s.AppendChunk(KindText, "world")
	if s.Len() != 1 {
		t.Fatalf("same-kind chunks must merge into one line, got %d", s.Len())
	}
	if s.lines[0].text != "Hello world" {
		t.Fatalf("merged text wrong: %q", s.lines[0].text)
	}
	// A different kind starts a new line.
	s.AppendChunk(KindThought, "hmm")
	if s.Len() != 2 || s.lines[1].kind != KindThought {
		t.Fatalf("thought chunk must start a new line, got %d", s.Len())
	}
	s.AppendChunk(KindText, "next") // tail is thought -> new text line
	if s.Len() != 3 {
		t.Fatalf("kind change must split lines, got %d", s.Len())
	}
	s.AppendChunk(KindText, "") // empty chunk ignored
	if s.Len() != 3 {
		t.Fatalf("empty chunk must be ignored, got %d", s.Len())
	}
}

func TestScrollTextWrapCache(t *testing.T) {
	s := NewScrollText(10)
	s.AppendLine(KindText, "one two three four five")
	v1 := s.wrapViews(10)
	if len(v1) == 0 || len(v1[0].rows) < 2 {
		t.Fatalf("expected wrapped rows at 10 cols, got %v", v1)
	}
	if s.wrapViews(10) == nil {
		t.Fatal("cached views must be reused")
	}
	s.AppendLine(KindText, "six") // invalidates cache
	v2 := s.wrapViews(10)
	if len(v2) != 2 {
		t.Fatalf("cache must recompute after append, got %d views", len(v2))
	}
	// Auto-scroll: appends re-pin to the bottom.
	s.ScrollWheel(5)
	s.AppendLine(KindText, "seven")
	if s.scroll != 0 {
		t.Fatalf("append must re-pin to bottom, scroll = %d", s.scroll)
	}
}

func TestKindStylesDistinct(t *testing.T) {
	t1, a1 := kindStyle(KindText)
	t2, a2 := kindStyle(KindThought)
	if t1 == t2 || a1 == a2 {
		t.Fatal("thought lines must render dimmer than text lines")
	}
	if t3, _ := kindStyle(KindLog); t3 == t1 {
		t.Fatal("log lines should use their own color")
	}
}
