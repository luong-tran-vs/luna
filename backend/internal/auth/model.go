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
	RoleAdmin Role = "admin"
	// RoleMember studies every lesson.
	RoleMember Role = "member"
	// RoleGuest studies only the first lesson of each roadmap; new accounts start as guests.
	RoleGuest Role = "guest"
	// roleLearner is the learner role before guests existed; it reads as RoleMember.
	roleLearner Role = "learner"
)

// Roles lists the roles an admin can give, from most to least access.
var Roles = []Role{RoleAdmin, RoleMember, RoleGuest}

// ParseRole reads a stored role: the old "learner" is a member, anything unknown a guest.
func ParseRole(s string) Role {
	switch r := Role(s); r {
	case RoleAdmin, RoleMember, RoleGuest:
		return r
	case roleLearner:
		return RoleMember
	}
	return RoleGuest
}

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
	// ErrOwnRole means an admin tried to change their own role (they could lock themselves out).
	ErrOwnRole = errors.New("auth: cannot change own role")
	// ErrOwnAccount means an admin tried to delete their own account.
	ErrOwnAccount = errors.New("auth: cannot delete own account")
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
