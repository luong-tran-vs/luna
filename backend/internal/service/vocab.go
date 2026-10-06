package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/luongtran/luna/backend/internal/lesson"
	"github.com/luongtran/luna/backend/internal/vocab"
)

// initVocab builds the vocabulary service: the notebook and the FSRS review (F5).
func (c *Container) initVocab() {
	c.vocab = vocab.NewService(vocab.Deps{
		Repo: c.store.Cards(),
		Logs: c.store.ReviewLogs(),
		LessonExists: func(ctx context.Context, id string) (bool, error) {
			_, err := c.lessons.Get(ctx, id)
			if errors.Is(err, lesson.ErrNotFound) {
				return false, nil
			}
			return err == nil, err
		},
		Timezones:  c.settings,
		Vocabulary: lessonVocabulary{c.reader},
		Titles:     lessonTitles{c.lessons},
		Now:        time.Now,
	})
}

// lessonTitles adapts the lesson repository to vocab.LessonTitles (progress uses it too).
type lessonTitles struct {
	repo lesson.Repository
}

func (l lessonTitles) Titles(ctx context.Context, ids []string) (map[string]string, error) {
	summaries, err := l.repo.Summaries(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("lesson summaries: %w", err)
	}
	out := make(map[string]string, len(summaries))
	for _, s := range summaries {
		out[s.ID] = s.Title
	}
	return out, nil
}

// lessonVocabulary adapts lesson.Reader to vocab.LessonVocabulary.
type lessonVocabulary struct {
	reader *lesson.Reader
}

func (l lessonVocabulary) Vocabulary(ctx context.Context, id string) ([]vocab.VocabItem, error) {
	items, _, err := l.reader.Vocabulary(ctx, id)
	if errors.Is(err, lesson.ErrNotFound) {
		return nil, vocab.ErrLessonNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lesson vocabulary: %w", err)
	}
	out := make([]vocab.VocabItem, len(items))
	for i, it := range items {
		out[i] = vocab.VocabItem{
			Lemma: it.Lemma, Text: it.Text, MeaningVi: it.MeaningVi, IPA: it.IPA,
			SentenceIndex: it.SentenceIndex, Sentence: it.Sentence,
		}
	}
	return out, nil
}
