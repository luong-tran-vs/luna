package vocab

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

const maxBulk = 200

// BulkResult is what SaveBulk added.
type BulkResult struct {
	Added int
	Cards []Card
}

// SaveBulk saves the given words of a lesson's Vocabulary section. Card data comes from the
// lesson's annotations, never from the client; words already in the notebook and words not in
// the lesson are skipped, so saving twice never creates duplicates.
func (s *Service) SaveBulk(ctx context.Context, userID, lessonID string, lemmas []string) (BulkResult, error) {
	if len(lemmas) == 0 || len(lemmas) > maxBulk {
		return BulkResult{}, &ValidationError{Fields: map[string]string{"lemmas": "Cần từ 1 đến 200 từ"}}
	}
	loc, err := s.location(ctx, userID)
	if err != nil {
		return BulkResult{}, err
	}
	items, err := s.vocabulary.Vocabulary(ctx, lessonID)
	if err != nil {
		return BulkResult{}, fmt.Errorf("vocab: lesson vocabulary: %w", err)
	}
	byLemma := make(map[string]VocabItem, len(items))
	for _, it := range items {
		byLemma[strings.ToLower(collapse(it.Lemma))] = it
	}

	res := BulkResult{Cards: []Card{}}
	now := s.now()
	done := map[string]bool{}
	for _, l := range lemmas {
		lemma := strings.ToLower(collapse(l))
		it, ok := byLemma[lemma]
		if !ok || done[lemma] {
			continue
		}
		done[lemma] = true
		c, err := s.repo.Create(ctx, Card{
			UserID: userID, Text: collapse(it.Text), Lemma: lemma, IPA: it.IPA, MeaningVi: it.MeaningVi,
			ContextSentence: it.Sentence, LessonID: lessonID, Source: SourceAI, CreatedAt: now,
			Schedule: Schedule{Due: FirstDue(now, loc), State: StateNew},
		})
		if errors.Is(err, ErrExists) {
			continue
		}
		if err != nil {
			return BulkResult{}, fmt.Errorf("vocab: create: %w", err)
		}
		res.Added++
		res.Cards = append(res.Cards, c)
	}
	return res, nil
}
