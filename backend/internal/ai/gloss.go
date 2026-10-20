package ai

import "context"

// GlossRequest asks for the meaning in context of every content word of a lesson (added
// 2026-10-20): what the reading popup shows first, "Trong bài này: …".
type GlossRequest struct {
	// Sentences are the lesson sentences, indexed from 0.
	Sentences []string
	Level     string
}

// Gloss is one word or phrase of a lesson as it is used there.
type Gloss struct {
	// Text is the word or phrase as written in the sentence; Lemma its dictionary form.
	Text  string `json:"text"`
	Lemma string `json:"lemma"`
	// MeaningVi is its Vietnamese meaning in this sentence, short.
	MeaningVi     string `json:"meaningVi"`
	SentenceIndex int    `json:"sentenceIndex"`
	POS           string `json:"pos"`
	// Phrase is the few words of the sentence around it ("a great way to relax") and PhraseVi their
	// Vietnamese, so the learner sees the word at work.
	Phrase   string `json:"phrase"`
	PhraseVi string `json:"phraseVi"`
}

// GlossProvider writes the glosses of a lesson. It is separate from Provider so that the many
// fakes of Provider need not implement it.
type GlossProvider interface {
	Glosses(ctx context.Context, req GlossRequest) ([]Gloss, error)
}

// DisabledGlosses is the gloss provider used when AI_PROVIDER=none or no key is set.
type DisabledGlosses struct{}

// Glosses always returns ErrNotConfigured.
func (DisabledGlosses) Glosses(context.Context, GlossRequest) ([]Gloss, error) {
	return nil, ErrNotConfigured
}
