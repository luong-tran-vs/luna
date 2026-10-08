package ai

import "context"

// FixRequest asks for a corrected version of one item the lesson check flagged (F22). Area is
// "sentence", "annotation", "question" or "translation"; exactly the matching item field is set.
// Problem is why the item was flagged, in Vietnamese.
type FixRequest struct {
	Level     string
	Title     string
	Sentences []string
	Area      string
	Problem   string
	// Sentence is the index of the flagged sentence in Sentences.
	Sentence    int
	Annotation  *ReviewAnnotation
	Question    *FixQuestion
	Translation *ReviewTranslation
}

// FixQuestion is a comprehension question with its stored answer.
type FixQuestion struct {
	Prompt        string   `json:"prompt"`
	Options       []string `json:"options"`
	AnswerIndex   int      `json:"answerIndex"`
	ExplanationVi string   `json:"explanationVi"`
}

// FixResult is the corrected item; only the fields of the asked area are used. NoteVi says what
// was changed and why, in Vietnamese.
type FixResult struct {
	// Text is the corrected sentence.
	Text string `json:"text"`
	// MeaningVi is the corrected meaning of an annotation.
	MeaningVi string `json:"meaningVi"`
	// Question is the corrected question.
	Question FixQuestion `json:"question"`
	// Vi and En are the corrected translation pair.
	Vi     string `json:"vi"`
	En     string `json:"en"`
	NoteVi string `json:"noteVi"`
}

// SuggestFix always returns ErrNotConfigured.
func (Disabled) SuggestFix(context.Context, FixRequest) (FixResult, error) {
	return FixResult{}, ErrNotConfigured
}
