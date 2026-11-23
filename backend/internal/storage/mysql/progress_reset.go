package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/luongtran/luna/backend/internal/progressreset"
)

// ProgressReset implements progressreset.Repository over the learners' progress tables.
type ProgressReset struct {
	db *sql.DB
}

// NewProgressReset returns the progress reset repository.
func NewProgressReset(db *sql.DB) *ProgressReset { return &ProgressReset{db: db} }

var _ progressreset.Repository = (*ProgressReset)(nil)

// resetStatements are what a part runs, each with the user id (and the time for the cards).
var resetStatements = map[progressreset.Part][]string{
	progressreset.PartLessons: {
		`DELETE FROM lesson_progress WHERE user_id = ?`,
		`DELETE FROM reading_answers WHERE user_id = ?`,
		`DELETE FROM dictation_results WHERE user_id = ?`,
		`DELETE FROM lesson_results WHERE user_id = ?`,
	},
	progressreset.PartGrammar: {`DELETE FROM grammar_progress WHERE user_id = ?`},
	progressreset.PartVocab: {
		`UPDATE cards SET due = ?, stability = 0, difficulty = 0, elapsed_days = 0, scheduled_days = 0, reps = 0,
  lapses = 0, state = 0, last_review = NULL WHERE user_id = ?`,
		`DELETE FROM review_logs WHERE user_id = ?`,
	},
	progressreset.PartToeic: {
		`DELETE g FROM toeic_attempt_groups g JOIN toeic_attempts a ON a.id = g.attempt_id WHERE a.user_id = ?`,
		`DELETE FROM toeic_attempts WHERE user_id = ?`,
		`DELETE FROM toeic_mistakes WHERE user_id = ?`,
	},
}

// Reset removes the user's progress in part, in one transaction.
func (r *ProgressReset) Reset(ctx context.Context, userID string, part progressreset.Part, now time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("mysql: reset begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, q := range resetStatements[part] {
		args := []any{userID}
		if strings.HasPrefix(q, "UPDATE cards") {
			args = []any{now.UTC(), userID}
		}
		if _, err := tx.ExecContext(ctx, q, args...); err != nil {
			return fmt.Errorf("mysql: reset %s: %w", part, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("mysql: reset commit: %w", err)
	}
	return nil
}
