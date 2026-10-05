package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/luongtran/luna/backend/internal/lesson"
)

// AILookups implements lesson.AskRepository on "ai_lookups" (F9): AI explanations shared by
// every learner, one per (lesson_id, revision, sentence_index, text_lower).
type AILookups struct {
	db *sql.DB
}

// NewAILookups returns the AILookups repository.
func NewAILookups(db *sql.DB) *AILookups { return &AILookups{db: db} }

var _ lesson.AskRepository = (*AILookups)(nil)

// aiLookupsSchema is the DDL of this domain (see migrate.go).
func aiLookupsSchema() []string {
	return []string{`CREATE TABLE IF NOT EXISTS ai_lookups (
		id             CHAR(24)     NOT NULL PRIMARY KEY,
		lesson_id      CHAR(24)     NOT NULL,
		revision       INT          NOT NULL,
		sentence_index INT          NOT NULL,
		text_lower     VARCHAR(191) NOT NULL,
		lemma          VARCHAR(191) NOT NULL,
		meaning_vi     TEXT         NOT NULL,
		note_vi        TEXT         NOT NULL,
		created_at     DATETIME(6)  NOT NULL,
		UNIQUE KEY ai_lookups_key (lesson_id, revision, sentence_index, text_lower)
	) ` + tableOptions}
}

// aiLookupValidID reports whether id has the shape of an id made by newID; anything else has no row.
func aiLookupValidID(id string) bool {
	if len(id) != 24 {
		return false
	}
	for _, c := range id {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

// Get returns the stored explanation for key.
func (r *AILookups) Get(ctx context.Context, key lesson.AskKey) (lesson.AskResult, bool, error) {
	if !aiLookupValidID(key.LessonID) {
		return lesson.AskResult{}, false, nil
	}
	res := lesson.AskResult{AskKey: key}
	err := r.db.QueryRowContext(ctx,
		"SELECT lemma, meaning_vi, note_vi, created_at FROM ai_lookups WHERE lesson_id = ? AND revision = ? AND sentence_index = ? AND text_lower = ?",
		key.LessonID, key.Revision, key.SentenceIndex, key.Text).
		Scan(&res.Lemma, &res.MeaningVi, &res.NoteVi, &res.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return lesson.AskResult{}, false, nil
	}
	if err != nil {
		return lesson.AskResult{}, false, fmt.Errorf("mysql find ai lookup: %w", err)
	}
	res.CreatedAt = res.CreatedAt.UTC()
	return res, true, nil
}

// Put stores an explanation; when the key is already stored it returns that one.
func (r *AILookups) Put(ctx context.Context, res lesson.AskResult) (lesson.AskResult, error) {
	if !aiLookupValidID(res.LessonID) {
		return lesson.AskResult{}, lesson.ErrNotFound
	}
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO ai_lookups (id, lesson_id, revision, sentence_index, text_lower, lemma, meaning_vi, note_vi, created_at) VALUES (?,?,?,?,?,?,?,?,?)",
		newID(), res.LessonID, res.Revision, res.SentenceIndex, res.Text, res.Lemma, res.MeaningVi, res.NoteVi, utc(res.CreatedAt))
	switch {
	case err == nil:
		return res, nil
	case !isDuplicate(err):
		return lesson.AskResult{}, fmt.Errorf("mysql insert ai lookup: %w", err)
	}
	stored, ok, err := r.Get(ctx, res.AskKey)
	if err != nil {
		return lesson.AskResult{}, err
	}
	if !ok {
		return lesson.AskResult{}, errors.New("mysql: ai lookup vanished after a duplicate key")
	}
	return stored, nil
}
