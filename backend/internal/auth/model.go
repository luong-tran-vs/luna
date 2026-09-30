// Package auth manages accounts, passwords and login sessions.
package auth

import (
	"errors"
	"fmt"
	"time"
)

// Role is the permission level of an account.
type Role string

const (
	RoleAdmin   Role = "admin"
	RoleLearner Role = "learner"
)

// User is a registered account.
type User struct {
	ID           string
	Email        string
	PasswordHash string
	Role         Role
	Timezone     string
	CreatedAt    time.Time
}

// Session is a login on one device. Only the SHA-256 hash of the token is stored.
type Session struct {
	TokenHash string
	UserID    string
	ExpiresAt time.Time
	CreatedAt time.Time
}

var (
	ErrNotFound           = errors.New("auth: not found")
	ErrEmailTaken         = errors.New("auth: email already registered")
	ErrInvalidCredentials = errors.New("auth: invalid email or password")
	ErrUnauthenticated    = errors.New("auth: no valid session")
)

// ValidationError lists invalid input fields with Vietnamese messages for the user.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("auth: invalid input %v", e.Fields)
}

// LockedError means login is temporarily blocked for an email.
type LockedError struct {
	RetryAfter time.Duration
}

func (e *LockedError) Error() string {
	return fmt.Sprintf("auth: login locked, retry after %s", e.RetryAfter)
}
