package lesson

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/luongtran/luna/backend/internal/wordmatch"
)

// maxFocusMeaning bounds a dictionary meaning copied into an annotation.
const maxFocusMeaning = 200

// focusWords returns the topic words found in the lesson content (F18), in topic order.
func focusWords(content string, topicWords []string) []string {
	text := wordmatch.New(content, nil)
	out := []string{}
	for _, w := range topicWords {
		if text.Contains(w) {
			out = append(out, w)
		}
	}
	return out
}

// cleanWordList trims the words and drops the empty ones and repeats (ignoring case).
func cleanWordList(words []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, w := range words {
		w = strings.Join(strings.Fields(w), " ")
		if w == "" || seen[normalize(w)] {
			continue
		}
		seen[normalize(w)] = true
		out = append(out, w)
	}
	return out
}

// onlyFocus keeps the annotations that are one of the focus words.
func onlyFocus(anns []Annotation, focus []string) []Annotation {
	out := make([]Annotation, 0, len(anns))
	for _, a := range anns {
		for _, w := range focus {
			if annotated([]Annotation{a}, w) {
				out = append(out, a)
				break
			}
		}
	}
	return out
}

// addMissedFocus adds an annotation for each focus word the AI did not annotate: the word as it
// appears in the first sentence containing it, its lowercase topic form as lemma and the first
// dictionary meaning. Words the dictionary does not know are skipped (F18, FR-018).
func addMissedFocus(ctx context.Context, anns []Annotation, focus, sentences []string, dict Dictionary) ([]Annotation, error) {
	if dict == nil {
		return anns, nil
	}
	for _, w := range focus {
		if annotated(anns, w) {
			continue
		}
		for i, s := range sentences {
			span, ok := wordmatch.New(s, nil).Find(w)
			if !ok {
				continue
			}
			lemma := normalize(w)
			e, ok, err := dict.Resolve(ctx, lemma)
			if err != nil {
				return nil, fmt.Errorf("lesson: resolve %q: %w", lemma, err)
			}
			if ok && len(e.Meanings) > 0 && utf8.RuneCountInString(e.Meanings[0].Text) <= maxFocusMeaning {
				anns = append(anns, Annotation{Text: span, Lemma: lemma, MeaningVi: e.Meanings[0].Text, SentenceIndex: i})
			}
			break
		}
	}
	return anns, nil
}

// annotated reports whether an annotation is exactly the word w (by its text or its lemma).
func annotated(anns []Annotation, w string) bool {
	for _, a := range anns {
		for _, s := range []string{a.Text, a.Lemma} {
			if span, ok := wordmatch.New(s, nil).Find(w); ok && span == strings.TrimSpace(s) {
				return true
			}
		}
	}
	return false
}
