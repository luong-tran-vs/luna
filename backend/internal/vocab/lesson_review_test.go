package vocab

import (
	"errors"
	"testing"
	"time"
)

func TestLessonStats(t *testing.T) {
	t.Parallel()
	e := newEnv()
	now := e.clock.Now()
	put := func(lemma, lesson string, s Schedule) {
		e.repo.put(Card{
			UserID: "u1", Text: lemma, Lemma: lemma, MeaningVi: lemma, LessonID: lesson,
			Source: SourceAI, CreatedAt: now.AddDate(0, 0, -10), Schedule: s,
		})
	}
	put("work", "L1", Schedule{Due: now.Add(-time.Hour), State: StateLearning, Reps: 1, LastReview: now.AddDate(0, 0, -3)})
	put("desk", "L1", Schedule{Due: now.AddDate(0, 0, 4), State: StateReview, Reps: 3, LastReview: now.AddDate(0, 0, -1)})
	put("boss", "L1", Schedule{Due: now.AddDate(0, 0, 2), State: StateReview, Reps: 2, LastReview: now.AddDate(0, 0, -2)})
	put("hello", "L2", Schedule{}) // never reviewed: due the day after it was saved
	put("mine", "", Schedule{Due: now.Add(-time.Hour), State: StateLearning, Reps: 1})
	e.repo.put(Card{UserID: "u2", Text: "x", Lemma: "x", MeaningVi: "x", LessonID: "L1", Source: SourceAI, CreatedAt: now})

	stats, err := e.svc.LessonStats(t.Context(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if len(stats) != 2 {
		t.Fatalf("stats = %+v, want 2 lessons", stats)
	}
	byID := map[string]LessonStats{}
	for _, s := range stats {
		byID[s.LessonID] = s
	}
	l1 := byID["L1"]
	if l1.Cards != 3 || l1.Due != 1 || l1.Remembered != 2 {
		t.Fatalf("L1 = %+v, want 3 cards, 1 due, 2 remembered", l1)
	}
	if !l1.LastReview.Equal(now.AddDate(0, 0, -1)) || !l1.NextDue.Equal(now.AddDate(0, 0, 2)) {
		t.Fatalf("L1 last/next = %v/%v", l1.LastReview, l1.NextDue)
	}
	l2 := byID["L2"]
	if l2.Cards != 1 || l2.Due != 1 || !l2.LastReview.IsZero() || !l2.NextDue.IsZero() {
		t.Fatalf("L2 = %+v, want 1 due card never reviewed", l2)
	}
}

func TestSaveLessonResult(t *testing.T) {
	t.Parallel()
	e := newEnv()
	if _, err := e.svc.SaveLessonResult(t.Context(), "u1", "lesson1", 2, 4, []string{"v:go", "v:go", "f:0"}); err != nil {
		t.Fatal(err)
	}
	// A later round replaces the first one.
	e.clock.t = e.clock.t.Add(time.Hour)
	r, err := e.svc.SaveLessonResult(t.Context(), "u1", "lesson1", 3, 4, []string{" t:1 "})
	if err != nil || r.Correct != 3 || len(r.Wrong) != 1 || r.Wrong[0] != "t:1" {
		t.Fatalf("second round = %+v, %v", r, err)
	}
	stats, _ := e.svc.LessonStats(t.Context(), "u1")
	if len(stats) != 1 || stats[0].LessonID != "lesson1" || stats[0].Cards != 0 || stats[0].Result == nil ||
		stats[0].Result.Correct != 3 || !stats[0].Result.ReviewedAt.Equal(e.clock.t) {
		t.Fatalf("stats = %+v", stats)
	}
	if other, _ := e.svc.LessonStats(t.Context(), "u2"); len(other) != 0 {
		t.Fatalf("another user sees %+v", other)
	}

	for _, bad := range []struct {
		correct, total int
		wrong          []string
	}{{0, 0, nil}, {5, 4, nil}, {1, 2, []string{"a", "b"}}, {0, 1, []string{""}}, {0, 101, nil}} {
		var verr *ValidationError
		if _, err := e.svc.SaveLessonResult(t.Context(), "u1", "lesson1", bad.correct, bad.total, bad.wrong); !errors.As(err, &verr) {
			t.Fatalf("%+v: err = %v", bad, err)
		}
	}
	if _, err := e.svc.SaveLessonResult(t.Context(), "u1", "missing", 1, 1, nil); !errors.Is(err, ErrLessonNotFound) {
		t.Fatalf("missing lesson: %v", err)
	}
}
