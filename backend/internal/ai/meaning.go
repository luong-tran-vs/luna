package ai

import "context"

// MeaningsRequest asks for what some words of one topic lack, the Vietnamese meaning, the IPA or
// both, all in one request (F24: filling the word bank).
type MeaningsRequest struct {
	TopicName string
	Words     []WordNeed
}

// WordNeed is one lowercase word or short phrase and what to give for it, so the answer spends no
// tokens on what the word already has.
type WordNeed struct {
	Word    string
	Meaning bool
	IPA     bool
}

// WordMeaning is the answer for one word; MeaningVi or IPA is empty when not asked or not given.
type WordMeaning struct {
	Word      string
	MeaningVi string
	IPA       string
}

// MeaningProvider fills in meanings and IPA. It is separate from Provider so that the many fakes
// of Provider need not implement it.
type MeaningProvider interface {
	WordMeanings(ctx context.Context, req MeaningsRequest) ([]WordMeaning, error)
}

// DisabledMeanings is the meaning provider used when AI_PROVIDER=none or no key is set.
type DisabledMeanings struct{}

// WordMeanings always returns ErrNotConfigured.
func (DisabledMeanings) WordMeanings(context.Context, MeaningsRequest) ([]WordMeaning, error) {
	return nil, ErrNotConfigured
}
