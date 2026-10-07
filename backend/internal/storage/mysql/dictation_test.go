package mysql

import (
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/progress"
)

func TestDictationUpsertListTotals(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	repo := NewDictationResults(testDB(t))
	u, l1, l2 := newID(), newID(), newID()
	at := time.Date(2026, 9, 30, 3, 4, 5, 123000000, time.UTC)

	got, err := repo.List(ctx, u, l1)
	if err != nil || len(got) != 0 {
		t.Fatalf("empty List = %v, %v", got, err)
	}
	if tot, err := repo.Totals(ctx, u, nil); err != nil || tot != (progress.DictationTotals{}) {
		t.Fatalf("empty Totals = %+v, %v", tot, err)
	}

	for i, r := range []progress.Result{
		{SentenceIndex: 2, Typed: "xin chào thế giới", CorrectWords: 1, TotalWords: 4, CheckedAt: at},
		{SentenceIndex: 0, Typed: "hello", CorrectWords: 2, TotalWords: 2, CheckedAt: at},
	} {
		if err := repo.Upsert(ctx, u, l1, 1, r); err != nil {
			t.Fatalf("Upsert %d: %v", i, err)
		}
	}
	if err := repo.Upsert(ctx, u, l2, 3, progress.Result{SentenceIndex: 0, Typed: "x", CorrectWords: 1, TotalWords: 3, CheckedAt: at}); err != nil {
		t.Fatal(err)
	}
	// Another user's result is not counted.
	if err := repo.Upsert(ctx, newID(), l1, 1, progress.Result{SentenceIndex: 0, CorrectWords: 9, TotalWords: 9, CheckedAt: at}); err != nil {
		t.Fatal(err)
	}

	got, err = repo.List(ctx, u, l1)
	if err != nil || len(got) != 2 {
		t.Fatalf("List = %v, %v", got, err)
	}
	if got[0].SentenceIndex != 0 || got[1].SentenceIndex != 2 || got[1].Typed != "xin chào thế giới" ||
		got[1].Revision != 1 || got[1].CorrectWords != 1 || got[1].TotalWords != 4 || !got[1].CheckedAt.Equal(at) {
		t.Fatalf("List = %+v", got)
	}

	// The latest check replaces the previous one, with its new revision.
	at2 := at.Add(time.Hour)
	if err := repo.Upsert(ctx, u, l1, 2, progress.Result{SentenceIndex: 2, Typed: "again", CorrectWords: 4, TotalWords: 4, CheckedAt: at2}); err != nil {
		t.Fatal(err)
	}
	got, _ = repo.List(ctx, u, l1)
	if len(got) != 2 || got[1].Typed != "again" || got[1].Revision != 2 || got[1].CorrectWords != 4 || !got[1].CheckedAt.Equal(at2) {
		t.Fatalf("after replace = %+v", got)
	}

	tot, err := repo.Totals(ctx, u, nil)
	if err != nil {
		t.Fatal(err)
	}
	if want := (progress.DictationTotals{Sentences: 3, CorrectWords: 2 + 4 + 1, TotalWords: 2 + 4 + 3, Lessons: 2}); tot != want {
		t.Fatalf("Totals = %+v, want %+v", tot, want)
	}
	// Only the results checked at or after since count.
	if tot, err := repo.Totals(ctx, u, &at); err != nil || tot.Sentences != 3 || tot.Lessons != 2 {
		t.Fatalf("Totals since at = %+v, %v", tot, err)
	}
	tot, err = repo.Totals(ctx, u, &at2)
	if want := (progress.DictationTotals{Sentences: 1, CorrectWords: 4, TotalWords: 4, Lessons: 1}); err != nil || tot != want {
		t.Fatalf("Totals since at2 = %+v, %v; want %+v", tot, err, want)
	}
}

func TestDictationDelete(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	repo := NewDictationResults(testDB(t))
	u, other, l1, l2 := newID(), newID(), newID(), newID()
	at := time.Date(2026, 9, 30, 3, 4, 5, 0, time.UTC)
	for _, c := range []struct {
		user, lesson string
		revision     int
		sentence     int
	}{{u, l1, 1, 0}, {u, l1, 2, 1}, {u, l2, 1, 0}, {other, l1, 1, 0}} {
		r := progress.Result{SentenceIndex: c.sentence, Typed: "x", CorrectWords: 1, TotalWords: 2, CheckedAt: at}
		if err := repo.Upsert(ctx, c.user, c.lesson, c.revision, r); err != nil {
			t.Fatal(err)
		}
	}

	if err := repo.Delete(ctx, u, l1); err != nil {
		t.Fatal(err)
	}
	if got, err := repo.List(ctx, u, l1); err != nil || len(got) != 0 {
		t.Fatalf("after delete = %v, %v", got, err)
	}
	if got, _ := repo.List(ctx, u, l2); len(got) != 1 {
		t.Fatalf("other lesson = %v", got)
	}
	if got, _ := repo.List(ctx, other, l1); len(got) != 1 {
		t.Fatalf("other user = %v", got)
	}
	if err := repo.Delete(ctx, u, l1); err != nil {
		t.Fatalf("delete again: %v", err)
	}
}
