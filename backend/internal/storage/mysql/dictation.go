package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/luongtran/luna/backend/internal/progress"
)

// DictationResults implements progress.DictationRepository on "dictation_results": the latest
// result of each sentence per (user, lesson, sentence), with the lesson revision it was checked on.
type DictationResults struct {
	db *sql.DB
}

var _ progress.DictationRepository = (*DictationResults)(nil)

// NewDictationResults returns the DictationResults repository.
func NewDictationResults(db *sql.DB) *DictationResults { return &DictationResults{db: db} }

// dictationSchema is the DDL of this domain (see migrate.go).
func dictationSchema() []string {
	return []string{`CREATE TABLE IF NOT EXISTS dictation_results (
  user_id         CHAR(24)     NOT NULL,
  lesson_id       CHAR(24)     NOT NULL,
  sentence_index  INT          NOT NULL,
  lesson_revision INT          NOT NULL,
  typed           TEXT         NOT NULL,
  correct_words   INT          NOT NULL,
  total_words     INT          NOT NULL,
  checked_at      DATETIME(6)  NOT NULL,
  PRIMARY KEY (user_id, lesson_id, sentence_index)
) ` + tableOptions}
}

// Upsert replaces the result of one sentence.
func (r *DictationResults) Upsert(ctx context.Context, userID, lessonID string, revision int, res progress.Result) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO dictation_results
  (user_id, lesson_id, sentence_index, lesson_revision, typed, correct_words, total_words, checked_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?) AS new
ON DUPLICATE KEY UPDATE lesson_revision = new.lesson_revision, typed = new.typed,
  correct_words = new.correct_words, total_words = new.total_words, checked_at = new.checked_at`,
		userID, lessonID, res.SentenceIndex, revision, res.Typed, res.CorrectWords, res.TotalWords, utc(res.CheckedAt))
	if err != nil {
		return fmt.Errorf("mysql upsert dictation result: %w", err)
	}
	return nil
}

// List returns the user's results for the lesson, ordered by sentence.
func (r *DictationResults) List(ctx context.Context, userID, lessonID string) ([]progress.StoredResult, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT sentence_index, lesson_revision, typed, correct_words, total_words, checked_at
FROM dictation_results WHERE user_id = ? AND lesson_id = ? ORDER BY sentence_index`, userID, lessonID)
	if err != nil {
		return nil, fmt.Errorf("mysql list dictation results: %w", err)
	}
	defer rows.Close()
	out := []progress.StoredResult{}
	for rows.Next() {
		var s progress.StoredResult
		if err := rows.Scan(&s.SentenceIndex, &s.Revision, &s.Typed, &s.CorrectWords, &s.TotalWords, &s.CheckedAt); err != nil {
			return nil, fmt.Errorf("mysql scan dictation result: %w", err)
		}
		s.CheckedAt = utc(s.CheckedAt)
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql list dictation results: %w", err)
	}
	return out, nil
}

// Totals sums the user's results over every lesson (one result per sentence), as the MongoDB
// repository does: the service filters by revision where it matters. Only results checked at or
// after since count (nil = all).
func (r *DictationResults) Totals(ctx context.Context, userID string, since *time.Time) (progress.DictationTotals, error) {
	var t progress.DictationTotals
	where, params := sinceClause("checked_at", since)
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*), CAST(COALESCE(SUM(correct_words), 0) AS SIGNED),
  CAST(COALESCE(SUM(total_words), 0) AS SIGNED), COUNT(DISTINCT lesson_id) FROM dictation_results WHERE user_id = ?`+where,
		append([]any{userID}, params...)...).
		Scan(&t.Sentences, &t.CorrectWords, &t.TotalWords, &t.Lessons)
	if err != nil {
		return progress.DictationTotals{}, fmt.Errorf("mysql dictation totals: %w", err)
	}
	return t, nil
}

// Delete removes the user's results for the lesson, of every revision.
func (r *DictationResults) Delete(ctx context.Context, userID, lessonID string) error {
	if _, err := r.db.ExecContext(ctx, "DELETE FROM dictation_results WHERE user_id = ? AND lesson_id = ?", userID, lessonID); err != nil {
		return fmt.Errorf("mysql delete dictation results: %w", err)
	}
	return nil
}
