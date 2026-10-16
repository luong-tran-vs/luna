// Package verbs keeps each learner's practice of the irregular verbs (the "Động từ" tab): which
// verbs they know and when each one is asked again. The verb list itself lives in the frontend;
// a verb is named by its base form ("go").
package verbs

import (
	"context"
	"fmt"
	"regexp"
	"time"
)

// Progress is one learner's record of one verb.
type Progress struct {
	UserID string
	Base   string
	// Level 0: not known yet (never right, or wrong the last time). Each right answer on a verb
	// that is due moves it up one box, so it is asked again later and later (Intervals).
	Level int
	// DueAt is when the verb is asked again; zero for a verb never answered right.
	DueAt time.Time
	// Right and Wrong count the answers given.
	Right     int
	Wrong     int
	UpdatedAt time.Time
}

// Known reports whether the verb counts as "đã thuộc".
func (p Progress) Known() bool { return p.Level > 0 }

// Result is one verb of a finished practice round: right means both V2 and V3 were right the first
// time they were asked in the round.
type Result struct {
	Base  string
	Right bool
}

// Repository stores the progress records.
type Repository interface {
	// List returns every record of the user.
	List(ctx context.Context, userID string) ([]Progress, error)
	// Save writes the records, each replacing the user's record of its verb.
	Save(ctx context.Context, records []Progress) error
}

// Intervals are the waits after a right answer: box 1 is asked again a day later, box 2 three
// days later, and so on; the last box keeps its wait.
var Intervals = []time.Duration{
	24 * time.Hour,
	3 * 24 * time.Hour,
	7 * 24 * time.Hour,
	14 * 24 * time.Hour,
	30 * 24 * time.Hour,
	60 * 24 * time.Hour,
}

// MaxResults bounds one request: a round is about ten verbs.
const MaxResults = 50

// basePattern is what a verb's base form looks like ("go", "lie", "dive").
var basePattern = regexp.MustCompile(`^[a-z]{1,30}$`)

// ValidationError lists invalid input fields with Vietnamese messages.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string { return fmt.Sprintf("verbs: invalid input %v", e.Fields) }

// Apply returns the record after one answer at now. A right answer moves a verb that is new or due
// up one box; one answered again before it is due (a second round the same day) keeps its box, so
// practising a lot in one day does not skip the waits. A wrong answer sends it back to box 0, asked
// again in the next round.
func Apply(p Progress, right bool, now time.Time) Progress {
	p.UpdatedAt = now
	if !right {
		p.Wrong++
		p.Level, p.DueAt = 0, time.Time{}
		return p
	}
	p.Right++
	if p.Level > 0 && now.Before(p.DueAt) {
		return p
	}
	p.Level = min(p.Level+1, len(Intervals))
	p.DueAt = now.Add(Intervals[p.Level-1])
	return p
}
