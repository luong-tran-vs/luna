package mysql

import (
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/lesson"
)

func readingAnswer(user, les string, version, q, choice int, correct bool) lesson.Answer {
	return lesson.Answer{
		UserID: user, LessonID: les, QuizVersion: version, QuestionIndex: q, Choice: choice, Correct: correct,
		AnsweredAt: time.Date(2026, 9, 30, 8, 0, 0, 123000, time.UTC),
	}
}

func TestReadingAnswersUpsertListAndDelete(t *testing.T) {
	t.Parallel()
	r := NewReadingAnswers(testDB(t))
	ctx := t.Context()
	u, l := newID(), newID()

	for _, q := range []int{2, 0, 1} {
		if err := r.Upsert(ctx, readingAnswer(u, l, 1, q, q, q != 1)); err != nil {
			t.Fatalf("insert %d: %v", q, err)
		}
	}
	got, err := r.List(ctx, u, l, 1)
	if err != nil || len(got) != 3 {
		t.Fatalf("list = %v, %v", got, err)
	}
	for i, a := range got {
		want := readingAnswer(u, l, 1, i, i, i != 1)
		if a.QuestionIndex != i || a.Choice != want.Choice || a.Correct != want.Correct || a.UserID != u || a.LessonID != l ||
			a.QuizVersion != 1 || !a.AnsweredAt.Equal(want.AnsweredAt) {
			t.Fatalf("answer %d = %+v", i, a)
		}
	}

	// A second answer replaces the first, with its new time.
	again := readingAnswer(u, l, 1, 1, 3, true)
	again.AnsweredAt = again.AnsweredAt.Add(time.Hour)
	if err := r.Upsert(ctx, again); err != nil {
		t.Fatalf("second answer: %v", err)
	}
	got, _ = r.List(ctx, u, l, 1)
	if len(got) != 3 || got[1].Choice != 3 || !got[1].Correct || !got[1].AnsweredAt.Equal(again.AnsweredAt) {
		t.Fatalf("after replace = %+v", got)
	}

	// Another version, user or lesson answers the same question again.
	other, otherLesson := newID(), newID()
	for _, a := range []lesson.Answer{
		readingAnswer(u, l, 2, 1, 0, true), readingAnswer(other, l, 1, 1, 0, true), readingAnswer(u, otherLesson, 1, 1, 0, true),
	} {
		if err := r.Upsert(ctx, a); err != nil {
			t.Fatalf("insert %+v: %v", a, err)
		}
	}
	if got, _ := r.List(ctx, u, l, 2); len(got) != 1 {
		t.Fatalf("version 2 = %v", got)
	}

	// Delete removes only the learner's answers to that version of that lesson.
	if err := r.Delete(ctx, u, l, 1); err != nil {
		t.Fatal(err)
	}
	if got, err := r.List(ctx, u, l, 1); err != nil || len(got) != 0 {
		t.Fatalf("after delete = %v, %v", got, err)
	}
	for _, c := range []struct {
		user, lesson string
		version      int
	}{{u, l, 2}, {other, l, 1}, {u, otherLesson, 1}} {
		if got, _ := r.List(ctx, c.user, c.lesson, c.version); len(got) != 1 {
			t.Fatalf("kept %+v = %v", c, got)
		}
	}
	if err := r.Delete(ctx, u, l, 1); err != nil {
		t.Fatalf("delete again: %v", err)
	}
}

func TestReadingAnswersTotalsCountLatest(t *testing.T) {
	t.Parallel()
	r := NewReadingAnswers(testDB(t))
	ctx := t.Context()
	u, l := newID(), newID()
	_ = r.Upsert(ctx, readingAnswer(u, l, 1, 0, 1, false))
	_ = r.Upsert(ctx, readingAnswer(u, l, 1, 0, 0, true))
	if a, c, err := r.Totals(ctx, u, nil); err != nil || a != 1 || c != 1 {
		t.Fatalf("totals = %d, %d, %v", a, c, err)
	}
}

func TestReadingAnswersListEmpty(t *testing.T) {
	t.Parallel()
	r := NewReadingAnswers(testDB(t))
	got, err := r.List(t.Context(), newID(), newID(), 1)
	if err != nil || len(got) != 0 {
		t.Fatalf("list = %v, %v", got, err)
	}
}

func TestReadingAnswersTotals(t *testing.T) {
	t.Parallel()
	r := NewReadingAnswers(testDB(t))
	ctx := t.Context()
	u := newID()
	if a, c, err := r.Totals(ctx, u, nil); err != nil || a != 0 || c != 0 {
		t.Fatalf("empty totals = %d, %d, %v", a, c, err)
	}
	l := newID()
	_ = r.Upsert(ctx, readingAnswer(u, l, 1, 0, 0, true))
	_ = r.Upsert(ctx, readingAnswer(u, l, 1, 1, 0, false))
	_ = r.Upsert(ctx, readingAnswer(u, l, 2, 0, 0, true)) // another version still counts
	_ = r.Upsert(ctx, readingAnswer(newID(), l, 1, 0, 0, true))
	a, c, err := r.Totals(ctx, u, nil)
	if err != nil || a != 3 || c != 2 {
		t.Fatalf("totals = %d, %d, %v", a, c, err)
	}
	// An answer given later counts alone since then.
	late := readingAnswer(u, l, 2, 1, 0, true)
	late.AnsweredAt = late.AnsweredAt.Add(24 * time.Hour)
	_ = r.Upsert(ctx, late)
	if a, c, err := r.Totals(ctx, u, &late.AnsweredAt); err != nil || a != 1 || c != 1 {
		t.Fatalf("totals since = %d, %d, %v", a, c, err)
	}
}
