package progress

import (
	"context"
	"fmt"
)

// StatsView is the stats page (F6): notebook, dictation and lessons over every topic.
type StatsView struct {
	Cards     int
	Dictation DictationTotals
	// Rate is CorrectWords / TotalWords; nil when nothing was checked.
	Rate    *float64
	Lessons StepCounts
	// Reading counts the comprehension answers (F15); ReadingRate is nil when there are none.
	Reading     ReadingTotals
	ReadingRate *float64
}

// ReadingTotals counts a learner's comprehension answers.
type ReadingTotals struct {
	Answered int
	Correct  int
}

// Stats returns the learner's totals.
func (s *StudyService) Stats(ctx context.Context, userID string) (StatsView, error) {
	var v StatsView
	var err error
	if v.Cards, err = s.d.Reviews.CardCount(ctx, userID); err != nil {
		return StatsView{}, fmt.Errorf("progress: card count: %w", err)
	}
	if v.Dictation, err = s.d.Dictation.Totals(ctx, userID); err != nil {
		return StatsView{}, err
	}
	if v.Dictation.TotalWords > 0 {
		rate := float64(v.Dictation.CorrectWords) / float64(v.Dictation.TotalWords)
		v.Rate = &rate
	}
	if v.Lessons, err = s.d.Progress.StepCounts(ctx, userID, nil); err != nil {
		return StatsView{}, fmt.Errorf("progress: step counts: %w", err)
	}
	if s.d.Quiz != nil {
		if v.Reading.Answered, v.Reading.Correct, err = s.d.Quiz.Totals(ctx, userID); err != nil {
			return StatsView{}, fmt.Errorf("progress: reading totals: %w", err)
		}
	}
	if v.Reading.Answered > 0 {
		rate := float64(v.Reading.Correct) / float64(v.Reading.Answered)
		v.ReadingRate = &rate
	}
	return v, nil
}
