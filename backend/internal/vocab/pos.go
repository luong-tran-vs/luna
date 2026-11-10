package vocab

import (
	"context"
	"strings"
)

// LessonPOS is what a LessonVocabulary may also offer: the lesson's words with only their part of
// speech, read faster than the whole vocabulary (added 2026-11-09). posOf uses it when there.
type LessonPOS interface {
	POS(ctx context.Context, lessonID string) ([]VocabItem, error)
}

// posCache keeps the parts of speech of the lessons met in one request, by lesson then lemma.
type posCache map[string]map[string]string

// posOf is the part of speech of a lesson card's word, as its lesson uses it (added 2026-10-20):
// the review picks wrong meanings of the same part of speech, which are harder to rule out. ""
// for a card typed by hand or a word the lesson no longer has. No AI request; best effort.
func (s *Service) posOf(ctx context.Context, cache posCache, c Card) string {
	if c.LessonID == "" || s.vocabulary == nil {
		return ""
	}
	byLemma, ok := cache[c.LessonID]
	if !ok {
		byLemma = map[string]string{}
		read := s.vocabulary.Vocabulary
		if p, ok := s.vocabulary.(LessonPOS); ok {
			read = p.POS
		}
		if items, err := read(ctx, c.LessonID); err == nil {
			for _, it := range items {
				byLemma[strings.ToLower(collapse(it.Lemma))] = it.POS
			}
		}
		cache[c.LessonID] = byLemma
	}
	return byLemma[strings.ToLower(collapse(c.Lemma))]
}
