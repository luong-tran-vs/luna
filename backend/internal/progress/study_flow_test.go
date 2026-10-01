package progress

import (
	"errors"
	"testing"
	"time"
)

// --- US2: steps ---

func TestTodayStepsAndReviewCount(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.setGoal(t, "family")
	e.reviews.setDue("u1", 45)

	v, _ := e.svc.Today(t.Context(), "u1")
	if v.CurrentStep != StepReview || v.Steps[StepReview] != StateCurrent || v.Steps[StepRead] != StateLocked ||
		v.Steps[StepListen] != StateLocked || v.ReviewCount != 30 {
		t.Fatalf("today = %+v", v)
	}

	// 12 cards reviewed in the review step: 18 left today.
	e.reviews.review("u1", 12, e.clock.now())
	if v, _ := e.svc.Today(t.Context(), "u1"); v.ReviewCount != 18 {
		t.Fatalf("after 12 = %d", v.ReviewCount)
	}
}

func TestReviewStepCompletesItselfWithoutDueCards(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.setGoal(t, "family")
	v, err := e.svc.Today(t.Context(), "u1")
	if err != nil || v.Steps[StepReview] != StateDone || v.CurrentStep != StepRead || v.ReviewCount != 0 {
		t.Fatalf("today = %+v, %v", v, err)
	}
	// Opening fixed the lesson for the day.
	if d, _ := e.days.Get(t.Context(), "u1", "2026-09-30"); d == nil || d.LessonID != "f1" {
		t.Fatalf("day = %+v", d)
	}
}

func TestCompleteStepOrderAndIdempotence(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.setGoal(t, "family")
	e.reviews.setDue("u1", 3)

	if _, err := e.svc.CompleteStep(t.Context(), "u1", StepRead); !errors.Is(err, ErrStepLocked) {
		t.Fatalf("read first: %v", err)
	}
	var verr *ValidationError
	if _, err := e.svc.CompleteStep(t.Context(), "u1", "speak"); !errors.As(err, &verr) {
		t.Fatalf("unknown step: %v", err)
	}
	e.reviews.review("u1", 3, e.clock.now())
	v := e.complete(t, StepReview)
	if v.CurrentStep != StepRead || v.Steps[StepReview] != StateDone {
		t.Fatalf("after review = %+v", v)
	}
	if d, _ := e.days.Get(t.Context(), "u1", "2026-09-30"); d.ReviewedCount != 3 {
		t.Fatalf("reviewed = %d", d.ReviewedCount)
	}
	e.complete(t, StepRead)

	// Listening cannot complete before every sentence is checked.
	if _, err := e.svc.CompleteStep(t.Context(), "u1", StepListen); !errors.Is(err, ErrListenIncomplete) {
		t.Fatalf("listen early: %v", err)
	}
	v = e.complete(t, StepListen)
	if v.Kind != TodayDone || v.Streak != 1 || v.Goal.CompletedLessons != 1 {
		t.Fatalf("done = %+v", v)
	}

	// Completing again changes nothing.
	again, err := e.svc.CompleteStep(t.Context(), "u1", StepListen)
	if err != nil || again.Streak != 1 || again.Goal.CompletedLessons != 1 {
		t.Fatalf("again = %+v, %v", again, err)
	}
	if p, _, _ := e.progress.Get(t.Context(), "u1", "f1"); p.CompletedAt.IsZero() || p.CurrentStep != StepDone {
		t.Fatalf("progress = %+v", p)
	}
}

func TestCompleteWithoutLesson(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	if _, err := e.svc.CompleteStep(t.Context(), "u1", StepReview); !errors.Is(err, ErrNoLesson) {
		t.Fatalf("no goal: %v", err)
	}
}

