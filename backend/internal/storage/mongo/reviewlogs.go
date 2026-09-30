package mongo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/luongtran/luna/backend/internal/vocab"
)

type reviewLogDoc struct {
	ID          bson.ObjectID `bson:"_id,omitempty"`
	UserID      bson.ObjectID `bson:"userId"`
	CardID      bson.ObjectID `bson:"cardId"`
	Rating      int           `bson:"rating"`
	Mode        string        `bson:"mode"`
	Context     string        `bson:"context"`
	ReviewedAt  time.Time     `bson:"reviewedAt"`
	StateBefore ScheduleDoc   `bson:"stateBefore"`
}

// ReviewLogs implements vocab.ReviewLogRepository on the "review_logs" collection.
type ReviewLogs struct {
	coll *mongo.Collection
}

// NewReviewLogs returns a review log repository on db.
func NewReviewLogs(db *mongo.Database) *ReviewLogs {
	return &ReviewLogs{coll: db.Collection("review_logs")}
}

var _ vocab.ReviewLogRepository = (*ReviewLogs)(nil)

// Add stores one review.
func (r *ReviewLogs) Add(ctx context.Context, l vocab.ReviewLog) error {
	uid, cid, err := ids(l.UserID, l.CardID)
	if err != nil {
		return fmt.Errorf("review log ids: %w", err)
	}
	d := reviewLogDoc{
		ID: bson.NewObjectID(), UserID: uid, CardID: cid, Rating: int(l.Rating), Mode: string(l.Mode), Context: string(l.Context),
		ReviewedAt: l.ReviewedAt.UTC(), StateBefore: fromSchedule(l.Before),
	}
	if _, err := r.coll.InsertOne(ctx, d); err != nil {
		return fmt.Errorf("insert review log: %w", err)
	}
	return nil
}

// DeleteByCard removes every log of one of the user's cards.
func (r *ReviewLogs) DeleteByCard(ctx context.Context, userID, cardID string) (int, error) {
	uid, cid, err := ids(userID, cardID)
	if err != nil {
		return 0, nil //nolint:nilerr // a malformed card id has no logs
	}
	res, err := r.coll.DeleteMany(ctx, bson.D{{Key: "userId", Value: uid}, {Key: "cardId", Value: cid}})
	if err != nil {
		return 0, fmt.Errorf("delete review logs: %w", err)
	}
	return int(res.DeletedCount), nil
}

// CountSince counts the user's reviews in context c at or after since.
func (r *ReviewLogs) CountSince(ctx context.Context, userID string, c vocab.Context, since time.Time) (int, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return 0, fmt.Errorf("review log user id: %w", err)
	}
	n, err := r.coll.CountDocuments(ctx, bson.D{
		{Key: "userId", Value: uid},
		{Key: "context", Value: string(c)},
		{Key: "reviewedAt", Value: bson.D{{Key: "$gte", Value: since.UTC()}}},
	})
	if err != nil {
		return 0, fmt.Errorf("count review logs: %w", err)
	}
	return int(n), nil
}
