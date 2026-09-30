package auth

import (
	"context"
	"strconv"
	"sync"
	"time"
)

// fakeUsers is an in-memory UserRepository for tests.
type fakeUsers struct {
	mu     sync.Mutex
	byID   map[string]User
	nextID int
}

func newFakeUsers() *fakeUsers { return &fakeUsers{byID: map[string]User{}} }

func (f *fakeUsers) Create(_ context.Context, u User) (User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, existing := range f.byID {
		if existing.Email == u.Email {
			return User{}, ErrEmailTaken
		}
	}
	f.nextID++
	u.ID = "u" + strconv.Itoa(f.nextID)
	f.byID[u.ID] = u
	return u, nil
}

func (f *fakeUsers) FindByEmail(_ context.Context, email string) (User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, u := range f.byID {
		if u.Email == email {
			return u, nil
		}
	}
	return User{}, ErrNotFound
}

func (f *fakeUsers) FindByID(_ context.Context, id string) (User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.byID[id]
	if !ok {
		return User{}, ErrNotFound
	}
	return u, nil
}

func (f *fakeUsers) Count(context.Context) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return int64(len(f.byID)), nil
}

// fakeSessions is an in-memory SessionRepository for tests.
type fakeSessions struct {
	mu     sync.Mutex
	byHash map[string]Session
}

func newFakeSessions() *fakeSessions { return &fakeSessions{byHash: map[string]Session{}} }

func (f *fakeSessions) Create(_ context.Context, s Session) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.byHash[s.TokenHash] = s
	return nil
}

func (f *fakeSessions) FindByTokenHash(_ context.Context, hash string) (Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.byHash[hash]
	if !ok {
		return Session{}, ErrNotFound
	}
	return s, nil
}

func (f *fakeSessions) Delete(_ context.Context, hash string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.byHash, hash)
	return nil
}

func (f *fakeSessions) all() []Session {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]Session, 0, len(f.byHash))
	for _, s := range f.byHash {
		out = append(out, s)
	}
	return out
}

// fakeClock is a manually advanced clock.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: time.Date(2026, 9, 29, 8, 0, 0, 0, time.UTC)}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}
