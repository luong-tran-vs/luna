package settings

import (
	"context"
	"sync"
)

// fakeRepo stores settings per user; users must be added before use (as accounts).
type fakeRepo struct {
	mu    sync.Mutex
	users map[string]Settings
}

func newFakeRepo(users ...string) *fakeRepo {
	f := &fakeRepo{users: map[string]Settings{}}
	for _, u := range users {
		f.users[u] = Settings{}
	}
	return f
}

func (f *fakeRepo) Get(_ context.Context, userID string) (Settings, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.users[userID]
	if !ok {
		return Settings{}, ErrNotFound
	}
	return s, nil
}

func (f *fakeRepo) Update(_ context.Context, userID string, p Patch) (Settings, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.users[userID]
	if !ok {
		return Settings{}, ErrNotFound
	}
	if p.Theme != nil {
		s.Theme = *p.Theme
	}
	if p.DailyReviewLimit != nil {
		s.DailyReviewLimit = *p.DailyReviewLimit
	}
	if p.Timezone != nil {
		s.Timezone = *p.Timezone
	}
	f.users[userID] = s
	return s, nil
}

func (f *fakeRepo) set(userID string, s Settings) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.users[userID] = s
}
