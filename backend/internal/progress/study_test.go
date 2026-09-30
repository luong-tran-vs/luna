package progress

import (
	"errors"
	"testing"
)

// complete finishes the given steps of today's lesson for u1, failing the test on error.
func (e *studyEnv) complete(t *testing.T, steps ...Step) TodayView {
	t.Helper()
	var v TodayView
	for _, s := range steps {
		if s == StepListen {
			e.finishDictation(t)
		}
		var err error
		if v, err = e.svc.CompleteStep(t.Context(), "u1", s); err != nil {
			t.Fatalf("complete %s: %v", s, err)
		}
	}
	return v
}

// finishDictation checks every sentence of today's lesson (F4) so the listen step can complete.
func (e *studyEnv) finishDictation(t *testing.T) {
	t.Helper()
	v, err := e.svc.Today(t.Context(), "u1")
	if err != nil || v.Lesson == nil {
		t.Fatalf("today: %+v %v", v, err)
	}
	for i := range 3 {
		if _, err := e.dictation.Record(t.Context(), "u1", v.Lesson.ID, Input{SentenceIndex: i, Typed: "x", CorrectWords: 1, TotalWords: 1}); err != nil {
			t.Fatal(err)
		}
	}
}

// studyLesson completes today's whole lesson for u1.
func (e *studyEnv) studyLesson(t *testing.T) TodayView {
	t.Helper()
	return e.complete(t, StepReview, StepRead, StepListen)
}

// nextDay moves the clock to 10:00 the next day in Viet Nam.
func (e *studyEnv) nextDay(days int) {
	e.clock.set(e.clock.now().AddDate(0, 0, days))
}

func (e *studyEnv) setGoal(t *testing.T, topicID string) SetGoalResult {
	t.Helper()
	r, err := e.svc.SetGoal(t.Context(), "u1", topicID)
	if err != nil {
		t.Fatalf("set goal %s: %v", topicID, err)
	}
	return r
}

// --- US1: goals ---

func TestSetGoalAndGoals(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	if _, err := e.svc.SetGoal(t.Context(), "u1", "nope"); !errors.Is(err, ErrTopicNotFound) {
		t.Fatalf("unknown topic: %v", err)
	}

	r := e.setGoal(t, "family")
	if r.StartsTomorrow || r.EffectiveFrom != "2026-09-30" || r.Active.TopicName != "Gia đình" || r.Active.TotalLessons != 3 {
		t.Fatalf("result = %+v", r)
	}

	goals, err := e.svc.Goals(t.Context(), "u1")
	if err != nil || goals.Active == nil || goals.Active.TopicID != "family" || goals.Active.Level != "A1" ||
		goals.Active.CompletedLessons != 0 || len(goals.Others) != 0 {
		t.Fatalf("goals = %+v, %v", goals, err)
	}

	// Another learner has no goal.
	if other, _ := e.svc.Goals(t.Context(), "u2"); other.Active != nil {
		t.Fatalf("u2 goals = %+v", other)
	}
}

func TestGoalsCountCompletedLessonsOfCurrentRoadmap(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.setGoal(t, "family")
	e.studyLesson(t)
	e.nextDay(1)
	e.studyLesson(t)

	goals, _ := e.svc.Goals(t.Context(), "u1")
	if goals.Active.CompletedLessons != 2 || goals.Active.TotalLessons != 3 {
		t.Fatalf("progress = %+v", goals.Active)
	}

	// The admin removes a studied lesson from the roadmap and adds a new one.
	e.roadmaps.set(TopicInfo{ID: "family", Name: "Gia đình", Level: "A1", LessonIDs: []string{"f2", "f3", "f4"}})
	goals, _ = e.svc.Goals(t.Context(), "u1")
	if goals.Active.CompletedLessons != 1 || goals.Active.TotalLessons != 3 {
		t.Fatalf("after roadmap edit = %+v", goals.Active)
	}

	// A deleted topic disappears.
	e.setGoal(t, "shopping")
	delete(e.roadmaps.topics, "family")
	goals, _ = e.svc.Goals(t.Context(), "u1")
	if goals.Active.TopicID != "shopping" || len(goals.Others) != 0 {
		t.Fatalf("goals = %+v", goals)
	}
}

func TestTodayWithoutAndWithGoal(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	v, err := e.svc.Today(t.Context(), "u1")
	if err != nil || v.Kind != TodayNoGoal || v.Lesson != nil || v.Goal != nil {
		t.Fatalf("no goal = %+v, %v", v, err)
	}

	e.reviews.setDue("u1", 5)
	e.setGoal(t, "family")
	v, _ = e.svc.Today(t.Context(), "u1")
	if v.Kind != TodayStudying || v.Lesson.ID != "f1" || v.Lesson.Title != "Family 1" || v.Goal.TopicID != "family" {
		t.Fatalf("with goal = %+v", v)
	}

	e.setGoal(t, "empty")
	if v, _ := e.svc.Today(t.Context(), "u1"); v.Kind != TodayNoNewLesson || v.Lesson != nil {
		t.Fatalf("empty roadmap = %+v", v)
	}
}

func TestGoalCompletedAtEndOfRoadmap(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.setGoal(t, "work") // one lesson
	v := e.studyLesson(t)
	if !v.GoalCompleted || v.Kind != TodayDone {
		t.Fatalf("last lesson = %+v", v)
	}
	goals, _ := e.svc.Goals(t.Context(), "u1")
	if goals.Active.CompletedLessons != 1 || goals.Active.TotalLessons != 1 {
		t.Fatalf("goal = %+v", goals.Active)
	}
	e.nextDay(1)
	if v, _ := e.svc.Today(t.Context(), "u1"); v.Kind != TodayNoNewLesson || !v.GoalCompleted {
		t.Fatalf("next day = %+v", v)
	}
}
