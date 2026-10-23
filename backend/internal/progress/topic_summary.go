package progress

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"
)

// The topic summary (added 2026-10-23): what the learner did in the lessons of one topic's roadmap,
// with a rating worked out from the scores, shown once the topic is finished (and at any time from
// the learning path). It needs no AI: everything comes from the results already kept per lesson.

// TopicRecords gives what the learner did in a set of lessons, beyond the study flow's own data
// (implemented over vocab, lesson and writing in main).
type TopicRecords interface {
	Records(ctx context.Context, userID string, lessonIDs []string) (Records, error)
}

// Score is a count of correct answers out of a total.
type Score struct {
	Correct int
	Total   int
}

// Percent is Correct out of Total, from 0 to 100; 0 when Total is 0.
func (s Score) Percent() int {
	if s.Total == 0 {
		return 0
	}
	return (s.Correct*100 + s.Total/2) / s.Total
}

// MistakeCount is how many corrections of one type the learner's graded writings had.
type MistakeCount struct {
	Type  string
	Count int
}

// Records are the learner's results over some lessons.
type Records struct {
	// Words counts the cards saved from the lessons; Remembered those in the review stage.
	Words      int
	Remembered int
	// Quiz sums the answers to the comprehension questions of the current question sets.
	Quiz Score
	// Review sums the latest lesson review round of each lesson (the review tab).
	Review Score
	// Writings counts the graded writings; WritingAverage is their average score, 1 to 5.
	Writings       int
	WritingAverage float64
	// Mistakes are the corrections of the graded writings by type, the most frequent first.
	Mistakes []MistakeCount
}

// Rating is the overall assessment of a topic.
type Rating string

const (
	RatingExcellent Rating = "excellent"
	RatingGood      Rating = "good"
	RatingFair      Rating = "fair"
	RatingPractice  Rating = "practice"
	// RatingNone means no score was recorded (no question answered, nothing written down).
	RatingNone Rating = "none"
)

// Skill names of the scores that make the rating.
const (
	SkillListening = "listening"
	SkillReading   = "reading"
	SkillWriting   = "writing"
	SkillReview    = "review"
)

// SkillScore is one score of the summary, as a percentage.
type SkillScore struct {
	Skill   string
	Percent int
	// Correct out of Total for listening (words), reading and review (answers); for writing,
	// Correct is the number of graded writings and Total is 0.
	Correct int
	Total   int
}

// SummaryLesson is one lesson of the topic.
type SummaryLesson struct {
	ID    string
	Title string
	Done  bool
}

// TopicSummary is the summary of one topic's roadmap at a level.
type TopicSummary struct {
	TopicID   string
	TopicName string
	Level     string
	// CanDoVi and GrammarVi come from the level's course; empty when the topic is not in it.
	CanDoVi   string
	GrammarVi []string
	Lessons   []SummaryLesson
	Completed int
	// StartedAt and FinishedAt span the lessons done; zero when none is done.
	StartedAt  time.Time
	FinishedAt time.Time
	// Days counts the days lessons of the topic were studied on.
	Days       int
	Words      int
	Remembered int
	// Skills are the scores recorded, in a fixed order; a skill with nothing recorded is left out.
	Skills         []SkillScore
	WritingAverage float64
	Mistakes       []MistakeCount
	// Overall is the average of the skills' percentages; Rating follows from it.
	Overall int
	Rating  Rating
}

// rate turns the average percentage into a rating.
func rate(overall int) Rating {
	switch {
	case overall >= 90:
		return RatingExcellent
	case overall >= 75:
		return RatingGood
	case overall >= 60:
		return RatingFair
	default:
		return RatingPractice
	}
}

// writingPercent maps a writing score out of 5 onto a percentage: 4/5 is 80.
func writingPercent(avg float64) int {
	return int(avg*20 + 0.5)
}

