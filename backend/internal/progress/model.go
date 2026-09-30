// Package progress stores a learner's results in lesson steps. F4 records dictation results.
package progress

import (
	"errors"
	"fmt"
	"time"
)

// Result is the latest check of one sentence in the Listening step.
type Result struct {
	SentenceIndex int
	Typed         string
	CorrectWords  int
	TotalWords    int
	CheckedAt     time.Time
}

// StoredResult is a Result with the lesson revision it was checked against.
type StoredResult struct {
	Result
	Revision int
}

// Summary totals a learner's dictation results for the current revision of a lesson.
type Summary struct {
	SentenceCount int
	CheckedCount  int
	CorrectWords  int
	TotalWords    int
	// Rate is CorrectWords / TotalWords, 0 when nothing is checked.
	Rate float64
	// Completed is true once every sentence is checked, correct or not.
	Completed bool
	// Results are sorted by sentence index.
	Results []Result
}

// DictationTotals sums a learner's dictation results over every lesson.
type DictationTotals struct {
	Sentences    int
	CorrectWords int
	TotalWords   int
}

// ErrLessonNotFound is returned when the lesson does not exist.
var ErrLessonNotFound = errors.New("progress: lesson not found")

// ValidationError lists invalid input fields with Vietnamese messages.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string { return fmt.Sprintf("progress: invalid input %v", e.Fields) }
