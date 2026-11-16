package ai

import "context"

// GrammarSpotRequest asks for sentences of the "Bắn lỗi" game (F26) for one grammar point: each has
// exactly one mistake of the point.
type GrammarSpotRequest struct {
	Level    string
	TitleEn  string
	Pattern  string
	Examples []string
	// Existing are the right sentences the point already has, not to be repeated.
	Existing []string
	// Count is how many sentences to write, at most 10.
	Count int
}

// GrammarSpotItem is one generated sentence: Sentence contains Wrong once; replacing it with Right
// makes the sentence correct; Distractors are two other fixes that are still wrong; RuleVi is the
// rule in one short Vietnamese sentence.
type GrammarSpotItem struct {
	Sentence    string   `json:"sentence"`
	Wrong       string   `json:"wrong"`
	Right       string   `json:"right"`
	Distractors []string `json:"distractors"`
	RuleVi      string   `json:"ruleVi"`
}

// GrammarSpotItems always returns ErrNotConfigured.
func (Disabled) GrammarSpotItems(context.Context, GrammarSpotRequest) ([]GrammarSpotItem, error) {
	return nil, ErrNotConfigured
}
