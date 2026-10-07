package progress

import (
	"testing"
	"time"
)

func load(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

func TestDayKey(t *testing.T) {
	t.Parallel()
	hcm := load(t, "Asia/Ho_Chi_Minh")
	ny := load(t, "America/New_York")
	cases := []struct {
		name string
		now  time.Time
		loc  *time.Location
		want string
	}{
		{"23:59 Viet Nam", time.Date(2026, 9, 30, 23, 59, 0, 0, hcm), hcm, "2026-09-30"},
		{"00:01 Viet Nam", time.Date(2026, 10, 1, 0, 1, 0, 0, hcm), hcm, "2026-10-01"},
		// 17:30 UTC is already the next day in Viet Nam but the same day in New York.
		{"same instant hcm", time.Date(2026, 9, 30, 17, 30, 0, 0, time.UTC), hcm, "2026-10-01"},
		{"same instant ny", time.Date(2026, 9, 30, 17, 30, 0, 0, time.UTC), ny, "2026-09-30"},
	}
	for _, tc := range cases {
		if got := DayKey(tc.now, tc.loc); got != tc.want {
			t.Errorf("%s: %s, want %s", tc.name, got, tc.want)
		}
	}
}

func TestNextAndPrevDayKey(t *testing.T) {
	t.Parallel()
	cases := map[string]string{"2026-09-30": "2026-10-01", "2026-12-31": "2027-01-01", "2028-02-28": "2028-02-29"}
	for in, want := range cases {
		if got := NextDayKey(in); got != want {
			t.Errorf("NextDayKey(%s) = %s, want %s", in, got, want)
		}
		if got := PrevDayKey(want); got != in {
			t.Errorf("PrevDayKey(%s) = %s, want %s", want, got, in)
		}
	}
}

func TestEffectiveGoal(t *testing.T) {
	t.Parallel()
	if _, ok := EffectiveGoal(nil); ok {
		t.Fatal("goal without goals")
	}
	goals := []Goal{{TopicID: "a", Status: GoalPaused}, {TopicID: "b", Status: GoalActive}}
	if g, ok := EffectiveGoal(goals); !ok || g.TopicID != "b" {
		t.Fatalf("goal = %+v, %v", g, ok)
	}
	if _, ok := EffectiveGoal(goals[:1]); ok {
		t.Fatal("paused goal is not effective")
	}
}

func TestCurrentLesson(t *testing.T) {
	t.Parallel()
	goal := &Goal{TopicID: "family", Status: GoalActive}
	roadmap := []string{"l1", "l2", "l3"}
	cases := []struct {
		name      string
		goal      *Goal
		roadmap   []string
		completed map[string]bool
		want      StudyState
	}{
		{"no goal", nil, nil, nil, StudyState{Kind: StudyNoGoal}},
		{"first lesson", goal, roadmap, nil, StudyState{Kind: StudyStudying, LessonID: "l1"}},
		// Returning to a topic, or right after a lesson: the first unfinished lesson.
		{"resume", goal, roadmap, map[string]bool{"l1": true, "l2": true}, StudyState{Kind: StudyStudying, LessonID: "l3"}},
		{"roadmap reordered", goal, []string{"l3", "l1", "l2"}, map[string]bool{"l1": true}, StudyState{Kind: StudyStudying, LessonID: "l3"}},
		{"all done", goal, roadmap, map[string]bool{"l1": true, "l2": true, "l3": true}, StudyState{Kind: StudyNoNewLesson}},
		{"empty roadmap", goal, nil, nil, StudyState{Kind: StudyNoNewLesson}},
	}
	for _, tc := range cases {
		if got := CurrentLesson(tc.goal, tc.roadmap, tc.completed); got != tc.want {
			t.Errorf("%s: %+v, want %+v", tc.name, got, tc.want)
		}
	}
}

func TestStreak(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		days []string
		want int
	}{
		{"none", nil, 0},
		{"today only", []string{"2026-09-30"}, 1},
		{"ends today", []string{"2026-09-28", "2026-09-29", "2026-09-30"}, 3},
		{"today not done yet", []string{"2026-09-27", "2026-09-28", "2026-09-29"}, 3},
		{"one full day missed is forgiven", []string{"2026-09-27", "2026-09-28"}, 2},
		{"two full days missed", []string{"2026-09-26", "2026-09-27"}, 0},
		{"three days missed", []string{"2026-09-25", "2026-09-26"}, 0},
		{"one missed day in the middle is forgiven", []string{"2026-09-25", "2026-09-27", "2026-09-28", "2026-09-29", "2026-09-30"}, 5},
		{"two missed days in the middle end it", []string{"2026-09-24", "2026-09-27", "2026-09-28", "2026-09-29", "2026-09-30"}, 4},
		{
			"a second forgiven day needs seven study days",
			[]string{"2026-09-20", "2026-09-22", "2026-09-24", "2026-09-26", "2026-09-27", "2026-09-28", "2026-09-29", "2026-09-30"},
			6,
		},
		{
			"a second forgiven day after seven study days",
			[]string{"2026-09-19", "2026-09-21", "2026-09-22", "2026-09-23", "2026-09-24", "2026-09-25", "2026-09-26", "2026-09-27", "2026-09-28", "2026-09-30"},
			10,
		},
		{"duplicates and order", []string{"2026-09-30", "2026-09-29", "2026-09-30"}, 2},
		{"across a month", []string{"2026-08-31", "2026-09-01"}, 0},
	}
	for _, tc := range cases {
		if got := Streak(tc.days, "2026-09-30"); got != tc.want {
			t.Errorf("%s: %d, want %d", tc.name, got, tc.want)
		}
	}
	if got := Streak([]string{"2026-08-31", "2026-09-01"}, "2026-09-01"); got != 2 {
		t.Errorf("month boundary: %d", got)
	}
}

