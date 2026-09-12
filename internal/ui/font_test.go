// G5.1: pure font tests — word wrapping, ellipsis, width math. No ebiten.
package ui

import (
	"strings"
	"testing"
)

func TestWrapTextBasic(t *testing.T) {
	got := WrapText("hello world foo", 11)
	want := []string{"hello world", "foo"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("line %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestWrapTextHardSplit(t *testing.T) {
	got := WrapText("abcdefghij", 4)
	want := []string{"abcd", "efgh", "ij"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestWrapTextNewlinesAndEmpty(t *testing.T) {
	got := WrapText("a\n\nb c", 10)
	want := []string{"a", "", "b c"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("got %v, want %v", got, want)
	}
	if lines := WrapText("", 10); len(lines) != 1 || lines[0] != "" {
		t.Fatalf("empty input should yield one empty line, got %v", lines)
	}
}

func TestWrapTextTinyWidthNeverFails(t *testing.T) {
	for w := 1; w <= 3; w++ {
		got := WrapText("neon tank agents", w)
		for _, l := range got {
			if len([]rune(l)) > w {
				t.Fatalf("width %d produced overlong line %q", w, l)
			}
		}
	}
}

func TestEllipsis(t *testing.T) {
	if got := Ellipsis("abcdef", 6); got != "abcdef" {
		t.Fatalf("short string should be untouched, got %q", got)
	}
	if got := Ellipsis("abcdefg", 5); got != "abc.." {
		t.Fatalf("got %q, want %q", got, "abc..")
	}
}

func TestTextWidth(t *testing.T) {
	// 5 chars -> 4*adv + glyphW = 4*6+5 = 29 at scale 1.
	if got := TextWidth("12345", 1); got != 29 {
		t.Fatalf("TextWidth(5 chars, 1) = %d, want 29", got)
	}
	if got := TextWidth("ab", 2); got != 2*6*2-2 {
		t.Fatalf("TextWidth(2 chars, 2) = %d, want %d", got, 2*6*2-2)
	}
	if TextWidth("", 1) != 0 {
		t.Fatal("empty string width must be 0")
	}
}
