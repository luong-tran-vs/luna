package mongo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/luongtran/luna/backend/internal/lesson"
)

// AskUsage implements lesson.AskUsageRepository on "ask_usage" (F9): the words each learner asked
// the AI about, one document per (userId, lessonId, textLower).
type AskUsage struct {
	coll *mongo.Collection
}

// NewAskUsage returns the AskUsage repository on db.
func NewAskUsage(db *mongo.Database) *AskUsage {
	return &AskUsage{coll: db.Collection("ask_usage")}
}

var _ lesson.AskUsageRepository = (*AskUsage)(nil)

type askUsageDoc struct {
	UserID    bson.ObjectID `bson:"userId"`
	LessonID  string        `bson:"lessonId"`
	TextLower string        `bson:"textLower"`
	CreatedAt time.Time     `bson:"createdAt"`
}

// Asked lists the texts the user asked about in the lesson, oldest first.
func (r *AskUsage) Asked(ctx context.Context, userID, lessonID string) ([]string, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return []string{}, nil
	}
	cur, err := r.coll.Find(ctx, bson.D{{Key: "userId", Value: uid}, {Key: "lessonId", Value: lessonID}},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}, {Key: "textLower", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("find asked words: %w", err)
	}
	var docs []askUsageDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("decode asked words: %w", err)
	}
	out := make([]string, len(docs))
	for i, d := range docs {
		out[i] = d.TextLower
	}
	return out, nil
}

// Add records an asked text; recording it again changes nothing.
func (r *AskUsage) Add(ctx context.Context, userID, lessonID, text string, at time.Time) error {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return fmt.Errorf("record ask: bad user id %q", userID)
	}
	filter := bson.D{{Key: "userId", Value: uid}, {Key: "lessonId", Value: lessonID}, {Key: "textLower", Value: text}}
	_, err = r.coll.UpdateOne(ctx, filter,
		bson.D{{Key: "$setOnInsert", Value: bson.D{{Key: "createdAt", Value: at.UTC()}}}},
		options.UpdateOne().SetUpsert(true))
	if err != nil && !mongo.IsDuplicateKeyError(err) {
		return fmt.Errorf("record ask: %w", err)
	}
	return nil
}
