// Package wordmatch finds words and phrases in a text by whole words, ignoring case and simple
// inflections (F18). Topics use it for vocabulary coverage, lessons for missing target words
// and the topic words to annotate.
package wordmatch

import (
	"regexp"
	"strings"

	"github.com/luongtran/luna/backend/internal/dictionary"
)

// wordPattern matches a word: letters, with apostrophes or hyphens inside ("o'clock", "T-shirt").
// Anything else, "/" included, separates words.
var wordPattern = regexp.MustCompile(`[\p{L}\p{N}]+(?:['’\-][\p{L}\p{N}]+)*`)

// Text is a text split into words, each with the base forms it may stand for.
type Text struct {
	text   string
	tokens []token
}

type token struct {
	start, end int
	forms      map[string]bool
}

// New indexes text. lemmas maps a lowercase word of the text to its base form when known (for
// example from the lesson annotations: "went" → "go"); it may be nil.
func New(text string, lemmas map[string]string) *Text {
	t := &Text{text: text}
	for _, loc := range wordPattern.FindAllStringIndex(text, -1) {
		word := text[loc[0]:loc[1]]
		forms := map[string]bool{}
		for _, c := range dictionary.Candidates(word) {
			forms[c] = true
		}
		if lemma, ok := lemmas[key(word)]; ok && lemma != "" {
			forms[key(lemma)] = true
		}
		t.tokens = append(t.tokens, token{start: loc[0], end: loc[1], forms: forms})
	}
	return t
}

// Find returns the first occurrence of term (a word or phrase) as written in the text. Each word
// of term matches a word of the text when they share a base form: "Grandmother" matches
// "grandmothers", "take a shower" matches "took a shower", "son" does not match "season".
func (t *Text) Find(term string) (string, bool) {
	want := termForms(term)
	if len(want) == 0 {
		return "", false
	}
	for i := 0; i+len(want) <= len(t.tokens); i++ {
		if t.matchesAt(i, want) {
			return t.text[t.tokens[i].start:t.tokens[i+len(want)-1].end], true
		}
	}
	return "", false
}

// Contains reports whether term occurs in the text (see Find).
func (t *Text) Contains(term string) bool {
	_, ok := t.Find(term)
	return ok
}

func (t *Text) matchesAt(i int, want [][]string) bool {
	for k, forms := range want {
		if !shares(t.tokens[i+k].forms, forms) {
			return false
		}
	}
	return true
}

// termForms splits term into words, each with its possible base forms.
func termForms(term string) [][]string {
	var out [][]string
	for _, w := range wordPattern.FindAllString(term, -1) {
		out = append(out, dictionary.Candidates(w))
	}
	return out
}

func shares(forms map[string]bool, candidates []string) bool {
	for _, c := range candidates {
		if forms[c] {
			return true
		}
	}
	return false
}

func key(s string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(s), "’", "'"))
}
