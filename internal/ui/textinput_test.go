// G5.1: TextInput pure-logic tests — insert, backspace, max length, masked
// flag. Keyboard/focus handling needs ebiten and stays untested here.
package ui

import "testing"

func TestTextInputInsertMaxLen(t *testing.T) {
	ti := TextInput{MaxLen: 5}
	ti.Insert([]rune("hello"))
	ti.Insert([]rune(" world")) // overflow ignored
	if ti.Value != "hello" {
		t.Fatalf("max length not enforced: %q", ti.Value)
	}
	ti.Backspace()
	ti.Insert([]rune("X"))
	if ti.Value != "hellX" {
		t.Fatalf("backspace+insert wrong: %q", ti.Value)
	}
	// Turkish runes are inserted as-is (font has the glyph map).
	ti2 := TextInput{}
	ti2.Insert([]rune("ğüşİöÜ"))
	if ti2.Value != "ğüşİöÜ" {
		t.Fatalf("Turkish runes must round-trip: %q", ti2.Value)
	}
	// Newlines are never inserted (single-line field).
	ti3 := TextInput{}
	ti3.Insert([]rune("a\nb"))
	if ti3.Value != "ab" {
		t.Fatalf("newlines must be dropped: %q", ti3.Value)
	}
}

func TestTextInputBackspaceEmpty(t *testing.T) {
	ti := TextInput{}
	ti.Backspace()
	if ti.Value != "" {
		t.Fatal("backspace on empty must stay empty (no panic, D4)")
	}
}

func TestTextInputMaskedFlag(t *testing.T) {
	ti := TextInput{Masked: true, MaxLen: 8}
	ti.Insert([]rune("hunter2"))
	if ti.Value != "hunter2" {
		t.Fatalf("masked input keeps the real value internally: %q", ti.Value)
	}
	if !ti.Masked {
		t.Fatal("masked flag must persist")
	}
}

func TestTextInputSet(t *testing.T) {
	ti := TextInput{}
	ti.Set("bound")
	if ti.Value != "bound" {
		t.Fatalf("Set must rebind the value: %q", ti.Value)
	}
}
