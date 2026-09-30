// Package ai defines the AI provider used for lesson annotation. Providers are chosen by
// configuration so the lesson logic never depends on a specific vendor.
package ai

import (
	"context"
	"errors"
)

// Annotation is a word or phrase worth learning, as returned by a provider.
type Annotation struct {
	Text          string `json:"text"`
	Lemma         string `json:"lemma"`
	MeaningVi     string `json:"meaningVi"`
	SentenceIndex int    `json:"sentenceIndex"`
}

// Provider annotates a lesson given its sentences (indexed from 0) and CEFR level.
type Provider interface {
	Annotate(ctx context.Context, sentences []string, level string) ([]Annotation, error)
}

var (
	// ErrNotConfigured means no provider or API key is set; retrying cannot help.
	ErrNotConfigured = errors.New("ai: provider not configured")
	// ErrInvalidKey means the provider rejected the API key; retrying cannot help.
	ErrInvalidKey = errors.New("ai: invalid API key")
	// ErrQuota means the free-tier quota is exhausted for now.
	ErrQuota = errors.New("ai: quota exceeded")
)

// Disabled is the provider used when AI_PROVIDER=none or no key is set.
type Disabled struct{}

// Annotate always returns ErrNotConfigured.
func (Disabled) Annotate(context.Context, []string, string) ([]Annotation, error) {
	return nil, ErrNotConfigured
}
