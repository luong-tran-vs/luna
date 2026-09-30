package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"
)

type testEnv struct {
	svc      *Service
	users    *fakeUsers
	sessions *fakeSessions
	clock    *fakeClock
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	users, sessions, clock := newFakeUsers(), newFakeSessions(), newFakeClock()
	svc, err := NewService(users, sessions, NewLockout(clock.Now), clock.Now)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	return &testEnv{svc: svc, users: users, sessions: sessions, clock: clock}
}

func (e *testEnv) register(t *testing.T, email string) (User, string) {
	t.Helper()
	u, token, err := e.svc.Register(context.Background(), email, "matkhau123", "Asia/Ho_Chi_Minh")
	if err != nil {
		t.Fatalf("Register(%s): %v", email, err)
	}
	return u, token
}

// --- Authenticate / sessions (foundation) ---

func TestAuthenticate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	env := newTestEnv(t)
	u, token := env.register(t, "an@example.com")

	got, err := env.svc.Authenticate(ctx, token)
	if err != nil || got.ID != u.ID {
		t.Fatalf("Authenticate = %+v, %v; want user %s", got, err, u.ID)
	}

	if _, err := env.svc.Authenticate(ctx, "not-a-token"); !errors.Is(err, ErrUnauthenticated) {
		t.Errorf("unknown token: err = %v, want ErrUnauthenticated", err)
	}
	if _, err := env.svc.Authenticate(ctx, ""); !errors.Is(err, ErrUnauthenticated) {
		t.Errorf("empty token: err = %v, want ErrUnauthenticated", err)
	}

	env.clock.Advance(SessionTTL + time.Second)
	if _, err := env.svc.Authenticate(ctx, token); !errors.Is(err, ErrUnauthenticated) {
		t.Errorf("expired session: err = %v, want ErrUnauthenticated", err)
	}
}

func TestSessionStoresOnlyTokenHash(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	_, token := env.register(t, "an@example.com")

	sessions := env.sessions.all()
	if len(sessions) != 1 {
		t.Fatalf("sessions = %d, want 1", len(sessions))
	}
	sum := sha256.Sum256([]byte(token))
	if sessions[0].TokenHash != hex.EncodeToString(sum[:]) {
		t.Errorf("stored hash is not SHA-256 of the token")
	}
	if sessions[0].TokenHash == token {
		t.Errorf("raw token stored")
	}
	if want := env.clock.Now().Add(SessionTTL); !sessions[0].ExpiresAt.Equal(want) {
		t.Errorf("ExpiresAt = %v, want %v", sessions[0].ExpiresAt, want)
	}
	if len(token) < 43 { // 32 bytes base64url
		t.Errorf("token %q too short", token)
	}
}

// --- Register (US1) ---

func TestRegisterRoles(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	first, _ := env.register(t, "Admin@Example.com")
	second, _ := env.register(t, "hoc@example.com")

	if first.Role != RoleAdmin {
		t.Errorf("first role = %s, want admin", first.Role)
	}
	if second.Role != RoleLearner {
		t.Errorf("second role = %s, want learner", second.Role)
	}
	if first.Email != "admin@example.com" {
		t.Errorf("email not normalized: %s", first.Email)
	}
}

func TestRegisterStoresHashAndTimezone(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	env := newTestEnv(t)

	u, _, err := env.svc.Register(ctx, "an@example.com", "matkhau123", "Mars/Base")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	stored, _ := env.users.FindByID(ctx, u.ID)
	if !strings.HasPrefix(stored.PasswordHash, "$argon2id$") || strings.Contains(stored.PasswordHash, "matkhau123") {
		t.Errorf("password not hashed: %q", stored.PasswordHash)
	}
	if stored.Timezone != DefaultTimezone {
		t.Errorf("timezone = %q, want %q", stored.Timezone, DefaultTimezone)
	}
	if !stored.CreatedAt.Equal(env.clock.Now()) {
		t.Errorf("CreatedAt = %v", stored.CreatedAt)
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	env.register(t, "An@Example.com")

	_, _, err := env.svc.Register(context.Background(), " an@example.COM ", "matkhau123", "")
	if !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("err = %v, want ErrEmailTaken", err)
	}
}

