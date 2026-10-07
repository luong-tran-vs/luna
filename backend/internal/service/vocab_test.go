package service

import (
	"context"
	"testing"

	"github.com/luongtran/luna/backend/internal/dictionary"
)

// fakeDict knows a few words by their exact form; "went" resolves to "go".
type fakeDict map[string]string

func (f fakeDict) Resolve(_ context.Context, word string) (dictionary.Entry, bool, error) {
	if word == "went" {
		return dictionary.Entry{Word: "go", IPA: f["go"]}, true, nil
	}
	ipa, ok := f[word]
	return dictionary.Entry{Word: word, IPA: ipa}, ok, nil
}

func TestPronunciations(t *testing.T) {
	t.Parallel()
	p := pronunciations{fakeDict{"good": "/ˈɡʊd/", "morning": "/ˈmɔr.nɪŋ/", "in": "ˈɪn", "go": "/ɡəʊ/", "love": "/ˈləv/"}}
	for lemma, want := range map[string]string{
		"good":         "/ˈɡʊd/",
		"good morning": "/ˈɡʊd ˈmɔr.nɪŋ/",
		"in love":      "/ˈɪn ˈləv/", // a word stored without slashes
		"good table":   "",           // one unknown word: no IPA
		"went":         "",           // an inflected form is not its lemma's IPA
		"went in":      "",
		"":             "",
	} {
		got, err := p.IPA(t.Context(), lemma)
		if err != nil || got != want {
			t.Errorf("IPA(%q) = %q, %v; want %q", lemma, got, err, want)
		}
	}
}
