// Package lesson manages lessons, their sentences, audio and AI annotations. Topics and their
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

// Status is the state of a lesson's audio or annotation work.
type Status string

const (
	// StatusNone means the work has never been queued (only used for practice).
	StatusNone    Status = ""
	StatusRunning Status = "running"
	StatusDone    Status = "done"
	StatusFailed  Status = "failed"
)

// Sentence is one sentence of a lesson. AudioPath is the URL of its mp3, empty until ready.
type Sentence struct {
	Index     int
	Text      string
	AudioPath string
}

// Annotation explains a word or phrase of a lesson in context.
type Annotation struct {
	Text          string
	Lemma         string
	MeaningVi     string
	SentenceIndex int
	EditedByAdmin bool
}

// Lesson is a text learners study for one day.
type Lesson struct {
	ID      string
	Title   string
	Content string
	// Level always equals the level of the lesson's topic.
	Level            Level
	TopicID          string
	Source           string
	License          string
	Revision         int
	Sentences        []Sentence
	AudioStatus      Status
	AudioError       string
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
	// PracticeVersion is bumped each time a practice is saved; practice audio files belong to one version.
	PracticeVersion int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// StatusOf returns the status of the given kind of background work.
func (l Lesson) StatusOf(t job.Type) Status {
	switch t {
	case job.TypeTTS:
		return l.AudioStatus
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
	AudioStatus      Status
	AnnotationStatus Status
	InRoadmap        bool
	CreatedAt        time.Time
}

// Filter narrows the lesson list; empty fields match everything.
type Filter struct {
	Level   Level
	TopicID string
}

// TopicRef is what lessons need to know about their topic.
type TopicRef struct {
	ID    string
	Name  string
	Level Level
	// Words is the topic vocabulary (F18).
	Words []string
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
	// ErrAlreadyAnswered is matched by *AlreadyAnsweredError.
	ErrAlreadyAnswered = errors.New("lesson: question already answered")
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
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("lesson: invalid input %v", e.Fields)
}

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

// AlreadyAnsweredError carries the answer stored first for a question answered again.
type AlreadyAnsweredError struct {
	Answer Answer
}

func (e *AlreadyAnsweredError) Error() string { return ErrAlreadyAnswered.Error() }

// Is makes errors.Is(err, ErrAlreadyAnswered) true.
func (e *AlreadyAnsweredError) Is(target error) bool { return target == ErrAlreadyAnswered }
