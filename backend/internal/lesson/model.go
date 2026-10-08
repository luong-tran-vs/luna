// Package lesson manages lessons, their sentences and AI annotations. Topics and their
// roadmaps live in package topic, reached through the Topics port.
package lesson

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/luongtran/luna/backend/internal/job"
)

// Level is a CEFR level.
type Level string

// Levels lists the valid CEFR levels in order.
var Levels = []Level{"A1", "A2", "B1", "B2", "C1", "C2"}

// ValidLevel reports whether s is one of Levels.
func ValidLevel(s string) bool { return slices.Contains(Levels, Level(s)) }

// Status is the state of a lesson's annotation or practice work.
type Status string

const (
	// StatusNone means the work has never been queued (only used for practice).
	StatusNone    Status = ""
	StatusRunning Status = "running"
	StatusDone    Status = "done"
	StatusFailed  Status = "failed"
)

// Sentence is one sentence of a lesson.
type Sentence struct {
	Index int
	Text  string
}

// Annotation explains a word or phrase of a lesson in context.
type Annotation struct {
	Text          string
	Lemma         string
	MeaningVi     string
	SentenceIndex int
	EditedByAdmin bool
	// POS is the part of speech as used in the sentence (see POSValues), "" when the AI gave none
	// (lessons annotated before it was asked for).
	POS string
}

// Lesson is a text learners study for one day.
type Lesson struct {
	ID      string
	Title   string
	Content string
	// Level always equals the level of the lesson's topic.
	Level   Level
	TopicID string
	Source  string
	License string
	// GrammarPointID is the syllabus point the lesson teaches, "" when none is assigned.
	GrammarPointID string
	// TargetWords are the topic words the lesson was generated to teach (F18), set when it is
	// created. While some of them are in the content, they are the lesson's whole vocabulary.
	TargetWords      []string
	Revision         int
	Sentences        []Sentence
	AnnotationStatus Status
	AnnotationError  string
	Annotations      []Annotation
	// Extras are the comprehension questions, grammar note and writing prompt (F15).
	Extras              Extras
	ExtrasEditedByAdmin bool
	// QuizVersion is bumped whenever the question set is replaced; answers belong to one version.
	QuizVersion int
	// Practice is the vocabulary practice (F17), nil until generated.
	Practice       *Practice
	PracticeStatus Status
	PracticeError  string
	// PracticeVersion is bumped each time a practice is saved, so an old practice job cannot overwrite a newer one.
	PracticeVersion int
	// Review is the AI check of the lesson (F22), nil until an admin runs it. Writing the content drops it.
	Review *Review
	// Draft hides the lesson from learners until an admin publishes it. Saved AI drafts start as
	// drafts; lessons added by hand, and those saved before publishing existed, are published.
	Draft     bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// StatusOf returns the status of the given kind of background work.
func (l Lesson) StatusOf(t job.Type) Status {
	switch t {
	case job.TypePractice:
		return l.PracticeStatus
	default:
		return l.AnnotationStatus
	}
}

// Summary is the list view of a lesson.
type Summary struct {
	ID               string
	Title            string
	Level            Level
	TopicID          string
	TopicName        string
	AnnotationStatus Status
	InRoadmap        bool
	// Flags counts the flags of the AI check the admin has not confirmed; Checked and Verified say
	// whether the lesson was checked and whether the admin marked the check as done (F22).
	Flags     int
	Checked   bool
	Verified  bool
	Draft     bool
	CreatedAt time.Time
}

// Filter narrows the lesson list; empty fields match everything.
type Filter struct {
	Level   Level
	TopicID string
}

// TopicWord is a word of the topic vocabulary (F18); Level is the lowest level it is meant for,
// "" for every level.
type TopicWord struct {
	Text  string
	Level Level
}

// fits reports whether the word may be given to a lesson of level.
func (w TopicWord) fits(level Level) bool {
	return w.Level == "" || slices.Index(Levels, w.Level) <= slices.Index(Levels, level)
}

// TopicRef is what lessons need to know about their topic. A topic is shared by every level.
type TopicRef struct {
	ID   string
	Name string
	// Words is the topic vocabulary (F18).
	Words []TopicWord
}

// WordTexts returns the texts of the topic words.
func (t TopicRef) WordTexts() []string {
	out := make([]string, len(t.Words))
	for i, w := range t.Words {
		out[i] = w.Text
	}
	return out
}

// Place is a roadmap: a topic at one level.
type Place struct {
	TopicID string
	Level   Level
}

var (
	ErrNotFound          = errors.New("lesson: not found")
	ErrInRoadmap         = errors.New("lesson: lesson is in the roadmap")
	ErrTopicNotFound     = errors.New("lesson: topic not found")
	ErrNotFailed         = errors.New("lesson: only failed work can be retried")
	ErrAnnotationRunning = errors.New("lesson: annotation is running")
	// ErrUnusableDraft means none of the generated drafts passed the checks.
	ErrUnusableDraft = errors.New("lesson: AI returned no usable draft")
	// ErrQuizChanged means an answer was sent for an older version of the questions.
	ErrQuizChanged = errors.New("lesson: questions changed")
	// ErrNoQuiz means the lesson has no comprehension questions.
	ErrNoQuiz = errors.New("lesson: lesson has no questions")
	// ErrNoValidPractice means nothing the AI wrote for the practice passed the checks.
	ErrNoValidPractice = errors.New("lesson: AI returned no usable practice")
	// ErrPracticeRunning means the practice is already being generated.
	ErrPracticeRunning = errors.New("lesson: practice is running")
	// ErrAnnotationNotDone means the practice needs the lesson annotations first.
	ErrAnnotationNotDone = errors.New("lesson: annotations are not done")
)

// ValidationError lists invalid input fields with Vietnamese messages for the admin.
type ValidationError struct {
	Fields map[string]string
	// cause is the error behind the message, if any, kept for the log.
	cause error
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("lesson: invalid input %v", e.Fields)
}

// Unwrap returns the error behind the message, nil for a plain invalid input.
func (e *ValidationError) Unwrap() error { return e.cause }

// Question is a multiple-choice comprehension question about a lesson (F15).
type Question struct {
	Prompt        string
	Options       []string
	AnswerIndex   int
	ExplanationVi string
}

// GrammarNote explains one grammar point of a lesson, with examples copied from it.
type GrammarNote struct {
	Title    string
	BodyVi   string
	Examples []string
}

// Extras are the parts of a lesson written with its annotations (F15). Any part may be empty.
type Extras struct {
	Questions     []Question
	GrammarNote   *GrammarNote
	WritingPrompt string
}

// Answer is a learner's answer to one question of one question-set version.
type Answer struct {
	UserID        string
	LessonID      string
	QuizVersion   int
	QuestionIndex int
	Choice        int
	Correct       bool
	AnsweredAt    time.Time
}
