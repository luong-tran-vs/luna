package mongo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/luongtran/luna/backend/internal/progressreset"
)

// ProgressReset implements progressreset.Repository over the learners' progress collections.
type ProgressReset struct {
	db *mongo.Database
}

// NewProgressReset returns the progress reset repository on db.
func NewProgressReset(db *mongo.Database) *ProgressReset { return &ProgressReset{db: db} }

var _ progressreset.Repository = (*ProgressReset)(nil)

// resetCollections are the collections whose documents of the user a part removes.
var resetCollections = map[progressreset.Part][]string{
	progressreset.PartLessons: {"lesson_progress", "reading_answers", "dictation_results", collLessonResults},
	progressreset.PartGrammar: {"grammar_progress"},
	progressreset.PartVocab:   {"review_logs"},
	progressreset.PartToeic:   {"toeic_attempts", "toeic_mistakes"},
}

// Reset removes the user's progress in part; for the notebook, its cards also become new again.
func (r *ProgressReset) Reset(ctx context.Context, userID string, part progressreset.Part, now time.Time) error {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return fmt.Errorf("mongo: reset user id: %w", err)
	}
	user := bson.D{{Key: "userId", Value: uid}}
	if part == progressreset.PartVocab {
		_, err := r.db.Collection("cards").UpdateMany(ctx, user, bson.D{
			{Key: "$set", Value: bson.D{{Key: "due", Value: now.UTC()}}},
			{Key: "$unset", Value: bson.D{
				{Key: "stability", Value: ""}, {Key: "difficulty", Value: ""}, {Key: "elapsedDays", Value: ""},
				{Key: "scheduledDays", Value: ""}, {Key: "reps", Value: ""}, {Key: "lapses", Value: ""},
				{Key: "state", Value: ""}, {Key: "lastReview", Value: ""},
			}},
		})
		if err != nil {
			return fmt.Errorf("mongo: reset cards: %w", err)
		}
	}
	for _, c := range resetCollections[part] {
		if _, err := r.db.Collection(c).DeleteMany(ctx, user); err != nil {
			return fmt.Errorf("mongo: reset %s: %w", c, err)
		}
	}
	return nil
}
