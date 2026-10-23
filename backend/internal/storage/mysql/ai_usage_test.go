package mysql

import (
	"fmt"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
)

func TestAIUsageKeepLatest(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	repo := NewAIUsage(testDB(t))
	t0 := time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC)
	for i := range 5 {
		if err := repo.Add(ctx, ai.Usage{At: t0.Add(time.Duration(i) * time.Minute), Op: fmt.Sprint(i), Key: i % 2}); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.KeepLatest(ctx, 3); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Since(ctx, t0)
	if err != nil || len(got) != 3 || got[0].Op != "2" || got[2].Op != "4" || got[1].Key != 1 {
		t.Fatalf("kept %+v, %v", got, err)
	}
}
