package settings

import (
	"context"
	"fmt"
	"time"
)

// Service reads and changes a learner's settings. The user is always the session user.
type Service struct {
	repo Repository
}

// NewService returns a Service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// withDefaults fills the fields an account never saved.
func withDefaults(s Settings) Settings {
	if s.Theme == "" {
		s.Theme = DefaultTheme
	}
	if s.DailyReviewLimit == 0 {
		s.DailyReviewLimit = DefaultReviewLimit
	}
	if !ValidTimezone(s.Timezone) {
		s.Timezone = DefaultTimezone
	}
	return s
}

// Get returns the user's settings, with defaults for fields never saved.
func (s *Service) Get(ctx context.Context, userID string) (Settings, error) {
	st, err := s.repo.Get(ctx, userID)
	if err != nil {
		return Settings{}, fmt.Errorf("settings: get: %w", err)
	}
	return withDefaults(st), nil
}

// Update changes the fields present in p after checking all of them; an invalid field changes
// nothing (*ValidationError).
func (s *Service) Update(ctx context.Context, userID string, p Patch) (Settings, error) {
	if err := ValidatePatch(p); err != nil {
		return Settings{}, err
	}
	if p == (Patch{}) {
		return s.Get(ctx, userID)
	}
	st, err := s.repo.Update(ctx, userID, p)
	if err != nil {
		return Settings{}, fmt.Errorf("settings: update: %w", err)
	}
	return withDefaults(st), nil
}

// Location is the timezone used to compute the user's study day.
func (s *Service) Location(ctx context.Context, userID string) (*time.Location, error) {
	st, err := s.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	loc, err := time.LoadLocation(st.Timezone)
	if err != nil {
		return nil, fmt.Errorf("settings: load timezone: %w", err)
	}
	return loc, nil
}

// ReviewLimit is the user's daily card limit for the review step.
func (s *Service) ReviewLimit(ctx context.Context, userID string) (int, error) {
	st, err := s.Get(ctx, userID)
	if err != nil {
		return 0, err
	}
	return st.DailyReviewLimit, nil
}
