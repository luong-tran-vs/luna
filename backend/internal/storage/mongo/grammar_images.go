package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/luongtran/luna/backend/internal/grammar"
)

// grammarImageDoc is the picture of one example of a grammar lesson in "grammar_images", unique by
// (pointId, key) (added 2026-11-06).
type grammarImageDoc struct {
	ID        bson.ObjectID `bson:"_id,omitempty"`
	PointID   string        `bson:"pointId"`
	Key       string        `bson:"key"`
	MIME      string        `bson:"mime"`
	Data      []byte        `bson:"data"`
	CreatedAt time.Time     `bson:"createdAt"`
}

// GrammarImages implements grammar.ImageRepository.
type GrammarImages struct {
	images *mongo.Collection
}

// NewGrammarImages returns the example picture repository on db.
func NewGrammarImages(db *mongo.Database) *GrammarImages {
	return &GrammarImages{images: db.Collection("grammar_images")}
}

var _ grammar.ImageRepository = (*GrammarImages)(nil)

// Keys lists the keys of the point's pictures.
func (r *GrammarImages) Keys(ctx context.Context, pointID string) ([]string, error) {
	cur, err := r.images.Find(ctx, bson.D{{Key: "pointId", Value: pointID}},
		options.Find().SetProjection(bson.D{{Key: "key", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("find grammar images: %w", err)
	}
	var docs []struct {
		Key string `bson:"key"`
	}
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("decode grammar images: %w", err)
	}
	out := make([]string, len(docs))
	for i, d := range docs {
		out[i] = d.Key
	}
	return out, nil
}

// Get returns grammar.ErrImageNotFound when the example has no picture.
func (r *GrammarImages) Get(ctx context.Context, pointID, key string) (grammar.ExampleImage, error) {
	var d grammarImageDoc
	err := r.images.FindOne(ctx, bson.D{{Key: "pointId", Value: pointID}, {Key: "key", Value: key}}).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return grammar.ExampleImage{}, grammar.ErrImageNotFound
	}
	if err != nil {
		return grammar.ExampleImage{}, fmt.Errorf("find grammar image: %w", err)
	}
	return grammar.ExampleImage{PointID: d.PointID, Key: d.Key, MIME: d.MIME, Data: d.Data, CreatedAt: d.CreatedAt}, nil
}

// Save stores a picture, replacing the example's previous one.
func (r *GrammarImages) Save(ctx context.Context, img grammar.ExampleImage) error {
	filter := bson.D{{Key: "pointId", Value: img.PointID}, {Key: "key", Value: img.Key}}
	d := grammarImageDoc{PointID: img.PointID, Key: img.Key, MIME: img.MIME, Data: img.Data, CreatedAt: img.CreatedAt.UTC()}
	if _, err := r.images.ReplaceOne(ctx, filter, d, options.Replace().SetUpsert(true)); err != nil {
		return fmt.Errorf("save grammar image: %w", err)
	}
	return nil
}

// Delete removes one picture.
func (r *GrammarImages) Delete(ctx context.Context, pointID, key string) error {
	if _, err := r.images.DeleteOne(ctx, bson.D{{Key: "pointId", Value: pointID}, {Key: "key", Value: key}}); err != nil {
		return fmt.Errorf("delete grammar image: %w", err)
	}
	return nil
}

// DeleteForPoint removes every picture of a point.
func (r *GrammarImages) DeleteForPoint(ctx context.Context, pointID string) error {
	if _, err := r.images.DeleteMany(ctx, bson.D{{Key: "pointId", Value: pointID}}); err != nil {
		return fmt.Errorf("delete grammar images: %w", err)
	}
	return nil
}
