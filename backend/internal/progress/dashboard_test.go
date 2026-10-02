package progress

import (
	"testing"
	"time"
)

func TestTomorrowCutoff(t *testing.T) {
	t.Parallel()
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]struct {
		now  time.Time
		loc  *time.Location
		want time.Time
	}{
		"just after midnight":  {time.Date(2026, 9, 30, 0, 1, 0, 0, hcm), hcm, time.Date(2026, 10, 2, 0, 0, 0, 0, hcm)},
		"just before midnight": {time.Date(2026, 9, 30, 23, 59, 0, 0, hcm), hcm, time.Date(2026, 10, 2, 0, 0, 0, 0, hcm)},
		"end of month":         {time.Date(2026, 12, 31, 8, 0, 0, 0, hcm), hcm, time.Date(2027, 1, 2, 0, 0, 0, 0, hcm)},
		// Clocks go back on 2026-11-01 in New York: the cutoff is still local midnight.
		"across DST": {time.Date(2026, 10, 31, 12, 0, 0, 0, ny), ny, time.Date(2026, 11, 2, 0, 0, 0, 0, ny)},
		// 2026-09-30 20:00 UTC is already October 1 in Viet Nam.
		"other timezone": {time.Date(2026, 9, 30, 20, 0, 0, 0, time.UTC), hcm, time.Date(2026, 10, 3, 0, 0, 0, 0, hcm)},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := TomorrowCutoff(tc.now, tc.loc); !got.Equal(tc.want) {
				t.Fatalf("cutoff = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAction(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		done    map[Step]bool
		current Step
		want    *Action
	}{
		"not started": {map[Step]bool{}, StepRead, &Action{Kind: ActionStart, Step: StepRead}},
		// Progress saved before 2026-10-02 may hold the dropped review step.
		"old review done": {map[Step]bool{"review": true}, StepRead, &Action{Kind: ActionStart, Step: StepRead}},
		"read done":       {map[Step]bool{StepRead: true}, StepListen, &Action{Kind: ActionContinue, Step: StepListen}},
		"all done":        {map[Step]bool{StepRead: true, StepListen: true, StepWrite: true}, StepDone, nil},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := NextAction(tc.done, tc.current)
			if (got == nil) != (tc.want == nil) || got != nil && *got != *tc.want {
				t.Fatalf("action = %+v, want %+v", got, tc.want)
			}
		})
	}
}
