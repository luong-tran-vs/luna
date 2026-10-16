package mysql

import (
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/vocab"
)

func TestLessonResultsSaveList(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	repo := NewLessonResults(testDB(t))
	u, l1, l2 := newID(), newID(), newID()
	at := time.Date(2026, 10, 16, 3, 4, 5, 123000000, time.UTC)

	if got, err := repo.List(ctx, u); err != nil || len(got) != 0 {
		t.Fatalf("empty List = %v, %v", got, err)
	}
	for _, r := range []vocab.LessonResult{
		{UserID: u, LessonID: l1, Correct: 1, Total: 4, Wrong: []string{"v:go", "f:0", "t:1"}, ReviewedAt: at},
		{UserID: u, LessonID: l1, Correct: 3, Total: 4, Wrong: []string{"t:1"}, ReviewedAt: at.Add(time.Hour)},
		{UserID: u, LessonID: l2, Correct: 2, Total: 2, ReviewedAt: at},
		{UserID: newID(), LessonID: l1, Correct: 0, Total: 1, Wrong: []string{"v:x"}, ReviewedAt: at},
	} {
		if err := repo.Save(ctx, r); err != nil {
			t.Fatal(err)
		}
	}
	got, err := repo.List(ctx, u)
	if err != nil || len(got) != 2 {
		t.Fatalf("List = %+v, %v", got, err)
	}
	byLesson := map[string]vocab.LessonResult{}
	for _, r := range got {
		byLesson[r.LessonID] = r
	}
	if r := byLesson[l1]; r.Correct != 3 || len(r.Wrong) != 1 || r.Wrong[0] != "t:1" || !r.ReviewedAt.Equal(at.Add(time.Hour)) {
		t.Fatalf("replaced result = %+v", r)
	}
	if r := byLesson[l2]; r.Correct != 2 || len(r.Wrong) != 0 {
		t.Fatalf("all right = %+v", r)
	}
}
