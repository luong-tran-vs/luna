package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/luongtran/luna/backend/internal/verbs"
	"github.com/luongtran/luna/backend/internal/wordbank"
)

// initVerbs builds the irregular verbs practice (the "Động từ" tab) and the verb pictures (added
// 2026-11-10). It runs after initWordBank, whose pictures the admin may reuse.
func (c *Container) initVerbs() {
	c.verbs = verbs.NewService(c.store.VerbProgress(), c.studyDays, time.Now)
	c.verbImages = verbs.NewImages(verbs.ImageDeps{
		Store:            c.store.VerbImages(),
		Gemini:           c.imageAI,
		Codex:            c.codexImageAI,
		CodexConcurrency: c.cfg.CodexImageConcurrency,
		Bank:             verbBank{c.wordBank},
		Log:              c.log,
		Now:              time.Now,
	})
}

// verbBank adapts wordbank.Service to verbs.Bank.
type verbBank struct {
	svc *wordbank.Service
}

func (b verbBank) Pictures(ctx context.Context, words []string) (map[string]verbs.BankPicture, error) {
	found, err := b.svc.Words(ctx, words)
	if err != nil {
		return nil, fmt.Errorf("word bank words: %w", err)
	}
	out := make(map[string]verbs.BankPicture, len(found))
	for lemma, w := range found {
		if w.HasImage() {
			out[lemma] = verbs.BankPicture{MeaningVi: w.MeaningVi, CreatedAt: w.ImageAt}
		}
	}
	return out, nil
}

func (b verbBank) Image(ctx context.Context, word string) (string, []byte, error) {
	img, err := b.svc.Image(ctx, word)
	if errors.Is(err, wordbank.ErrNotFound) || errors.Is(err, wordbank.ErrImageNotFound) {
		return "", nil, verbs.ErrNoBankImage
	}
	if err != nil {
		return "", nil, fmt.Errorf("word bank image: %w", err)
	}
	return img.MIME, img.Data, nil
}
