package vocab

import (
	"time"

	fsrs "github.com/open-spaced-repetition/go-fsrs/v3"
)

// newScheduler returns an FSRS scheduler with the default parameters; fuzz is off, so schedules
// are deterministic. A new one per call: go-fsrs writes a seed into its parameters on every
// Repeat/Next, so a shared scheduler is not safe for concurrent requests.
func newScheduler() *fsrs.FSRS {
	return fsrs.NewFSRS(fsrs.DefaultParam())
}

// FirstDue is when a card saved at t is first due: the start of the next day in loc.
func FirstDue(t time.Time, loc *time.Location) time.Time {
	y, m, d := t.In(loc).Date()
	return time.Date(y, m, d+1, 0, 0, 0, 0, loc)
}

// startOfDay is midnight of t's day in loc.
func startOfDay(t time.Time, loc *time.Location) time.Time {
	y, m, d := t.In(loc).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, loc)
}

// dayOf formats t's day in loc as YYYY-MM-DD.
func dayOf(t time.Time, loc *time.Location) string {
	return t.In(loc).Format(time.DateOnly)
}

// effective is the schedule to use for c: cards saved before F5 have none and are new cards
// due the day after they were saved.
func effective(c Card, loc *time.Location) Schedule {
	if c.Schedule.Due.IsZero() {
		return Schedule{Due: FirstDue(c.CreatedAt, loc), State: StateNew}
	}
	return c.Schedule
}

func toFSRS(s Schedule) fsrs.Card {
	return fsrs.Card{
		Due: s.Due, Stability: s.Stability, Difficulty: s.Difficulty, ElapsedDays: s.ElapsedDays,
		ScheduledDays: s.ScheduledDays, Reps: s.Reps, Lapses: s.Lapses,
		State:      fsrs.State(s.State), //nolint:gosec // State is 0..3
		LastReview: s.LastReview,
	}
}

func fromFSRS(c fsrs.Card) Schedule {
	return Schedule{
		Due: c.Due, Stability: c.Stability, Difficulty: c.Difficulty, ElapsedDays: c.ElapsedDays,
		ScheduledDays: c.ScheduledDays, Reps: c.Reps, Lapses: c.Lapses, State: State(c.State),
		LastReview: c.LastReview,
	}
}

// intervalsFor is the time until the next review for each rating given at now.
func intervalsFor(s Schedule, now time.Time) Intervals {
	log := newScheduler().Repeat(toFSRS(s), now)
	return Intervals{
		Again: log[fsrs.Again].Card.Due.Sub(now),
		Hard:  log[fsrs.Hard].Card.Due.Sub(now),
		Good:  log[fsrs.Good].Card.Due.Sub(now),
		Easy:  log[fsrs.Easy].Card.Due.Sub(now),
	}
}

// next is the schedule after rating r at now.
func next(s Schedule, now time.Time, r Rating) Schedule {
	return fromFSRS(newScheduler().Next(toFSRS(s), now, fsrs.Rating(r)).Card) //nolint:gosec // Rating is validated 1..4
}
