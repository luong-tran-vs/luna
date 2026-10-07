package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
		Timezones:      c.settings,
		Vocabulary:     lessonVocabulary{c.reader},
		Titles:         lessonTitles{c.lessons},
		Pronunciations: pronunciations{c.dict},
		Now:            time.Now,
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

// pronunciations adapts the offline dictionary to vocab.Pronunciations.
type pronunciations struct {
	dict lesson.Dictionary
}

// IPA is the transcription of lemma when the dictionary has that word itself. A phrase joins the
// transcriptions of its words ("good morning" → "/ˈɡʊd ˈmɔr.nɪŋ/"), only when it knows every word.
func (p pronunciations) IPA(ctx context.Context, lemma string) (string, error) {
	words := strings.Fields(lemma)
	if len(words) < 2 {
		return p.word(ctx, lemma)
	}
	parts := make([]string, 0, len(words))
	for _, w := range words {
		ipa, err := p.word(ctx, w)
		if err != nil || ipa == "" {
			return "", err
		}
		parts = append(parts, strings.Trim(ipa, "/[] "))
	}
	return "/" + strings.Join(parts, " ") + "/", nil
}

// word is the transcription of one word, "" unless the dictionary has that exact form.
func (p pronunciations) word(ctx context.Context, w string) (string, error) {
	e, ok, err := p.dict.Resolve(ctx, w)
	if err != nil {
		return "", fmt.Errorf("dictionary: resolve %q: %w", w, err)
	}
	if !ok || e.Word != w {
		return "", nil
	}
	return e.IPA, nil
}
