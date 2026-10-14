package vocab

import (
	"context"
	"strings"
)

// SentenceKey is how an example is matched with a lesson sentence: case and spacing ignored.
func SentenceKey(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}

// sentenceCache keeps the translations of the lessons met in one request (nil: none).
type sentenceCache map[string]map[string]string

// sentenceVi is the card's translation of its example; a lesson card without one gets the
// lesson's translation of that sentence. No AI request: only translations the lesson already has.
// Best effort: a failed lookup gives "".
func (s *Service) sentenceVi(ctx context.Context, cache sentenceCache, c Card) string {
	if c.ContextSentenceVi != "" || c.LessonID == "" || c.ContextSentence == "" || s.translations == nil {
		return c.ContextSentenceVi
	}
	tr, ok := cache[c.LessonID]
	if !ok {
		var err error
		if tr, err = s.translations.Translations(ctx, c.LessonID); err != nil {
			tr = nil
		}
		cache[c.LessonID] = tr
	}
	return tr[SentenceKey(c.ContextSentence)]
}

// fillSentenceVi is sentenceVi for a saved card: a translation found now is stored, so the next
// read has it without looking it up. A failed save still returns it.
func (s *Service) fillSentenceVi(ctx context.Context, userID string, cache sentenceCache, c Card) string {
	vi := s.sentenceVi(ctx, cache, c)
	if vi != "" && c.ContextSentenceVi == "" {
		_, _ = s.repo.UpdateDetails(ctx, userID, c.ID, Details{ContextSentenceVi: &vi})
	}
	return vi
}
