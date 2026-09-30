package progress

import (
	"errors"
	"slices"
	"time"
)

// Step is one step of a day's lesson (L).
type Step string

const (
	StepReview Step = "review"
	StepRead   Step = "read"
	StepListen Step = "listen"
	// StepDone is the current step once every step is done.
	StepDone Step = "done"
)

// Steps lists the steps in study order (phase 1: review, read, listen).
var Steps = []Step{StepReview, StepRead, StepListen}

// ValidStep reports whether s is one of Steps.
func ValidStep(s Step) bool { return slices.Contains(Steps, s) }

// DefaultReviewLimit is the number of due cards reviewed in the review step per day.
const DefaultReviewLimit = 30

// GoalStatus is active for the topic being studied, paused for the others.
type GoalStatus string

const (
	GoalActive GoalStatus = "active"
	GoalPaused GoalStatus = "paused"
)

// Goal is a topic a learner chose to study. Its lessons are the topic's roadmap as it is now.
type Goal struct {
	ID      string
	UserID  string
	TopicID string
	Level   string
	Status  GoalStatus
	// EffectiveFrom is the day (YYYY-MM-DD) from which the goal gives today's lesson: today, or
	// tomorrow when it was chosen after today's lesson had started.
	EffectiveFrom string
	StartedAt     time.Time
}

// LessonProgress is a learner's progress on one lesson; there is one per lesson ever started.
type LessonProgress struct {
	UserID   string
	LessonID string
	TopicID  string
	// DayKey is the day the lesson was studied.
	DayKey        string
	Done          map[Step]bool
	CurrentStep   Step
	SentenceIndex int
	StartedAt     time.Time
	CompletedAt   time.Time
}

// StudyDay is a learner's day: its lesson (fixed once a step is done), the cards reviewed in the
// review step and whether the lesson was completed (for the streak).
type StudyDay struct {
	DayKey        string
	LessonID      string
	ReviewedCount int
	Completed     bool
}

// TodayKind is the state of the day's lesson.
type TodayKind string

const (
	TodayNoGoal      TodayKind = "noGoal"
	TodayStudying    TodayKind = "studying"
	TodayDone        TodayKind = "doneToday"
	TodayNoNewLesson TodayKind = "noNewLesson"
)

// TodayState is which lesson the learner has today.
type TodayState struct {
	Kind     TodayKind
	LessonID string
	// Started is true once a step of today's lesson is done (the lesson is then fixed).
	Started bool
}

// StepCounts counts lessons by the steps done: Read and Listen steps, and completed lessons.
type StepCounts struct {
	Read, Listen, Completed int
}

// TopicInfo is a topic with its roadmap (from F14).
type TopicInfo struct {
	ID        string
	Name      string
	Level     string
	LessonIDs []string
}

var (
	ErrNoGoal           = errors.New("progress: no goal")
	ErrTopicNotFound    = errors.New("progress: topic not found")
	ErrStepLocked       = errors.New("progress: previous step not done")
	ErrListenIncomplete = errors.New("progress: dictation not finished")
	ErrNoLesson         = errors.New("progress: no lesson today")
	ErrNotCurrentStep   = errors.New("progress: not the current step")
	ErrLessonLocked     = errors.New("progress: lesson not open yet")
)
