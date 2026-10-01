// Package ai defines the AI provider used for lesson annotation (F2) and lesson generation
// (F7). Providers are chosen by configuration so the lesson logic never depends on a
// specific vendor.
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

// Question is a multiple-choice comprehension question about a lesson (F15).
type Question struct {
	Prompt        string   `json:"prompt"`
	Options       []string `json:"options"`
	AnswerIndex   int      `json:"answerIndex"`
	ExplanationVi string   `json:"explanationVi"`
}

// GrammarNote explains one grammar point of a lesson in Vietnamese, with examples copied from
// the lesson.
type GrammarNote struct {
	Title    string   `json:"title"`
	BodyVi   string   `json:"bodyVi"`
	Examples []string `json:"examples"`
}

// LessonExtras is everything one annotation request returns: word annotations plus, since F15,
// comprehension questions, a grammar note and a writing prompt. Any part may be missing.
type LessonExtras struct {
	Annotations   []Annotation `json:"annotations"`
	Questions     []Question   `json:"questions"`
	GrammarNote   *GrammarNote `json:"grammarNote"`
	WritingPrompt string       `json:"writingPrompt"`
}

// LessonKind is the form of a generated lesson.
type LessonKind string

const (
	// KindReading is a continuous text.
	KindReading LessonKind = "reading"
	// KindDialogue is a conversation, one "Name: line" per turn.
	KindDialogue LessonKind = "dialogue"
)

// ValidKind reports whether s is a known LessonKind.
func ValidKind(s string) bool {
	return LessonKind(s) == KindReading || LessonKind(s) == KindDialogue
}

// GenerateRequest asks for Count new lessons of about Words words each.
type GenerateRequest struct {
	Level     string
	TopicName string
	Count     int
	Words     int
	Kind      LessonKind
	// Idea is an optional hint from the admin.
	Idea string
	// ExistingTitles are the lessons already in the topic, to avoid repeating them.
	ExistingTitles []string
}

// LessonDraft is a generated lesson as returned by a provider, before any checks.
type LessonDraft struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// Provider is the AI used by lessons.
type Provider interface {
	// Annotate annotates a lesson given its sentences (indexed from 0) and CEFR level, and writes
	// its questions, grammar note and writing prompt, all in one request.
	Annotate(ctx context.Context, sentences []string, level string) (LessonExtras, error)
	// GenerateLessons writes lesson drafts in a single request.
	GenerateLessons(ctx context.Context, req GenerateRequest) ([]LessonDraft, error)
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
func (Disabled) Annotate(context.Context, []string, string) (LessonExtras, error) {
	return LessonExtras{}, ErrNotConfigured
}

// GenerateLessons always returns ErrNotConfigured.
func (Disabled) GenerateLessons(context.Context, GenerateRequest) ([]LessonDraft, error) {
	return nil, ErrNotConfigured
}
