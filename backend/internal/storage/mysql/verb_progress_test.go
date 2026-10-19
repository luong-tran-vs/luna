package mysql

import (
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/verbs"
)

func TestVerbProgressSaveList(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	repo := NewVerbProgress(testDB(t))
	u, other := newID(), newID()
	at := time.Date(2026, 10, 16, 3, 4, 5, 123000000, time.UTC)

	if got, err := repo.List(ctx, u); err != nil || len(got) != 0 {
		t.Fatalf("empty List = %v, %v", got, err)
	}
	if err := repo.Save(ctx, []verbs.Progress{
		{UserID: u, Base: "go", LearnedAt: at, Level: 1, DueAt: at.Add(24 * time.Hour), Right: 1, UpdatedAt: at},
		{UserID: u, Base: "see", Wrong: 1, UpdatedAt: at},
		{UserID: other, Base: "go", Level: 3, DueAt: at, Right: 3, UpdatedAt: at},
	}); err != nil {
		t.Fatal(err)
	}
	// Saving again replaces the row.
	if err := repo.Save(ctx, []verbs.Progress{{UserID: u, Base: "go", LearnedAt: at, Level: 2, DueAt: at.Add(72 * time.Hour), Right: 2, UpdatedAt: at.Add(time.Hour)}}); err != nil {
		t.Fatal(err)
	}
	got, err := repo.List(ctx, u)
	if err != nil || len(got) != 2 {
		t.Fatalf("List = %+v, %v", got, err)
	}
	if g := got[0]; g.Base != "go" || !g.LearnedAt.Equal(at) || g.Level != 2 || !g.DueAt.Equal(at.Add(72*time.Hour)) || g.Right != 2 || !g.UpdatedAt.Equal(at.Add(time.Hour)) {
		t.Errorf("go = %+v", g)
	}
	if s := got[1]; s.Base != "see" || !s.LearnedAt.IsZero() || !s.DueAt.IsZero() || s.Wrong != 1 {
		t.Errorf("see = %+v", s)
	}
	if err := repo.Save(ctx, nil); err != nil {
		t.Errorf("empty Save: %v", err)
	}
}
