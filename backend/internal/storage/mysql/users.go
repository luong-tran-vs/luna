package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/luongtran/luna/backend/internal/auth"
	"github.com/luongtran/luna/backend/internal/export"
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
	u.Role = auth.ParseRole(role)
	u.CreatedAt = created.UTC()
	return u, nil
}

// List returns every account, oldest first.
func (r *Users) List(ctx context.Context) ([]auth.User, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT "+usersCols+" FROM users ORDER BY created_at, id")
	if err != nil {
		return nil, fmt.Errorf("mysql list users: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := []auth.User{}
	for rows.Next() {
		var (
			u       auth.User
			role    string
			created time.Time
		)
		if err := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, &role, &u.Timezone, &created); err != nil {
			return nil, fmt.Errorf("mysql read users: %w", err)
		}
		u.Role, u.CreatedAt = auth.ParseRole(role), created.UTC()
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql read users: %w", err)
	}
	return out, nil
}

// Update changes the email, role and (when PasswordHash is set) password of an account. MySQL
// counts only changed rows, so an unchanged account is told apart from a missing one by looking
// it up.
func (r *Users) Update(ctx context.Context, id string, c auth.AccountChange) error {
	set, params := "email = ?, role = ?", []any{c.Email, string(c.Role)}
	if c.PasswordHash != "" {
		set += ", password_hash = ?"
		params = append(params, c.PasswordHash)
	}
	res, err := r.db.ExecContext(ctx, "UPDATE users SET "+set+" WHERE id = ?", append(params, id)...)
	if isDuplicate(err) {
		return auth.ErrEmailTaken
	}
	if err != nil {
		return fmt.Errorf("mysql update user: %w", err)
	}
	if ok, err := changed(res); err != nil || ok {
		return err
	}
	_, err = r.FindByID(ctx, id)
	return err
}

// userData lists the tables holding a user's own rows (by user_id); they go with the account.
// Grammar reports stay for the admin.
var userData = []string{
	"sessions", export.CollCards, export.CollReviewLogs, export.CollGoals, export.CollLessonProgress,
	export.CollStudyDays, export.CollDictationResults, export.CollReadingAnswers, export.CollWritings,
	export.CollGrammarProgress,
}

// Delete removes an account and its rows in one transaction.
func (r *Users) Delete(ctx context.Context, id string) error {
	if _, err := r.FindByID(ctx, id); err != nil {
		return err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("mysql begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, table := range userData {
		if _, err := tx.ExecContext(ctx, "DELETE FROM "+table+" WHERE user_id = ?", id); err != nil {
			return fmt.Errorf("mysql delete user %s: %w", table, err)
		}
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM users WHERE id = ?", id); err != nil {
		return fmt.Errorf("mysql delete user: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("mysql commit: %w", err)
	}
	return nil
}
