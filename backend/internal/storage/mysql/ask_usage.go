package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/luongtran/luna/backend/internal/lesson"
)

// AskUsage implements lesson.AskUsageRepository on "ask_usage" (F9): the words each learner asked
// the AI about, one row per (user_id, lesson_id, text_lower).
type AskUsage struct {
	db *sql.DB
}

// NewAskUsage returns the AskUsage repository.
func NewAskUsage(db *sql.DB) *AskUsage { return &AskUsage{db: db} }

var _ lesson.AskUsageRepository = (*AskUsage)(nil)

// askUsageSchema is the DDL of this domain (see migrate.go).
func askUsageSchema() []string {
	return []string{`CREATE TABLE IF NOT EXISTS ask_usage (
		user_id    CHAR(24)     NOT NULL,
		lesson_id  CHAR(24)     NOT NULL,
		text_lower VARCHAR(191) NOT NULL,
		created_at DATETIME(6)  NOT NULL,
		PRIMARY KEY (user_id, lesson_id, text_lower)
	) ` + tableOptions}
}

// Asked lists the texts the user asked about in the lesson, oldest first.
func (r *AskUsage) Asked(ctx context.Context, userID, lessonID string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT text_lower FROM ask_usage WHERE user_id = ? AND lesson_id = ? ORDER BY created_at, text_lower", userID, lessonID)
	if err != nil {
		return nil, fmt.Errorf("mysql asked words: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := []string{}
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, fmt.Errorf("mysql read asked words: %w", err)
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql read asked words: %w", err)
	}
	return out, nil
}

// Add records an asked text; recording it again changes nothing.
func (r *AskUsage) Add(ctx context.Context, userID, lessonID, text string, at time.Time) error {
	_, err := r.db.ExecContext(ctx,
		"INSERT IGNORE INTO ask_usage (user_id, lesson_id, text_lower, created_at) VALUES (?, ?, ?, ?)",
		userID, lessonID, text, utc(at))
	if err != nil {
		return fmt.Errorf("mysql record ask: %w", err)
	}
	return nil
}
