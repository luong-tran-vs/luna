package vocab

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

const maxMisses = 30

// MissResult is what PracticeMisses did: cards created, and saved cards brought forward.
type MissResult struct {
	Added       int
	Rescheduled int
}

// PracticeMisses puts words the learner got wrong in a lesson's practice into the review queue,
// due now. A word not yet in the notebook becomes a card (data from the lesson's annotations, never
// from the client); a saved card due later is brought forward without touching its FSRS state, so
// the next real review decides its interval. Words not in the lesson are skipped.
func (s *Service) PracticeMisses(ctx context.Context, userID, lessonID string, words []string) (MissResult, error) {
	if len(words) == 0 || len(words) > maxMisses {
		return MissResult{}, &ValidationError{Fields: map[string]string{"words": "Cần từ 1 đến 30 từ"}}
	}
	loc, err := s.location(ctx, userID)
	if err != nil {
		return MissResult{}, err
	}
	items, err := s.vocabulary.Vocabulary(ctx, lessonID)
	if err != nil {
		return MissResult{}, fmt.Errorf("vocab: lesson vocabulary: %w", err)
	}
	byWord := make(map[string]VocabItem, 2*len(items))
	for _, it := range items {
		byWord[strings.ToLower(collapse(it.Text))] = it
	}
	for _, it := range items { // a lemma wins over another word's text
		byWord[strings.ToLower(collapse(it.Lemma))] = it
	}

	var res MissResult
	now := s.now()
	done := map[string]bool{}
	for _, w := range words {
		it, ok := byWord[strings.ToLower(collapse(w))]
		lemma := strings.ToLower(collapse(it.Lemma))
		if !ok || done[lemma] {
			continue
		}
		done[lemma] = true

		c, err := s.repo.FindByLemma(ctx, userID, lemma)
		switch {
		case err == nil:
			sched := effective(c, loc)
			if !sched.Due.After(now) {
				continue
			}
			sched.Due = now
			updated, err := s.repo.UpdateSchedule(ctx, userID, c.ID, sched.Reps, sched)
			if err != nil {
				return MissResult{}, fmt.Errorf("vocab: update schedule: %w", err)
			}
			if updated {
				res.Rescheduled++
			}
		case errors.Is(err, ErrNotFound):
			_, err := s.repo.Create(ctx, Card{
				UserID: userID, Text: collapse(it.Text), Lemma: lemma, IPA: it.IPA, MeaningVi: it.MeaningVi,
				ContextSentence: it.Sentence, LessonID: lessonID, Source: SourceAI, CreatedAt: now,
				Schedule: Schedule{Due: now, State: StateNew},
			})
			if errors.Is(err, ErrExists) {
				continue
			}
			if err != nil {
				return MissResult{}, fmt.Errorf("vocab: create: %w", err)
			}
			res.Added++
		default:
			return MissResult{}, fmt.Errorf("vocab: find card: %w", err)
		}
	}
	return res, nil
}

// MaxGameMisses bounds the words of one game round brought back to the review queue (F26).
const MaxGameMisses = 60

// GameMisses brings the words a learner got wrong in a game (F26) back to the review queue: a
// saved card due later becomes due now, its FSRS state kept, like a saved card in PracticeMisses.
// A word without a card is skipped (a game never creates cards); each lemma counts once. It
// returns how many cards were brought forward.
func (s *Service) GameMisses(ctx context.Context, userID string, lemmas []string) (int, error) {
	if len(lemmas) > MaxGameMisses {
		return 0, &ValidationError{Fields: map[string]string{"wrongWords": fmt.Sprintf("Tối đa %d từ mỗi lượt", MaxGameMisses)}}
	}
	if len(lemmas) == 0 {
		return 0, nil
	}
	loc, err := s.location(ctx, userID)
	if err != nil {
		return 0, err
	}
	now := s.now()
	done := map[string]bool{}
	moved := 0
	for _, w := range lemmas {
		lemma := strings.ToLower(collapse(w))
		if lemma == "" || done[lemma] {
			continue
		}
		done[lemma] = true
		c, err := s.repo.FindByLemma(ctx, userID, lemma)
		if errors.Is(err, ErrNotFound) {
			continue
		}
		if err != nil {
			return moved, fmt.Errorf("vocab: find card: %w", err)
		}
		sched := effective(c, loc)
		if !sched.Due.After(now) {
			continue
		}
		sched.Due = now
		updated, err := s.repo.UpdateSchedule(ctx, userID, c.ID, sched.Reps, sched)
		if err != nil {
			return moved, fmt.Errorf("vocab: update schedule: %w", err)
		}
		if updated {
			moved++
		}
	}
	return moved, nil
}

// maxGameCards bounds the cards read to build a game round (F26).
const maxGameCards = 5000

// GameCards returns the learner's cards for the games (F26), newest first, each with the schedule
// it is reviewed on (a card saved before F5 gets its first due day).
func (s *Service) GameCards(ctx context.Context, userID string) ([]Card, error) {
	loc, err := s.location(ctx, userID)
	if err != nil {
		return nil, err
	}
	cards, _, err := s.repo.List(ctx, userID, ListQuery{}, maxGameCards, 0)
	if err != nil {
		return nil, fmt.Errorf("vocab: list cards: %w", err)
	}
	for i := range cards {
		cards[i].Schedule = effective(cards[i], loc)
	}
	return cards, nil
}
