package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/luongtran/luna/backend/internal/auth"
)

// Users implements auth.UserRepository on the "users" table. The same table also holds the learner's
// settings (theme, daily_review_limit, timezone); see Settings.
type Users struct {
	db *sql.DB
}

// NewUsers returns the Users repository.
func NewUsers(db *sql.DB) *Users { return &Users{db: db} }

var _ auth.UserRepository = (*Users)(nil)

// usersSchema is the DDL of this domain (see migrate.go).
func usersSchema() []string {
	return []string{`CREATE TABLE IF NOT EXISTS users (
	id                 CHAR(24)     NOT NULL PRIMARY KEY,
	email              VARCHAR(191) NOT NULL,
	password_hash      VARCHAR(255) NOT NULL,
	role               VARCHAR(32)  NOT NULL,
	theme              VARCHAR(16)  NOT NULL DEFAULT '',
	daily_review_limit INT          NOT NULL DEFAULT 0,
	timezone           VARCHAR(64)  NOT NULL DEFAULT '',
	created_at         DATETIME(6)  NOT NULL,
	UNIQUE KEY users_email (email)
) ` + tableOptions}
}

const usersCols = "id, email, password_hash, role, timezone, created_at"

// Create inserts u; a duplicate email returns auth.ErrEmailTaken.
func (r *Users) Create(ctx context.Context, u auth.User) (auth.User, error) {
	u.ID = newID()
	u.CreatedAt = utc(u.CreatedAt)
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO users (id, email, password_hash, role, timezone, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		u.ID, u.Email, u.PasswordHash, string(u.Role), u.Timezone, u.CreatedAt)
	if err != nil {
		if isDuplicate(err) {
			return auth.User{}, auth.ErrEmailTaken
		}
		return auth.User{}, fmt.Errorf("mysql insert user: %w", err)
	}
	return u, nil
}

// FindByEmail looks up a normalized email.
func (r *Users) FindByEmail(ctx context.Context, email string) (auth.User, error) {
	return r.usersFindOne(ctx, "SELECT "+usersCols+" FROM users WHERE email = ?", email)
}

// FindByID looks up an id; a malformed id is simply not found.
func (r *Users) FindByID(ctx context.Context, id string) (auth.User, error) {
	return r.usersFindOne(ctx, "SELECT "+usersCols+" FROM users WHERE id = ?", id)
}

// Count returns the number of accounts.
func (r *Users) Count(ctx context.Context) (int64, error) {
	var n int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&n); err != nil {
		return 0, fmt.Errorf("mysql count users: %w", err)
	}
	return n, nil
}

func (r *Users) usersFindOne(ctx context.Context, query string, arg any) (auth.User, error) {
	var (
		u       auth.User
		role    string
		created time.Time
	)
	err := r.db.QueryRowContext(ctx, query, arg).Scan(&u.ID, &u.Email, &u.PasswordHash, &role, &u.Timezone, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return auth.User{}, auth.ErrNotFound
	}
	if err != nil {
		return auth.User{}, fmt.Errorf("mysql find user: %w", err)
	}
	u.Role = auth.Role(role)
	u.CreatedAt = created.UTC()
	return u, nil
}
