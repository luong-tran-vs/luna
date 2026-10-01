package progress

import (
	"testing"
	"time"
)

func (e *studyEnv) dashboard(t *testing.T, userID string) DashboardView {
	t.Helper()
	v, err := e.svc.Dashboard(t.Context(), userID)
	if err != nil {
		t.Fatalf("dashboard: %v", err)
	}
	return v
}

func wantAction(t *testing.T, got *Action, kind ActionKind, step Step) {
	t.Helper()
	if got == nil || got.Kind != kind || got.Step != step {
		t.Fatalf("action = %+v, want %s %s", got, kind, step)
	}
}

// --- US1: studying ---

func TestDashboardStudying(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.setGoal(t, "family")
	e.reviews.setDue("u1", 5)

	v := e.dashboard(t, "u1")
	if v.Kind != TodayStudying || v.Goal == nil || v.Goal.TopicName != "Gia đình" || v.Goal.Level != "A1" ||
		v.Goal.CompletedLessons != 0 || v.Goal.TotalLessons != 3 || v.GoalCompleted {
		t.Fatalf("goal = %+v", v)
	}
	if v.Lesson == nil || *v.Lesson != (DashboardLesson{ID: "f1", Title: "Family 1", TopicName: "Gia đình", Level: "A1"}) {
		t.Fatalf("lesson = %+v", v.Lesson)
	}
	if v.Steps[StepReview] != StateCurrent || v.Steps[StepRead] != StateLocked || v.CurrentStep != StepReview {
		t.Fatalf("steps = %v %s", v.Steps, v.CurrentStep)
	}
	wantAction(t, v.Action, ActionStart, StepReview)

	e.complete(t, StepReview)
	v = e.dashboard(t, "u1")
	if v.Steps[StepReview] != StateDone || v.Steps[StepRead] != StateCurrent {
		t.Fatalf("after review steps = %v", v.Steps)
	}
	wantAction(t, v.Action, ActionContinue, StepRead)

	// Another learner sees their own home page.
	if other := e.dashboard(t, "u2"); other.Kind != TodayNoGoal || other.Goal != nil {
		t.Fatalf("u2 = %+v", other)
	}
}

func TestDashboardIsReadOnly(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.setGoal(t, "family") // no card due

	for range 2 {
		wantAction(t, e.dashboard(t, "u1").Action, ActionStart, StepRead)
	}
	if len(e.days.days) != 0 || len(e.progress.rows) != 0 {
		t.Fatalf("dashboard wrote days %v, progress %v", e.days.days, e.progress.rows)
	}
	// Switching topic is still immediate: today's lesson has not started.
	if r := e.setGoal(t, "shopping"); r.StartsTomorrow {
		t.Fatalf("switch = %+v", r)
	}
}

func TestDashboardStreakAndTomorrowCards(t *testing.T) {
	t.Parallel()
	e := newStudyEnv() // 2026-09-30 10:00 in Viet Nam
	e.setGoal(t, "family")
	e.days.put("u1", "2026-09-28", "x")
	e.days.put("u1", "2026-09-29", "y")
	for _, c := range []fakeCard{
		{due: time.Date(2026, 9, 28, 8, 0, 0, 0, hcm)},     // overdue
		{due: time.Date(2026, 10, 1, 23, 0, 0, 0, hcm)},    // tomorrow night
		{due: time.Date(2026, 10, 2, 0, 0, 0, 0, hcm)},     // the day after tomorrow
		{created: time.Date(2026, 9, 30, 9, 0, 0, 0, hcm)}, // saved today without a schedule: due tomorrow
	} {
		e.reviews.addCard("u1", c)
	}
	e.reviews.addCard("u2", fakeCard{due: time.Date(2026, 9, 28, 8, 0, 0, 0, hcm)})

	v := e.dashboard(t, "u1")
	if v.Streak != 2 || v.TomorrowCards != 3 {
		t.Fatalf("streak %d, tomorrow cards %d; want 2, 3", v.Streak, v.TomorrowCards)
	}
	// In New York it is still 2026-09-29 23:00: the cutoff is 2026-10-01 00:00 there, and the card
	// saved without a schedule counts only once saved before 2026-09-30 00:00 New York time.
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	e.zones["u1"] = ny
	if v := e.dashboard(t, "u1"); v.TomorrowCards != 2 {
		t.Fatalf("tomorrow cards in New York = %d, want 2", v.TomorrowCards)
	}
}

func TestDashboardLessonTopic(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.setGoal(t, "family")
	e.complete(t, StepReview)
	// Switching after starting: today's lesson stays f1 of "Gia đình".
	if r := e.setGoal(t, "shopping"); !r.StartsTomorrow {
		t.Fatalf("switch = %+v", r)
	}
	v := e.dashboard(t, "u1")
	if v.Lesson == nil || v.Lesson.ID != "f1" || v.Lesson.TopicName != "Gia đình" {
		t.Fatalf("lesson = %+v", v.Lesson)
	}

	// A deleted lesson has no title.
	e2 := newStudyEnv()
	e2.roadmaps.set(TopicInfo{ID: "ghost", Name: "Ma", Level: "A2", LessonIDs: []string{"gone"}})
	e2.setGoal(t, "ghost")
	if v := e2.dashboard(t, "u1"); v.Lesson == nil || v.Lesson.Title != "" || v.Lesson.TopicName != "Ma" {
		t.Fatalf("deleted lesson = %+v", v.Lesson)
	}
}

