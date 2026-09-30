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
	return v, nil
}
