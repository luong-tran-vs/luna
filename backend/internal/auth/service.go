package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"time"
)

// SessionTTL is how long a login lasts on one device. It is fixed, not extended on use.
const SessionTTL = 30 * 24 * time.Hour

// Service implements registration, login, logout and session checks.
type Service struct {
	users    UserRepository
	sessions SessionRepository
	lockout  *Lockout
	now      func() time.Time
	// dummyHash is verified when an email is unknown so both login failures take similar time.
	dummyHash string
}

// NewService wires the repositories, the login lockout and the clock.
func NewService(users UserRepository, sessions SessionRepository, lockout *Lockout, now func() time.Time) (*Service, error) {
	dummy, err := HashPassword("luna-dummy-password")
	if err != nil {
		return nil, err
	}
	return &Service{users: users, sessions: sessions, lockout: lockout, now: now, dummyHash: dummy}, nil
}

// Register creates an account and logs it in. The first account is an admin, the others guests.
func (s *Service) Register(ctx context.Context, email, password, timezone string) (User, string, error) {
	email = NormalizeEmail(email)
	if err := ValidateCredentials(email, password); err != nil {
		return User{}, "", err
	}

	count, err := s.users.Count(ctx)
	if err != nil {
		return User{}, "", fmt.Errorf("auth: count users: %w", err)
	}
	role := RoleGuest
	if count == 0 {
		role = RoleAdmin
	}

	hash, err := HashPassword(password)
	if err != nil {
		return User{}, "", err
	}
	u, err := s.users.Create(ctx, User{
		Email:        email,
		PasswordHash: hash,
		Role:         role,
		Timezone:     NormalizeTimezone(timezone),
		CreatedAt:    s.now(),
	})
	if err != nil {
		if errors.Is(err, ErrEmailTaken) {
			return User{}, "", ErrEmailTaken
		}
		return User{}, "", fmt.Errorf("auth: create user: %w", err)
	}

	token, err := s.startSession(ctx, u.ID)
	if err != nil {
		return User{}, "", err
	}
	return u, token, nil
}

// Login checks the credentials and starts a session. Unknown email and wrong password both
// return ErrInvalidCredentials; repeated failures return *LockedError.
func (s *Service) Login(ctx context.Context, email, password string) (User, string, error) {
	email = NormalizeEmail(email)
	if email == "" || password == "" {
		fields := map[string]string{}
		if email == "" {
			fields["email"] = "Vui lòng nhập email"
		}
		if password == "" {
			fields["password"] = "Vui lòng nhập mật khẩu"
		}
		return User{}, "", &ValidationError{Fields: fields}
	}

	if remaining, locked := s.lockout.Check(email); locked {
		return User{}, "", &LockedError{RetryAfter: remaining}
	}

	u, err := s.users.FindByEmail(ctx, email)
	switch {
	case errors.Is(err, ErrNotFound):
		_, _ = VerifyPassword(password, s.dummyHash)
		return User{}, "", s.fail(email)
	case err != nil:
		return User{}, "", fmt.Errorf("auth: find user: %w", err)
	}

	ok, err := VerifyPassword(password, u.PasswordHash)
	if err != nil {
		return User{}, "", fmt.Errorf("auth: verify password: %w", err)
	}
	if !ok {
		return User{}, "", s.fail(email)
	}

	s.lockout.Success(email)
	token, err := s.startSession(ctx, u.ID)
	if err != nil {
		return User{}, "", err
	}
	return u, token, nil
}

func (s *Service) fail(email string) error {
	if remaining, locked := s.lockout.Fail(email); locked {
		return &LockedError{RetryAfter: remaining}
	}
	return ErrInvalidCredentials
}

// Logout ends the session for token. Unknown tokens are ignored.
func (s *Service) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	if err := s.sessions.Delete(ctx, hashToken(token)); err != nil {
		return fmt.Errorf("auth: delete session: %w", err)
	}
	return nil
}

// Authenticate returns the user owning a valid, unexpired session token.
func (s *Service) Authenticate(ctx context.Context, token string) (User, error) {
	if token == "" {
		return User{}, ErrUnauthenticated
	}
	sess, err := s.sessions.FindByTokenHash(ctx, hashToken(token))
	switch {
	case errors.Is(err, ErrNotFound):
		return User{}, ErrUnauthenticated
	case err != nil:
		return User{}, fmt.Errorf("auth: find session: %w", err)
	}
	// The TTL index removes expired sessions only about once a minute, so check here too.
	if !s.now().Before(sess.ExpiresAt) {
		return User{}, ErrUnauthenticated
	}
	return s.UserByID(ctx, sess.UserID)
}

