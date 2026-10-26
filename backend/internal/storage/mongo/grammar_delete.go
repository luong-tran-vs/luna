package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/luongtran/luna/backend/internal/grammar"
)

// Delete removes the lesson of a point (added 2026-10-26).
func (r *GrammarLessons) Delete(ctx context.Context, pointID string) error {
	res, err := r.coll.DeleteOne(ctx, bson.D{{Key: "pointId", Value: pointID}})
	if err != nil {
		return fmt.Errorf("delete grammar lesson: %w", err)
	}
	if res.DeletedCount == 0 {
		return grammar.ErrNotFound
	}
	return nil
}
