package wordmatch_test

import (
	"testing"

	"github.com/luongtran/luna/backend/internal/wordmatch"
)

func TestFind(t *testing.T) {
	t.Parallel()

	text := "My Grandmothers watched TV. Then we took a shower at nine o'clock, wearing T-shirts. " +
		"It was the season of love. He went home; FAMILY and/or friends came."
	m := wordmatch.New(text, map[string]string{"went": "go"})

	tests := []struct {
		term string
		want string // "" means no match
	}{
		{"family", "FAMILY"},
		{"Grandmother", "Grandmothers"},
		{"watch", "watched"},
		{"take a shower", "took a shower"},
		{"o'clock", "o'clock"},
		{"T-shirt", "T-shirts"},
		{"go", "went"},
		{"son", ""},
		{"friend", "friends"},
		{"and/or", "and/or"},
		{"take a bath", ""},
		{"  ", ""},
	}
	for _, tt := range tests {
		span, ok := m.Find(tt.term)
		if ok != (tt.want != "") || span != tt.want {
			t.Errorf("Find(%q) = %q, %v; want %q", tt.term, span, ok, tt.want)
		}
	}
}

func TestFindPluralTermAndCurlyApostrophe(t *testing.T) {
	t.Parallel()
	m := wordmatch.New("My parent’s house is near the school.", nil)
	for _, term := range []string{"Parents", "parent's", "schools"} {
		if !m.Contains(term) {
			t.Errorf("Contains(%q) = false", term)
		}
	}
	if m.Contains("house keeper") {
		t.Error("a phrase must match whole")
	}
}

func TestFindWatching(t *testing.T) {
	t.Parallel()
	m := wordmatch.New("She is watching the birds.", nil)
	if span, ok := m.Find("watch"); !ok || span != "watching" {
		t.Fatalf("Find = %q, %v", span, ok)
	}
	if span, ok := m.Find("bird"); !ok || span != "birds" {
		t.Fatalf("Find = %q, %v", span, ok)
	}
}