// --- US2: special states ---

func TestDashboardNoGoal(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.days.put("u1", "2026-09-29", "x")
	e.reviews.addCard("u1", fakeCard{due: time.Date(2026, 9, 30, 8, 0, 0, 0, hcm)})
	v := e.dashboard(t, "u1")
	if v.Kind != TodayNoGoal || v.Goal != nil || v.Skills != nil || v.Lesson != nil || v.Action != nil {
		t.Fatalf("no goal = %+v", v)
	}
	if v.Streak != 1 || v.TomorrowCards != 1 || len(v.Steps) != 4 || v.Steps[StepReview] != StateLocked {
		t.Fatalf("no goal numbers = %+v", v)
	}
}

func TestDashboardDoneToday(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.setGoal(t, "family")
	e.studyLesson(t)
	v := e.dashboard(t, "u1")
	if v.Kind != TodayDone || v.Lesson == nil || v.Lesson.ID != "f1" || v.Action != nil || v.Streak != 1 ||
		v.CurrentStep != StepDone {
		t.Fatalf("done today = %+v", v)
	}
	for _, s := range Steps {
		if v.Steps[s] != StateDone {
			t.Fatalf("steps = %v", v.Steps)
		}
	}
}

func TestDashboardNoNewLesson(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.setGoal(t, "shopping")
	e.studyLesson(t)
	e.nextDay(1)
	e.studyLesson(t)
	e.nextDay(1)
	v := e.dashboard(t, "u1")
	if v.Kind != TodayNoNewLesson || !v.GoalCompleted || v.Lesson != nil || v.Action != nil ||
		v.Goal.CompletedLessons != 2 || v.Streak != 2 {
		t.Fatalf("no new lesson = %+v", v)
	}

	// An empty roadmap: 0/0, not completed.
	e.setGoal(t, "empty")
	v = e.dashboard(t, "u1")
	if v.Kind != TodayNoNewLesson || v.GoalCompleted || v.Goal.TotalLessons != 0 || v.Skills == nil || v.Skills.Total != 0 {
		t.Fatalf("empty roadmap = %+v %+v", v, v.Skills)
	}
}

// --- US3: skills ---

func TestDashboardSkills(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.setGoal(t, "family")
	e.studyLesson(t) // f1
	e.nextDay(1)

	e.complete(t, StepReview, StepRead) // f2
	v := e.dashboard(t, "u1")
	if *v.Skills != (SkillCounts{Read: 2, Listen: 1, Write: 1, Total: 3}) || v.Goal.CompletedLessons != 1 || v.Streak != 1 {
		t.Fatalf("after read = %+v %+v", v.Skills, v.Goal)
	}
	wantAction(t, v.Action, ActionContinue, StepListen)

	e.complete(t, StepListen)
	v = e.dashboard(t, "u1")
	if *v.Skills != (SkillCounts{Read: 2, Listen: 2, Write: 2, Total: 3}) || v.Goal.CompletedLessons != 2 || v.Streak != 2 ||
		v.Kind != TodayDone {
		t.Fatalf("after listen = %+v %+v", v.Skills, v)
	}

	// Lessons of another topic do not count toward the new goal.
	e.nextDay(1)
	e.setGoal(t, "shopping")
	if v := e.dashboard(t, "u1"); *v.Skills != (SkillCounts{Total: 2}) {
		t.Fatalf("new topic skills = %+v", v.Skills)
	}
}

// --- US4: stats ---

func TestStats(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()

	v, err := e.svc.Stats(t.Context(), "u1")
	if err != nil || v.Cards != 0 || v.Dictation != (DictationTotals{}) || v.Rate != nil || v.Lessons != (StepCounts{}) {
		t.Fatalf("new learner = %+v, %v", v, err)
	}

	e.setGoal(t, "family")
	e.studyLesson(t) // f1: 3 sentences checked, 1/1 word each
	e.nextDay(1)
	e.complete(t, StepReview, StepRead) // f2 read only
	e.nextDay(1)
	e.setGoal(t, "shopping") // older topics still count
	for range 25 {
		e.reviews.addCard("u1", fakeCard{})
	}
	if _, err := e.dictation.Record(t.Context(), "u1", "s1", Input{SentenceIndex: 0, Typed: "x", CorrectWords: 2, TotalWords: 5}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.dictation.Record(t.Context(), "u2", "s1", Input{SentenceIndex: 0, Typed: "x", CorrectWords: 0, TotalWords: 9}); err != nil {
		t.Fatal(err)
	}

	v, err = e.svc.Stats(t.Context(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if v.Cards != 25 || v.Dictation != (DictationTotals{Sentences: 4, CorrectWords: 5, TotalWords: 8}) ||
		v.Rate == nil || *v.Rate != 5.0/8 || v.Lessons != (StepCounts{Read: 2, Listen: 1, Write: 1, Completed: 1}) {
		t.Fatalf("stats = %+v rate %v", v, v.Rate)
	}
}
