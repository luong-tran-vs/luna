package lesson

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"unicode/utf8"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/job"
)

// Gloss is one word or phrase of a lesson with its meaning there (added 2026-10-20): what the
// reading popup shows first, "Trong bài này: thư giãn", with the few words around it. Unlike the
// annotations, glosses are not the lesson's vocabulary: they cover every content word.
type Gloss struct {
	Text          string
	Lemma         string
	MeaningVi     string
	SentenceIndex int
	POS           string
	Phrase        string
	PhraseVi      string
}

// GlossRepository keeps the glosses of each lesson for one revision. A lesson whose content changed
// since has none (Get reports the revision they belong to).
type GlossRepository interface {
	// Get returns the glosses and the revision they were written for; ok is false when there are none.
	Get(ctx context.Context, lessonID string) (revision int, glosses []Gloss, ok bool, err error)
	// Save replaces the lesson's glosses; an empty list records that there is nothing to show.
	Save(ctx context.Context, lessonID string, revision int, glosses []Gloss) error
	// Revisions gives the revision the glosses of each lesson that has some were written for.
	Revisions(ctx context.Context) (map[string]int, error)
}

const (
	// maxGlosses bounds what is kept of one answer: a lesson has a few hundred words.
	maxGlosses = 300
	// maxGlossMeaning is the longest meaning kept, in characters: a few words, not a definition.
	maxGlossMeaning = 80
)

// ProcessGlosses asks the AI once for the glosses of a lesson whose annotations are done and keeps
// those that pass CleanGlosses. A lesson rewritten meanwhile is skipped.
func (s *Service) ProcessGlosses(ctx context.Context, j job.Job) error {
	if s.GlossAI == nil || s.Glosses == nil {
		return nil
	}
	l, ok, err := s.current(ctx, j)
	if !ok {
		return err
	}
	if l.AnnotationStatus != StatusDone {
		return job.Permanent(ErrAnnotationNotDone)
	}
	res, err := s.GlossAI.Glosses(ctx, ai.GlossRequest{Sentences: sentenceTexts(l.Sentences), Level: string(l.Level)})
	if err != nil {
		if errors.Is(err, ai.ErrNotConfigured) || errors.Is(err, ai.ErrInvalidKey) {
			return job.Permanent(err)
		}
		return err
	}
	if err := s.Glosses.Save(ctx, l.ID, l.Revision, CleanGlosses(res, sentenceTexts(l.Sentences))); err != nil {
		return fmt.Errorf("lesson: save glosses: %w", err)
	}
	return nil
}

// glossesFailed records an empty list for the revision, so the lesson is not queued again at every
// start; the popup then falls back to the dictionary as before.
func (s *Service) glossesFailed(ctx context.Context, j job.Job, err error) {
	s.Log.WarnContext(ctx, "glosses failed", slog.String("lesson_id", j.LessonID), slog.Any("error", err))
	if s.Glosses == nil {
		return
	}
	if serr := s.Glosses.Save(ctx, j.LessonID, j.Revision, []Gloss{}); serr != nil {
		s.Log.ErrorContext(ctx, "record glosses failure", slog.String("lesson_id", j.LessonID), slog.Any("error", serr))
	}
}

// queueGlosses queues the glosses of a lesson just annotated; without a gloss provider it does nothing.
func (s *Service) queueGlosses(ctx context.Context, id string, revision int) error {
	if s.GlossAI == nil || s.Glosses == nil {
		return nil
	}
	return s.enqueue(ctx, id, revision, job.TypeGlosses)
}

// QueueMissingGlosses queues the glosses of the annotated lessons that have none for their current
// revision (lessons annotated before glosses existed, or rewritten since). It returns how many.
func (s *Service) QueueMissingGlosses(ctx context.Context) (int, error) {
	if s.GlossAI == nil || s.Glosses == nil {
		return 0, nil
	}
	have, err := s.Glosses.Revisions(ctx)
	if err != nil {
		return 0, fmt.Errorf("lesson: gloss revisions: %w", err)
	}
	list, err := s.Lessons.List(ctx, Filter{})
	if err != nil {
		return 0, fmt.Errorf("lesson: list: %w", err)
	}
	n := 0
	for _, sum := range list {
		if sum.AnnotationStatus != StatusDone {
			continue
		}
		l, err := s.Lessons.Get(ctx, sum.ID)
		if errors.Is(err, ErrNotFound) {
			continue
		}
		if err != nil {
			return n, fmt.Errorf("lesson: load: %w", err)
		}
		if rev, ok := have[l.ID]; ok && rev == l.Revision {
			continue
		}
		if err := s.enqueue(ctx, l.ID, l.Revision, job.TypeGlosses); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

// CleanGlosses keeps the glosses that belong to the lesson: a sentence that exists and holds the
// text (case ignored), a short meaning, one per lemma and sentence. A phrase the sentence does not
// hold is dropped with its translation; the gloss stays.
func CleanGlosses(raw []ai.Gloss, sentences []string) []Gloss {
	out := []Gloss{}
	seen := map[string]bool{}
	for _, g := range raw {
		if len(out) >= maxGlosses {
			break
		}
		if g.SentenceIndex < 0 || g.SentenceIndex >= len(sentences) {
			continue
		}
		sentence := strings.ToLower(sentences[g.SentenceIndex])
		text := strings.Join(strings.Fields(g.Text), " ")
		meaning := strings.TrimSpace(g.MeaningVi)
		lemma := strings.ToLower(strings.Join(strings.Fields(g.Lemma), " "))
		if lemma == "" {
			lemma = strings.ToLower(text)
		}
		if text == "" || meaning == "" || utf8.RuneCountInString(meaning) > maxGlossMeaning ||
			!strings.Contains(sentence, strings.ToLower(text)) {
			continue
		}
		key := fmt.Sprintf("%d|%s", g.SentenceIndex, lemma)
		if seen[key] {
			continue
		}
		seen[key] = true
		phrase := strings.Join(strings.Fields(g.Phrase), " ")
		phraseVi := strings.TrimSpace(g.PhraseVi)
		if phrase == "" || phraseVi == "" || !strings.Contains(sentence, strings.ToLower(phrase)) {
			phrase, phraseVi = "", ""
		}
		out = append(out, Gloss{
			Text: text, Lemma: lemma, MeaningVi: meaning, SentenceIndex: g.SentenceIndex,
			POS: strings.TrimSpace(g.POS), Phrase: phrase, PhraseVi: phraseVi,
		})
	}
	return out
}
