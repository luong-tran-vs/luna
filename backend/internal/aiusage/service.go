package aiusage

import (
	"cmp"
	"context"
	"log/slog"
	"slices"
	"sync"
	"time"
	_ "time/tzdata" // Pacific time on servers without a zone database

	"github.com/luongtran/luna/backend/internal/ai"
)

const (
	// recordTimeout bounds saving one request, so a slow database never holds up the AI call.
	recordTimeout = 3 * time.Second
	// pruneEvery is how often recording also deletes the requests older than Keep.
	pruneEvery = time.Hour
	// recentCount is how many of the last requests the summary lists.
	recentCount = 30
)

// pacific is the time zone of the daily limits of Google's free tier.
var pacific = mustZone("America/Los_Angeles")

func mustZone(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err) // embedded by time/tzdata
	}
	return loc
}

// Service records the AI requests and sums them up. It implements ai.UsageRecorder.
type Service struct {
	repo Repository
	now  func() time.Time
	log  *slog.Logger

	mu       sync.Mutex
	prunedAt time.Time
}

// NewService returns a Service.
func NewService(repo Repository, now func() time.Time, log *slog.Logger) *Service {
	return &Service{repo: repo, now: now, log: log}
}

var _ ai.UsageRecorder = (*Service)(nil)

// RecordUsage saves one request; a failure is only logged. It keeps going when the caller's
// context ends, since the request was made anyway.
func (s *Service) RecordUsage(ctx context.Context, u ai.Usage) {
	if u.At.IsZero() {
		u.At = s.now()
	}
	rctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), recordTimeout)
	defer cancel()
	if err := s.repo.Add(rctx, u); err != nil {
		s.log.WarnContext(ctx, "ai usage: record request", slog.Any("error", err))
	}
	s.mu.Lock()
	prune := u.At.Sub(s.prunedAt) >= pruneEvery
	if prune {
		s.prunedAt = u.At
	}
	s.mu.Unlock()
	if prune {
		if err := s.repo.DeleteBefore(rctx, u.At.Add(-Keep)); err != nil {
			s.log.WarnContext(ctx, "ai usage: delete old requests", slog.Any("error", err))
		}
	}
}

// Summary sums up the requests of the last 7 days.
func (s *Service) Summary(ctx context.Context) (Summary, error) {
	now := s.now()
	calls, err := s.repo.Since(ctx, now.Add(-7*24*time.Hour))
	if err != nil {
		return Summary{}, err
	}
	return summarize(calls, now), nil
}

// summarize builds the summary of calls (oldest first) as of now.
func summarize(calls []ai.Usage, now time.Time) Summary {
	p := now.In(pacific)
	out := Summary{Now: now, DayStart: time.Date(p.Year(), p.Month(), p.Day(), 0, 0, 0, 0, pacific)}
	hourStart := now.Truncate(time.Minute).Add(-59 * time.Minute)
	out.Minutes = make([]Minute, 60)
	for i := range out.Minutes {
		out.Minutes[i].Start = hourStart.Add(time.Duration(i) * time.Minute)
	}
	lastMinute := map[string]*Totals{}
	today := map[string]*Totals{}
	week := map[string]*Totals{}
	for _, u := range calls {
		if u.At.After(now) {
			continue
		}
		bucket(week, u.Op).add(u)
		if !u.At.Before(out.DayStart) {
			bucket(today, u.Model).add(u)
		}
		if u.At.After(now.Add(-time.Minute)) {
			bucket(lastMinute, u.Model).add(u)
		}
		if i := int(u.At.Sub(hourStart) / time.Minute); !u.At.Before(hourStart) && i < len(out.Minutes) {
			out.Minutes[i].add(u)
		}
	}
	for _, m := range out.Minutes {
		if m.Requests > out.PeakMinute.Requests ||
			(m.Requests == out.PeakMinute.Requests && m.TotalTokens > out.PeakMinute.TotalTokens) {
			out.PeakMinute = m.Totals
		}
	}
	out.LastMinute = byModel(lastMinute)
	out.Today = byModel(today)
	for op, t := range week {
		out.Week = append(out.Week, OpTotals{Op: op, Totals: *t})
	}
	slices.SortFunc(out.Week, func(a, b OpTotals) int {
		return cmp.Or(cmp.Compare(b.Requests, a.Requests), cmp.Compare(a.Op, b.Op))
	})
	for i := len(calls) - 1; i >= 0 && len(out.Recent) < recentCount; i-- {
		if !calls[i].At.After(now) {
			out.Recent = append(out.Recent, calls[i])
		}
	}
	return out
}

func bucket(m map[string]*Totals, key string) *Totals {
	t, ok := m[key]
	if !ok {
		t = &Totals{}
		m[key] = t
	}
	return t
}

func byModel(m map[string]*Totals) []ModelTotals {
	out := make([]ModelTotals, 0, len(m))
	for model, t := range m {
		out = append(out, ModelTotals{Model: model, Totals: *t})
	}
	slices.SortFunc(out, func(a, b ModelTotals) int { return cmp.Compare(a.Model, b.Model) })
	return out
}
