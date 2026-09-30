package auth

import "context"

// UserRepository stores accounts. Implementations live in internal/storage.
type UserRepository interface {
	// Create stores u and returns it with its ID; ErrEmailTaken if the email exists.
	Create(ctx context.Context, u User) (User, error)
	// FindByEmail returns ErrNotFound when no account has this (normalized) email.
	FindByEmail(ctx context.Context, email string) (User, error)
	// FindByID returns ErrNotFound when the account does not exist.
	FindByID(ctx context.Context, id string) (User, error)
	Count(ctx context.Context) (int64, error)
}

// SessionRepository stores login sessions by token hash.
type SessionRepository interface {
	Create(ctx context.Context, s Session) error
	// FindByTokenHash returns ErrNotFound when no session has this hash.
	FindByTokenHash(ctx context.Context, hash string) (Session, error)
	// Delete removes the session; deleting a missing session is not an error.
	Delete(ctx context.Context, hash string) error
}
