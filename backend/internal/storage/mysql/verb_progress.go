package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/luongtran/luna/backend/internal/verbs"
)

// VerbProgress implements verbs.Repository on "verb_progress": one row per (user, verb).
type VerbProgress struct {
	db *sql.DB
}

var _ verbs.Repository = (*VerbProgress)(nil)

// NewVerbProgress returns the VerbProgress repository.
func NewVerbProgress(db *sql.DB) *VerbProgress { return &VerbProgress{db: db} }

// verbProgressSchema is the DDL of this domain (see migrate.go). due_at is NULL for a verb not
// learned.
func verbProgressSchema() []string {
	return []string{`CREATE TABLE IF NOT EXISTS verb_progress (
  user_id     CHAR(24)    NOT NULL,
  base        VARCHAR(40) NOT NULL,
  level       INT         NOT NULL,
  due_at      DATETIME(6) NULL,
  right_count INT         NOT NULL,
  wrong_count INT         NOT NULL,
  updated_at  DATETIME(6) NOT NULL,
  PRIMARY KEY (user_id, base)
) ` + tableOptions}
}

// verbLearnedSchema adds when each verb was confirmed as learned (NULL: not learned). A verb
// already in a review box before this column existed counts as learned from its last update.
func verbLearnedSchema() []string {
	return []string{
		`ALTER TABLE verb_progress ADD COLUMN learned_at DATETIME(6) NULL AFTER base`,
		`UPDATE verb_progress SET learned_at = updated_at WHERE level > 0 AND learned_at IS NULL`,
	}
}

// List returns every record of the user, by verb.
func (r *VerbProgress) List(ctx context.Context, userID string) ([]verbs.Progress, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT base, learned_at, level, due_at, right_count, wrong_count, updated_at
FROM verb_progress WHERE user_id = ? ORDER BY base`, userID)
	if err != nil {
		return nil, fmt.Errorf("mysql list verb progress: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []verbs.Progress
	for rows.Next() {
		p := verbs.Progress{UserID: userID}
		var learned, due sql.NullTime
		if err := rows.Scan(&p.Base, &learned, &p.Level, &due, &p.Right, &p.Wrong, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("mysql scan verb progress: %w", err)
		}
		if learned.Valid {
			p.LearnedAt = utc(learned.Time)
		}
		if due.Valid {
			p.DueAt = utc(due.Time)
		}
		p.UpdatedAt = utc(p.UpdatedAt)
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql list verb progress: %w", err)
	}
	return out, nil
}

// Save writes the records in one statement, each replacing the user's row of its verb.
func (r *VerbProgress) Save(ctx context.Context, records []verbs.Progress) error {
	if len(records) == 0 {
		return nil
	}
	orNull := func(zero bool, v any) any {
		if zero {
			return nil
		}
		return v
	}
	values := make([]string, len(records))
	args := make([]any, 0, len(records)*8)
	for i, p := range records {
		values[i] = "(?, ?, ?, ?, ?, ?, ?, ?)"
		args = append(args, p.UserID, p.Base, orNull(p.LearnedAt.IsZero(), utc(p.LearnedAt)), p.Level,
			orNull(p.DueAt.IsZero(), utc(p.DueAt)), p.Right, p.Wrong, utc(p.UpdatedAt))
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO verb_progress (user_id, base, learned_at, level, due_at, right_count, wrong_count, updated_at)
VALUES `+strings.Join(values, ", ")+` AS new
ON DUPLICATE KEY UPDATE learned_at = new.learned_at, level = new.level, due_at = new.due_at,
  right_count = new.right_count, wrong_count = new.wrong_count, updated_at = new.updated_at`, args...)
	if err != nil {
		return fmt.Errorf("mysql save verb progress: %w", err)
	}
	return nil
}