// TopicSummary sums up the learner's work on the roadmap of topicID at level.
func (s *StudyService) TopicSummary(ctx context.Context, userID, topicID, level string) (TopicSummary, error) {
	if !slices.Contains(goalLevels, level) {
		return TopicSummary{}, &ValidationError{Fields: map[string]string{"level": "Trình độ không hợp lệ"}}
	}
	info, err := s.d.Roadmaps.Roadmap(ctx, topicID, level)
	if err != nil {
		return TopicSummary{}, err
	}
	sum := TopicSummary{TopicID: info.ID, TopicName: info.Name, Level: level, Lessons: []SummaryLesson{}}
	if s.d.Course != nil {
		if c, ok := s.d.Course(ctx, level, info.ID, info.Name); ok {
			sum.CanDoVi, sum.GrammarVi = c.CanDoVi, c.GrammarVi
		}
	}

	completed, err := s.d.Progress.Completed(ctx, userID)
	if err != nil {
		return TopicSummary{}, fmt.Errorf("progress: completed lessons: %w", err)
	}
	inTopic := map[string]bool{}
	for _, id := range info.LessonIDs {
		inTopic[id] = true
	}
	done := map[string]bool{}
	days := map[string]bool{}
	for _, p := range completed {
		if !inTopic[p.LessonID] {
			continue
		}
		done[p.LessonID] = true
		days[p.DayKey] = true
		if sum.StartedAt.IsZero() || (!p.StartedAt.IsZero() && p.StartedAt.Before(sum.StartedAt)) {
			sum.StartedAt = p.StartedAt
		}
		if p.CompletedAt.After(sum.FinishedAt) {
			sum.FinishedAt = p.CompletedAt
		}
	}
	sum.Completed, sum.Days = len(done), len(days)

	titles, err := s.d.Titles.Titles(ctx, info.LessonIDs)
	if err != nil {
		return TopicSummary{}, fmt.Errorf("progress: lesson titles: %w", err)
	}
	for _, id := range info.LessonIDs {
		sum.Lessons = append(sum.Lessons, SummaryLesson{ID: id, Title: titles[id], Done: done[id]})
	}

	// Only the lessons done count: a lesson not studied yet has no results to judge.
	var doneIDs []string
	for _, id := range info.LessonIDs {
		if done[id] {
			doneIDs = append(doneIDs, id)
		}
	}
	var listening Score
	if s.d.Dictation != nil {
		for _, id := range doneIDs {
			d, err := s.d.Dictation.Summary(ctx, userID, id)
			if errors.Is(err, ErrLessonNotFound) {
				continue
			}
			if err != nil {
				return TopicSummary{}, err
			}
			listening.Correct += d.CorrectWords
			listening.Total += d.TotalWords
		}
	}
	var rec Records
	if s.d.Records != nil && len(doneIDs) > 0 {
		if rec, err = s.d.Records.Records(ctx, userID, doneIDs); err != nil {
			return TopicSummary{}, fmt.Errorf("progress: topic records: %w", err)
		}
	}
	sum.Words, sum.Remembered, sum.Mistakes = rec.Words, rec.Remembered, rec.Mistakes
	if sum.Mistakes == nil {
		sum.Mistakes = []MistakeCount{}
	}

	add := func(skill string, sc Score) {
		if sc.Total > 0 {
			sum.Skills = append(sum.Skills, SkillScore{Skill: skill, Percent: sc.Percent(), Correct: sc.Correct, Total: sc.Total})
		}
	}
	add(SkillListening, listening)
	add(SkillReading, rec.Quiz)
	if rec.Writings > 0 {
		sum.WritingAverage = rec.WritingAverage
		sum.Skills = append(sum.Skills, SkillScore{Skill: SkillWriting, Percent: writingPercent(rec.WritingAverage), Correct: rec.Writings})
	}
	add(SkillReview, rec.Review)

	sum.Rating = RatingNone
	if len(sum.Skills) > 0 {
		total := 0
		for _, sk := range sum.Skills {
			total += sk.Percent
		}
		sum.Overall = (total + len(sum.Skills)/2) / len(sum.Skills)
		sum.Rating = rate(sum.Overall)
	}
	if sum.Skills == nil {
		sum.Skills = []SkillScore{}
	}
	return sum, nil
}
