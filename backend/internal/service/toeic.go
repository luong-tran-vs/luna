package service

import (
	"context"
	"errors"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/toeic"
	"github.com/luongtran/luna/backend/internal/vocab"
)

// initToeic builds the TOEIC practice (F27). It saves words to the vocabulary and counts study days,
// so it runs after initVocab, initProgress and initSettings. The pictures are drawn with the
// image AI the admin picks (Gemini or Codex) and the groups drafted with the text AI.
func (c *Container) initToeic() {
	c.toeic = toeic.NewService(toeic.Deps{
		Groups:    c.store.ToeicGroups(),
		Images:    c.store.ToeicImages(),
		Tests:     c.store.ToeicTests(),
		Attempts:  c.store.ToeicAttempts(),
		Mistakes:  c.store.ToeicMistakes(),
		Goals:     c.store.ToeicGoals(),
		StudyDays: c.studyDays,
		Cards:     toeicCards{c.vocab},
		Timezones: c.settings,
		Writer:    c.ai,
		Drawer:    toeicDrawer{gemini: c.imageAI, codex: c.codexImageAI},
		Now:       time.Now,
	})
}

// toeicCards adapts vocab.Service to toeic.Cards: a word of an answer goes to the notebook as a
// manual card; one already there is fine.
type toeicCards struct{ svc *vocab.Service }

func (t toeicCards) SaveWord(ctx context.Context, userID string, w toeic.CardInput) error {
	_, err := t.svc.Save(ctx, userID, vocab.Input{
		Text: w.Word, Lemma: w.Word, IPA: w.IPA, MeaningVi: w.MeaningVi, ContextSentence: w.Sentence, Source: string(vocab.SourceManual),
	})
	var exists *vocab.ExistsError
	var verr *vocab.ValidationError
	switch {
	case errors.As(err, &exists):
		return nil
	case errors.As(err, &verr):
		return &toeic.ValidationError{Fields: verr.Fields}
	}
	return err
}

// toeicDrawer draws the TOEIC pictures with the engine the admin picked: Gemini (the image model) or
// the Codex CLI of this machine (nil when CODEX_IMAGES is off).
type toeicDrawer struct {
	gemini ai.ImageProvider
	codex  ai.ImageProvider
}

func (d toeicDrawer) Draw(ctx context.Context, engine string, req ai.ImageRequest) (ai.Image, error) {
	switch engine {
	case toeic.EngineCodex:
		if d.codex == nil {
			return ai.Image{}, toeic.ErrCodexUnavailable
		}
		return d.codex.GenerateImage(ctx, req)
	default:
		if d.gemini == nil {
			return ai.Image{}, toeic.ErrDrawUnavailable
		}
		return d.gemini.GenerateImage(ctx, req)
	}
}

func (d toeicDrawer) CodexAvailable() bool { return d.codex != nil }
