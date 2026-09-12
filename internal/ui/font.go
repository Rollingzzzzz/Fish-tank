// G5.1: built-in 5x7 bitmap font (Latin + Turkish), text drawing, measuring
// and word wrapping. No external font file is available (no go get allowed,
// golang.org/x/image not in go.mod), so glyphs are authored here and drawn
// from a cached per-color atlas. Zero emoji (D6).
package ui

import (
	"image"
	"sort"
	"strings"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

// Glyph metrics in atlas pixels (5x7 cell, 1 px advance gap, 1 px leading).
const (
	glyphW    = 5
	glyphH    = 7
	glyphAdv  = 6
	glyphLead = 8
	atlasCols = 16
)

// glyphSrc maps runes to 7 rows of 5 cells ('#' on, '.' off), '/' separated.
var glyphSrc = map[rune]string{
	' ':  "...../...../...../...../...../...../.....",
	'A':  "..#../.#.#./#...#/#####/#...#/#...#/#...#",
	'B':  "####./#...#/#...#/####./#...#/#...#/####.",
	'C':  ".###./#...#/#..../#..../#..../#...#/.###.",
	'D':  "####./#...#/#...#/#...#/#...#/#...#/####.",
	'E':  "#####/#..../#..../####./#..../#..../#####",
	'F':  "#####/#..../#..../####./#..../#..../#....",
	'G':  ".###./#...#/#..../#.###/#...#/#...#/.###.",
	'H':  "#...#/#...#/#...#/#####/#...#/#...#/#...#",
	'I':  "#####/..#../..#../..#../..#../..#../#####",
	'J':  "..###/...#./...#./...#./...#./#..#./.##..",
	'K':  "#...#/#..#./#.#../##.../#.#../#..#./#...#",
	'L':  "#..../#..../#..../#..../#..../#..../#####",
	'M':  "#...#/##.##/#.#.#/#...#/#...#/#...#/#...#",
	'N':  "#...#/##..#/#.#.#/#..##/#...#/#...#/#...#",
	'O':  ".###./#...#/#...#/#...#/#...#/#...#/.###.",
	'P':  "####./#...#/#...#/####./#..../#..../#....",
	'Q':  ".###./#...#/#...#/#...#/#.#.#/#..#./.##.#",
	'R':  "####./#...#/#...#/####./#.#../#..#./#...#",
	'S':  ".####/#..../#..../.###./....#/....#/####.",
	'T':  "#####/..#../..#../..#../..#../..#../..#..",
	'U':  "#...#/#...#/#...#/#...#/#...#/#...#/.###.",
	'V':  "#...#/#...#/#...#/#...#/#...#/.#.#./..#..",
	'W':  "#...#/#...#/#...#/#...#/#.#.#/##.##/#...#",
	'X':  "#...#/#...#/.#.#./..#../.#.#./#...#/#...#",
	'Y':  "#...#/#...#/.#.#./..#../..#../..#../..#..",
	'Z':  "#####/....#/...#./..#../.#.../#..../#####",
	'a':  "...../...../.###./....#/.####/#...#/.####",
	'b':  "#..../#..../####./#...#/#...#/#...#/####.",
	'c':  "...../...../.###./#..../#..../#...#/.###.",
	'd':  "....#/....#/.####/#...#/#...#/#...#/.####",
	'e':  "...../...../.###./#...#/#####/#..../.###.",
	'f':  "..##./.#..#/.#.../###../.#.../.#.../.#...",
	'g':  "...../.###./#...#/#...#/.####/....#/.###.",
	'h':  "#..../#..../####./#...#/#...#/#...#/#...#",
	'i':  "..#../...../..#../..#../..#../..#../..#..",
	'j':  "...#./...../...#./...#./...#./#..#./.##..",
	'k':  "#..../#..../#...#/#..#./###../#..#./#...#",
	'l':  "..#../..#../..#../..#../..#../..#../.###.",
	'm':  "...../...../##.#./#.#.#/#.#.#/#.#.#/#.#.#",
	'n':  "...../...../####./#...#/#...#/#...#/#...#",
	'o':  "...../...../.###./#...#/#...#/#...#/.###.",
	'p':  "...../...../####./#...#/#...#/####./#....",
	'q':  "...../...../.####/#...#/#...#/.####/....#",
	'r':  "...../...../#.##./##..#/#..../#..../#....",
	's':  "...../...../.####/#..../.###./....#/####.",
	't':  ".#.../.#.../###../.#.../.#.../.#.../..##.",
	'u':  "...../...../#...#/#...#/#...#/#...#/.####",
	'v':  "...../...../#...#/#...#/#...#/.#.#./..#..",
	'w':  "...../...../#...#/#...#/#.#.#/#.#.#/.#.#.",
	'x':  "...../...../#...#/.#.#./..#../.#.#./#...#",
	'y':  "...../...../#...#/#...#/.####/....#/.###.",
	'z':  "...../...../#####/...#./..#../.#.../#####",
	'0':  ".###./#...#/#..##/#.#.#/##..#/#...#/.###.",
	'1':  "..#../.##../..#../..#../..#../..#../.###.",
	'2':  ".###./#...#/....#/...#./..#../.#.../#####",
	'3':  "####./....#/....#/.###./....#/....#/####.",
	'4':  "...#./..##./.#.#./#..#./#####/...#./...#.",
	'5':  "#####/#..../#..../####./....#/....#/####.",
	'6':  "..##./.#.../#..../####./#...#/#...#/.###.",
	'7':  "#####/....#/...#./..#../.#.../.#.../.#...",
	'8':  ".###./#...#/#...#/.###./#...#/#...#/.###.",
	'9':  ".###./#...#/#...#/.####/....#/...#./.##..",
	'.':  "...../...../...../...../...../.##../.##..",
	',':  "...../...../...../...../.##../.##../.#...",
	':':  "...../.##../.##../...../.##../.##../.....",
	';':  "...../.##../.##../...../.##../.##../.#...",
	'!':  "..#../..#../..#../..#../..#../...../..#..",
	'?':  ".###./#...#/....#/..##./..#../...../..#..",
	'\'': ".#.../.#.../.#.../...../...../...../.....",
	'"':  ".#.#./.#.#./...../...../...../...../.....",
	'-':  "...../...../...../#####/...../...../.....",
	'_':  "...../...../...../...../...../...../#####",
	'/':  "....#/....#/...#./..#../.#.../#..../#....",
	'\\': "#..../#..../.#.../..#../...#./....#/....#",
	'(':  "..#../.#.../#..../#..../#..../.#.../..#..",
	')':  "..#../...#./....#/....#/....#/...#./..#..",
	'[':  ".###./.#.../.#.../.#.../.#.../.#.../.###.",
	']':  ".###./...#./...#./...#./...#./...#./.###.",
	'{':  "..##./.#.../.#.../##.../.#.../.#.../..##.",
	'}':  ".##../..#../..#../...##/..#../..#../.##..",
	'+':  "...../..#../..#../#####/..#../..#../.....",
	'=':  "...../...../#####/...../#####/...../.....",
	'*':  "...../..#../#.#.#/.###./#.#.#/..#../.....",
	'<':  "...#./..#../.#.../#..../.#.../..#../...#.",
	'>':  ".#.../..#../...#./....#/...#./..#../.#...",
	'%':  "#...#/#..#./...#./..#../.#.../.#..#/#...#",
	'#':  ".#.#./#####/.#.#./#####/.#.#./...../.....",
	'@':  ".###./#...#/#.###/#.#.#/#.##./#..../.###.",
	'&':  ".##../#..#./#.#../.#.../#.#.#/#..#./.##.#",
	'$':  "..#../.####/#.#../.###./..#.#/####./..#..",
	'|':  "..#../..#../..#../..#../..#../..#../..#..",
	'~':  "...../...../.#..#/#..#./...../...../.....",
	'^':  "..#../.#.#./#...#/...../...../...../.....",
	// Turkish set (README C3/D6 note: UI strings stay English, but user
	// content such as agent-streamed names must not crash the renderer).
	'ç': "...../.###./#..../#..../#...#/.###./.#...",
	'Ç': ".###./#..../#..../#...#/.###./.#.../.....",
	'ğ': ".#.#./.###./#...#/#...#/.####/....#/.###.",
	'Ğ': ".#.#./.###./#...#/#..../#.###/#...#/.###.",
	'ı': "...../...../..#../..#../..#../..#../..#..",
	'İ': "..#../#####/..#../..#../..#../..#../#####",
	'ö': ".#.#./...../.###./#...#/#...#/#...#/.###.",
	'Ö': ".#.#./.###./#...#/#...#/#...#/#...#/.###.",
	'ş': "...../.####/#..../.###./....#/####./.#...",
	'Ş': ".###./#...#/#..../.###./....#/####./.#...",
	'ü': ".#.#./...../#...#/#...#/#...#/#...#/.####",
	'Ü': ".#.#./#...#/#...#/#...#/#...#/#...#/.###.",
	// Fallback shown for any rune outside the map (never crash, D4).
	0: "#####/#...#/#...#/#...#/#...#/#...#/#####",
}

var (
	fontMu    sync.Mutex
	fontOnce  sync.Once
	glyphIdx  map[rune]int    // rune -> atlas cell index
	glyphCuts []*ebiten.Image // pre-cut sub-image per atlas cell
)

// fontEnsure builds the glyph atlas on first use (never at package init, so
// tests and constructors stay GPU-free).
func fontEnsure() {
	fontMu.Lock()
	defer fontMu.Unlock()
	fontOnce.Do(buildAtlas)
}

// buildAtlas rasterizes and cuts one opaque-white glyph atlas into per-rune
// sub-images; tinting happens per draw via ColorScale (no per-color cache).
func buildAtlas() {
	runes := make([]rune, 0, len(glyphSrc))
	for r := range glyphSrc {
		runes = append(runes, r)
	}
	sort.Slice(runes, func(i, j int) bool { return runes[i] < runes[j] })
	glyphIdx = make(map[rune]int, len(runes))
	rows := (len(runes) + atlasCols - 1) / atlasCols
	atlas := ebiten.NewImage(atlasCols*glyphAdv, rows*glyphLead)
	for y := 0; y < rows; y++ {
		for x := 0; x < atlasCols && y*atlasCols+x < len(runes); x++ {
			r := runes[y*atlasCols+x]
			glyphIdx[r] = y*atlasCols + x
			for ry, row := range strings.Split(glyphSrc[r], "/") {
				for rx := 0; rx < glyphW; rx++ {
					if row[rx] == '#' {
						atlas.Set(x*glyphAdv+rx, y*glyphLead+ry, white)
					}
				}
			}
		}
	}
	// Pre-cut sub-images once; DrawText never allocates afterwards.
	cuts := make([]*ebiten.Image, len(runes))
	for _, r := range runes {
		i := glyphIdx[r]
		cx, cy := (i%atlasCols)*glyphAdv, (i/atlasCols)*glyphLead
		sub := atlas.SubImage(image.Rect(cx, cy, cx+glyphW, cy+glyphH))
		cuts[i] = sub.(*ebiten.Image)
	}
	glyphCuts = cuts
}

// DrawText draws s with the bitmap font at integer scale, top-left at (x, y).
// Each on-cell paints through FillRect (the white-sprite path) — the glyph
// atlas route measured ~55% dimmer end-to-end (F12), while rect fills stay
// at full brightness; a 5x7 bitmap is rect-shaped by nature anyway.
func DrawText(dst *ebiten.Image, s string, x, y, scale int, hex string, alpha float64) {
	if dst == nil || s == "" || scale < 1 {
		return
	}
	if alpha < 0 {
		alpha = 0
	}
	if alpha > 1 {
		alpha = 1
	}
	_ = fontEnsure // keep the atlas warm for callers that measure via cuts
	cx, cy := x, y
	for _, rn := range s {
		if rn == '\n' {
			break
		}
		src, ok := glyphSrc[rn]
		if !ok {
			src = glyphSrc[0] // fallback box
		}
		for ry, row := range strings.Split(src, "/") {
			for rx := 0; rx < glyphW; rx++ {
				if row[rx] == '#' {
					FillRect(dst, cx+rx*scale, cy+ry*scale, scale, scale, hex, alpha)
				}
			}
		}
		cx += glyphAdv * scale
	}
}

func fontLookup(r rune) (int, bool) {
	fontMu.Lock()
	defer fontMu.Unlock()
	i, ok := glyphIdx[r]
	return i, ok
}

// TextWidth returns the rendered width of s in pixels.
func TextWidth(s string, scale int) int {
	n := 0
	for range s {
		n++
	}
	if n == 0 {
		return 0
	}
	return (n*glyphAdv - (glyphAdv - glyphW)) * scale
}

// LineHeight returns the line pitch of the font at the given scale.
func LineHeight(scale int) int { return glyphLead * scale }

// WrapText word-wraps s to at most maxChars columns. Explicit newlines are
// honored; words longer than maxChars are hard-split. Pure logic (tested).
func WrapText(s string, maxChars int) []string {
	if maxChars < 1 {
		maxChars = 1
	}
	var out []string
	for _, para := range strings.Split(s, "\n") {
		words := strings.Split(para, " ")
		line := ""
		flush := func() {
			out = append(out, line)
			line = ""
		}
		for _, w := range words {
			for len([]rune(w)) > maxChars { // hard-split oversized words
				part := string([]rune(w)[:maxChars])
				w = string([]rune(w)[maxChars:])
				if line != "" {
					flush()
				}
				out = append(out, part)
			}
			if line == "" {
				line = w
			} else if len([]rune(line))+1+len([]rune(w)) <= maxChars {
				line += " " + w
			} else {
				flush()
				line = w
			}
		}
		flush()
	}
	return out
}

// Ellipsis truncates s to maxChars runes, appending ".." when cut.
func Ellipsis(s string, maxChars int) string {
	rs := []rune(s)
	if maxChars < 3 {
		maxChars = 3
	}
	if len(rs) <= maxChars {
		return s
	}
	return string(rs[:maxChars-2]) + ".."
}