func TestPosition(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.setGoal(t, "family")
	e.complete(t, StepReview)

	if err := e.svc.SetPosition(t.Context(), "u1", StepRead, 2); err != nil {
		t.Fatal(err)
	}
	if v, _ := e.svc.Today(t.Context(), "u1"); v.CurrentStep != StepRead || v.SentenceIndex != 2 {
		t.Fatalf("position = %+v", v)
	}
	if err := e.svc.SetPosition(t.Context(), "u1", StepListen, 1); !errors.Is(err, ErrNotCurrentStep) {
		t.Fatalf("other step: %v", err)
	}
	var verr *ValidationError
	for _, bad := range []int{-1, 3} {
		if err := e.svc.SetPosition(t.Context(), "u1", StepRead, bad); !errors.As(err, &verr) {
			t.Fatalf("sentence %d: %v", bad, err)
		}
	}
	// Moving to the next step starts at its first sentence.
	if v := e.complete(t, StepRead); v.SentenceIndex != 0 || v.CurrentStep != StepListen {
		t.Fatalf("after read = %+v", v)
	}
}

// --- US3: days and streak ---

func TestOneLessonPerDay(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.setGoal(t, "family")
	e.clock.set(time.Date(2026, 9, 30, 23, 50, 0, 0, hcm))
	e.studyLesson(t)

	e.clock.set(time.Date(2026, 9, 30, 23, 59, 0, 0, hcm))
	if v, _ := e.svc.Today(t.Context(), "u1"); v.Kind != TodayDone || v.Lesson.ID != "f1" {
		t.Fatalf("23:59 = %+v", v)
	}
	e.clock.set(time.Date(2026, 10, 1, 0, 1, 0, 0, hcm))
	v, _ := e.svc.Today(t.Context(), "u1")
	if v.Kind != TodayStudying || v.Lesson.ID != "f2" || v.Streak != 1 {
		t.Fatalf("00:01 = %+v", v)
	}
}

func TestDaysOffAndStreak(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.setGoal(t, "family")
	e.days.put("u1", "2026-09-28", "x")
	e.days.put("u1", "2026-09-29", "y")
	if v, _ := e.svc.Today(t.Context(), "u1"); v.Streak != 2 {
		t.Fatalf("today not done yet: %d", v.Streak)
	}
	if v := e.studyLesson(t); v.Streak != 3 {
		t.Fatalf("after today: %d", v.Streak)
	}

	// Three days off: still one lesson, streak 0, then 1.
	e.nextDay(4)
	v, _ := e.svc.Today(t.Context(), "u1")
	if v.Kind != TodayStudying || v.Lesson.ID != "f2" || v.Streak != 0 {
		t.Fatalf("after days off = %+v", v)
	}
	if v := e.studyLesson(t); v.Streak != 1 {
		t.Fatalf("back = %d", v.Streak)
	}
}

func TestReviewQuotaMovesToNextDay(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.setGoal(t, "family")
	e.reviews.setDue("u1", 45)
	e.reviews.review("u1", 30, e.clock.now()) // 15 still due
	v, _ := e.svc.Today(t.Context(), "u1")
	if v.ReviewCount != 0 || v.Steps[StepReview] != StateDone {
		t.Fatalf("quota used = %+v", v)
	}
	e.studyLesson(t)
	e.nextDay(1)
	if v, _ := e.svc.Today(t.Context(), "u1"); v.ReviewCount != 15 {
		t.Fatalf("next day = %d", v.ReviewCount)
	}
}

func TestTimezoneChangeKeepsOneLessonPerDay(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.setGoal(t, "family")
	e.clock.set(time.Date(2026, 9, 30, 8, 0, 0, 0, hcm)) // 2026-09-29 21:00 in New York
	e.studyLesson(t)                                     // f1 on 2026-09-30 (Viet Nam)

	// In New York it is still 2026-09-29, but the study day never goes back (F12): 2026-09-30 stays
	// today, done, and no second lesson is given in the same real day.
	e.zones["u1"] = time.FixedZone("NY", -4*3600)
	if v, _ := e.svc.Today(t.Context(), "u1"); v.Kind != TodayDone || v.Lesson.ID != "f1" || v.Streak != 1 {
		t.Fatalf("new york = %+v", v)
	}
	// Back in Viet Nam, 2026-09-30 is done.
	delete(e.zones, "u1")
	if v, _ := e.svc.Today(t.Context(), "u1"); v.Kind != TodayDone || v.Lesson.ID != "f1" {
		t.Fatalf("back = %+v", v)
	}
}

