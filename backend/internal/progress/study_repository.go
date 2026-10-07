package progress

import (
	"context"
	"time"
)

// GoalRepository stores learners' goals. At most one goal per user is active.
type GoalRepository interface {
	List(ctx context.Context, userID string) ([]Goal, error)
	// Activate makes the goal of topicID active (creating it if needed) and pauses the others.
	Activate(ctx context.Context, userID, topicID, level, effectiveFrom string, now time.Time) (Goal, error)
}

// ProgressRepository stores lesson progress, one per (user, lesson).
type ProgressRepository interface {
	// Get returns ok=false when the learner never started the lesson.
	Get(ctx context.Context, userID, lessonID string) (LessonProgress, bool, error)
	// Completed returns the completed lessons, newest first.
	Completed(ctx context.Context, userID string) ([]LessonProgress, error)
	Upsert(ctx context.Context, p LessonProgress) error
	SetPosition(ctx context.Context, userID, lessonID string, step Step, sentence int) error
	// StepCounts counts the learner's lessons among lessonIDs (nil = every lesson) by steps done.
	// With since set, a step counts only when done at or after since (by DoneAt, so steps done
	// before DoneAt was recorded never count) and a lesson only when completed at or after since.
	StepCounts(ctx context.Context, userID string, lessonIDs []string, since *time.Time) (StepCounts, error)
}

// DayRepository stores the days with a completed lesson, for the streak.
type DayRepository interface {
	// MarkCompleted records that the learner completed a lesson on dayKey.
	MarkCompleted(ctx context.Context, userID, dayKey string) error
	// CompletedKeys lists the days with a completed lesson.
	CompletedKeys(ctx context.Context, userID string) ([]string, error)
}

// Roadmaps reads topics and their roadmaps (implemented over topic.Service in main).
type Roadmaps interface {
	// Roadmap returns ErrTopicNotFound when the topic does not exist.
	Roadmap(ctx context.Context, topicID string) (TopicInfo, error)
}

// ReadingQuiz is what the Reading step needs from the comprehension questions (F15, implemented
// over lesson.Reader in main).
type ReadingQuiz interface {
	// Status returns how many questions the lesson has and how many the learner answered in the
	// current question set; 0 questions means the lesson has no quiz.
	Status(ctx context.Context, userID, lessonID string) (questions, answered int, err error)
	// Totals counts the learner's answers and the correct ones, answered at or after since (nil = all).
	Totals(ctx context.Context, userID string, since *time.Time) (answered, correct int, err error)
}

// Writings is what the Write step and the stats need from the writings (F8, implemented over
// writing.Service in main).
type Writings interface {
	// Submitted reports whether the learner has submitted the writing of the lesson.
	Submitted(ctx context.Context, userID, lessonID string) (bool, error)
	// Stats counts the writings submitted at or after since (nil = all) and averages the graded
	// ones among them (nil when none is graded).
	Stats(ctx context.Context, userID string, since *time.Time) (submitted int, average *float64, err error)
}

// LessonTitles gives the titles of the lessons that still exist among ids.
type LessonTitles interface {
	Titles(ctx context.Context, ids []string) (map[string]string, error)
}

// Reviews is what the home page and the stats need from the notebook (implemented over vocab in
// main).
type Reviews interface {
	// DueCount is the number of cards due now.
	DueCount(ctx context.Context, userID string) (int, error)
	// DueBefore counts cards due before `before`; cards without a schedule count when saved
	// before createdBefore (they are due the day after they were saved).
	DueBefore(ctx context.Context, userID string, before, createdBefore time.Time) (int, error)
	// CardCount is the number of cards in the notebook created at or after since (nil = all).
	CardCount(ctx context.Context, userID string, since *time.Time) (int, error)
}

// Grammar is what the stats need from the grammar lessons (F20, implemented over grammar.Service
// in main).
type Grammar interface {
	// MasteredCount counts the grammar points the learner mastered at or after since (nil = all).
	MasteredCount(ctx context.Context, userID string, since *time.Time) (int, error)
}

// Timezones gives a learner's timezone.
type Timezones interface {
	Location(ctx context.Context, userID string) (*time.Location, error)
}
