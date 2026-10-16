package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/luongtran/luna/backend/internal/vocab"
)

// LessonResults implements vocab.LessonResultRepository on "lesson_results": the latest lesson
// review round of each lesson per (user, lesson), the ids of the questions got wrong as JSON.
type LessonResults struct {
	db *sql.DB
}

var _ vocab.LessonResultRepository = (*LessonResults)(nil)

// NewLessonResults returns the LessonResults repository.
func NewLessonResults(db *sql.DB) *LessonResults { return &LessonResults{db: db} }

// lessonResultsSchema is the DDL of this domain (see migrate.go).
func lessonResultsSchema() []string {
	return []string{`CREATE TABLE IF NOT EXISTS lesson_results (
  user_id     CHAR(24)    NOT NULL,
  lesson_id   CHAR(24)    NOT NULL,
  correct     INT         NOT NULL,
  total       INT         NOT NULL,
  wrong       JSON        NOT NULL,
  reviewed_at DATETIME(6) NOT NULL,
  PRIMARY KEY (user_id, lesson_id)
) ` + tableOptions}
}

// Save replaces the user's result for the lesson.
func (r *LessonResults) Save(ctx context.Context, res vocab.LessonResult) error {
	wrong := res.Wrong
	if wrong == nil {
		wrong = []string{}
	}
	raw, err := json.Marshal(wrong)
	if err != nil {
		return fmt.Errorf("mysql encode lesson result: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO lesson_results (user_id, lesson_id, correct, total, wrong, reviewed_at)
VALUES (?, ?, ?, ?, ?, ?) AS new
ON DUPLICATE KEY UPDATE correct = new.correct, total = new.total, wrong = new.wrong, reviewed_at = new.reviewed_at`,
		res.UserID, res.LessonID, res.Correct, res.Total, string(raw), utc(res.ReviewedAt))
	if err != nil {
		return fmt.Errorf("mysql save lesson result: %w", err)
	}
	return nil
}

// List returns every result of the user.
func (r *LessonResults) List(ctx context.Context, userID string) ([]vocab.LessonResult, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT lesson_id, correct, total, wrong, reviewed_at FROM lesson_results WHERE user_id = ?`, userID)
	if err != nil {
		return nil, fmt.Errorf("mysql list lesson results: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []vocab.LessonResult
	for rows.Next() {
		res := vocab.LessonResult{UserID: userID}
		var raw []byte
		if err := rows.Scan(&res.LessonID, &res.Correct, &res.Total, &raw, &res.ReviewedAt); err != nil {
			return nil, fmt.Errorf("mysql scan lesson result: %w", err)
		}
		if err := json.Unmarshal(raw, &res.Wrong); err != nil {
			return nil, fmt.Errorf("mysql decode lesson result: %w", err)
		}
		res.ReviewedAt = utc(res.ReviewedAt)
		out = append(out, res)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql list lesson results: %w", err)
	}
	return out, nil
}
