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
		At: u.At.UTC(), Model: u.Model, Op: u.Op, Status: u.Status, PromptTokens: u.PromptTokens,
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
			At: d.At, Model: d.Model, Op: d.Op, Status: d.Status, PromptTokens: d.PromptTokens,
			OutputTokens: d.OutputTokens, ThoughtTokens: d.ThoughtTokens, TotalTokens: d.TotalTokens,
			Duration: time.Duration(d.DurationMs) * time.Millisecond,
		}
	}
	return out, nil
}

// DeleteBefore removes the requests made before t (the TTL index does it too, in its own time).
func (r *AIUsage) DeleteBefore(ctx context.Context, t time.Time) error {
	if _, err := r.coll.DeleteMany(ctx, bson.D{{Key: "at", Value: bson.D{{Key: "$lt", Value: t.UTC()}}}}); err != nil {
		return fmt.Errorf("delete ai usage: %w", err)
	}
	return nil
}
