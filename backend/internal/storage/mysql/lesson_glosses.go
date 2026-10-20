package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/luongtran/luna/backend/internal/lesson"
)

// LessonGlosses implements lesson.GlossRepository on "lesson_glosses": one row per lesson, the
// glosses of one revision as JSON (added 2026-10-20).
type LessonGlosses struct {
	db *sql.DB
}

var _ lesson.GlossRepository = (*LessonGlosses)(nil)

// NewLessonGlosses returns the LessonGlosses repository.
func NewLessonGlosses(db *sql.DB) *LessonGlosses { return &LessonGlosses{db: db} }

// lessonGlossesSchema is the DDL of this domain (see migrate.go).
func lessonGlossesSchema() []string {
	return []string{`CREATE TABLE IF NOT EXISTS lesson_glosses (
  lesson_id  CHAR(24)    NOT NULL PRIMARY KEY,
  revision   INT         NOT NULL,
  glosses    JSON        NOT NULL,
  updated_at DATETIME(6) NOT NULL
) ` + tableOptions}
}

type glossJSON struct {
	Text          string `json:"text"`
	Lemma         string `json:"lemma"`
	MeaningVi     string `json:"meaningVi"`
	SentenceIndex int    `json:"sentenceIndex"`
	POS           string `json:"pos,omitempty"`
	Phrase        string `json:"phrase,omitempty"`
	PhraseVi      string `json:"phraseVi,omitempty"`
}

// Get returns the lesson's glosses and the revision they were written for.
func (r *LessonGlosses) Get(ctx context.Context, lessonID string) (int, []lesson.Gloss, bool, error) {
	var revision int
	var raw []byte
	err := r.db.QueryRowContext(ctx, `SELECT revision, glosses FROM lesson_glosses WHERE lesson_id = ?`, lessonID).
		Scan(&revision, &raw)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil, false, nil
	}
	if err != nil {
		return 0, nil, false, fmt.Errorf("mysql get glosses: %w", err)
	}
	var docs []glossJSON
	if err := fromJSON(raw, &docs); err != nil {
		return 0, nil, false, fmt.Errorf("mysql decode glosses: %w", err)
	}
	out := make([]lesson.Gloss, len(docs))
	for i, d := range docs {
		out[i] = lesson.Gloss(d)
	}
	return revision, out, true, nil
}

// Save replaces the lesson's glosses.
func (r *LessonGlosses) Save(ctx context.Context, lessonID string, revision int, glosses []lesson.Gloss) error {
	docs := make([]glossJSON, len(glosses))
	for i, g := range glosses {
		docs[i] = glossJSON(g)
	}
	raw, err := toJSON(docs)
	if err != nil {
		return fmt.Errorf("mysql encode glosses: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO lesson_glosses (lesson_id, revision, glosses, updated_at)
VALUES (?, ?, ?, ?) AS new
ON DUPLICATE KEY UPDATE revision = new.revision, glosses = new.glosses, updated_at = new.updated_at`,
		lessonID, revision, string(raw), utc(time.Now()))
	if err != nil {
		return fmt.Errorf("mysql save glosses: %w", err)
	}
	return nil
}

// Revisions gives the revision of each lesson's glosses.
func (r *LessonGlosses) Revisions(ctx context.Context) (map[string]int, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT lesson_id, revision FROM lesson_glosses`)
	if err != nil {
		return nil, fmt.Errorf("mysql gloss revisions: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := map[string]int{}
	for rows.Next() {
		var id string
		var rev int
		if err := rows.Scan(&id, &rev); err != nil {
			return nil, fmt.Errorf("mysql scan gloss revision: %w", err)
		}
		out[id] = rev
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql gloss revisions: %w", err)
	}
	return out, nil
}
