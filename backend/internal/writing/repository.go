package writing

import (
	"context"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/job"
)

// Repository stores writings. Implementations live in internal/storage.
type Repository interface {
	// GetByLesson returns the learner's writing for a lesson; ok is false when there is none.
	GetByLesson(ctx context.Context, userID, lessonID string) (w Writing, ok bool, err error)
	// Get returns ErrNotFound when the writing does not exist (or the id is malformed).
	Get(ctx context.Context, id string) (Writing, error)
	// SaveDraft creates or updates the draft text; ErrSubmitted once submitted.
	SaveDraft(ctx context.Context, userID, lessonID, text string, now time.Time) (Writing, error)
	// Submit turns the learner's writing for w.LessonID into a submitted one with w's prompt,
	// text, lesson title and revision, submittedAt and a pending, seen grade. It creates the
	// writing when there was no draft; ErrSubmitted when it was already submitted.
	Submit(ctx context.Context, w Writing) (Writing, error)
	// SetGrade replaces the grade of a submitted writing.
	SetGrade(ctx context.Context, id string, g Grade) error
	// MarkSeen marks the writing's result as seen.
	MarkSeen(ctx context.Context, id string) error
	// List returns the learner's submitted writings, newest first.
	List(ctx context.Context, userID string) ([]Writing, error)
	// Unseen counts the learner's new results and gradings in progress.
	Unseen(ctx context.Context, userID string) (UnseenCount, error)
	// Stats counts the writings submitted at or after since (nil = all) and averages the graded
	// ones among them (nil when none).
	Stats(ctx context.Context, userID string, since *time.Time) (submitted int, average *float64, err error)
}

// Lessons gives lesson details (implemented over lesson.Repository in main).
type Lessons interface {
	// Info returns ErrNotFound when the lesson does not exist.
	Info(ctx context.Context, lessonID string) (LessonInfo, error)
}

// Steps tells whether the learner is at the Write step of this lesson today (implemented over
// progress.StudyService in main).
type Steps interface {
	CanWrite(ctx context.Context, userID, lessonID string) (bool, error)
}

// Jobs queues background work.
type Jobs interface {
	Enqueue(ctx context.Context, j job.Job) error
}

// Grader grades writings (ai.Provider).
type Grader interface {
	GradeWriting(ctx context.Context, req ai.GradeRequest) (ai.Grade, error)
}
