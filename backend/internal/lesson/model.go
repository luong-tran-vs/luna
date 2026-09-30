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
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// StatusOf returns the status of the given kind of background work.
func (l Lesson) StatusOf(t job.Type) Status {
	if t == job.TypeTTS {
		return l.AudioStatus
	}
	return l.AnnotationStatus
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
}

var (
	ErrNotFound          = errors.New("lesson: not found")
	ErrInRoadmap         = errors.New("lesson: lesson is in the roadmap")
	ErrTopicNotFound     = errors.New("lesson: topic not found")
	ErrNotFailed         = errors.New("lesson: only failed work can be retried")
	ErrAnnotationRunning = errors.New("lesson: annotation is running")
)

// ValidationError lists invalid input fields with Vietnamese messages for the admin.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("lesson: invalid input %v", e.Fields)
}