// UserByID returns the account for id, or ErrUnauthenticated if it no longer exists.
func (s *Service) UserByID(ctx context.Context, id string) (User, error) {
	u, err := s.users.FindByID(ctx, id)
	switch {
	case errors.Is(err, ErrNotFound):
		return User{}, ErrUnauthenticated
	case err != nil:
		return User{}, fmt.Errorf("auth: find user: %w", err)
	}
	return u, nil
}

func (s *Service) startSession(ctx context.Context, userID string) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("auth: read token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	now := s.now()
	err := s.sessions.Create(ctx, Session{
		TokenHash: hashToken(token),
		UserID:    userID,
		ExpiresAt: now.Add(SessionTTL),
		CreatedAt: now,
	})
	if err != nil {
		return "", fmt.Errorf("auth: create session: %w", err)
	}
	return token, nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// Users lists every account for the admin, oldest first.
func (s *Service) Users(ctx context.Context) ([]User, error) {
	users, err := s.users.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("auth: list users: %w", err)
	}
	return users, nil
}

// AccountInput is what an admin enters for an account. On update an empty Password keeps the
// current one.
type AccountInput struct {
	Email    string
	Password string
	Role     Role
}

// validateAccount normalizes the email and checks every field; the password may be empty only
// when keepPassword is set.
func validateAccount(in AccountInput, keepPassword bool) (AccountInput, error) {
	in.Email = NormalizeEmail(in.Email)
	fields := map[string]string{}
	checkEmail(fields, in.Email)
	if !keepPassword || in.Password != "" {
		checkPassword(fields, in.Password)
	}
	if !slices.Contains(Roles, in.Role) {
		fields["role"] = "Quyền không hợp lệ"
	}
	if len(fields) > 0 {
		return in, &ValidationError{Fields: fields}
	}
	return in, nil
}

// CreateAccount adds an account with the given role, without logging it in.
func (s *Service) CreateAccount(ctx context.Context, in AccountInput) (User, error) {
	in, err := validateAccount(in, false)
	if err != nil {
		return User{}, err
	}
	hash, err := HashPassword(in.Password)
	if err != nil {
		return User{}, err
	}
	u, err := s.users.Create(ctx, User{
		Email: in.Email, PasswordHash: hash, Role: in.Role, Timezone: DefaultTimezone, CreatedAt: s.now(),
	})
	if errors.Is(err, ErrEmailTaken) {
		return User{}, ErrEmailTaken
	}
	if err != nil {
		return User{}, fmt.Errorf("auth: create user: %w", err)
	}
	return u, nil
}

// UpdateAccount changes the email, role and (when given) password of an account. An admin may edit
// their own email and password but not their own role, so the app always keeps an admin.
func (s *Service) UpdateAccount(ctx context.Context, actorID, id string, in AccountInput) (User, error) {
	in, err := validateAccount(in, true)
	if err != nil {
		return User{}, err
	}
	cur, err := s.users.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return User{}, err
		}
		return User{}, fmt.Errorf("auth: find user: %w", err)
	}
	if id == actorID && in.Role != cur.Role {
		return User{}, ErrOwnRole
	}
	change := AccountChange{Email: in.Email, Role: in.Role}
	if in.Password != "" {
		if change.PasswordHash, err = HashPassword(in.Password); err != nil {
			return User{}, err
		}
	}
	if err := s.users.Update(ctx, id, change); err != nil {
		if errors.Is(err, ErrNotFound) || errors.Is(err, ErrEmailTaken) {
			return User{}, err
		}
		return User{}, fmt.Errorf("auth: update user: %w", err)
	}
	u, err := s.users.FindByID(ctx, id)
	if err != nil {
		return User{}, fmt.Errorf("auth: find user: %w", err)
	}
	return u, nil
}

// DeleteAccount removes an account with its sessions and learning data. An admin cannot delete
// their own account.
func (s *Service) DeleteAccount(ctx context.Context, actorID, id string) error {
	if id == actorID {
		return ErrOwnAccount
	}
	if err := s.users.Delete(ctx, id); err != nil {
		if errors.Is(err, ErrNotFound) {
			return err
		}
		return fmt.Errorf("auth: delete user: %w", err)
	}
	return nil
}
