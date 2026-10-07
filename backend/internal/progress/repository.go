package progress

import (
	"context"
	"time"
)

// DictationRepository stores dictation results. Implementations live in internal/storage.
type DictationRepository interface {
	// Upsert replaces the user's result for r.SentenceIndex of the lesson.
	Upsert(ctx context.Context, userID, lessonID string, revision int, r Result) error
	// List returns all of the user's results for the lesson, of any revision, in any order.
	List(ctx context.Context, userID, lessonID string) ([]StoredResult, error)
	// Delete removes all of the user's results for the lesson, of any revision.
	Delete(ctx context.Context, userID, lessonID string) error
	// Totals sums the user's results over every lesson (one result per sentence), counting only
	// the results checked at or after since (nil = every result).
	Totals(ctx context.Context, userID string, since *time.Time) (DictationTotals, error)
}

// Lessons reads what progress needs from a lesson (implemented over lesson.Repository in main).
type Lessons interface {
	// Info returns ErrLessonNotFound when the lesson does not exist.
	Info(ctx context.Context, lessonID string) (revision, sentenceCount int, err error)
}