// --- US4: switching topics ---

func TestSwitchBeforeStarting(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.reviews.setDue("u1", 2)
	e.setGoal(t, "family")
	if v, _ := e.svc.Today(t.Context(), "u1"); v.Lesson.ID != "f1" {
		t.Fatalf("family = %+v", v)
	}
	r := e.setGoal(t, "shopping")
	if r.StartsTomorrow || r.EffectiveFrom != "2026-09-30" {
		t.Fatalf("result = %+v", r)
	}
	if v, _ := e.svc.Today(t.Context(), "u1"); v.Lesson.ID != "s1" {
		t.Fatalf("today = %+v", v)
	}
}

func TestSwitchAfterStarting(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.setGoal(t, "family")
	e.studyLesson(t) // f1
	e.nextDay(1)
	e.studyLesson(t) // f2
	e.nextDay(1)
	e.complete(t, StepReview) // f3 started on 2026-10-02

	r := e.setGoal(t, "shopping")
	if !r.StartsTomorrow || r.EffectiveFrom != "2026-10-03" {
		t.Fatalf("result = %+v", r)
	}
	v, _ := e.svc.Today(t.Context(), "u1")
	if v.Lesson.ID != "f3" || v.Goal.TopicID != "shopping" {
		t.Fatalf("today after switch = %+v", v)
	}
	if v = e.complete(t, StepRead, StepListen); v.Streak != 3 { // finishing f3 today still counts
		t.Fatalf("streak = %d", v.Streak)
	}

	e.nextDay(1)
	if v, _ := e.svc.Today(t.Context(), "u1"); v.Lesson.ID != "s1" || v.Streak != 3 {
		t.Fatalf("tomorrow = %+v", v)
	}
	goals, _ := e.svc.Goals(t.Context(), "u1")
	if len(goals.Others) != 1 || goals.Others[0].TopicID != "family" || goals.Others[0].Status != GoalPaused ||
		goals.Others[0].CompletedLessons != 3 {
		t.Fatalf("goals = %+v", goals)
	}

	// Another level: opening today auto-completed the review step of s1, so B1 starts tomorrow;
	// from then on only B1 lessons.
	if r := e.setGoal(t, "work"); !r.StartsTomorrow {
		t.Fatalf("work = %+v", r)
	}
	e.nextDay(1)
	if v, _ := e.svc.Today(t.Context(), "u1"); v.Lesson.ID != "w1" || v.Goal.Level != "B1" {
		t.Fatalf("B1 = %+v", v)
	}
}

func TestReturnToOldTopicResumes(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.setGoal(t, "family")
	e.studyLesson(t) // f1
	e.nextDay(1)
	e.setGoal(t, "shopping")
	e.studyLesson(t) // s1
	e.nextDay(1)
	e.setGoal(t, "family")
	if v, _ := e.svc.Today(t.Context(), "u1"); v.Lesson.ID != "f2" || v.Goal.CompletedLessons != 1 || v.Streak != 2 {
		t.Fatalf("resume = %+v", v)
	}
}

// --- US5: lessons and access ---

func TestMyLessonsAndAccess(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.setGoal(t, "family")
	e.studyLesson(t) // f1
	e.nextDay(1)
	e.setGoal(t, "shopping")
	e.studyLesson(t) // s1
	e.nextDay(1)
	e.setGoal(t, "family")

	mine, err := e.svc.MyLessons(t.Context(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if mine.Today == nil || mine.Today.ID != "f2" {
		t.Fatalf("today = %+v", mine.Today)
	}
	if len(mine.Completed) != 2 || mine.Completed[0].ID != "s1" || mine.Completed[0].TopicName != "Mua sắm" ||
		mine.Completed[1].ID != "f1" || mine.Completed[1].Title != "Family 1" {
		t.Fatalf("completed = %+v", mine.Completed)
	}
	if len(mine.Upcoming) != 1 || mine.Upcoming[0].ID != "f3" || mine.Upcoming[0].Title != "Family 3" {
		t.Fatalf("upcoming = %+v", mine.Upcoming)
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
