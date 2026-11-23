package toeic

import (
	"context"
	"time"
)

// GroupFilter narrows the admin list of groups; zero values match everything.
type GroupFilter struct {
	Part   Part
	Status Status
}

// GroupRepository stores the question groups, keyed by Key.
type GroupRepository interface {
	// List returns the groups matching f, by part then key.
	List(ctx context.Context, f GroupFilter) ([]Group, error)
	// ByKeys returns the groups of the keys that exist, in no particular order.
	ByKeys(ctx context.Context, keys []string) ([]Group, error)
	// Published returns the published groups of a part (every part when part is 0).
	Published(ctx context.Context, part Part) ([]Group, error)
	// Upsert stores the groups: a new key is created as given; a known key gets the new content
	// and level, keeps its status and creation time, and its version goes up by one.
	Upsert(ctx context.Context, gs []Group) (created, replaced []string, err error)
	// SetStatus changes the status of a group; ErrNotFound for an unknown key.
	SetStatus(ctx context.Context, key string, s Status) error
	// Delete removes the groups of the keys; unknown keys are ignored.
	Delete(ctx context.Context, keys []string) error
}

// ImageRepository stores the pictures of the groups, one per (group, slot).
type ImageRepository interface {
	// Get returns ErrNotFound when there is no picture.
	Get(ctx context.Context, groupKey, slot string) (Image, error)
	// Save stores a picture, replacing the previous one of the slot.
	Save(ctx context.Context, img Image) error
	// Delete removes a picture; a missing one is not an error.
	Delete(ctx context.Context, groupKey, slot string) error
	// Has returns the slots that have a picture, by group key, for the given groups.
	Has(ctx context.Context, keys []string) (map[string][]string, error)
}

// TestRepository stores the fixed tests, keyed by Key.
type TestRepository interface {
	// List returns the tests by order then key; only the published ones when onlyPublished.
	List(ctx context.Context, onlyPublished bool) ([]Test, error)
	// Get returns ErrNotFound for an unknown key.
	Get(ctx context.Context, key string) (Test, error)
	// Save creates or replaces the test of t.Key (keeping its creation time).
	Save(ctx context.Context, t Test) error
	// SetStatus changes the status of a test; ErrNotFound for an unknown key.
	SetStatus(ctx context.Context, key string, s Status) error
	// Delete removes a test; ErrNotFound for an unknown key.
	Delete(ctx context.Context, key string) error
}

// AttemptFilter narrows the attempts of a learner; zero values match everything.
type AttemptFilter struct {
	TestKey string
	Status  AttemptStatus
	Counted bool
	// Limit keeps the newest ones; 0 means all.
	Limit int
}

// AttemptRepository stores the learners' attempts.
type AttemptRepository interface {
	// Create stores a new attempt and returns it with its id.
	Create(ctx context.Context, a Attempt) (Attempt, error)
	// Get returns ErrNotFound when the attempt does not exist or is not the user's.
	Get(ctx context.Context, userID, id string) (Attempt, error)
	// Save stores the progress of an attempt in progress.
	Save(ctx context.Context, a Attempt) error
	// Submit stores the result and the submitted status.
	Submit(ctx context.Context, a Attempt) error
	// List returns the user's attempts matching f, newest first.
	List(ctx context.Context, userID string, f AttemptFilter) ([]Attempt, error)
	// InProgress returns the user's attempt in progress of a test; ErrNotFound when none.
	InProgress(ctx context.Context, userID, testKey string) (Attempt, error)
	// AnsweredQuestions returns the questions of the user's submitted attempts.
	AnsweredQuestions(ctx context.Context, userID string) (map[string]bool, error)
	// WithGroup returns every submitted attempt (of any learner) that has the group.
	WithGroup(ctx context.Context, groupKey string) ([]Attempt, error)
	// CountWithTest returns how many attempts (of any learner, in any status) are of the test.
	CountWithTest(ctx context.Context, testKey string) (int, error)
	// UpdateResult replaces the result of a submitted attempt.
	UpdateResult(ctx context.Context, id string, r Result) error
}

// MistakeRepository stores the learners' mistakes notebooks, one entry per (user, question).
type MistakeRepository interface {
	// List returns the user's entries of a part (every part when part is 0), oldest wrong first.
	List(ctx context.Context, userID string, part Part) ([]Mistake, error)
	// Get returns ErrNotFound when the question is not in the notebook.
	Get(ctx context.Context, userID, questionID string) (Mistake, error)
	// Upsert stores an entry.
	Upsert(ctx context.Context, m Mistake) error
	// Delete removes an entry; a missing one is not an error.
	Delete(ctx context.Context, userID, questionID string) error
	// Count returns how many entries the user has.
	Count(ctx context.Context, userID string) (int, error)
}

// GoalRepository stores the learners' goals, one per user.
type GoalRepository interface {
	// Get returns ErrNotFound when the user has no goal.
	Get(ctx context.Context, userID string) (Goal, error)
	Save(ctx context.Context, g Goal) error
	// Delete removes the goal; a missing one is not an error.
	Delete(ctx context.Context, userID string) error
}

// StudyDays counts today as a study day for the streak.
type StudyDays interface {
	MarkStudied(ctx context.Context, userID string) error
}

// CardInput is a word added to the learner's notebook from an answer.
type CardInput struct {
	Word      string
	IPA       string
	MeaningVi string
	Sentence  string
}

// Cards adds words to the learner's vocabulary notebook (vocab.Service in internal/service); a word
// already there is not an error.
type Cards interface {
	SaveWord(ctx context.Context, userID string, w CardInput) error
}

// Timezones gives a learner's timezone.
type Timezones interface {
	Location(ctx context.Context, userID string) (*time.Location, error)
}
