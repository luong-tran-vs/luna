package mysql

import (
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/grammar"
)

func TestGrammarReports(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	repo := NewGrammarReports(testDB(t))
	t0 := time.Date(2026, 10, 2, 9, 0, 0, 0, time.UTC)
	u1, u2 := newID(), newID()
	rep := func(user, point, ex string, reason grammar.ReportReason, note string, at time.Time) grammar.Report {
		return grammar.Report{UserID: user, PointID: point, ExerciseID: ex, Reason: reason, Note: note, CreatedAt: at, UpdatedAt: at}
	}

	if got, err := repo.ListOpen(ctx); err != nil || got == nil || len(got) != 0 {
		t.Fatalf("empty ListOpen = %v, %v", got, err)
	}

	// Insert three reports, one a minute apart.
	for i, r := range []grammar.Report{
		rep(u1, "a1-to-be", "p1", grammar.ReasonWrongAnswer, "sai", t0),
		rep(u2, "a1-to-be", "p1", grammar.ReasonTypo, "", t0.Add(time.Minute)),
		rep(u1, "a1-to-be", "p2", grammar.ReasonAmbiguous, "mơ hồ", t0.Add(2*time.Minute)),
	} {
		if err := repo.Upsert(ctx, r); err != nil {
			t.Fatalf("upsert %d: %v", i, err)
		}
	}
	open, err := repo.ListOpen(ctx)
	if err != nil || len(open) != 3 {
		t.Fatalf("ListOpen = %v, %v", open, err)
	}
	if open[0].ExerciseID != "p2" || open[1].UserID != u2 || open[2].UserID != u1 || open[2].ExerciseID != "p1" {
		t.Fatalf("order = %+v", open)
	}
	first := open[2]
	if first.ID == "" || len(first.ID) != 24 || first.Status != grammar.ReportOpen || first.Note != "sai" ||
		!first.CreatedAt.Equal(t0) || !first.ResolvedAt.IsZero() {
		t.Fatalf("first = %+v", first)
	}

	// Overwrite: same id and createdAt, new reason and note, newest now.
	if err := repo.Upsert(ctx, rep(u1, "a1-to-be", "p1", grammar.ReasonOther, "đổi", t0.Add(time.Hour))); err != nil {
		t.Fatal(err)
	}
	open, _ = repo.ListOpen(ctx)
	if len(open) != 3 || open[0].UserID != u1 || open[0].ExerciseID != "p1" {
		t.Fatalf("after overwrite = %+v", open)
	}
	if o := open[0]; o.ID != first.ID || !o.CreatedAt.Equal(t0) || o.Reason != grammar.ReasonOther || o.Note != "đổi" ||
		!o.UpdatedAt.Equal(t0.Add(time.Hour)) {
		t.Fatalf("overwritten = %+v (first %+v)", o, first)
	}

	// Resolve touches only that pair, counts the open ones.
	n, err := repo.Resolve(ctx, "a1-to-be", "p1", t0.Add(2*time.Hour))
	if err != nil || n != 2 {
		t.Fatalf("Resolve = %d, %v", n, err)
	}
	if n, err := repo.Resolve(ctx, "a1-to-be", "p1", t0.Add(3*time.Hour)); err != nil || n != 0 {
		t.Fatalf("second Resolve = %d, %v", n, err)
	}
	if n, err := repo.Resolve(ctx, "a1-other", "p1", t0); err != nil || n != 0 {
		t.Fatalf("other point Resolve = %d, %v", n, err)
	}
	open, _ = repo.ListOpen(ctx)
	if len(open) != 1 || open[0].ExerciseID != "p2" {
		t.Fatalf("after resolve = %+v", open)
	}

	// Reporting a resolved exercise again reopens the same row.
	if err := repo.Upsert(ctx, rep(u2, "a1-to-be", "p1", grammar.ReasonTypo, "lại", t0.Add(4*time.Hour))); err != nil {
		t.Fatal(err)
	}
	open, _ = repo.ListOpen(ctx)
	if len(open) != 2 || open[0].UserID != u2 || open[0].Note != "lại" || !open[0].ResolvedAt.IsZero() ||
		open[0].Status != grammar.ReportOpen || !open[0].CreatedAt.Equal(t0.Add(time.Minute)) {
		t.Fatalf("after reopen = %+v", open)
	}
}
