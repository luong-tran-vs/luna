package ai

import "context"

// ReviewRequest asks a second AI to read over what the first one wrote for a lesson (F22). The
// questions carry no answer on purpose: the reviewer answers them on its own.
type ReviewRequest struct {
	Level        string
	Title        string
	Sentences    []string
	Annotations  []ReviewAnnotation
	Questions    []ReviewQuestion
	Translations []ReviewTranslation
}

// ReviewAnnotation is an annotation of the lesson; Index is its place in the lesson's list and
// SentenceIndex the sentence it comes from.
type ReviewAnnotation struct {
	Index         int    `json:"index"`
	Text          string `json:"text"`
	Lemma         string `json:"lemma"`
	MeaningVi     string `json:"meaningVi"`
	SentenceIndex int    `json:"sentenceIndex"`
}

// ReviewQuestion is a comprehension question without its answer or explanation.
type ReviewQuestion struct {
	Index   int      `json:"index"`
	Prompt  string   `json:"prompt"`
	Options []string `json:"options"`
}

// ReviewTranslation is a Vietnamese sentence of the practice with its English answer.
type ReviewTranslation struct {
	Index int    `json:"index"`
	Vi    string `json:"vi"`
	En    string `json:"en"`
}

// ReviewIssue is something the reviewer finds wrong at Index of the list it belongs to.
type ReviewIssue struct {
	Index  int    `json:"index"`
	NoteVi string `json:"noteVi"`
}

// ReviewAnswer is the reviewer's own answer to the question at Index. Ambiguous is true when, to
// the reviewer, more than one option is right or none is.
type ReviewAnswer struct {
	Index       int    `json:"index"`
	ChoiceIndex *int   `json:"choiceIndex,omitempty"`
	Ambiguous   bool   `json:"ambiguous"`
	NoteVi      string `json:"noteVi"`
}

// ReviewResult lists only the items the reviewer finds wrong, plus its answer to each question.
type ReviewResult struct {
	Sentences    []ReviewIssue  `json:"sentences"`
	Annotations  []ReviewIssue  `json:"annotations"`
	Translations []ReviewIssue  `json:"translations"`
	Answers      []ReviewAnswer `json:"answers"`
}

// ReviewLesson always returns ErrNotConfigured.
func (Disabled) ReviewLesson(context.Context, ReviewRequest) (ReviewResult, error) {
	return ReviewResult{}, ErrNotConfigured
}
