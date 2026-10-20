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
	"good":      {Word: "good", IPA: "/ˈɡʊd/", Meanings: []Meaning{{Text: "tốt"}, {Text: "giỏi"}, {Text: "hay"}}},
	"morning":   {Word: "morning", IPA: "/ˈmɔr.nɪŋ/"},
	"in":        {Word: "in", IPA: "ˈɪn"},
	"go":        {Word: "go", IPA: "/ɡəʊ/", Meanings: []Meaning{{Text: "đi"}}},
	"love":      {Word: "love", IPA: "/ˈləv/"},
	"a":         {Word: "a", IPA: "/eɪ̯/"},
	"book":      {Word: "book", IPA: "/ˈbʊk/"},
	"read":      {Word: "read", IPA: "ˈɹiːd"},
	"dish":      {Word: "dish", IPA: "/ˈdɪʃ/"},
	"dishes":    {Word: "dish", IPA: "/ˈdɪʃ/"}, // resolves to its lemma
	"cats":      {Word: "cat", IPA: "/ˈkæt/"},
	"dogs":      {Word: "dog", IPA: "/ˈdɒɡ/"},
	"wash":      {Word: "wash", IPA: "/ˈwɒʃ/"},
	"lunch":     {Word: "lunch", IPA: "/ˈlʌntʃ/"},
	"have":      {Word: "have"}, // in the dictionary without a transcription
	"lunaword":  {Word: "lunaword", IPA: "/ˈlɑʊ.vəɹ, ˈlaʊ.və/"},
	"lunawords": {Word: "lunaword", IPA: "/ˈlɑʊ.vəɹ, ˈlaʊ.və/"},
}

func TestIPA(t *testing.T) {
	t.Parallel()
	for text, want := range map[string]string{
		// One standard, General American from the CMU dictionary (updated 2026-10-19); one syllable,
		// no stress mark.
		"sound":    "/saʊnd/",
		"dance":    "/dæns/",
		"water":    "/ˈwɔːtər/",
		"exercise": "/ˈeksərˌsaɪz/",
		"explain":  "/ɪkˈspleɪn/", // the stress mark goes before the onset "sp"
		"happy":    "/ˈhæpi/",
		"read":     "/riːd/", // the base form, not the past /red/
		"went":     "/went/", // its own transcription, not its lemma's
		"TV":       "/ˌtiːˈviː/",
		// A phrase: small words weak, the others stressed.
		"read a book":     "/ˈriːd ə ˈbʊk/",
		"dance to":        "/ˈdæns tə/",
		"have dinner":     "/ˈhæv ˈdɪnər/",
		"wash the dishes": "/ˈwɑːʃ ðə ˈdɪʃəz/",
		"":                "",
		// Not in the CMU dictionary: the offline dictionary's, cleaned up, first variant only.
		"lunaword":       "/ˈlaʊvər/",
		"lunawords":      "/ˈlaʊvərz/",
		"lunaword lunax": "", // one unknown word: no IPA
		"lunaword in":    "/ˈlaʊvər ˈɪn/",
	} {
		got, err := IPA(t.Context(), testWords, text)
		if err != nil || got != want {
			t.Errorf("IPA(%q) = %q, %v; want %q", text, got, err, want)
		}
	}
}

func TestCleanIPA(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]string{
		"/hæv, həv, v/": "/hæv/",
		"[ˈdɪnɚ]":       "/ˈdɪnər/",
		"ˈɛk.sɜː.ˌsɑɪz": "/ˈeksɜːˌsaɪz/",
		"/ˈwɔːtə(r)/":   "/ˈwɔːtər/",
		"/tɹæk/":        "/træk/",
		"/eɪ̯/":         "/eɪ/",
		"  ":            "",
	} {
		if got := cleanIPA(in); got != want {
			t.Errorf("cleanIPA(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIPASuggestion(t *testing.T) {
	t.Parallel()
	for _, c := range []struct{ text, stored, want string }{
		{"sound", "/ˈsɑʊnd/", "/saʊnd/"}, // another convention
		{"have dinner", "/hæv, həv, v ˈdɪnə/", "/ˈhæv ˈdɪnər/"},
		{"track", "/tɹæk/", "/træk/"},
		{"dance", "/ˈdænts/", "/dæns/"}, // other sounds
		{"home", "/ˈhoʊm/", ""},         // the same sounds, a stress mark aside
		{"home", "/hoʊm/", ""},
		{"home", "", ""},            // nothing stored: nothing to check
		{"lunax", "/ˈluːnæks/", ""}, // no standard known
	} {
		got, err := IPASuggestion(t.Context(), testWords, c.text, c.stored)
		if err != nil || got != c.want {
			t.Errorf("IPASuggestion(%q, %q) = %q, %v; want %q", c.text, c.stored, got, err, c.want)
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
