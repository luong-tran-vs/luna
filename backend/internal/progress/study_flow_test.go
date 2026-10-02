package progress

import (
	"errors"
	"testing"
	"time"
)

// --- US2: steps ---

func TestCompleteStepOrderAndIdempotence(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.setGoal(t, "family")

	if _, err := e.svc.CompleteStep(t.Context(), "u1", "f1", StepListen); !errors.Is(err, ErrStepLocked) {
		t.Fatalf("listen first: %v", err)
	}
	var verr *ValidationError
	if _, err := e.svc.CompleteStep(t.Context(), "u1", "f1", "speak"); !errors.As(err, &verr) {
		t.Fatalf("unknown step: %v", err)
	}
	// The review step was dropped (2026-10-02).
	if _, err := e.svc.CompleteStep(t.Context(), "u1", "f1", "review"); !errors.As(err, &verr) {
		t.Fatalf("review step: %v", err)
	}
	v := e.complete(t, StepRead)
	if v.CurrentStep != StepListen || v.Steps[StepRead] != StateDone || v.Steps[StepListen] != StateCurrent {
		t.Fatalf("after read = %+v", v)
	}

	// Listening cannot complete before every sentence is checked.
	if _, err := e.svc.CompleteStep(t.Context(), "u1", "f1", StepListen); !errors.Is(err, ErrListenIncomplete) {
		t.Fatalf("listen early: %v", err)
	}
	v = e.complete(t, StepListen)
	if v.Status != LessonCompleted || v.CurrentStep != StepDone || v.Streak != 1 || v.Goal.CompletedLessons != 1 ||
		v.Next == nil || v.Next.ID != "f2" || v.Next.Title != "Family 2" {
		t.Fatalf("done = %+v", v)
	}

	// Completing again changes nothing.
	again, err := e.svc.CompleteStep(t.Context(), "u1", "f1", StepListen)
	if err != nil || again.Streak != 1 || again.Goal.CompletedLessons != 1 || again.Status != LessonCompleted {
		t.Fatalf("again = %+v, %v", again, err)
	}
	if p, _, _ := e.progress.Get(t.Context(), "u1", "f1"); p.CompletedAt.IsZero() || p.CurrentStep != StepDone {
		t.Fatalf("progress = %+v", p)
	}
}

func TestOldProgressWithReviewStep(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.setGoal(t, "family")
	// Saved before 2026-10-02, when a lesson began with the review step.
	_ = e.progress.Upsert(t.Context(), LessonProgress{
		UserID: "u1", LessonID: "f1", Done: map[Step]bool{"review": true}, CurrentStep: StepRead, SentenceIndex: 1,
	})
	v, _ := e.svc.LessonStudy(t.Context(), "u1", "f1")
	if v.CurrentStep != StepRead || v.SentenceIndex != 1 || len(v.Steps) != 3 {
		t.Fatalf("old progress = %+v", v)
	}
}

func TestStepsOnlyForTheCurrentLesson(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	if _, err := e.svc.CompleteStep(t.Context(), "u1", "f1", StepRead); !errors.Is(err, ErrNotCurrentLesson) {
		t.Fatalf("no goal: %v", err)
	}
	e.setGoal(t, "family")
	if _, err := e.svc.CompleteStep(t.Context(), "u1", "f2", StepRead); !errors.Is(err, ErrNotCurrentLesson) {
		t.Fatalf("upcoming lesson: %v", err)
	}
	if err := e.svc.SetPosition(t.Context(), "u1", "f2", StepRead, 1); !errors.Is(err, ErrNotCurrentLesson) {
		t.Fatalf("position of upcoming lesson: %v", err)
	}
}

func TestPosition(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.setGoal(t, "family")

	// The first position starts the lesson.
	if err := e.svc.SetPosition(t.Context(), "u1", "f1", StepRead, 2); err != nil {
		t.Fatal(err)
	}
	if v, _ := e.svc.LessonStudy(t.Context(), "u1", "f1"); v.CurrentStep != StepRead || v.SentenceIndex != 2 {
		t.Fatalf("position = %+v", v)
	}
	if p, ok, _ := e.progress.Get(t.Context(), "u1", "f1"); !ok || p.TopicID != "family" || p.DayKey != "2026-09-30" {
		t.Fatalf("started = %+v %v", p, ok)
	}
	if err := e.svc.SetPosition(t.Context(), "u1", "f1", StepListen, 1); !errors.Is(err, ErrNotCurrentStep) {
		t.Fatalf("other step: %v", err)
	}
	var verr *ValidationError
	for _, bad := range []int{-1, 3} {
		if err := e.svc.SetPosition(t.Context(), "u1", "f1", StepRead, bad); !errors.As(err, &verr) {
			t.Fatalf("sentence %d: %v", bad, err)
		}
	}
	// Moving to the next step starts at its first sentence.
	if v := e.complete(t, StepRead); v.SentenceIndex != 0 || v.CurrentStep != StepListen {
		t.Fatalf("after read = %+v", v)
	}
}

// --- US3: one lesson after another, streak ---

func TestNextLessonOpensAtOnce(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.setGoal(t, "family")
	e.studyLesson(t)
	if id := e.current(t); id != "f2" {
		t.Fatalf("after f1 = %s", id)
	}
	if ok, _ := e.svc.CanOpen(t.Context(), "u1", false, "f2"); !ok {
		t.Fatal("f2 locked after f1")
	}
	// Same day, a second and a third lesson: the day counts once for the streak.
	e.studyLesson(t)
	v := e.studyLesson(t)
	if v.Streak != 1 || !v.GoalCompleted || v.Goal.CompletedLessons != 3 {
		t.Fatalf("three lessons in a day = %+v", v)
	}
}

