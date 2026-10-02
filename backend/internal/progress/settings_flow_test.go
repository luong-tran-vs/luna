package progress

import (
	"testing"
	"time"
)

// --- F12: the timezone comes from the settings ---

func TestTimezoneChangeKeepsProgress(t *testing.T) {
	t.Parallel()
	e := newStudyEnv() // 2026-09-30 10:00 in Viet Nam = 04:00 in London
	e.setGoal(t, "family")
	if err := e.svc.SetPosition(t.Context(), "u1", "f1", StepRead, 2); err != nil {
		t.Fatal(err)
	}

	london, err := time.LoadLocation("Europe/London")
	if err != nil {
		t.Fatal(err)
	}
	e.zones["u1"] = london
	v, err := e.svc.LessonStudy(t.Context(), "u1", "f1")
	if err != nil || v.Status != LessonStudying || v.CurrentStep != StepRead || v.SentenceIndex != 2 {
		t.Fatalf("london = %+v, %v", v, err)
	}
}

func TestStreakDayFollowsTheTimezone(t *testing.T) {
	t.Parallel()
	e := newStudyEnv()
	e.clock.set(time.Date(2026, 9, 30, 23, 0, 0, 0, hcm)) // 2026-09-30 in Viet Nam
	e.days.put("u1", "2026-09-30")
	e.setGoal(t, "family")

	e.zones["u1"] = time.FixedZone("UTC+10", 10*3600) // already 2026-10-01 02:00
	if v := e.studyLesson(t); v.Streak != 2 {
		t.Fatalf("finished = %+v", v)
	}
	if keys, _ := e.days.CompletedKeys(t.Context(), "u1"); len(keys) != 2 {
		t.Fatalf("days = %v", keys)
	}
}
