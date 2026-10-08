package progress

import (
	"errors"
	"slices"
	"time"
)

// Step is one step of a lesson (L).
type Step string

const (
	StepRead   Step = "read"
	StepListen Step = "listen"
	StepWrite  Step = "write"
	// StepDone is the current step once every step is done.
	StepDone Step = "done"
)

// Steps lists the steps in study order (F8 adds write after listen). The review step was dropped
// on 2026-10-02: due cards are reviewed in the notebook, not in a lesson; progress saved before
// may still hold it and it is ignored.
var Steps = []Step{StepRead, StepListen, StepWrite}

// ValidStep reports whether s is one of Steps.
func ValidStep(s Step) bool { return slices.Contains(Steps, s) }

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
	// EffectiveFrom is the day (YYYY-MM-DD) the goal was chosen; it takes effect at once.
	EffectiveFrom string
	StartedAt     time.Time
}

// LessonProgress is a learner's progress on one lesson; there is one per lesson ever started.
type LessonProgress struct {
	UserID   string
	LessonID string
	TopicID  string
	// DayKey is the day the lesson was started.
	DayKey string
	Done   map[Step]bool
	// DoneAt is when each step was done. It is recorded since 2026-10-07: older progress has
	// steps in Done without a time.
	DoneAt        map[Step]time.Time
	CurrentStep   Step
	SentenceIndex int
	StartedAt     time.Time
	CompletedAt   time.Time
}

// StudyKind is where the learner stands in the goal's roadmap.
type StudyKind string

const (
	StudyNoGoal      StudyKind = "noGoal"
	StudyStudying    StudyKind = "studying"
	StudyNoNewLesson StudyKind = "noNewLesson"
	// StudyMembersOnly: a guest finished the first lesson of the roadmap; the others are for members.
	StudyMembersOnly StudyKind = "membersOnly"
)

// StudyState is the lesson the learner studies now: the first lesson of the roadmap not completed.
type StudyState struct {
	Kind     StudyKind
	LessonID string
}

// LessonStatus is where one lesson stands for the learner.
type LessonStatus string

const (
	// LessonStudying is the lesson the learner studies now.
	LessonStudying LessonStatus = "studying"
	// LessonCompleted is a lesson the learner finished.
	LessonCompleted LessonStatus = "completed"
	// LessonOther is any other lesson (started in another topic, or opened by an admin).
	LessonOther LessonStatus = "other"
)

// StepCounts counts lessons by the steps done (Read, Listen, Write) and completed lessons.
type StepCounts struct {
	Read, Listen, Write, Completed int
}

// TopicInfo is a topic with its roadmap at one level (from F14); Level is "" when no level was asked.
type TopicInfo struct {
	ID        string
	Name      string
	Level     string
	LessonIDs []string
}

var (
	ErrNoGoal           = errors.New("progress: no goal")
	ErrTopicNotFound    = errors.New("progress: topic not found")
	ErrListenIncomplete = errors.New("progress: dictation not finished")
	ErrReadIncomplete   = errors.New("progress: comprehension questions not answered")
	ErrWriteIncomplete  = errors.New("progress: writing not submitted")
	ErrNotCurrentLesson = errors.New("progress: not the lesson being studied")
	ErrNotCurrentStep   = errors.New("progress: step already done")
	ErrLessonLocked     = errors.New("progress: lesson not open yet")
)
