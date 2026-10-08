package progress

import (
	"errors"
	"testing"
)

// current is the lesson u1 studies now, failing the test when there is none.
func (e *studyEnv) current(t *testing.T) string {
	t.Helper()
	mine, err := e.svc.MyLessons(t.Context(), "u1")
	if err != nil || mine.Current == nil {
		t.Fatalf("current lesson: %+v %v", mine, err)
	}
	return mine.Current.ID
}

// complete finishes the given steps of u1's current lesson, failing the test on error. Ending
// with listen also submits and completes the writing (F8), so tests written for two steps still
// finish the lesson.
func (e *studyEnv) complete(t *testing.T, steps ...Step) LessonStudyView {
	t.Helper()
	if len(steps) > 0 && steps[len(steps)-1] == StepListen {
		steps = append(steps, StepWrite)
	}
	id := e.current(t)
	var v LessonStudyView
	for _, s := range steps {
		switch s {
		case StepListen:
			e.finishDictation(t)
		case StepWrite:
			e.submitWriting(t)
		}
		var err error
		if v, err = e.svc.CompleteStep(t.Context(), "u1", id, s); err != nil {
			t.Fatalf("complete %s of %s: %v", s, id, err)
		}
	}
	return v
}

// submitWriting submits the writing of u1's current lesson (F8) so the write step can complete.
func (e *studyEnv) submitWriting(t *testing.T) {
	t.Helper()
	e.writings.submit("u1", e.current(t))
}

// finishDictation checks every sentence of u1's current lesson (F4) so the listen step can
// complete.
func (e *studyEnv) finishDictation(t *testing.T) {
	t.Helper()
	id := e.current(t)
	for i := range 3 {
		if _, err := e.dictation.Record(t.Context(), "u1", id, Input{SentenceIndex: i, Typed: "x", CorrectWords: 1, TotalWords: 1}); err != nil {
			t.Fatal(err)
		}
	}
}

// studyLesson completes u1's whole current lesson.
func (e *studyEnv) studyLesson(t *testing.T) LessonStudyView {
	t.Helper()
	return e.complete(t, StepRead, StepListen)
}

// nextDay moves the clock to 10:00 the next day in Viet Nam.
func (e *studyEnv) nextDay(days int) {
	e.clock.set(e.clock.now().AddDate(0, 0, days))
}

func (e *studyEnv) setGoal(t *testing.T, topicID string) GoalView {
	t.Helper()
	g, err := e.svc.SetGoal(t.Context(), "u1", topicID, e.roadmaps.topics[topicID].Level)
	if err != nil {
		t.Fatalf("set goal %s: %v", topicID, err)
	}
	return g
}

// --- US1: goals ---

