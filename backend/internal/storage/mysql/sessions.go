package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/luongtran/luna/backend/internal/auth"
)

// Sessions implements auth.SessionRepository on the "sessions" table. MySQL has no TTL index, so a
// session past expires_at is treated as missing on lookup, and expired rows are purged whenever a
// new session is created.
type Sessions struct {
	db *sql.DB
}

// NewSessions returns the Sessions repository.
func NewSessions(db *sql.DB) *Sessions { return &Sessions{db: db} }

var _ auth.SessionRepository = (*Sessions)(nil)

// sessionsSchema is the DDL of this domain (see migrate.go).
func sessionsSchema() []string {
	return []string{`CREATE TABLE IF NOT EXISTS sessions (
	id         CHAR(24)     NOT NULL PRIMARY KEY,
	token_hash VARCHAR(191) NOT NULL,
	user_id    CHAR(24)     NOT NULL,
	expires_at DATETIME(6)  NOT NULL,
	created_at DATETIME(6)  NOT NULL,
	UNIQUE KEY sessions_token_hash (token_hash),
	KEY sessions_expires_at (expires_at)
) ` + tableOptions}
}

// Create inserts a session and purges the expired ones.
func (r *Sessions) Create(ctx context.Context, s auth.Session) error {
	if !sessionsIsID(s.UserID) {
		return fmt.Errorf("mysql session user id: %q is not a valid id", s.UserID)
	}
	if _, err := r.db.ExecContext(ctx, "DELETE FROM sessions WHERE expires_at <= ?", time.Now().UTC()); err != nil {
		return fmt.Errorf("mysql purge sessions: %w", err)
	}
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO sessions (id, token_hash, user_id, expires_at, created_at) VALUES (?, ?, ?, ?, ?)",
		newID(), s.TokenHash, s.UserID, utc(s.ExpiresAt), utc(s.CreatedAt))
	if err != nil {
		return fmt.Errorf("mysql insert session: %w", err)
	}
	return nil
}

// FindByTokenHash returns auth.ErrNotFound when no live session has this hash.
func (r *Sessions) FindByTokenHash(ctx context.Context, hash string) (auth.Session, error) {
	var s auth.Session
	err := r.db.QueryRowContext(ctx,
		"SELECT token_hash, user_id, expires_at, created_at FROM sessions WHERE token_hash = ? AND expires_at > ?",
		hash, time.Now().UTC()).Scan(&s.TokenHash, &s.UserID, &s.ExpiresAt, &s.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return auth.Session{}, auth.ErrNotFound
	}
	if err != nil {
		return auth.Session{}, fmt.Errorf("mysql find session: %w", err)
	}
	s.ExpiresAt, s.CreatedAt = s.ExpiresAt.UTC(), s.CreatedAt.UTC()
	return s, nil
}

// Delete removes the session; a missing session is not an error.
func (r *Sessions) Delete(ctx context.Context, hash string) error {
	if _, err := r.db.ExecContext(ctx, "DELETE FROM sessions WHERE token_hash = ?", hash); err != nil {
		return fmt.Errorf("mysql delete session: %w", err)
	}
	return nil
}

// sessionsIsID reports whether s has the shape of an id made by newID (24 lowercase hex digits).
func sessionsIsID(s string) bool {
	if len(s) != 24 {
		return false
	}
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}
