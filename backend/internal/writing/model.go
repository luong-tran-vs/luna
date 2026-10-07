// Package writing runs the Write step (F8): drafts, submission, AI grading in the background
// and the learner's list of writings. It meets lessons, the daily flow and the job queue only
// through ports wired in main.
package writing

import (
	"errors"
	"fmt"
	"math"
	"time"
)

// Status is where a writing is in its life.
type Status string

const (
	StatusDraft     Status = "draft"
	StatusSubmitted Status = "submitted"
)

// GradeStatus is the state of the AI grading of a submitted writing.
type GradeStatus string

const (
	GradePending GradeStatus = "pending"
	GradeDone    GradeStatus = "done"
	GradeFailed  GradeStatus = "failed"
)

// Criteria names, in display order.
const (
	CriterionTask       = "task"
	CriterionGrammar    = "grammar"
	CriterionVocabulary = "vocabulary"
	CriterionCoherence  = "coherence"
)

// Limits of a writing.
const (
	MinWords = 5
	MaxWords = 400
	// maxChars bounds a draft so autosave cannot store huge texts.
	maxChars = 4000
)

// DefaultPrompt is used when the lesson has no writing prompt (F15).
const DefaultPrompt = "Tóm tắt bài bằng 3–5 câu."

// Criterion is one graded criterion.
type Criterion struct {
	Name      string
	Score     int
	CommentVi string
}

// Grade is the result of grading a submitted writing.
type Grade struct {
	Status        GradeStatus
	Error         string
	Criteria      []Criterion
	OverallVi     string
	CorrectedText string
	GradedAt      time.Time
	// Seen is false while a new result (done or failed) has not been opened.
	Seen bool
}

// Writing is a learner's writing for one lesson; there is at most one per learner and lesson.
type Writing struct {
	ID             string
	UserID         string
	LessonID       string
	LessonRevision int
	LessonTitle    string
	// Prompt is stored at submission time; empty for a draft.
	Prompt      string
	Text        string
	Status      Status
	Grade       *Grade
	CreatedAt   time.Time
	UpdatedAt   time.Time
	SubmittedAt time.Time
}

// LessonView is the Write step of a lesson for one learner.
type LessonView struct {
	Prompt string
	Level  string
	// CanWrite is true for the lesson being studied while its Write step is not done.
	CanWrite bool
	Writing  *Writing
}

// Summary is a submitted writing in the learner's list.
type Summary struct {
	ID          string
	LessonID    string
	LessonTitle string
	SubmittedAt time.Time
	GradeStatus GradeStatus
	Average     *float64
	Seen        bool
}

// UnseenCount tells the app whether to show a new-result notice.
type UnseenCount struct {
	Unseen  int
	Pending int
	// Latest is the most recent unseen result, nil when there is none.
	Latest *Latest
}

// Latest names a writing with a new result.
type Latest struct {
	ID     string
	Status GradeStatus
}

// LessonInfo is what writing needs from a lesson.
type LessonInfo struct {
	Title         string
	Level         string
	Content       string
	WritingPrompt string
	Revision      int
}

var (
	// ErrNotFound means no such writing or lesson for this learner (also another learner's).
	ErrNotFound = errors.New("writing: not found")
	// ErrSubmitted means the writing was already submitted and cannot change.
	ErrSubmitted = errors.New("writing: already submitted")
	// ErrLocked means the lesson is not today's lesson at the Write step.
	ErrLocked = errors.New("writing: not at the write step")
	// ErrNotFailed means only a failed grading can be run again.
	ErrNotFailed = errors.New("writing: grade not failed")
	// ErrUnusableGrade means the AI answer missed scores or comments.
	ErrUnusableGrade = errors.New("writing: AI returned an unusable grade")
)

// ValidationError lists invalid fields with Vietnamese messages.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string { return fmt.Sprintf("writing: invalid input %v", e.Fields) }

// Average is the mean of the criteria scores rounded to one decimal, nil until graded.
func Average(g *Grade) *float64 {
	if g == nil || g.Status != GradeDone || len(g.Criteria) == 0 {
		return nil
	}
	sum := 0
	for _, c := range g.Criteria {
		sum += c.Score
	}
	avg := math.Round(float64(sum)/float64(len(g.Criteria))*10) / 10
	return &avg
}
