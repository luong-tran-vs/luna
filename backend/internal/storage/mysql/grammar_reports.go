package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/luongtran/luna/backend/internal/grammar"
)

// grammarReportsSchema is the DDL of this domain (see migrate.go): a learner's reports on grammar exercises,
// one per learner and exercise.
func grammarReportsSchema() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS grammar_reports (
  id CHAR(24) NOT NULL PRIMARY KEY,
  user_id CHAR(24) NOT NULL,
  point_id VARCHAR(64) NOT NULL,
  exercise_id VARCHAR(16) NOT NULL,
  reason VARCHAR(16) NOT NULL,
  note TEXT NOT NULL,
  status VARCHAR(16) NOT NULL,
  created_at DATETIME(6) NOT NULL,
  updated_at DATETIME(6) NOT NULL,
  resolved_at DATETIME(6) NULL,
  UNIQUE KEY uq_grammar_reports_user_exercise (user_id, point_id, exercise_id),
  KEY idx_grammar_reports_status (status, updated_at)
) ` + tableOptions,
	}
}

// GrammarReports implements grammar.ReportRepository on "grammar_reports".
type GrammarReports struct {
	db *sql.DB
}

// NewGrammarReports returns the grammar report repository.
func NewGrammarReports(db *sql.DB) *GrammarReports { return &GrammarReports{db: db} }

var _ grammar.ReportRepository = (*GrammarReports)(nil)

// Upsert inserts the report, or reopens the learner's existing one (keeping its id and creation time) with
// the new reason and note.
func (r *GrammarReports) Upsert(ctx context.Context, rep grammar.Report) error {
	updated := rep.UpdatedAt
	if updated.IsZero() {
		updated = time.Now()
	}
	created := rep.CreatedAt
	if created.IsZero() {
		created = updated
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO grammar_reports (id, user_id, point_id, exercise_id, reason, note, status, created_at, updated_at, resolved_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NULL)
ON DUPLICATE KEY UPDATE reason = VALUES(reason), note = VALUES(note), status = VALUES(status),
  updated_at = VALUES(updated_at), resolved_at = NULL`,
		newID(), rep.UserID, rep.PointID, rep.ExerciseID, string(rep.Reason), rep.Note, string(grammar.ReportOpen),
		utc(created), utc(updated))
	if err != nil {
		return fmt.Errorf("mysql upsert grammar report: %w", err)
	}
	return nil
}

// ListOpen returns the open reports, newest first.
func (r *GrammarReports) ListOpen(ctx context.Context) ([]grammar.Report, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, point_id, exercise_id, reason, note, status, created_at, updated_at, resolved_at
FROM grammar_reports WHERE status = ? ORDER BY updated_at DESC, id DESC`, string(grammar.ReportOpen))
	if err != nil {
		return nil, fmt.Errorf("mysql find grammar reports: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := []grammar.Report{}
	for rows.Next() {
		var (
			rep            grammar.Report
			reason, status string
			resolved       sql.NullTime
		)
		if err := rows.Scan(&rep.ID, &rep.UserID, &rep.PointID, &rep.ExerciseID, &reason, &rep.Note, &status,
			&rep.CreatedAt, &rep.UpdatedAt, &resolved); err != nil {
			return nil, fmt.Errorf("mysql read grammar reports: %w", err)
		}
		rep.Reason, rep.Status = grammar.ReportReason(reason), grammar.ReportStatus(status)
		rep.CreatedAt, rep.UpdatedAt, rep.ResolvedAt = rep.CreatedAt.UTC(), rep.UpdatedAt.UTC(), timeOf(resolved)
		out = append(out, rep)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql read grammar reports: %w", err)
	}
	return out, nil
}

// Resolve marks the open reports on the exercise resolved in one statement and returns how many.
func (r *GrammarReports) Resolve(ctx context.Context, pointID, exerciseID string, at time.Time) (int, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE grammar_reports SET status = ?, resolved_at = ? WHERE point_id = ? AND exercise_id = ? AND status = ?`,
		string(grammar.ReportResolved), utc(at), pointID, exerciseID, string(grammar.ReportOpen))
	if err != nil {
		return 0, fmt.Errorf("mysql resolve grammar reports: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("mysql resolve grammar reports: %w", err)
	}
	return int(n), nil
}
