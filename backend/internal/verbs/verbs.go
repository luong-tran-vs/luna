// Package verbs keeps each learner's irregular verbs (the "Động từ" tab): which verbs they have
// confirmed as learned and when each learned verb is reviewed again. The verb list and its lessons
// live in the frontend; a verb is named by its base form ("go").
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
	// LearnedAt is when the learner confirmed the verb as learned ("Đã học"); zero for a verb not
	// learned. Only learned verbs are reviewed.
	LearnedAt time.Time
	// Level is the review box of a learned verb, from 1: each right review once it is due moves it
	// up, so it is asked again later and later (Intervals). 0 for a verb not learned.
	Level int
	// DueAt is when a learned verb is reviewed again; zero for a verb not learned.
	DueAt time.Time
	// Right and Wrong count the review answers given.
	Right     int
	Wrong     int
	UpdatedAt time.Time
}

// Learned reports whether the learner confirmed the verb as learned.
func (p Progress) Learned() bool { return !p.LearnedAt.IsZero() }

// Result is one verb of a finished review round: right means both V2 and V3 were right the first
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

// Intervals are the waits before the next review: box 1 a day, box 2 three days, and so on; the
// last box keeps its wait.
var Intervals = []time.Duration{
	24 * time.Hour,
	3 * 24 * time.Hour,
	7 * 24 * time.Hour,
	14 * 24 * time.Hour,
	30 * 24 * time.Hour,
	60 * 24 * time.Hour,
}

// MaxResults bounds one request: a round is about ten verbs, a lesson six.
const MaxResults = 50

// basePattern is what a verb's base form looks like ("go", "lie", "dive").
var basePattern = regexp.MustCompile(`^[a-z]{1,30}$`)

// ValidationError lists invalid input fields with Vietnamese messages.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string { return fmt.Sprintf("verbs: invalid input %v", e.Fields) }

// Learn returns the record marked as learned at now: box 1, first review a day later. A verb
// already learned is left as it is.
func Learn(p Progress, now time.Time) Progress {
	if p.Learned() {
		return p
	}
	p.LearnedAt, p.Level, p.DueAt, p.UpdatedAt = now, 1, now.Add(Intervals[0]), now
	return p
}

// Unlearn returns the record no longer learned: it leaves the reviews. The answer counts stay.
func Unlearn(p Progress, now time.Time) Progress {
	p.LearnedAt, p.Level, p.DueAt, p.UpdatedAt = time.Time{}, 0, time.Time{}, now
	return p
}

// Apply returns the record after one review answer at now. A verb not learned only counts the
// answer. For a learned verb, a right answer once it is due moves it up a box; a right answer
// before it is due (a second round the same day) keeps its box, so reviewing a lot in one day does
// not skip the waits. A wrong answer keeps it learned but back in box 1, asked again a day later.
func Apply(p Progress, right bool, now time.Time) Progress {
	p.UpdatedAt = now
	if right {
		p.Right++
	} else {
		p.Wrong++
	}
	if !p.Learned() {
		return p
	}
	switch {
	case !right:
		p.Level = 1
	case now.Before(p.DueAt):
		return p
	default:
		p.Level = min(p.Level+1, len(Intervals))
	}
	p.DueAt = now.Add(Intervals[p.Level-1])
	return p
}
