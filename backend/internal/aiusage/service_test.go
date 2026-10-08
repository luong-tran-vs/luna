package aiusage

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
)

type fakeRepo struct {
	mu      sync.Mutex
	calls   []ai.Usage
	deleted []time.Time
	err     error
}

func (f *fakeRepo) Add(_ context.Context, u ai.Usage) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	f.calls = append(f.calls, u)
	return nil
}

func (f *fakeRepo) Since(_ context.Context, from time.Time) ([]ai.Usage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []ai.Usage
	for _, u := range f.calls {
		if !u.At.Before(from) {
			out = append(out, u)
		}
	}
	slices.SortFunc(out, func(a, b ai.Usage) int { return a.At.Compare(b.At) })
	return out, nil
}

func (f *fakeRepo) DeleteBefore(_ context.Context, t time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deleted = append(f.deleted, t)
	return nil
}

func TestSummary(t *testing.T) {
	t.Parallel()
	// 20:30 UTC on 8 Oct 2026 is 13:30 in Los Angeles (PDT): the Google day began at 07:00 UTC.
	now := time.Date(2026, 10, 8, 20, 30, 20, 0, time.UTC)
	repo := &fakeRepo{}
	svc := NewService(repo, func() time.Time { return now }, slog.New(slog.DiscardHandler))
	add := func(ago time.Duration, model, op string, status, tokens int) {
		svc.RecordUsage(t.Context(), ai.Usage{At: now.Add(-ago), Model: model, Op: op, Status: status, TotalTokens: tokens, PromptTokens: tokens / 2})
	}
	add(10*time.Second, "flash", "annotate", 200, 1000)
	add(30*time.Second, "flash", "practice", 429, 0)
	add(45*time.Second, "image", "image", 200, 1300)
	add(5*time.Minute, "flash", "annotate", 200, 800)
	add(5*time.Minute+10*time.Second, "flash", "annotate", 200, 700)
	add(3*time.Hour, "flash", "explain", 200, 100)
	add(14*time.Hour, "flash", "grade_writing", 500, 0) // yesterday for Google
	add(8*24*time.Hour, "flash", "annotate", 200, 9)    // older than a week

	s, err := svc.Summary(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if !s.DayStart.Equal(time.Date(2026, 10, 8, 7, 0, 0, 0, time.UTC)) {
		t.Fatalf("day start = %v", s.DayStart)
	}
	want := []ModelTotals{
		{Model: "flash", Totals: Totals{Requests: 2, Quota: 1, PromptTokens: 500, TotalTokens: 1000}},
		{Model: "image", Totals: Totals{Requests: 1, PromptTokens: 650, TotalTokens: 1300}},
	}
	if len(s.LastMinute) != 2 || s.LastMinute[0] != want[0] || s.LastMinute[1] != want[1] {
		t.Fatalf("last minute = %+v", s.LastMinute)
	}
	if len(s.Today) != 2 || s.Today[0].Requests != 5 || s.Today[1].Requests != 1 {
		t.Fatalf("today = %+v", s.Today)
	}
	if len(s.Minutes) != 60 || s.Minutes[59].Requests != 1 || s.Minutes[58].Requests != 2 || s.Minutes[54].Requests != 2 || s.Minutes[54].TotalTokens != 1500 {
		t.Fatalf("minutes = %+v / %+v / %+v", s.Minutes[59], s.Minutes[58], s.Minutes[54])
	}
	// Clock minutes: 20:30 has one request, 20:29 two; 20:25 also two, with more tokens.
	if s.PeakMinute.Requests != 2 || s.PeakMinute.TotalTokens != 1500 {
		t.Fatalf("peak = %+v", s.PeakMinute)
	}
	if len(s.Week) != 5 || s.Week[0].Op != "annotate" || s.Week[0].Requests != 3 {
		t.Fatalf("week = %+v", s.Week)
	}
	if len(s.Recent) != 7 || s.Recent[0].Op != "annotate" || s.Recent[0].TotalTokens != 1000 {
		t.Fatalf("recent = %+v", s.Recent)
	}
}

func TestRecordUsagePrunesHourly(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 10, 8, 9, 0, 0, 0, time.UTC)
	repo := &fakeRepo{}
	svc := NewService(repo, func() time.Time { return now }, slog.New(slog.DiscardHandler))
	svc.RecordUsage(t.Context(), ai.Usage{Op: "a"})
	now = now.Add(time.Minute)
	svc.RecordUsage(t.Context(), ai.Usage{Op: "b"})
	now = now.Add(time.Hour)
	svc.RecordUsage(t.Context(), ai.Usage{Op: "c"})
	if len(repo.calls) != 3 || len(repo.deleted) != 2 || !repo.deleted[1].Equal(now.Add(-Keep)) {
		t.Fatalf("calls %d, deleted %v", len(repo.calls), repo.deleted)
	}
	// A failing database is only logged.
	repo.err = errors.New("down")
	svc.RecordUsage(t.Context(), ai.Usage{Op: "d"})
}
