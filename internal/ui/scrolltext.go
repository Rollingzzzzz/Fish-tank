// G5.1: ScrollText — word-wrapped streaming text buffer with auto-scroll to
// the bottom and dimmed thought lines. Buffer caps at maxLines (oldest
// dropped, C3). Wrap/view computation is pure and cached per width.
package ui

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// LineKind classifies a ScrollText line; KindThought renders dim.
type LineKind int

const (
	KindText    LineKind = iota // agent output (normal text color)
	KindThought                 // reasoning_content stream (dimmed, C6)
	KindLog                     // tank log line (soft blue)
)

type uiLine struct {
	kind LineKind
	text string
}

// lineView caches the wrapped rows of one line at a given width.
type lineView struct {
	kind LineKind
	rows []string
}

// ScrollText is a bounded streaming buffer. Zero value is usable but call
// NewScrollText to set the cap.
type ScrollText struct {
	lines  []uiLine
	views  []lineView
	width  int // wrap width in characters (0 = not wrapped yet)
	scroll int // 0 = pinned to bottom; >0 = rows scrolled back
	max    int
}

// NewScrollText returns a buffer capped at maxLines (hard ceiling 500, C3).
func NewScrollText(maxLines int) *ScrollText {
	if maxLines <= 0 || maxLines > 500 {
		maxLines = 500
	}
	return &ScrollText{max: maxLines}
}

// AppendLine adds one complete line.
func (s *ScrollText) AppendLine(kind LineKind, text string) {
	s.lines = append(s.lines, uiLine{kind: kind, text: text})
	s.trim()
	s.dirty()
}

// AppendChunk streams into the last line of the same kind (agent deltas);
// starts a new line when the tail is missing or of another kind.
func (s *ScrollText) AppendChunk(kind LineKind, chunk string) {
	if chunk == "" {
		return
	}
	if n := len(s.lines); n > 0 && s.lines[n-1].kind == kind {
		s.lines[n-1].text += chunk
	} else {
		s.lines = append(s.lines, uiLine{kind: kind, text: chunk})
	}
	s.trim()
	s.dirty()
}

// trim enforces the cap (oldest dropped) — pure.
func (s *ScrollText) trim() {
	for len(s.lines) > s.max {
		s.lines = s.lines[1:]
		if len(s.views) > 0 {
			s.views = s.views[1:]
		}
	}
	if len(s.lines) > 0 && s.lines[len(s.lines)-1].text == "" {
		s.lines = s.lines[:len(s.lines)-1]
	}
	s.scroll = 0 // auto-scroll: new content always re-pins to the bottom
}

func (s *ScrollText) dirty() {
	s.views = nil // recompute lazily on next draw
}

// Len returns the number of buffered lines. Pure.
func (s *ScrollText) Len() int { return len(s.lines) }

// ScrollWheel scrolls by delta rows (positive = towards older lines).
func (s *ScrollText) ScrollWheel(delta int) { s.scroll += delta }

// wrapViews computes cached wrapped views at maxCols columns. Pure apart
// from WrapText; called only when content or width changed.
func (s *ScrollText) wrapViews(maxCols int) []lineView {
	if s.views != nil && s.width == maxCols {
		return s.views
	}
	views := make([]lineView, 0, len(s.lines))
	for _, ln := range s.lines {
		views = append(views, lineView{kind: ln.kind, rows: WrapText(ln.text, maxCols)})
	}
	s.views = views
	s.width = maxCols
	return views
}

// Draw renders the visible slice of the buffer into (x, y, w, h) at the given
// font scale, pinned to the bottom unless scrolled back.
func (s *ScrollText) Draw(dst *ebiten.Image, x, y, w, h, scale int) {
	if dst == nil || w < glyphAdv*scale || h < glyphLead {
		return
	}
	maxCols := w / (glyphAdv * scale)
	views := s.wrapViews(maxCols)
	// Total rendered rows across all lines.
	total := 0
	for _, v := range views {
		total += len(v.rows)
	}
	if total == 0 {
		return
	}
	rowsFit := h / LineHeight(scale)
	if rowsFit < 1 {
		return
	}
	off := s.scroll
	if off > total-rowsFit {
		off = total - rowsFit
	}
	if off < 0 {
		off = 0
	}
	startRow := total - rowsFit - off
	if startRow < 0 {
		startRow = 0
	}
	// Find (lineIdx, rowOffset) for startRow.
	lineIdx, rowOff, counted := 0, 0, 0
	for ; lineIdx < len(views); lineIdx++ {
		nr := len(views[lineIdx].rows)
		if counted+nr > startRow {
			rowOff = startRow - counted
			break
		}
		counted += nr
	}
	cy := y
	for rowsFit > 0 && lineIdx < len(views) {
		v := views[lineIdx]
		for r := rowOff; r < len(v.rows) && rowsFit > 0; r++ {
			hex, alpha := kindStyle(v.kind)
			DrawText(dst, v.rows[r], x, cy, scale, hex, alpha)
			cy += LineHeight(scale)
			rowsFit--
		}
		rowOff = 0
		lineIdx++
	}
	// Fade cap: a 2 px soft line marks the cut when scrolled content exists.
	if startRow > 0 {
		FillRect(dst, x, y, w, 2, ColPanel, 0.9)
	}
}

func kindStyle(k LineKind) (string, float64) {
	switch k {
	case KindThought:
		return ColDim, 0.75
	case KindLog:
		return ColLogText, 0.9
	default:
		return ColText, 1
	}
}

// AppendLinesFromText splits a multi-line chunk and appends each part.
func AppendLinesFromText(s *ScrollText, kind LineKind, text string) {
	for _, part := range strings.Split(text, "\n") {
		if part != "" {
			s.AppendLine(kind, part)
		}
	}
}
