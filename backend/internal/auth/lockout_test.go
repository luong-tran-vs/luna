package auth

import (
	"sync"
	"testing"
	"time"
)

func TestLockout(t *testing.T) {
	t.Parallel()

	clock := newFakeClock()
	l := NewLockout(clock.Now)
	const email = "hoc@example.com"

	for i := 1; i <= 4; i++ {
		if _, locked := l.Fail(email); locked {
			t.Fatalf("locked after %d failures, want 5", i)
		}
	}
	if _, locked := l.Check(email); locked {
		t.Fatal("Check reports locked after 4 failures")
	}

	remaining, locked := l.Fail(email)
	if !locked || remaining != LockoutDuration {
		t.Fatalf("5th failure: locked=%v remaining=%v, want true %v", locked, remaining, LockoutDuration)
	}

	clock.Advance(10 * time.Minute)
	if remaining, locked := l.Check(email); !locked || remaining != 5*time.Minute {
		t.Fatalf("after 10 min: locked=%v remaining=%v, want true 5m", locked, remaining)
	}

	clock.Advance(5 * time.Minute)
	if _, locked := l.Check(email); locked {
		t.Fatal("still locked after 15 minutes")
	}

	// Counter restarts from zero after the lock expires.
	for i := 1; i <= 4; i++ {
		if _, locked := l.Fail(email); locked {
			t.Fatalf("relocked after %d failures following expiry", i)
		}
	}
}

func TestLockoutSuccessResets(t *testing.T) {
	t.Parallel()

	l := NewLockout(newFakeClock().Now)
	for range 3 {
		l.Fail("a@b.vn")
	}
	l.Success("a@b.vn")
	for i := 1; i <= 4; i++ {
		if _, locked := l.Fail("a@b.vn"); locked {
			t.Fatalf("locked after %d failures following success", i)
		}
	}
}

func TestLockoutKeysAreIndependent(t *testing.T) {
	t.Parallel()

	l := NewLockout(newFakeClock().Now)
	for range MaxFailedLogins {
		l.Fail("a@b.vn")
	}
	if _, locked := l.Check("c@d.vn"); locked {
		t.Fatal("other email is locked")
	}
}

func TestLockoutConcurrent(t *testing.T) {
	t.Parallel()

	l := NewLockout(newFakeClock().Now)
	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() {
			l.Fail("a@b.vn")
			l.Check("a@b.vn")
		})
	}
	wg.Wait()
	if _, locked := l.Check("a@b.vn"); !locked {
		t.Fatal("50 concurrent failures did not lock")
	}
}
