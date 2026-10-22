// Package cefr gives the CEFR level of English words, from the CEFR-J Wordlist (A1–B2), to warn
// when a lesson uses words above its level (added 2026-10-22, A1 curriculum).
//
// Source: CEFR-J Wordlist Version 1.5, Tono Laboratory, Tokyo University of Foreign Studies
// (https://github.com/openlanguageprofiles/olp-en-cefrj), free for research and commercial use
// with this citation. cefrj-levels.tsv keeps each headword (and each spelling of "a/b" headwords)
// once, lowercased, with its lowest level over its parts of speech.
package cefr

import (
	"bufio"
	_ "embed"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/luongtran/luna/backend/internal/dictionary"
)

// Levels lists the levels of the list in order.
var Levels = []string{"A1", "A2", "B1", "B2"}

//go:embed cefrj-levels.tsv
var levelsTSV string

var (
	loadOnce sync.Once
	byWord   map[string]string
)

func table() map[string]string {
	loadOnce.Do(func() {
		byWord = make(map[string]string, 7100)
		sc := bufio.NewScanner(strings.NewReader(levelsTSV))
		for sc.Scan() {
			if w, l, ok := strings.Cut(sc.Text(), "\t"); ok {
				byWord[w] = l
			}
		}
	})
	return byWord
}

// Rank orders levels: A1 is 1, C2 is 6, anything else 0.
func Rank(level string) int {
	switch level {
	case "C1":
		return 5
	case "C2":
		return 6
	}
	return slices.Index(Levels, level) + 1
}

// Level returns the level of word, trying its base forms ("plays" → "play"); ok is false when the
// list does not have it.
func Level(word string) (string, bool) {
	t := table()
	for _, c := range dictionary.Candidates(word) {
		if l, ok := t[c]; ok {
			return l, true
		}
	}
	return "", false
}

// Word is a word of a text above the text's level.
type Word struct {
	// Text is the word as written in the text (first time it appears).
	Text  string
	Level string
}

var (
	tokenRe    = regexp.MustCompile(`[A-Za-z]+(?:['’][A-Za-z]+)?`)
	sentenceRe = regexp.MustCompile(`[.!?]\s*$`)
)

// Above returns the words of text whose level is above level, once each, in text order. Words
// the list lacks are left out (names, numbers, rare words), as are capitalised words inside a
// sentence (names) and the endings of contractions ("isn't" is "is").
func Above(text, level string) []Word {
	want := Rank(level)
	if want == 0 || want >= Rank("B2") {
		return nil
	}
	var out []Word
	seen := map[string]bool{}
	for _, loc := range tokenRe.FindAllStringIndex(text, -1) {
		tok := text[loc[0]:loc[1]]
		if i := strings.IndexAny(tok, "'’"); i > 0 {
			tok = tok[:i]
		}
		startOfSentence := loc[0] == 0 || sentenceRe.MatchString(text[:loc[0]])
		if !startOfSentence && tok[0] >= 'A' && tok[0] <= 'Z' && tok != "I" {
			continue
		}
		key := strings.ToLower(tok)
		if seen[key] {
			continue
		}
		seen[key] = true
		if l, ok := Level(key); ok && Rank(l) > want {
			out = append(out, Word{Text: tok, Level: l})
		}
	}
	return out
}
