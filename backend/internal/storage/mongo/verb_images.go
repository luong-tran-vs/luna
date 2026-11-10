package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/luongtran/luna/backend/internal/verbs"
)

// verbImageDoc is the picture of one irregular verb in "verb_images", keyed by its base form
// (added 2026-11-10).
type verbImageDoc struct {
	Base      string    `bson:"_id"`
	MIME      string    `bson:"mime"`
	Data      []byte    `bson:"data"`
	CreatedAt time.Time `bson:"createdAt"`
}

// VerbImages implements verbs.ImageRepository.
type VerbImages struct {
	images *mongo.Collection
}

// NewVerbImages returns the verb picture repository on db.
func NewVerbImages(db *mongo.Database) *VerbImages {
	return &VerbImages{images: db.Collection("verb_images")}
}

var _ verbs.ImageRepository = (*VerbImages)(nil)

// List tells every verb that has a picture, without the pictures.
func (r *VerbImages) List(ctx context.Context) ([]verbs.ImageInfo, error) {
	cur, err := r.images.Find(ctx, bson.D{}, options.Find().SetProjection(bson.D{{Key: "createdAt", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("find verb images: %w", err)
	}
	var docs []struct {
		Base      string    `bson:"_id"`
		CreatedAt time.Time `bson:"createdAt"`
	}
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("decode verb images: %w", err)
	}
	out := make([]verbs.ImageInfo, len(docs))
	for i, d := range docs {
		out[i] = verbs.ImageInfo{Base: d.Base, CreatedAt: d.CreatedAt.UTC()}
	}
	return out, nil
}

// Get returns verbs.ErrImageNotFound when the verb has no picture.
func (r *VerbImages) Get(ctx context.Context, base string) (verbs.VerbImage, error) {
	var d verbImageDoc
	err := r.images.FindOne(ctx, bson.D{{Key: "_id", Value: base}}).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return verbs.VerbImage{}, verbs.ErrImageNotFound
	}
	if err != nil {
		return verbs.VerbImage{}, fmt.Errorf("find verb image: %w", err)
	}
	return verbs.VerbImage{Base: d.Base, MIME: d.MIME, Data: d.Data, CreatedAt: d.CreatedAt.UTC()}, nil
}

// Save stores a picture, replacing the verb's previous one.
func (r *VerbImages) Save(ctx context.Context, img verbs.VerbImage) error {
	d := verbImageDoc{Base: img.Base, MIME: img.MIME, Data: img.Data, CreatedAt: img.CreatedAt.UTC()}
	if _, err := r.images.ReplaceOne(ctx, bson.D{{Key: "_id", Value: img.Base}}, d, options.Replace().SetUpsert(true)); err != nil {
		return fmt.Errorf("save verb image: %w", err)
	}
	return nil
}

// Delete removes a verb's picture.
func (r *VerbImages) Delete(ctx context.Context, base string) error {
	if _, err := r.images.DeleteOne(ctx, bson.D{{Key: "_id", Value: base}}); err != nil {
		return fmt.Errorf("delete verb image: %w", err)
	}
	return nil
}