func TestNextStep(t *testing.T) {
	t.Parallel()
	cases := []struct {
		done map[Step]bool
		want Step
	}{
		{nil, StepRead},
		{map[Step]bool{StepRead: true}, StepListen},
		{map[Step]bool{StepRead: true, StepListen: true}, StepWrite},
		{map[Step]bool{StepRead: true, StepListen: true, StepWrite: true}, StepDone},
		{map[Step]bool{StepListen: true}, StepRead},
		// The review step of progress saved before 2026-10-02 is ignored.
		{map[Step]bool{"review": true}, StepRead},
	}
	for _, tc := range cases {
		if got := NextStep(tc.done); got != tc.want {
			t.Errorf("NextStep(%v) = %s, want %s", tc.done, got, tc.want)
		}
	}
}

func TestPeriodStart(t *testing.T) {
	t.Parallel()
	hcm := load(t, "Asia/Ho_Chi_Minh")
	ny := load(t, "America/New_York")
	for _, tt := range []struct {
		name   string
		period Period
		now    time.Time
		loc    *time.Location
		want   time.Time
	}{
		{"week midweek", PeriodWeek, time.Date(2026, 9, 30, 10, 0, 0, 0, hcm), hcm, time.Date(2026, 9, 28, 0, 0, 0, 0, hcm)},
		{"week on Monday 0:00", PeriodWeek, time.Date(2026, 9, 28, 0, 0, 0, 0, hcm), hcm, time.Date(2026, 9, 28, 0, 0, 0, 0, hcm)},
		{"week on Sunday night", PeriodWeek, time.Date(2026, 10, 4, 23, 59, 0, 0, hcm), hcm, time.Date(2026, 9, 28, 0, 0, 0, 0, hcm)},
		{"week across a month", PeriodWeek, time.Date(2026, 10, 1, 8, 0, 0, 0, hcm), hcm, time.Date(2026, 9, 28, 0, 0, 0, 0, hcm)},
		{"month", PeriodMonth, time.Date(2026, 9, 30, 10, 0, 0, 0, hcm), hcm, time.Date(2026, 9, 1, 0, 0, 0, 0, hcm)},
		{"month on the 1st", PeriodMonth, time.Date(2026, 10, 1, 0, 0, 0, 0, hcm), hcm, time.Date(2026, 10, 1, 0, 0, 0, 0, hcm)},
		// 2026-09-30 20:00 UTC is already Thursday 2026-10-01 03:00 in Viet Nam.
		{"month in the learner's zone", PeriodMonth, time.Date(2026, 9, 30, 20, 0, 0, 0, time.UTC), hcm, time.Date(2026, 10, 1, 0, 0, 0, 0, hcm)},
		// 2026-09-28 02:00 UTC is still Sunday 2026-09-27 in New York.
		{"week in the learner's zone", PeriodWeek, time.Date(2026, 9, 28, 2, 0, 0, 0, time.UTC), ny, time.Date(2026, 9, 21, 0, 0, 0, 0, ny)},
	} {
		got := PeriodStart(tt.period, tt.now, tt.loc)
		if got == nil || !got.Equal(tt.want) {
			t.Errorf("%s: start = %v, want %v", tt.name, got, tt.want)
		}
	}
	if got := PeriodStart(PeriodAll, time.Now(), hcm); got != nil {
		t.Errorf("all: start = %v, want nil", got)
	}
}
