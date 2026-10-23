package mysql

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/curriculum"
)

func TestCurricula(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	repo := NewCurricula(testDB(t))
	if _, err := repo.Get(ctx, "A2"); !errors.Is(err, curriculum.ErrNoPlan) {
		t.Fatalf("empty: %v", err)
	}
	t0 := time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC)
	p := &curriculum.Plan{Level: "A2", Topics: []curriculum.Topic{{
		TopicID: "t1", Name: "Du lịch", CanDoVi: "kể về chuyến đi.", Grammar: []string{"a2-past-simple-regular"},
		Slots: []curriculum.Slot{{Key: "a2-01-1", Kind: curriculum.KindMain, Title: "My Trip", Grammar: []string{"a2-past-simple-regular"}, Idea: "Lan visits Hue."}},
	}}}
	if err := repo.Save(ctx, curriculum.Course{Level: "A2", Draft: p, UpdatedAt: t0}); err != nil {
		t.Fatal(err)
	}
	c, err := repo.Get(ctx, "A2")
	if err != nil || c.Approved != nil || !c.ApprovedAt.IsZero() || !reflect.DeepEqual(c.Draft, p) || !c.UpdatedAt.Equal(t0) {
		t.Fatalf("draft: %+v, %v", c, err)
	}
	// Approving replaces the row.
	if err := repo.Save(ctx, curriculum.Course{Level: "A2", Approved: p, ApprovedAt: t0.Add(time.Hour), UpdatedAt: t0.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	c, err = repo.Get(ctx, "A2")
	if err != nil || c.Draft != nil || !reflect.DeepEqual(c.Approved, p) || !c.ApprovedAt.Equal(t0.Add(time.Hour)) {
		t.Fatalf("approved: %+v, %v", c, err)
	}
}
