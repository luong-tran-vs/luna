package verbs

import (
	"context"
	"fmt"
	"time"
)

// StudyDays counts today as a study day for the streak (implemented over progress in
// internal/service).
type StudyDays interface {
	MarkStudied(ctx context.Context, userID string) error
}

// Service keeps the learned verbs and records their review rounds.
type Service struct {
	repo      Repository
	studyDays StudyDays
	now       func() time.Time
}

// NewService returns a Service on repo; studyDays counts a lesson or review round for the streak
// (nil counts none); now is the clock (time.Now outside tests).
func NewService(repo Repository, studyDays StudyDays, now func() time.Time) *Service {
	return &Service{repo: repo, studyDays: studyDays, now: now}
}

// List returns the user's records.
func (s *Service) List(ctx context.Context, userID string) ([]Progress, error) {
	out, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("verbs: list: %w", err)
	}
	return out, nil
}

// Record applies a finished review round and returns the records it changed. A verb listed twice
// counts once, with its last result.
func (s *Service) Record(ctx context.Context, userID string, results []Result) ([]Progress, error) {
	if err := checkCount("results", len(results)); err != nil {
		return nil, err
	}
	last := make(map[string]bool, len(results))
	bases := make([]string, 0, len(results))
	for _, r := range results {
		bases = append(bases, r.Base)
		last[r.Base] = r.Right
	}
	out, err := s.update(ctx, userID, "results", bases, func(p Progress, now time.Time) Progress {
		return Apply(p, last[p.Base], now)
	})
	if err != nil {
		return nil, err
	}
	if err := s.markStudied(ctx, userID); err != nil {
		return nil, err
	}
	return out, nil
}

// SetLearned marks the verbs as learned (the learner's "Đã học"), or no longer learned, and
// returns their records.
func (s *Service) SetLearned(ctx context.Context, userID string, bases []string, learned bool) ([]Progress, error) {
	if err := checkCount("bases", len(bases)); err != nil {
		return nil, err
	}
	change := Unlearn
	if learned {
		change = Learn
	}
	out, err := s.update(ctx, userID, "bases", bases, change)
	if err != nil || !learned {
		return out, err
	}
	if err := s.markStudied(ctx, userID); err != nil {
		return nil, err
	}
	return out, nil
}

// markStudied counts today for the streak: a finished lesson or review round is a study day.
func (s *Service) markStudied(ctx context.Context, userID string) error {
	if s.studyDays == nil {
		return nil
	}
	if err := s.studyDays.MarkStudied(ctx, userID); err != nil {
		return fmt.Errorf("verbs: study day: %w", err)
	}
	return nil
}

func checkCount(field string, n int) error {
	switch {
	case n == 0:
		return &ValidationError{Fields: map[string]string{field: "Chưa có động từ nào"}}
	case n > MaxResults:
		return &ValidationError{Fields: map[string]string{field: fmt.Sprintf("Tối đa %d động từ mỗi lần", MaxResults)}}
	}
	return nil
}

// update applies change to the user's record of each verb (a new one for a verb never seen), once
// per verb in the order first listed, and saves them.
func (s *Service) update(ctx context.Context, userID, field string, bases []string, change func(Progress, time.Time) Progress) ([]Progress, error) {
	seen := make(map[string]bool, len(bases))
	var order []string
	for _, b := range bases {
		if !basePattern.MatchString(b) {
			return nil, &ValidationError{Fields: map[string]string{field: "Động từ không hợp lệ: " + b}}
		}
		if !seen[b] {
			seen[b] = true
			order = append(order, b)
		}
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
		out = append(out, change(p, now))
	}
	if err := s.repo.Save(ctx, out); err != nil {
		return nil, fmt.Errorf("verbs: save: %w", err)
	}
	return out, nil
}
