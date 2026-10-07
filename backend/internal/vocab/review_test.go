package vocab

import (
	"errors"
	"testing"
	"time"
)

// dueCard stores a card for u1 that is due one hour before the env clock.
func (e *testEnv) dueCard(lemma string, s Schedule) Card {
	if s.Due.IsZero() {
		s.Due = e.clock.Now().Add(-time.Hour)
	}
	return e.repo.put(Card{
		UserID: "u1", Text: lemma, Lemma: lemma, MeaningVi: "nghĩa", LessonID: "lesson1", Source: SourceAI,
		CreatedAt: e.clock.Now().AddDate(0, 0, -3), Schedule: s,
	})
}

func TestReviewValidation(t *testing.T) {
	t.Parallel()
	e := newEnv()
	c := e.dueCard("go", Schedule{State: StateNew})
	cases := map[string]struct {
		in    ReviewInput
		field string
	}{
		"rating 0":   {ReviewInput{Rating: 0, Mode: "flip"}, "rating"},
		"rating 5":   {ReviewInput{Rating: 5, Mode: "flip"}, "rating"},
		"bad mode":   {ReviewInput{Rating: 3, Mode: "write"}, "mode"},
		"empty mode": {ReviewInput{Rating: 3}, "mode"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := e.svc.Review(t.Context(), "u1", c.ID, tc.in)
			var verr *ValidationError
			if !errors.As(err, &verr) || verr.Fields[tc.field] == "" {
				t.Fatalf("err = %v, want field %s", err, tc.field)
			}
		})
	}
}

