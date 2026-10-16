package verbs

import (
	"context"
	"fmt"
	"time"
)

// Service records the practice rounds of the irregular verbs.
type Service struct {
	repo Repository
	now  func() time.Time
}

// NewService returns a Service on repo; now is the clock (time.Now outside tests).
func NewService(repo Repository, now func() time.Time) *Service {
	return &Service{repo: repo, now: now}
}

// List returns the user's records.
func (s *Service) List(ctx context.Context, userID string) ([]Progress, error) {
	out, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("verbs: list: %w", err)
	}
	return out, nil
}

// Record applies a finished round and returns the records it changed. A verb listed twice counts
// once, with its last result.
func (s *Service) Record(ctx context.Context, userID string, results []Result) ([]Progress, error) {
	switch {
	case len(results) == 0:
		return nil, &ValidationError{Fields: map[string]string{"results": "Chưa có kết quả nào"}}
	case len(results) > MaxResults:
		return nil, &ValidationError{Fields: map[string]string{"results": fmt.Sprintf("Tối đa %d động từ mỗi lượt", MaxResults)}}
	}
	last := make(map[string]bool, len(results))
	var order []string
	for _, r := range results {
		if !basePattern.MatchString(r.Base) {
			return nil, &ValidationError{Fields: map[string]string{"results": "Động từ không hợp lệ: " + r.Base}}
		}
		if _, seen := last[r.Base]; !seen {
			order = append(order, r.Base)
		}
		last[r.Base] = r.Right
	}

	current, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("verbs: list: %w", err)
	}
	byBase := make(map[string]Progress, len(current))
	for _, p := range current {
		byBase[p.Base] = p
	}
	now := s.now().UTC()
	out := make([]Progress, 0, len(order))
	for _, base := range order {
		p, ok := byBase[base]
		if !ok {
			p = Progress{UserID: userID, Base: base}
		}
		out = append(out, Apply(p, last[base], now))
	}
	if err := s.repo.Save(ctx, out); err != nil {
		return nil, fmt.Errorf("verbs: save: %w", err)
	}
	return out, nil
}
