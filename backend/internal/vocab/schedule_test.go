package vocab

import (
	"testing"
	"time"

	fsrs "github.com/open-spaced-repetition/go-fsrs/v3"
)

func mustLoad(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

func TestFirstDue(t *testing.T) {
	t.Parallel()
	hcm := mustLoad(t, "Asia/Ho_Chi_Minh")
	ny := mustLoad(t, "America/New_York")
	paris := mustLoad(t, "Europe/Paris")

	cases := []struct {
		name  string
		saved time.Time
		loc   *time.Location
		want  time.Time
	}{
		{"23:59 local", time.Date(2026, 9, 30, 23, 59, 0, 0, hcm), hcm, time.Date(2026, 10, 1, 0, 0, 0, 0, hcm)},
		{"00:01 local", time.Date(2026, 10, 1, 0, 1, 0, 0, hcm), hcm, time.Date(2026, 10, 2, 0, 0, 0, 0, hcm)},
		// 17:30 UTC is already the next day in Viet Nam.
		{"utc evening", time.Date(2026, 9, 30, 17, 30, 0, 0, time.UTC), hcm, time.Date(2026, 10, 2, 0, 0, 0, 0, hcm)},
		{"new york", time.Date(2026, 9, 30, 2, 0, 0, 0, time.UTC), ny, time.Date(2026, 9, 30, 0, 0, 0, 0, ny)},
		{"dst change", time.Date(2026, 3, 28, 22, 0, 0, 0, paris), paris, time.Date(2026, 3, 29, 0, 0, 0, 0, paris)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := FirstDue(tc.saved, tc.loc); !got.Equal(tc.want) {
				t.Fatalf("FirstDue = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestFSRSRoundTrip(t *testing.T) {
	t.Parallel()
	s := Schedule{
		Due: time.Date(2026, 10, 3, 1, 0, 0, 0, time.UTC), Stability: 3.2, Difficulty: 5.1, ElapsedDays: 2,
		ScheduledDays: 3, Reps: 4, Lapses: 1, State: StateReview, LastReview: time.Date(2026, 9, 30, 1, 0, 0, 0, time.UTC),
	}
	if got := fromFSRS(toFSRS(s)); got != s {
		t.Fatalf("round trip = %+v, want %+v", got, s)
	}
}

func TestEffectiveScheduleOfLegacyCard(t *testing.T) {
	t.Parallel()
	hcm := mustLoad(t, "Asia/Ho_Chi_Minh")
	c := Card{CreatedAt: time.Date(2026, 9, 29, 10, 0, 0, 0, hcm)}
	s := effective(c, hcm)
	if s.State != StateNew || s.Reps != 0 || !s.Due.Equal(time.Date(2026, 9, 30, 0, 0, 0, 0, hcm)) {
		t.Fatalf("legacy schedule = %+v", s)
	}

	c.Schedule = Schedule{Due: time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC), Reps: 2, State: StateReview}
	if got := effective(c, hcm); got != c.Schedule {
		t.Fatalf("stored schedule changed: %+v", got)
	}
}

func TestIntervalsOfNewCard(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	iv := intervalsFor(Schedule{State: StateNew, Due: now}, now)
	if iv.Again != time.Minute || iv.Hard != 5*time.Minute || iv.Good != 10*time.Minute || iv.Easy < 24*time.Hour {
		t.Fatalf("intervals = %+v", iv)
	}
}

func TestNextMatchesFSRS(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	ref := fsrs.NewFSRS(fsrs.DefaultParam())
	newCard := Schedule{State: StateNew, Due: now}

	wantState := map[Rating]State{Again: StateLearning, Hard: StateLearning, Good: StateLearning, Easy: StateReview}
	for r, st := range wantState {
		got := next(newCard, now, r)
		want := fromFSRS(ref.Next(toFSRS(newCard), now, fsrs.Rating(r)).Card)
		if got != want {
			t.Fatalf("rating %d: %+v, want %+v", r, got, want)
		}
		if got.State != st || got.Reps != 1 || !got.LastReview.Equal(now) {
			t.Fatalf("rating %d: state %d reps %d", r, got.State, got.Reps)
		}
	}

	// A card in review that is forgotten goes to relearning and counts a lapse.
	review := next(newCard, now, Easy)
	later := review.Due.Add(time.Hour)
	forgot := next(review, later, Again)
	if forgot.State != StateRelearning || forgot.Lapses != review.Lapses+1 || !forgot.Due.After(later) {
		t.Fatalf("forgot = %+v", forgot)
	}
	remembered := next(review, later, Good)
	if remembered.State != StateReview || remembered.ScheduledDays <= review.ScheduledDays {
		t.Fatalf("remembered = %+v (before %+v)", remembered, review)
	}
}
