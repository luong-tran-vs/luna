package mysql

import (
	"errors"
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

func TestReadingAnswersInsertListAndFinal(t *testing.T) {
	t.Parallel()
	r := NewReadingAnswers(testDB(t))
	ctx := t.Context()
	u, l := newID(), newID()

	for _, q := range []int{2, 0, 1} {
		if err := r.Insert(ctx, readingAnswer(u, l, 1, q, q, q != 1)); err != nil {
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

	// A second answer is refused and returns the first.
	err = r.Insert(ctx, readingAnswer(u, l, 1, 1, 3, true))
	var already *lesson.AlreadyAnsweredError
	if !errors.As(err, &already) || !errors.Is(err, lesson.ErrAlreadyAnswered) {
		t.Fatalf("second insert = %v", err)
	}
	if already.Answer.Choice != 1 || already.Answer.Correct {
		t.Fatalf("first answer = %+v", already.Answer)
	}
	if got, _ := r.List(ctx, u, l, 1); got[1].Choice != 1 || got[1].Correct {
		t.Fatalf("answer was overwritten: %+v", got[1])
	}

	// Another version, user or lesson answers the same question again.
	for _, a := range []lesson.Answer{
		readingAnswer(u, l, 2, 1, 0, true), readingAnswer(newID(), l, 1, 1, 0, true), readingAnswer(u, newID(), 1, 1, 0, true),
	} {
		if err := r.Insert(ctx, a); err != nil {
			t.Fatalf("insert %+v: %v", a, err)
		}
	}
	if got, _ := r.List(ctx, u, l, 2); len(got) != 1 {
		t.Fatalf("version 2 = %v", got)
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
	if a, c, err := r.Totals(ctx, u); err != nil || a != 0 || c != 0 {
		t.Fatalf("empty totals = %d, %d, %v", a, c, err)
	}
	l := newID()
	_ = r.Insert(ctx, readingAnswer(u, l, 1, 0, 0, true))
	_ = r.Insert(ctx, readingAnswer(u, l, 1, 1, 0, false))
	_ = r.Insert(ctx, readingAnswer(u, l, 2, 0, 0, true)) // another version still counts
	_ = r.Insert(ctx, readingAnswer(newID(), l, 1, 0, 0, true))
	a, c, err := r.Totals(ctx, u)
	if err != nil || a != 3 || c != 2 {
		t.Fatalf("totals = %d, %d, %v", a, c, err)
	}
}
