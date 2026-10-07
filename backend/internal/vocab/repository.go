package vocab

import (
	"context"
	"time"
)

// Repository stores cards. Implementations live in internal/storage. Every method is scoped to
// userID: another user's card is ErrNotFound.
type Repository interface {
	// Create stores c; ErrExists when the user already has a card with c.Lemma.
	Create(ctx context.Context, c Card) (Card, error)
	// FindByLemma returns ErrNotFound when the user has no card with lemma.
	FindByLemma(ctx context.Context, userID, lemma string) (Card, error)
	// Words lists the user's saved lemmas and texts.
	Words(ctx context.Context, userID string) ([]WordRef, error)
	// Get returns ErrNotFound when the user has no card with id.
	Get(ctx context.Context, userID, id string) (Card, error)
	// List returns up to limit cards after skip, newest first, and whether more follow.
	// q.Q matches text or lemma case-insensitively as a plain substring.
	List(ctx context.Context, userID string, q ListQuery, limit, skip int) ([]Card, bool, error)
	// LessonCounts counts cards per lesson id; "" counts cards added by hand.
	LessonCounts(ctx context.Context, userID string) (map[string]int, error)
	// UpdateDetails sets the non-nil fields of d and returns the card; the schedule is untouched.
	UpdateDetails(ctx context.Context, userID, id string, d Details) (Card, error)
	// UpdateSchedule stores s if the card still has expectedReps reviews (missing = 0);
	// ok is false when it does not.
	UpdateSchedule(ctx context.Context, userID, id string, expectedReps uint64, s Schedule) (ok bool, err error)
	// Due returns up to limit cards due at now, most overdue first, and how many are due.
	// Cards without a schedule are due when saved before startOfToday.
	Due(ctx context.Context, userID string, now, startOfToday time.Time, limit int) ([]Card, int, error)
	// NextDue is the earliest due time after now (cards without a schedule saved since
	// startOfToday are due the next day); ok is false when there is none.
	NextDue(ctx context.Context, userID string, now, startOfToday time.Time) (t time.Time, ok bool, err error)
	// CountDue counts cards due before `before`; cards without a schedule count when saved
	// before createdBefore.
	CountDue(ctx context.Context, userID string, before, createdBefore time.Time) (int, error)
	// Count is the number of the user's cards created at or after since (nil = every card).
	Count(ctx context.Context, userID string, since *time.Time) (int, error)
	// Delete removes the card; ok is false when it did not exist.
	Delete(ctx context.Context, userID, id string) (ok bool, err error)
}

// ReviewLogRepository stores review history.
type ReviewLogRepository interface {
	Add(ctx context.Context, l ReviewLog) error
	// DeleteByCard removes the logs of one card and returns how many there were.
	DeleteByCard(ctx context.Context, userID, cardID string) (int, error)
	// CountSince counts the user's reviews in context c made at or after since.
	CountSince(ctx context.Context, userID string, c Context, since time.Time) (int, error)
}

// Timezones gives a learner's timezone (implemented over auth in main).
type Timezones interface {
	Location(ctx context.Context, userID string) (*time.Location, error)
}

// LessonVocabulary gives the annotated words of a lesson (implemented over lesson.Reader);
// ErrLessonNotFound when the lesson does not exist.
type LessonVocabulary interface {
	Vocabulary(ctx context.Context, lessonID string) ([]VocabItem, error)
}

// LessonTitles gives the titles of the lessons that still exist among ids.
type LessonTitles interface {
	Titles(ctx context.Context, ids []string) (map[string]string, error)
}
