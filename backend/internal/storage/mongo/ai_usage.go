package mongo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/aiusage"
)

// aiUsageDoc is one AI request in "ai_usage"; a TTL index on at drops it after aiusage.Keep.
type aiUsageDoc struct {
	At            time.Time `bson:"at"`
	Model         string    `bson:"model"`
	Op            string    `bson:"op"`
	Key           int       `bson:"key,omitempty"`
	Status        int       `bson:"status"`
	PromptTokens  int       `bson:"promptTokens"`
	OutputTokens  int       `bson:"outputTokens"`
	ThoughtTokens int       `bson:"thoughtTokens"`
	TotalTokens   int       `bson:"totalTokens"`
	DurationMs    int64     `bson:"durationMs"`
}

// AIUsage implements aiusage.Repository.
type AIUsage struct {
	coll *mongo.Collection
}

// NewAIUsage returns the AI usage repository on db.
func NewAIUsage(db *mongo.Database) *AIUsage { return &AIUsage{coll: db.Collection("ai_usage")} }

var _ aiusage.Repository = (*AIUsage)(nil)

// Add stores one request.
func (r *AIUsage) Add(ctx context.Context, u ai.Usage) error {
	_, err := r.coll.InsertOne(ctx, aiUsageDoc{
		At: u.At.UTC(), Model: u.Model, Op: u.Op, Key: u.Key, Status: u.Status, PromptTokens: u.PromptTokens,
		OutputTokens: u.OutputTokens, ThoughtTokens: u.ThoughtTokens, TotalTokens: u.TotalTokens,
		DurationMs: u.Duration.Milliseconds(),
	})
	if err != nil {
		return fmt.Errorf("insert ai usage: %w", err)
	}
	return nil
}

// Since returns the requests made at or after from, oldest first.
func (r *AIUsage) Since(ctx context.Context, from time.Time) ([]ai.Usage, error) {
	cur, err := r.coll.Find(ctx, bson.D{{Key: "at", Value: bson.D{{Key: "$gte", Value: from.UTC()}}}},
		options.Find().SetSort(bson.D{{Key: "at", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("find ai usage: %w", err)
	}
	var docs []aiUsageDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("read ai usage: %w", err)
	}
	out := make([]ai.Usage, len(docs))
	for i, d := range docs {
		out[i] = ai.Usage{
			At: d.At, Model: d.Model, Op: d.Op, Key: d.Key, Status: d.Status, PromptTokens: d.PromptTokens,
			OutputTokens: d.OutputTokens, ThoughtTokens: d.ThoughtTokens, TotalTokens: d.TotalTokens,
			Duration: time.Duration(d.DurationMs) * time.Millisecond,
		}
	}
	return out, nil
}

// KeepLatest removes every request but the n most recent ones.
func (r *AIUsage) KeepLatest(ctx context.Context, n int) error {
	cur, err := r.coll.Find(ctx, bson.D{}, options.Find().
		SetSort(bson.D{{Key: "at", Value: -1}, {Key: "_id", Value: -1}}).
		SetLimit(int64(n)).
		SetProjection(bson.D{{Key: "_id", Value: 1}}))
	if err != nil {
		return fmt.Errorf("find latest ai usage: %w", err)
	}
	var keep []struct {
		ID bson.ObjectID `bson:"_id"`
	}
	if err := cur.All(ctx, &keep); err != nil {
		return fmt.Errorf("read latest ai usage: %w", err)
	}
	if len(keep) < n {
		return nil
	}
	ids := make(bson.A, len(keep))
	for i, k := range keep {
		ids[i] = k.ID
	}
	if _, err := r.coll.DeleteMany(ctx, bson.D{{Key: "_id", Value: bson.D{{Key: "$nin", Value: ids}}}}); err != nil {
		return fmt.Errorf("delete ai usage: %w", err)
	}
	return nil
}
