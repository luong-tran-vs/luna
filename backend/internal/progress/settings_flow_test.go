package progress

import (
	"testing"
	"time"
)

// --- F12: the daily card limit and the timezone come from the settings ---

func TestReviewLimitFromSettings(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.setGoal(t, "family")
	e.reviews.setDue("u1", 25)
	e.limit.Store(10)
	if v, _ := e.svc.Today(t.Context(), "u1"); v.ReviewCount != 10 {
		t.Fatalf("limit 10 = %d", v.ReviewCount)
	}

	// 20 cards reviewed today, then the limit lowered to 10: nothing left, the step completes.
	e2 := newStudyEnv()
	e2.setGoal(t, "family")
	e2.reviews.setDue("u1", 25)
	e2.reviews.review("u1", 20, e2.clock.now())
	e2.limit.Store(10)
	if v, _ := e2.svc.Today(t.Context(), "u1"); v.ReviewCount != 0 || v.Steps[StepReview] != StateDone {
		t.Fatalf("lowered = %+v", v)
	}

	// 30 reviewed, the limit raised to 50: 20 more.
	e3 := newStudyEnv()
	e3.setGoal(t, "family")
	e3.reviews.setDue("u1", 45)
	e3.reviews.review("u1", 30, e3.clock.now()) // 15 still due
	e3.limit.Store(50)
	if v, _ := e3.svc.Today(t.Context(), "u1"); v.ReviewCount != 15 {
		t.Fatalf("raised, 15 due = %d", v.ReviewCount)
	}
	e3.reviews.setDue("u1", 40)
	if v, _ := e3.svc.Today(t.Context(), "u1"); v.ReviewCount != 20 {
		t.Fatalf("raised, 40 due = %d", v.ReviewCount)
	}
}

func TestTimezoneChangeSameDayKeepsProgress(t *testing.T) {
	t.Parallel()
	e := newStudyEnv() // 2026-09-30 10:00 in Viet Nam = 04:00 in London
	e.setGoal(t, "family")
	e.reviews.setDue("u1", 3)
	e.complete(t, StepReview)
	if err := e.svc.SetPosition(t.Context(), "u1", StepRead, 2); err != nil {
		t.Fatal(err)
	}

	london, err := time.LoadLocation("Europe/London")
	if err != nil {
		t.Fatal(err)
	}
	e.zones["u1"] = london
	v, err := e.svc.Today(t.Context(), "u1")
	if err != nil || v.Lesson == nil || v.Lesson.ID != "f1" || v.Steps[StepReview] != StateDone ||
		v.CurrentStep != StepRead || v.SentenceIndex != 2 {
		t.Fatalf("london = %+v, %v", v, err)
	}
}

func TestTimezoneChangeToNextDayKeepsUnfinishedLesson(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.clock.set(time.Date(2026, 9, 30, 23, 0, 0, 0, hcm)) // 2026-09-30 in Viet Nam, 2026-10-01 in Tokyo+2
	e.setGoal(t, "family")
	e.reviews.setDue("u1", 3)
	e.complete(t, StepReview, StepRead)

	e.zones["u1"] = time.FixedZone("UTC+10", 10*3600) // already 2026-10-01 01:00
	v, err := e.svc.Today(t.Context(), "u1")
	if err != nil || v.Kind != TodayStudying || v.Lesson == nil || v.Lesson.ID != "f1" ||
		v.Steps[StepReview] != StateDone || v.Steps[StepRead] != StateDone || v.CurrentStep != StepListen {
		t.Fatalf("next day = %+v, %v", v, err)
	}
	// Finishing it counts for the new day; the previous days are kept.
	if v := e.complete(t, StepListen); v.Kind != TodayDone || v.Streak != 1 {
		t.Fatalf("finished = %+v", v)
	}
}

func TestTimezoneChangeToPreviousDayGivesNoNewLesson(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.clock.set(time.Date(2026, 10, 1, 0, 30, 0, 0, hcm)) // 2026-09-30 10:30 in Los Angeles
	e.days.put("u1", "2026-09-30", "x")
	e.setGoal(t, "family")
	e.studyLesson(t) // f1 on 2026-10-01 (Viet Nam), streak 2

	la, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatal(err)
	}
	e.zones["u1"] = la
	v, err := e.svc.Today(t.Context(), "u1")
	if err != nil || v.Kind != TodayDone || v.Lesson == nil || v.Lesson.ID != "f1" || v.Streak != 2 {
		t.Fatalf("los angeles = %+v, %v", v, err)
	}
	if d := e.dashboard(t, "u1"); d.Kind != TodayDone || d.Action != nil || d.Streak != 2 {
		t.Fatalf("dashboard = %+v", d)
	}
}
