package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/luongtran/luna/backend/internal/settings"
)

// Settings implements settings.Repository on the settings columns of the "users" table.
type Settings struct {
	db *sql.DB
}

// NewSettings returns the Settings repository.
func NewSettings(db *sql.DB) *Settings { return &Settings{db: db} }

var _ settings.Repository = (*Settings)(nil)

// settingsSchema is the DDL of this domain (see migrate.go). The settings live in the columns
// theme, daily_review_limit and timezone of the users table, created by usersSchema.
func settingsSchema() []string { return nil }

// Get returns the user's stored settings; empty/zero values mean "not set".
func (r *Settings) Get(ctx context.Context, userID string) (settings.Settings, error) {
	return settingsGet(ctx, r.db, userID)
}

func settingsGet(ctx context.Context, q execer, userID string) (settings.Settings, error) {
	var (
		s     settings.Settings
		theme string
	)
	err := q.QueryRowContext(ctx, "SELECT theme, daily_review_limit, timezone FROM users WHERE id = ?", userID).
		Scan(&theme, &s.DailyReviewLimit, &s.Timezone)
	if errors.Is(err, sql.ErrNoRows) {
		return settings.Settings{}, settings.ErrNotFound
	}
	if err != nil {
		return settings.Settings{}, fmt.Errorf("mysql find settings: %w", err)
	}
	s.Theme = settings.Theme(theme)
	return s, nil
}

// Update sets the non-nil fields of p and returns the stored settings.
func (r *Settings) Update(ctx context.Context, userID string, p settings.Patch) (settings.Settings, error) {
	var (
		sets []string
		vals []any
	)
	if p.Theme != nil {
		sets, vals = append(sets, "theme = ?"), append(vals, string(*p.Theme))
	}
	if p.DailyReviewLimit != nil {
		sets, vals = append(sets, "daily_review_limit = ?"), append(vals, *p.DailyReviewLimit)
	}
	if p.Timezone != nil {
		sets, vals = append(sets, "timezone = ?"), append(vals, *p.Timezone)
	}
	if len(sets) == 0 {
		return r.Get(ctx, userID)
	}
	var out settings.Settings
	err := inTx(ctx, r.db, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, "UPDATE users SET "+strings.Join(sets, ", ")+" WHERE id = ?", append(vals, userID)...)
		if err != nil {
			return fmt.Errorf("mysql update settings: %w", err)
		}
		ok, err := changed(res)
		if err != nil {
			return err
		}
		if !ok {
			return settings.ErrNotFound
		}
		out, err = settingsGet(ctx, tx, userID)
		return err
	})
	if err != nil {
		return settings.Settings{}, err
	}
	return out, nil
}
