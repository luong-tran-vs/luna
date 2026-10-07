package dictionary

import (
	"context"
	"testing"
)

// fakeResolver knows a few words by their exact form; "went" resolves to "go".
type fakeResolver map[string]Entry

func (f fakeResolver) Resolve(_ context.Context, word string) (Entry, bool, error) {
	if word == "went" {
		return f["go"], true, nil
	}
	e, ok := f[word]
	return e, ok, nil
}

var testWords = fakeResolver{
	"good":    {Word: "good", IPA: "/ˈɡʊd/", Meanings: []Meaning{{Text: "tốt"}, {Text: "giỏi"}, {Text: "hay"}}},
	"morning": {Word: "morning", IPA: "/ˈmɔr.nɪŋ/"},
	"in":      {Word: "in", IPA: "ˈɪn"},
	"go":      {Word: "go", IPA: "/ɡəʊ/", Meanings: []Meaning{{Text: "đi"}}},
	"love":    {Word: "love", IPA: "/ˈləv/"},
}

func TestIPA(t *testing.T) {
	t.Parallel()
	for text, want := range map[string]string{
		"good":         "/ˈɡʊd/",
		"good morning": "/ˈɡʊd ˈmɔr.nɪŋ/",
		"in love":      "/ˈɪn ˈləv/", // a word stored without slashes
		"good table":   "",           // one unknown word: no IPA
		"went":         "",           // an inflected form is not given its lemma's IPA
		"went in":      "",
		"":             "",
	} {
		got, err := IPA(t.Context(), testWords, text)
		if err != nil || got != want {
			t.Errorf("IPA(%q) = %q, %v; want %q", text, got, err, want)
		}
	}
}

func TestMeaningVi(t *testing.T) {
	t.Parallel()
	for text, want := range map[string]string{
		"good":  "tốt; giỏi", // the first two meanings
		"go":    "đi",
		"went":  "", // not that exact form
		"table": "",
	} {
		got, err := MeaningVi(t.Context(), testWords, text)
		if err != nil || got != want {
			t.Errorf("MeaningVi(%q) = %q, %v; want %q", text, got, err, want)
		}
	}
}
