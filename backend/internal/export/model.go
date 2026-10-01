// Package export builds a learner's data export (F13): everything the learner produced, as JSON.
package export

import (
	"errors"
	"time"
)

// Version is the export format version.
const Version = 1

// Account is the part of the account that is exported; the password hash is never read.
type Account struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
}

// Doc is a stored document with JSON-friendly values (ids as hex strings, dates as RFC 3339).
type Doc = map[string]any

// Export is the file a learner downloads. Every list is present, empty when there is nothing.
type Export struct {
	Version          int            `json:"version"`
	ExportedAt       time.Time      `json:"exportedAt"`
	Account          Account        `json:"account"`
	Settings         map[string]any `json:"settings"`
	Cards            []Doc          `json:"cards"`
	ReviewLogs       []Doc          `json:"reviewLogs"`
	Goals            []Doc          `json:"goals"`
	LessonProgress   []Doc          `json:"lessonProgress"`
	StudyDays        []Doc          `json:"studyDays"`
	DictationResults []Doc          `json:"dictationResults"`
	ReadingAnswers   []Doc          `json:"readingAnswers"`
	Lessons          []Doc          `json:"lessons"`
}

// Collections a learner's documents are exported from. Nothing else (sessions, users) is ever
// read by collection name.
const (
	CollCards            = "cards"
	CollReviewLogs       = "review_logs"
	CollGoals            = "goals"
	CollLessonProgress   = "lesson_progress"
	CollStudyDays        = "study_days"
	CollDictationResults = "dictation_results"
	CollReadingAnswers   = "reading_answers"
)

// Allowed reports whether collection may be read by UserDocs.
func Allowed(collection string) bool {
	switch collection {
	case CollCards, CollReviewLogs, CollGoals, CollLessonProgress, CollStudyDays, CollDictationResults, CollReadingAnswers:
		return true
	}
	return false
}

var (
	// ErrNotFound is returned when the session's account no longer exists.
	ErrNotFound = errors.New("export: user not found")
	// ErrCollection is returned for a collection outside the allowed list.
	ErrCollection = errors.New("export: collection not allowed")
)
