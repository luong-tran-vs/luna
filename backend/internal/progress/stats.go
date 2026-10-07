package progress

import (
	"context"
	"fmt"
	"time"
)

// StatsView is the stats page (F6): notebook, dictation and lessons over every topic, in a period.
type StatsView struct {
	Period Period
	Cards  int
	// Dictation.Lessons counts the lessons with a dictation result in the period.
	Dictation DictationTotals
	// Rate is CorrectWords / TotalWords; nil when nothing was checked.
	Rate    *float64
	Lessons StepCounts
	// Reading counts the comprehension answers (F15); ReadingRate is nil when there are none.
	Reading     ReadingTotals
	ReadingRate *float64
	// Writing counts submitted writings (F8); AverageScore is nil until one is graded.
	Writing WritingTotals
	// GrammarLessons counts the grammar points mastered in the period (F20).
	GrammarLessons int
}

// ReadingTotals counts a learner's comprehension answers.
type ReadingTotals struct {
	Answered int
	Correct  int
}

// Stats returns the learner's figures for the calendar period (in the learner's timezone) that
// holds now; an empty period means PeriodAll.
func (s *StudyService) Stats(ctx context.Context, userID string, period Period) (StatsView, error) {
	if period == "" {
		period = PeriodAll
	}
	if !ValidPeriod(period) {
		return StatsView{}, &ValidationError{Fields: map[string]string{"period": "Khoảng thời gian không hợp lệ"}}
	}
	var since *time.Time
	if period != PeriodAll {
		loc, err := s.d.Timezones.Location(ctx, userID)
		if err != nil {
			return StatsView{}, fmt.Errorf("progress: timezone: %w", err)
		}
		since = PeriodStart(period, s.d.Now(), loc)
	}
	v := StatsView{Period: period}
	var err error
	if v.Cards, err = s.d.Reviews.CardCount(ctx, userID, since); err != nil {
		return StatsView{}, fmt.Errorf("progress: card count: %w", err)
	}
	if v.Dictation, err = s.d.Dictation.Totals(ctx, userID, since); err != nil {
		return StatsView{}, err
	}
	if v.Dictation.TotalWords > 0 {
		rate := float64(v.Dictation.CorrectWords) / float64(v.Dictation.TotalWords)
		v.Rate = &rate
	}
	if v.Lessons, err = s.d.Progress.StepCounts(ctx, userID, nil, since); err != nil {
		return StatsView{}, fmt.Errorf("progress: step counts: %w", err)
	}
	if s.d.Quiz != nil {
		if v.Reading.Answered, v.Reading.Correct, err = s.d.Quiz.Totals(ctx, userID, since); err != nil {
			return StatsView{}, fmt.Errorf("progress: reading totals: %w", err)
		}
	}
	if v.Reading.Answered > 0 {
		rate := float64(v.Reading.Correct) / float64(v.Reading.Answered)
		v.ReadingRate = &rate
	}
	if s.d.Writings != nil {
		if v.Writing.Submitted, v.Writing.AverageScore, err = s.d.Writings.Stats(ctx, userID, since); err != nil {
			return StatsView{}, fmt.Errorf("progress: writing stats: %w", err)
		}
	}
	if s.d.Grammar != nil {
		if v.GrammarLessons, err = s.d.Grammar.MasteredCount(ctx, userID, since); err != nil {
			return StatsView{}, fmt.Errorf("progress: grammar stats: %w", err)
		}
	}
	return v, nil
}

// WritingTotals counts a learner's writings (F8).
type WritingTotals struct {
	Submitted    int
	AverageScore *float64
}
