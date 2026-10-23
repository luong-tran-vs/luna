package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/luongtran/luna/backend/internal/curriculum"
)

// Curricula implements curriculum.Repository on "curricula" (added 2026-10-23): one row per level,
// the approved course and the draft each a JSON body.
type Curricula struct {
	db *sql.DB
}

var _ curriculum.Repository = (*Curricula)(nil)

// NewCurricula returns the Curricula repository.
func NewCurricula(db *sql.DB) *Curricula { return &Curricula{db: db} }

// curriculaSchema is the DDL of this domain (see migrate.go).
func curriculaSchema() []string {
	return []string{`CREATE TABLE IF NOT EXISTS curricula (
  level       VARCHAR(8)  NOT NULL PRIMARY KEY,
  approved    JSON        NULL,
  draft       JSON        NULL,
  approved_at DATETIME(6) NULL,
  updated_at  DATETIME(6) NOT NULL
) ` + tableOptions}
}

func planBody(p *curriculum.Plan) (any, error) {
	if p == nil {
		return nil, nil
	}
	data, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return string(data), nil
}

func planOf(body sql.NullString, level string) (*curriculum.Plan, error) {
	if !body.Valid {
		return nil, nil
	}
	var p curriculum.Plan
	if err := json.Unmarshal([]byte(body.String), &p); err != nil {
		return nil, err
	}
	p.Level = level
	return &p, nil
}

// Get returns the course of level, or curriculum.ErrNoPlan.
func (r *Curricula) Get(ctx context.Context, level string) (curriculum.Course, error) {
	var (
		approved, draft sql.NullString
		approvedAt      sql.NullTime
		updatedAt       time.Time
	)
	err := r.db.QueryRowContext(ctx,
		`SELECT approved, draft, approved_at, updated_at FROM curricula WHERE level = ?`, level,
	).Scan(&approved, &draft, &approvedAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return curriculum.Course{}, curriculum.ErrNoPlan
	}
	if err != nil {
		return curriculum.Course{}, fmt.Errorf("mysql: get curriculum: %w", err)
	}
	c := curriculum.Course{Level: level, UpdatedAt: updatedAt.UTC()}
	if c.Approved, err = planOf(approved, level); err != nil {
		return curriculum.Course{}, fmt.Errorf("mysql: decode curriculum: %w", err)
	}
	if c.Draft, err = planOf(draft, level); err != nil {
		return curriculum.Course{}, fmt.Errorf("mysql: decode curriculum draft: %w", err)
	}
	if approvedAt.Valid {
		c.ApprovedAt = approvedAt.Time.UTC()
	}
	return c, nil
}

// Save writes the course of c.Level, replacing it.
func (r *Curricula) Save(ctx context.Context, c curriculum.Course) error {
	approved, err := planBody(c.Approved)
	if err != nil {
		return fmt.Errorf("mysql: encode curriculum: %w", err)
	}
	draft, err := planBody(c.Draft)
	if err != nil {
		return fmt.Errorf("mysql: encode curriculum draft: %w", err)
	}
	var approvedAt any
	if !c.ApprovedAt.IsZero() {
		approvedAt = c.ApprovedAt.UTC()
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO curricula (level, approved, draft, approved_at, updated_at)
VALUES (?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE approved = VALUES(approved), draft = VALUES(draft),
  approved_at = VALUES(approved_at), updated_at = VALUES(updated_at)`,
		c.Level, approved, draft, approvedAt, c.UpdatedAt.UTC())
	if err != nil {
		return fmt.Errorf("mysql: save curriculum: %w", err)
	}
	return nil
}