func TestRegisterValidation(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	_, _, err := env.svc.Register(context.Background(), "khong-hop-le", "ngan", "")
	var verr *ValidationError
	if !errors.As(err, &verr) || verr.Fields["email"] == "" || verr.Fields["password"] == "" {
		t.Fatalf("err = %v, want ValidationError on email and password", err)
	}
	if n, _ := env.users.Count(context.Background()); n != 0 {
		t.Fatalf("user created despite invalid input")
	}
}

// --- Login / Logout (US2) ---

func TestLogin(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	env := newTestEnv(t)
	u, _ := env.register(t, "an@example.com")

	got, token, err := env.svc.Login(ctx, " AN@example.com", "matkhau123")
	if err != nil || got.ID != u.ID || token == "" {
		t.Fatalf("Login = %+v, %q, %v", got, token, err)
	}
	if _, err := env.svc.Authenticate(ctx, token); err != nil {
		t.Fatalf("new session not usable: %v", err)
	}

	if _, _, err := env.svc.Login(ctx, "an@example.com", "sai-mat-khau"); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("wrong password: err = %v, want ErrInvalidCredentials", err)
	}
	if _, _, err := env.svc.Login(ctx, "khong-co@example.com", "matkhau123"); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("unknown email: err = %v, want ErrInvalidCredentials", err)
	}
}

func TestLoginMissingFields(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	_, _, err := env.svc.Login(context.Background(), "", "")
	var verr *ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("err = %v, want ValidationError", err)
	}
}

// Not parallel: timing is measured while no other argon2id test runs.
func TestLoginUnknownEmailTakesComparableTime(t *testing.T) {
	ctx := context.Background()
	env := newTestEnv(t)
	env.register(t, "an@example.com")

	measure := func(email string) time.Duration {
		start := time.Now()
		_, _, _ = env.svc.Login(ctx, email, "sai-mat-khau")
		return time.Since(start)
	}
	known, unknown := measure("an@example.com"), measure("khong-co@example.com")
	// Both paths run argon2id; the unknown path must not return almost instantly.
	if unknown < known/4 {
		t.Fatalf("unknown email took %v vs %v for a known one; dummy hash not verified", unknown, known)
	}
}

func TestLogout(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	env := newTestEnv(t)
	_, token := env.register(t, "an@example.com")

	if err := env.svc.Logout(ctx, token); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if _, err := env.svc.Authenticate(ctx, token); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("session still valid after logout: %v", err)
	}
	if err := env.svc.Logout(ctx, "unknown"); err != nil {
		t.Fatalf("Logout unknown token: %v", err)
	}
}

// --- Lockout (US3) ---

func TestLoginLockout(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	env := newTestEnv(t)
	env.register(t, "an@example.com")

	for i := 1; i < MaxFailedLogins; i++ {
		if _, _, err := env.svc.Login(ctx, "an@example.com", "sai"); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("attempt %d: err = %v, want ErrInvalidCredentials", i, err)
		}
	}

	var locked *LockedError
	if _, _, err := env.svc.Login(ctx, "an@example.com", "sai"); !errors.As(err, &locked) || locked.RetryAfter != LockoutDuration {
		t.Fatalf("5th attempt: err = %v, want LockedError(15m)", err)
	}
	if _, _, err := env.svc.Login(ctx, "an@example.com", "matkhau123"); !errors.As(err, &locked) {
		t.Fatalf("correct password while locked: err = %v, want LockedError", err)
	}

	env.clock.Advance(LockoutDuration)
	if _, _, err := env.svc.Login(ctx, "an@example.com", "matkhau123"); err != nil {
		t.Fatalf("after lock expiry: %v", err)
	}
}

func TestLoginSuccessResetsFailures(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	env := newTestEnv(t)
	env.register(t, "an@example.com")

	for range 3 {
		_, _, _ = env.svc.Login(ctx, "an@example.com", "sai")
	}
	if _, _, err := env.svc.Login(ctx, "an@example.com", "matkhau123"); err != nil {
		t.Fatalf("login: %v", err)
	}
	for i := 1; i < MaxFailedLogins; i++ {
		if _, _, err := env.svc.Login(ctx, "an@example.com", "sai"); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("attempt %d after reset: err = %v", i, err)
		}
	}
}

func TestLoginLockoutUnknownEmail(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	env := newTestEnv(t)

	var err error
	for range MaxFailedLogins {
		_, _, err = env.svc.Login(ctx, "khong-co@example.com", "sai")
	}
	var locked *LockedError
	if !errors.As(err, &locked) {
		t.Fatalf("unknown email not locked: %v", err)
	}
}
