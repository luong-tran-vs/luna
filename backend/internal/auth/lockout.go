package auth

import (
	"sync"
	"time"
)

const (
	// MaxFailedLogins consecutive failures lock an email.
	MaxFailedLogins = 5
	// LockoutDuration is how long a locked email cannot log in.
	LockoutDuration = 15 * time.Minute
)

type lockEntry struct {
	failures    int
	lockedUntil time.Time
}

// Lockout counts consecutive failed logins per email in memory.
// Counters are lost on restart, which is acceptable for a single-user app.
type Lockout struct {
	mu      sync.Mutex
	entries map[string]*lockEntry
	now     func() time.Time
}

// NewLockout returns an empty Lockout using now as its clock.
func NewLockout(now func() time.Time) *Lockout {
	return &Lockout{entries: map[string]*lockEntry{}, now: now}
}

// Check reports whether key is locked and for how much longer.
func (l *Lockout) Check(key string) (time.Duration, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.remaining(key)
}

// Fail records a failed login. When it reaches MaxFailedLogins the key is locked for
// LockoutDuration and the counter restarts; the returned values report that new lock.
func (l *Lockout) Fail(key string) (time.Duration, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if d, locked := l.remaining(key); locked {
		return d, true
	}
	e := l.entries[key]
	if e == nil {
		e = &lockEntry{}
		l.entries[key] = e
	}
	e.failures++
	if e.failures >= MaxFailedLogins {
		e.failures = 0
		e.lockedUntil = l.now().Add(LockoutDuration)
		return LockoutDuration, true
	}
	return 0, false
}

// Success clears the failure count after a correct login.
func (l *Lockout) Success(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, key)
}

// remaining must be called with l.mu held. Expired locks with no pending failures are removed.
func (l *Lockout) remaining(key string) (time.Duration, bool) {
	e := l.entries[key]
	if e == nil {
		return 0, false
	}
	if d := e.lockedUntil.Sub(l.now()); d > 0 {
		return d, true
	}
	if e.failures == 0 {
		delete(l.entries, key)
	}
	return 0, false
}