func TestSetGoalAndGoals(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	if _, err := e.svc.SetGoal(t.Context(), "u1", "nope", "A1"); !errors.Is(err, ErrTopicNotFound) {
		t.Fatalf("unknown topic: %v", err)
	}

	g := e.setGoal(t, "family")
	if g.EffectiveFrom != "2026-09-30" || g.TopicName != "Gia đình" || g.TotalLessons != 3 {
		t.Fatalf("result = %+v", g)
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

func TestCurrentLessonWithoutAndWithGoal(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	mine, err := e.svc.MyLessons(t.Context(), "u1")
	if err != nil || mine.Current != nil || len(mine.Upcoming) != 0 {
		t.Fatalf("no goal = %+v, %v", mine, err)
	}
	if v, _ := e.svc.LessonStudy(t.Context(), "u1", "f1"); v.Status != LessonOther || v.Goal != nil {
		t.Fatalf("f1 without goal = %+v", v)
	}

	e.setGoal(t, "family")
	v, _ := e.svc.LessonStudy(t.Context(), "u1", "f1")
	if v.Status != LessonStudying || v.CurrentStep != StepRead || v.Steps[StepRead] != StateCurrent ||
		v.Steps[StepListen] != StateLocked || v.Steps[StepWrite] != StateLocked || v.Goal.TopicID != "family" || v.Next != nil {
		t.Fatalf("with goal = %+v", v)
	}
	if v, _ := e.svc.LessonStudy(t.Context(), "u1", "f2"); v.Status != LessonOther || len(v.Steps) != 0 {
		t.Fatalf("f2 = %+v", v)
	}

	e.setGoal(t, "empty")
	if mine, _ := e.svc.MyLessons(t.Context(), "u1"); mine.Current != nil {
		t.Fatalf("empty roadmap = %+v", mine)
	}
}

func TestGoalCompletedAtEndOfRoadmap(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.setGoal(t, "work") // one lesson
	v := e.studyLesson(t)
	if !v.GoalCompleted || v.Status != LessonCompleted || v.Next != nil {
		t.Fatalf("last lesson = %+v", v)
	}
	goals, _ := e.svc.Goals(t.Context(), "u1")
	if goals.Active.CompletedLessons != 1 || goals.Active.TotalLessons != 1 {
		t.Fatalf("goal = %+v", goals.Active)
	}
	d, _ := e.svc.Dashboard(t.Context(), "u1")
	if d.Kind != StudyNoNewLesson || !d.GoalCompleted {
		t.Fatalf("dashboard = %+v", d)
	}
}

func TestSameTopicAtTwoLevelsIsTwoGoals(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.setGoal(t, "family")
	e.studyLesson(t)

	g, err := e.svc.SetGoal(t.Context(), "u1", "family", "A2")
	if err != nil || g.Level != "A2" || g.TotalLessons != 0 {
		t.Fatalf("A2 goal = %+v, %v", g, err)
	}
	goals, _ := e.svc.Goals(t.Context(), "u1")
	if goals.Active.Level != "A2" || len(goals.Others) != 1 || goals.Others[0].Level != "A1" || goals.Others[0].CompletedLessons != 1 {
		t.Fatalf("goals = %+v", goals)
	}
}

// --- guests study only the first lesson of each roadmap ---

func TestGuestStudiesOnlyTheFirstLesson(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.members.setGuest("u1", true)
	e.setGoal(t, "family") // f1, f2, f3

	mine, err := e.svc.MyLessons(t.Context(), "u1")
	if err != nil || mine.Current == nil || mine.Current.ID != "f1" || len(mine.Upcoming) != 2 ||
		!mine.Upcoming[0].MembersOnly || !mine.Upcoming[1].MembersOnly {
		t.Fatalf("guest lessons = %+v, %v", mine, err)
	}
	v := e.studyLesson(t)
	if v.Status != LessonCompleted || v.Next != nil || !v.MembersOnly || v.GoalCompleted {
		t.Fatalf("after first lesson = %+v", v)
	}
	d, _ := e.svc.Dashboard(t.Context(), "u1")
	if d.Kind != StudyMembersOnly || d.Lesson != nil || d.GoalCompleted {
		t.Fatalf("dashboard = %+v", d)
	}
	if _, err := e.svc.CompleteStep(t.Context(), "u1", "f2", StepRead); !errors.Is(err, ErrNotCurrentLesson) {
		t.Fatalf("guest studies f2: %v", err)
	}
	if ok, _ := e.svc.CanOpen(t.Context(), "u1", false, "f2"); ok {
		t.Fatal("guest opens f2")
	}

	// Another topic: its first lesson is open to the guest too.
	e.setGoal(t, "shopping")
	if id := e.current(t); id != "s1" {
		t.Fatalf("shopping current = %s", id)
	}

	// Made a member: the next lessons open.
	e.members.setGuest("u1", false)
	e.setGoal(t, "family")
	if id := e.current(t); id != "f2" {
		t.Fatalf("member current = %s", id)
	}
	if mine, _ := e.svc.MyLessons(t.Context(), "u1"); mine.Upcoming[0].MembersOnly {
		t.Fatalf("member upcoming = %+v", mine.Upcoming)
	}
}

func TestGuestWithOneLessonRoadmapFinishesTheGoal(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.members.setGuest("u1", true)
	e.setGoal(t, "work") // one lesson
	if v := e.studyLesson(t); !v.GoalCompleted || v.MembersOnly {
		t.Fatalf("one-lesson roadmap = %+v", v)
	}
}
