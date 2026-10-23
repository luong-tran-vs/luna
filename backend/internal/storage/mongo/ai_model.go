package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/luongtran/luna/backend/internal/aimodel"
)

// AIModel implements aimodel.Repository on "ai_settings" (added 2026-10-23): one document, its _id
// "text", with the model the admin chose and the ops given their own model.
type AIModel struct {
	coll *mongo.Collection
}

// NewAIModel returns the AIModel repository on db.
func NewAIModel(db *mongo.Database) *AIModel {
	return &AIModel{coll: db.Collection("ai_settings")}
}

var _ aimodel.Repository = (*AIModel)(nil)

const aiModelID = "text"

type aiModelDoc struct {
	ID        string            `bson:"_id"`
	Model     string            `bson:"model"`
	ByOp      map[string]string `bson:"byOp,omitempty"`
	UpdatedAt time.Time         `bson:"updatedAt"`
}

// Get returns the saved choice, or a zero Choice.
func (r *AIModel) Get(ctx context.Context) (aimodel.Choice, error) {
	var d aiModelDoc
	err := r.coll.FindOne(ctx, bson.D{{Key: "_id", Value: aiModelID}}).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return aimodel.Choice{}, nil
	}
	if err != nil {
		return aimodel.Choice{}, fmt.Errorf("mongo: get ai model: %w", err)
	}
	return aimodel.Choice{Model: d.Model, ByOp: d.ByOp, UpdatedAt: d.UpdatedAt}, nil
}

// Save replaces the choice.
func (r *AIModel) Save(ctx context.Context, c aimodel.Choice) error {
	d := aiModelDoc{ID: aiModelID, Model: c.Model, ByOp: c.ByOp, UpdatedAt: c.UpdatedAt.UTC()}
	_, err := r.coll.ReplaceOne(ctx, bson.D{{Key: "_id", Value: aiModelID}}, d, options.Replace().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("mongo: save ai model: %w", err)
	}
	return nil
}
