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
// time they were asked in the round. Quick marks a verb reviewed by flipping its card and grading
// oneself ("Ôn nhanh", added 2026-11-10) instead of typing: Right is then "Nhớ" or "Hơi nhớ"
// (Hard), not right is "Quên".
type Result struct {
	Base  string
	Right bool
	Quick bool
	Hard  bool
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

// QuickMaxLevel is the highest box a quick review ("Ôn nhanh") moves a verb to: its wait is
// QuickMaxWait. Waiting longer takes a typed review, so a learner grading too kindly cannot push a
// verb far away (added 2026-11-10).
const QuickMaxLevel = 4

// QuickMaxWait is the longest wait a quick review gives.
const QuickMaxWait = 14 * 24 * time.Hour

// HardWait is the wait after "Hơi nhớ" in a quick review, when the box's own wait is longer.
const HardWait = 2 * 24 * time.Hour

// ApplyResult returns the record after one answer of a round at now: typed answers as Apply,
// quick ones as ApplyQuick.
func ApplyResult(p Progress, r Result, now time.Time) Progress {
	if r.Quick {
		return ApplyQuick(p, r.Right, r.Hard, now)
	}
	return Apply(p, r.Right, now)
}

// ApplyQuick returns the record after a self-graded answer at now. "Quên" (not right) counts as a
// wrong answer. "Hơi nhớ" (hard) keeps the box and asks again within HardWait. "Nhớ" moves the
// verb up a box once it is due, but not past QuickMaxLevel, and never waits more than QuickMaxWait.
// A verb never typed right (Right == 0) does not move up on a quick answer: the frontend has it
// typed, and this guards against a page that does not.
func ApplyQuick(p Progress, right, hard bool, now time.Time) Progress {
	if !right {
		return Apply(p, false, now)
	}
	neverTyped := p.Right == 0
	p.UpdatedAt = now
	p.Right++
	if !p.Learned() || now.Before(p.DueAt) {
		return p
	}
	if !hard && !neverTyped {
		p.Level = max(p.Level, min(p.Level+1, QuickMaxLevel))
	}
	wait := min(Intervals[p.Level-1], QuickMaxWait)
	if hard || neverTyped {
		wait = min(Intervals[p.Level-1], HardWait)
	}
	p.DueAt = now.Add(wait)
	return p
}