func TestReviewNotFound(t *testing.T) {
	t.Parallel()
	e := newEnv()
	c := e.dueCard("go", Schedule{State: StateNew})
	if _, err := e.svc.Review(t.Context(), "u2", c.ID, ReviewInput{Rating: 3, Mode: "flip"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other user: %v", err)
	}
	if _, err := e.svc.Review(t.Context(), "u1", "nope", ReviewInput{Rating: 3, Mode: "flip"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown card: %v", err)
	}
}

func TestReviewEachRatingOfNewCard(t *testing.T) {
	t.Parallel()
	for _, r := range []Rating{Again, Hard, Good, Easy} {
		t.Run(string(rune('0'+r)), func(t *testing.T) {
			t.Parallel()
			e := newEnv()
			c := e.dueCard("go", Schedule{State: StateNew})
			before := c.Schedule
			now := e.clock.Now()

			got, err := e.svc.Review(t.Context(), "u1", c.ID, ReviewInput{Rating: int(r), Mode: "listen", Reps: 0})
			if err != nil {
				t.Fatal(err)
			}
			want := next(before, now, r)
			if got.Schedule != want {
				t.Fatalf("schedule = %+v, want %+v", got.Schedule, want)
			}
			if got.Intervals != intervalsFor(want, now) {
				t.Fatalf("intervals = %+v", got.Intervals)
			}
			stored, _ := e.repo.Get(t.Context(), "u1", c.ID)
			if stored.Schedule != want {
				t.Fatalf("stored = %+v", stored.Schedule)
			}
			logs := e.logs.forCard(c.ID)
			if len(logs) != 1 || logs[0].Rating != r || logs[0].Mode != ModeListen || !logs[0].ReviewedAt.Equal(now) ||
				logs[0].Before != before || logs[0].UserID != "u1" {
				t.Fatalf("logs = %+v", logs)
			}
		})
	}
}

func TestReviewForgottenCard(t *testing.T) {
	t.Parallel()
	e := newEnv()
	now := e.clock.Now()
	review := next(Schedule{State: StateNew}, now.AddDate(0, 0, -20), Easy)
	review.Due = now.Add(-time.Hour)
	c := e.dueCard("go", review)

	got, err := e.svc.Review(t.Context(), "u1", c.ID, ReviewInput{Rating: int(Again), Mode: "flip", Reps: review.Reps})
	if err != nil {
		t.Fatal(err)
	}
	if got.Schedule.State != StateRelearning || got.Schedule.Lapses != review.Lapses+1 {
		t.Fatalf("schedule = %+v", got.Schedule)
	}
}

func TestReviewConflict(t *testing.T) {
	t.Parallel()
	e := newEnv()
	c := e.dueCard("go", Schedule{State: StateNew})
	in := ReviewInput{Rating: int(Good), Mode: "flip", Reps: 0}
	if _, err := e.svc.Review(t.Context(), "u1", c.ID, in); err != nil {
		t.Fatal(err)
	}

	// The same answer sent again (e.g. a retry after a lost response) is not counted twice.
	_, err := e.svc.Review(t.Context(), "u1", c.ID, in)
	var conflict *ConflictError
	if !errors.As(err, &conflict) || conflict.Card.Schedule.Reps != 1 || conflict.Card.ID != c.ID {
		t.Fatalf("err = %v", err)
	}
	if n := len(e.logs.forCard(c.ID)); n != 1 {
		t.Fatalf("logs = %d, want 1", n)
	}
}

func TestReviewLegacyCard(t *testing.T) {
	t.Parallel()
	e := newEnv()
	// Saved two days ago before F5: no schedule stored.
	c := e.repo.put(Card{
		UserID: "u1", Text: "went", Lemma: "go", MeaningVi: "đi", LessonID: "lesson1", Source: SourceAI,
		CreatedAt: e.clock.Now().AddDate(0, 0, -2),
	})

	got, err := e.svc.Review(t.Context(), "u1", c.ID, ReviewInput{Rating: int(Good), Mode: "flip", Reps: 0})
	if err != nil || got.Schedule.Reps != 1 || got.Schedule.State != StateLearning {
		t.Fatalf("legacy review = %+v, %v", got.Schedule, err)
	}
	if logs := e.logs.forCard(c.ID); len(logs) != 1 || logs[0].Before.State != StateNew || logs[0].Before.Due.IsZero() {
		t.Fatalf("logs = %+v", logs)
	}
}

func TestDue(t *testing.T) {
	t.Parallel()
	e := newEnv() // 2026-09-29 15:00 in Viet Nam
	now := e.clock.Now()
	older := e.dueCard("older", Schedule{Due: now.Add(-48 * time.Hour), State: StateReview, Reps: 3})
	newer := e.dueCard("newer", Schedule{Due: now.Add(-time.Minute), State: StateLearning, Reps: 1})
	e.dueCard("later", Schedule{Due: now.Add(3 * time.Hour), State: StateReview, Reps: 2})
	// Legacy cards: saved yesterday (local) is due, saved this morning (local) is not.
	yesterday := e.repo.put(Card{
		UserID: "u1", Text: "y", Lemma: "y", MeaningVi: "m", Source: SourceManual,
		CreatedAt: time.Date(2026, 9, 28, 22, 0, 0, 0, hcm),
	})
	e.repo.put(Card{
		UserID: "u1", Text: "t", Lemma: "t", MeaningVi: "m", Source: SourceManual,
		CreatedAt: time.Date(2026, 9, 29, 0, 30, 0, 0, hcm),
	})
	// Another learner's due card.
	e.repo.put(Card{
		UserID: "u2", Text: "x", Lemma: "x", MeaningVi: "m", Source: SourceManual,
		CreatedAt: now.AddDate(0, 0, -5), Schedule: Schedule{Due: now.Add(-time.Hour)},
	})

	list, err := e.svc.Due(t.Context(), "u1", 50)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, c := range list.Cards {
		got = append(got, c.ID)
	}
	want := []string{yesterday.ID, older.ID, newer.ID}
	if len(got) != 3 || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] || list.Total != 3 {
		t.Fatalf("due = %v (total %d), want %v", got, list.Total, want)
	}
	// Legacy cards show their effective schedule and intervals.
	if first := list.Cards[0]; first.Schedule.Due.IsZero() || first.Intervals.Good != 10*time.Minute {
		t.Fatalf("legacy due card = %+v", first)
	}
	// The earliest not-yet-due card is "later" (in 3 h), before the legacy card due tomorrow.
	if !list.NextDue.Equal(now.Add(3 * time.Hour)) {
		t.Fatalf("nextDue = %v", list.NextDue)
	}

	limited, _ := e.svc.Due(t.Context(), "u1", 2)
	if len(limited.Cards) != 2 || limited.Total != 3 {
		t.Fatalf("limited = %d cards, total %d", len(limited.Cards), limited.Total)
	}
	for _, bad := range []int{0, 201} {
		var verr *ValidationError
		if _, err := e.svc.Due(t.Context(), "u1", bad); !errors.As(err, &verr) || verr.Fields["limit"] == "" {
			t.Fatalf("limit %d: %v", bad, err)
		}
	}
}

func TestDueEmpty(t *testing.T) {
	t.Parallel()
	e := newEnv()
	list, err := e.svc.Due(t.Context(), "u1", 50)
	if err != nil || len(list.Cards) != 0 || list.Total != 0 || !list.NextDue.IsZero() {
		t.Fatalf("empty = %+v, %v", list, err)
	}

	// A card saved today is due tomorrow at 00:00 local.
	if _, err := e.svc.Save(t.Context(), "u1", validInput()); err != nil {
		t.Fatal(err)
	}
	list, _ = e.svc.Due(t.Context(), "u1", 50)
	if len(list.Cards) != 0 || !list.NextDue.Equal(time.Date(2026, 9, 30, 0, 0, 0, 0, hcm)) {
		t.Fatalf("after save = %+v", list)
	}
}

func TestReviewContext(t *testing.T) {
	t.Parallel()
	e := newEnv()
	a := e.dueCard("a", Schedule{State: StateNew})
	b := e.dueCard("b", Schedule{State: StateNew})
	c := e.dueCard("c", Schedule{State: StateNew})
	start := e.clock.Now()

	if _, err := e.svc.Review(t.Context(), "u1", a.ID, ReviewInput{Rating: 3, Mode: "flip", Context: "daily"}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.Review(t.Context(), "u1", b.ID, ReviewInput{Rating: 3, Mode: "flip"}); err != nil {
		t.Fatal(err)
	}
	if logs := e.logs.forCard(b.ID); logs[0].Context != ContextFree {
		t.Fatalf("default context = %q", logs[0].Context)
	}
	var verr *ValidationError
	if _, err := e.svc.Review(t.Context(), "u1", c.ID, ReviewInput{Rating: 3, Mode: "flip", Context: "weekly"}); !errors.As(err, &verr) || verr.Fields["context"] == "" {
		t.Fatalf("bad context: %v", err)
	}

	if n, err := e.svc.ReviewedSince(t.Context(), "u1", ContextDaily, start); err != nil || n != 1 {
		t.Fatalf("daily since start = %d, %v", n, err)
	}
	if n, _ := e.svc.ReviewedSince(t.Context(), "u1", ContextDaily, start.Add(time.Minute)); n != 0 {
		t.Fatalf("daily later = %d", n)
	}
	if n, _ := e.svc.ReviewedSince(t.Context(), "u2", ContextDaily, start); n != 0 {
		t.Fatalf("other user = %d", n)
	}
}

func TestDueBeforeAndCount(t *testing.T) {
	t.Parallel()
	e := newEnv() // 2026-09-29 15:00 in Viet Nam
	now := e.clock.Now()
	cutoff := time.Date(2026, 10, 1, 0, 0, 0, 0, hcm)   // 00:00 the day after tomorrow
	tomorrow := time.Date(2026, 9, 30, 0, 0, 0, 0, hcm) // cards without a schedule saved before it are due by then
	e.dueCard("overdue", Schedule{Due: now.Add(-48 * time.Hour), State: StateReview, Reps: 3})
	e.dueCard("tomorrow-late", Schedule{Due: cutoff.Add(-time.Minute), State: StateReview, Reps: 3})
	e.dueCard("at-cutoff", Schedule{Due: cutoff, State: StateReview, Reps: 3})
	e.repo.put(Card{UserID: "u1", Text: "l", Lemma: "l", MeaningVi: "m", Source: SourceManual, CreatedAt: now})
	e.repo.put(Card{UserID: "u1", Text: "n", Lemma: "n", MeaningVi: "m", Source: SourceManual, CreatedAt: tomorrow})
	e.repo.put(Card{
		UserID: "u2", Text: "x", Lemma: "x", MeaningVi: "m", Source: SourceManual,
		CreatedAt: now.AddDate(0, 0, -5), Schedule: Schedule{Due: now.Add(-time.Hour)},
	})

	if n, err := e.svc.DueBefore(t.Context(), "u1", cutoff, tomorrow); err != nil || n != 3 {
		t.Fatalf("due before = %d, %v; want 3", n, err)
	}
	if n, err := e.svc.Count(t.Context(), "u1", nil); err != nil || n != 5 {
		t.Fatalf("count = %d, %v; want 5", n, err)
	}
	if n, _ := e.svc.Count(t.Context(), "nobody", nil); n != 0 {
		t.Fatalf("count nobody = %d", n)
	}
	since := now.Add(-time.Hour) // only "l" and "n" were saved since
	if n, err := e.svc.Count(t.Context(), "u1", &since); err != nil || n != 2 {
		t.Fatalf("count since = %d, %v; want 2", n, err)
	}
}
