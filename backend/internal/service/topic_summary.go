package service

import (
	"context"
	"fmt"
	"slices"

	"github.com/luongtran/luna/backend/internal/lesson"
	"github.com/luongtran/luna/backend/internal/progress"
	"github.com/luongtran/luna/backend/internal/vocab"
	"github.com/luongtran/luna/backend/internal/writing"
)

// topicRecords adapts the notebook, the comprehension questions and the writings to
// progress.TopicRecords, for the topic summary (added 2026-10-23).
type topicRecords struct {
	vocab   *vocab.Service
	reader  *lesson.Reader
	writing *writing.Service
}

func (t topicRecords) Records(ctx context.Context, userID string, lessonIDs []string) (progress.Records, error) {
	var out progress.Records
	stats, err := t.vocab.LessonStats(ctx, userID)
	if err != nil {
		return out, fmt.Errorf("lesson stats: %w", err)
	}
	for _, st := range stats {
		if !slices.Contains(lessonIDs, st.LessonID) {
			continue
		}
		out.Words += st.Cards
		out.Remembered += st.Remembered
		if st.Result != nil {
			out.Review.Correct += st.Result.Correct
			out.Review.Total += st.Result.Total
		}
	}
	for _, id := range lessonIDs {
		answered, correct, err := t.reader.QuizScore(ctx, userID, id)
		if err != nil {
			return out, fmt.Errorf("quiz score: %w", err)
		}
		out.Quiz.Total += answered
		out.Quiz.Correct += correct
	}
	graded, avg, mistakes, err := t.writing.LessonResults(ctx, userID, lessonIDs)
	if err != nil {
		return out, err
	}
	out.Writings, out.WritingAverage = graded, avg
	for _, m := range mistakes {
		out.Mistakes = append(out.Mistakes, progress.MistakeCount{Type: m.Type, Count: m.Count})
	}
	return out, nil
}
