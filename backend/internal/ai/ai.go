// Package ai defines the AI provider used for lesson annotation (F2, F15), lesson generation
// (F7), writing grades (F8), word explanations (F9) and vocabulary practice (F17). Providers
// are chosen by configuration so the app never depends on a specific vendor.
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
	// TargetWords lists, for each lesson in order, the topic words it must use (F18); a group may be
	// empty and the slice may be nil.
	TargetWords [][]string
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
	Annotate(ctx context.Context, req AnnotateRequest) (LessonExtras, error)
	// GenerateLessons writes lesson drafts in a single request.
	GenerateLessons(ctx context.Context, req GenerateRequest) ([]LessonDraft, error)
	// GradeWriting grades a learner's writing on four criteria in one request (F8).
	GradeWriting(ctx context.Context, req GradeRequest) (Grade, error)
	// Explain gives the meaning of a word or phrase in one sentence (F9).
	Explain(ctx context.Context, req ExplainRequest) (Explanation, error)
	// Practice writes the vocabulary practice of a lesson in one request (F17).
	Practice(ctx context.Context, req PracticeRequest) (Practice, error)
	// SuggestWords proposes new core words of a topic in one request (F18).
	SuggestWords(ctx context.Context, req SuggestWordsRequest) ([]string, error)
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
func (Disabled) Annotate(context.Context, AnnotateRequest) (LessonExtras, error) {
	return LessonExtras{}, ErrNotConfigured
}

// GenerateLessons always returns ErrNotConfigured.
func (Disabled) GenerateLessons(context.Context, GenerateRequest) ([]LessonDraft, error) {
	return nil, ErrNotConfigured
}

// GradeWriting always returns ErrNotConfigured.
func (Disabled) GradeWriting(context.Context, GradeRequest) (Grade, error) {
	return Grade{}, ErrNotConfigured
}

// GradeRequest is a learner's writing about a lesson.
type GradeRequest struct {
	Level      string
	LessonText string
	Prompt     string
	Text       string
}

// Criterion is the score (1–5) and Vietnamese comment of one grading criterion.
type Criterion struct {
	Score     int    `json:"score"`
	CommentVi string `json:"commentVi"`
}

// Grade is the AI's assessment of a writing, before any checks.
type Grade struct {
	Task          Criterion `json:"task"`
	Grammar       Criterion `json:"grammar"`
	Vocabulary    Criterion `json:"vocabulary"`
	Coherence     Criterion `json:"coherence"`
	OverallVi     string    `json:"overallVi"`
	CorrectedText string    `json:"correctedText"`
}

// Explain always returns ErrNotConfigured.
func (Disabled) Explain(context.Context, ExplainRequest) (Explanation, error) {
	return Explanation{}, ErrNotConfigured
}

// ExplainRequest asks for the meaning of Text as used in Sentence.
type ExplainRequest struct {
	Text     string
	Sentence string
	Level    string
}

// Explanation is the meaning of a word or phrase in context, as returned by a provider.
type Explanation struct {
	Lemma     string `json:"lemma"`
	MeaningVi string `json:"meaningVi"`
	NoteVi    string `json:"noteVi"`
}

// Practice always returns ErrNotConfigured.
func (Disabled) Practice(context.Context, PracticeRequest) (Practice, error) {
	return Practice{}, ErrNotConfigured
}

// PracticeWord is one vocabulary word of a lesson, as sent to the provider (F17).
type PracticeWord struct {
	Lemma     string
	Text      string
	MeaningVi string
}

// PracticeRequest asks for the vocabulary practice of a lesson (F17).
type PracticeRequest struct {
	Level     string
	Title     string
	Sentences []string
	Words     []PracticeWord
}

// Example is an English example sentence for one vocabulary word.
type Example struct {
	Lemma    string `json:"lemma"`
	Sentence string `json:"sentence"`
}

// Turn is one line of a dialogue; Speaker is 0 or 1.
type Turn struct {
	Speaker   int    `json:"speaker"`
	Text      string `json:"text"`
	MeaningVi string `json:"meaningVi"`
}

// Dialogue is a sample conversation between two named speakers.
type Dialogue struct {
	Speakers []string `json:"speakers"`
	Turns    []Turn   `json:"turns"`
}

// Translation is a Vietnamese sentence with its English answer and distractor words.
type Translation struct {
	Vi          string   `json:"vi"`
	En          string   `json:"en"`
	Distractors []string `json:"distractors"`
}

// Practice is the vocabulary practice of a lesson as returned by a provider, before any checks.
type Practice struct {
	ObjectiveVi  string        `json:"objectiveVi"`
	Examples     []Example     `json:"examples"`
	Dialogue     Dialogue      `json:"dialogue"`
	GrammarTipVi string        `json:"grammarTipVi"`
	Translations []Translation `json:"translations"`
}

// AnnotateRequest is a lesson to annotate.
type AnnotateRequest struct {
	// Sentences are the lesson sentences, indexed from 0.
	Sentences []string
	Level     string
	// FocusWords are topic words found in the lesson that must be annotated (F18); may be empty.
	FocusWords []string
}

// SuggestWordsRequest asks for Count new core English words or short phrases of a topic, at a
// CEFR level, none of them in Existing (F18).
type SuggestWordsRequest struct {
	Level     string
	TopicName string
	Existing  []string
	Count     int
}

// SuggestWords always returns ErrNotConfigured.
func (Disabled) SuggestWords(context.Context, SuggestWordsRequest) ([]string, error) {
	return nil, ErrNotConfigured
}