func TestDaysOffAndStreak(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.setGoal(t, "family")
	e.days.put("u1", "2026-09-28")
	e.days.put("u1", "2026-09-29")
	if v, _ := e.svc.LessonStudy(t.Context(), "u1", "f1"); v.Streak != 2 {
		t.Fatalf("today not done yet: %d", v.Streak)
	}
	if v := e.studyLesson(t); v.Streak != 3 {
		t.Fatalf("after today: %d", v.Streak)
	}

	// Three days off: streak 0, then 1.
	e.nextDay(4)
	if v, _ := e.svc.LessonStudy(t.Context(), "u1", "f2"); v.Status != LessonStudying || v.Streak != 0 {
		t.Fatalf("after days off = %+v", v)
	}
	if v := e.studyLesson(t); v.Streak != 1 {
		t.Fatalf("back = %d", v.Streak)
	}
}

func TestDayFollowsTheLearnersTimezone(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.setGoal(t, "family")
	e.clock.set(time.Date(2026, 9, 30, 23, 50, 0, 0, hcm))
	e.studyLesson(t) // 2026-09-30
	e.clock.set(time.Date(2026, 10, 1, 0, 10, 0, 0, hcm))
	if v := e.studyLesson(t); v.Streak != 2 {
		t.Fatalf("after midnight = %+v", v)
	}
}

// --- US4: switching topics ---

func TestSwitchTakesEffectAtOnce(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.setGoal(t, "family")
	e.studyLesson(t)        // f1
	e.complete(t, StepRead) // f2 started
	e.setGoal(t, "shopping")
	if id := e.current(t); id != "s1" {
		t.Fatalf("after switch = %s", id)
	}
	// f2 was started: it still opens, but its steps wait for the family topic.
	if ok, _ := e.svc.CanOpen(t.Context(), "u1", false, "f2"); !ok {
		t.Fatal("started lesson locked")
	}
	if v, _ := e.svc.LessonStudy(t.Context(), "u1", "f2"); v.Status != LessonOther {
		t.Fatalf("f2 = %+v", v)
	}
	if _, err := e.svc.CompleteStep(t.Context(), "u1", "f2", StepListen); !errors.Is(err, ErrNotCurrentLesson) {
		t.Fatalf("f2 listen: %v", err)
	}

	goals, _ := e.svc.Goals(t.Context(), "u1")
	if len(goals.Others) != 1 || goals.Others[0].TopicID != "family" || goals.Others[0].Status != GoalPaused ||
		goals.Others[0].CompletedLessons != 1 {
		t.Fatalf("goals = %+v", goals)
	}

	// Another level: only B1 lessons from now on.
	e.setGoal(t, "work")
	if id := e.current(t); id != "w1" {
		t.Fatalf("B1 = %s", id)
	}
}

func TestReturnToOldTopicResumes(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.setGoal(t, "family")
	e.studyLesson(t)        // f1
	e.complete(t, StepRead) // f2 started
	e.setGoal(t, "shopping")
	e.studyLesson(t) // s1
	e.setGoal(t, "family")
	v, _ := e.svc.LessonStudy(t.Context(), "u1", "f2")
	if v.Status != LessonStudying || v.CurrentStep != StepListen || v.Goal.CompletedLessons != 1 || v.Streak != 1 {
		t.Fatalf("resume = %+v", v)
	}
}

// --- US5: lessons and access ---

func TestMyLessonsAndAccess(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.setGoal(t, "family")
	e.studyLesson(t) // f1
	e.setGoal(t, "shopping")
	e.studyLesson(t) // s1
	e.setGoal(t, "family")

	mine, err := e.svc.MyLessons(t.Context(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if mine.Current == nil || mine.Current.ID != "f2" || mine.Current.Title != "Family 2" {
		t.Fatalf("current = %+v", mine.Current)
	}
	// Only the goal's topic: s1 was finished in "shopping", so it is listed there, not here.
	if len(mine.Completed) != 1 || mine.Completed[0].ID != "f1" || mine.Completed[0].Title != "Family 1" ||
		mine.Completed[0].TopicName != "Gia đình" {
		t.Fatalf("completed = %+v", mine.Completed)
	}
	if len(mine.Upcoming) != 1 || mine.Upcoming[0].ID != "f3" || mine.Upcoming[0].Title != "Family 3" {
		t.Fatalf("upcoming = %+v", mine.Upcoming)
	}
	if v, _ := e.svc.LessonStudy(t.Context(), "u1", "f1"); v.Status != LessonCompleted || v.CurrentStep != StepDone ||
		v.Steps[StepWrite] != StateDone || v.Next == nil || v.Next.ID != "f2" {
		t.Fatalf("completed f1 = %+v", v)
	}

	for id, want := range map[string]bool{"f2": true, "f1": true, "s1": true, "f3": false, "s2": false, "w1": false} {
		if ok, err := e.svc.CanOpen(t.Context(), "u1", false, id); err != nil || ok != want {
			t.Errorf("CanOpen(%s) = %v, want %v (%v)", id, ok, want, err)
		}
	}
	if ok, _ := e.svc.CanOpen(t.Context(), "admin", true, "f3"); !ok {
		t.Error("admin blocked")
	}
	if ok, _ := e.svc.CanOpen(t.Context(), "u2", false, "f1"); ok {
		t.Error("another learner opens f1 without a goal")
	}
}
