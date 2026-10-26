package mysql

import (
	"context"
	"fmt"

	"github.com/luongtran/luna/backend/internal/grammar"
)

// Delete removes the lesson of a point (added 2026-10-26).
func (r *GrammarLessons) Delete(ctx context.Context, pointID string) error {
	res, err := r.db.ExecContext(ctx, "DELETE FROM grammar_lessons WHERE point_id = ?", pointID)
	if err != nil {
		return fmt.Errorf("mysql delete grammar lesson: %w", err)
	}
	if n, err := res.RowsAffected(); err == nil && n == 0 {
		return grammar.ErrNotFound
	}
	return nil
}
