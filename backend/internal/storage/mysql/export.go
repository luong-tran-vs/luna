package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/luongtran/luna/backend/internal/export"
)

// Export implements export.Repository: read-only access to one learner's rows (F13). Rows leave
// in the shape the MongoDB export has, so a file does not depend on the database it came from:
// camelCase keys, "_id" for the id, times as RFC 3339, JSON columns decoded, flags as booleans, and
// no key at all for a NULL.
type Export struct {
	db *sql.DB
}

// NewExport returns an export repository on db.
func NewExport(db *sql.DB) *Export { return &Export{db: db} }

var _ export.Repository = (*Export)(nil)

// exportOrder is the ORDER BY of each exportable table, "oldest first" (the Mongo export sorts by _id).
// These are the only table names that ever reach a query: export.Allowed gates the rest.
var exportOrder = map[string]string{
	export.CollCards:            "seq",
	export.CollReviewLogs:       "id",
	export.CollGoals:            "id",
	export.CollLessonProgress:   "lesson_id",
	export.CollStudyDays:        "day_key",
	export.CollDictationResults: "lesson_id, sentence_index",
	export.CollReadingAnswers:   "id",
	export.CollWritings:         "id",
	export.CollGrammarProgress:  "point_id",
}

// exportBools are the TINYINT(1) columns that are flags; they are exported as true/false.
var exportBools = map[string]bool{
	"extras_edited": true, "grade_seen": true, "correct": true, "completed": true,
	"step_read": true, "step_listen": true, "step_write": true, "words_seeded": true,
}

// Account reads only the exported account fields; the password hash is never loaded.
func (r *Export) Account(ctx context.Context, userID string) (export.Account, error) {
	var a export.Account
	err := r.db.QueryRowContext(ctx, "SELECT id, email, role, created_at FROM users WHERE id = ?", userID).
		Scan(&a.ID, &a.Email, &a.Role, &a.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return export.Account{}, export.ErrNotFound
	}
	if err != nil {
		return export.Account{}, fmt.Errorf("mysql export account: %w", err)
	}
	a.CreatedAt = utc(a.CreatedAt)
	return a, nil
}

// UserDocs returns the user's rows of an allowed table, oldest first.
func (r *Export) UserDocs(ctx context.Context, collection, userID string) ([]export.Doc, error) {
	order, ok := exportOrder[collection]
	if !ok || !export.Allowed(collection) {
		return nil, export.ErrCollection
	}
	return r.docs(ctx, "SELECT * FROM "+collection+" WHERE user_id = ? ORDER BY "+order, userID)
}

// Lessons returns the lessons among ids that still exist.
func (r *Export) Lessons(ctx context.Context, ids []string) ([]export.Doc, error) {
	if len(ids) == 0 {
		return []export.Doc{}, nil
	}
	return r.docs(ctx, "SELECT * FROM lessons WHERE id IN ("+placeholders(len(ids))+") ORDER BY id", args(ids)...)
}

func (r *Export) docs(ctx context.Context, query string, a ...any) ([]export.Doc, error) {
	rows, err := r.db.QueryContext(ctx, query, a...)
	if err != nil {
		return nil, fmt.Errorf("mysql export query: %w", err)
	}
	defer func() { _ = rows.Close() }()
	cols, err := rows.ColumnTypes()
	if err != nil {
		return nil, fmt.Errorf("mysql export columns: %w", err)
	}
	out := []export.Doc{}
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("mysql export scan: %w", err)
		}
		doc := make(export.Doc, len(cols))
		for i, c := range cols {
			v, err := exportValue(c.Name(), c.DatabaseTypeName(), vals[i])
			if err != nil {
				return nil, err
			}
			if v != nil {
				doc[exportKey(c.Name())] = v
			}
		}
		out = append(out, doc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql export rows: %w", err)
	}
	return out, nil
}

// exportKey turns a column name into the document key: id -> _id, user_id -> userId.
func exportKey(col string) string {
	if col == "id" {
		return "_id"
	}
	parts := strings.Split(col, "_")
	for i := 1; i < len(parts); i++ {
		if parts[i] != "" {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

// exportValue converts one scanned column to a JSON-friendly value; nil means "leave the key out".
func exportValue(col, typ string, v any) (any, error) {
	switch x := v.(type) {
	case nil:
		return nil, nil
	case time.Time:
		return x.UTC().Format(time.RFC3339Nano), nil
	case []byte:
		if typ == "JSON" {
			var decoded any
			if err := fromJSON(x, &decoded); err != nil {
				return nil, err
			}
			return decoded, nil
		}
		return string(x), nil
	case int64:
		if exportBools[col] {
			return x != 0, nil
		}
		return x, nil
	default:
		return v, nil
	}
}
