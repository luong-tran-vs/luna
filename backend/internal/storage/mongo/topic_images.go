package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/luongtran/luna/backend/internal/topicimage"
)

// topicImageDoc is the picture of one topic in "topic_images", keyed by its id
// (added 2026-11-16).
type topicImageDoc struct {
	TopicID   string    `bson:"_id"`
	MIME      string    `bson:"mime"`
	Data      []byte    `bson:"data"`
	CreatedAt time.Time `bson:"createdAt"`
}

// TopicImages implements topicimage.Repository.
type TopicImages struct {
	images *mongo.Collection
}

// NewTopicImages returns the topic picture repository on db.
func NewTopicImages(db *mongo.Database) *TopicImages {
	return &TopicImages{images: db.Collection("topic_images")}
}

var _ topicimage.Repository = (*TopicImages)(nil)

// List tells every topic that has a picture, without the pictures.
func (r *TopicImages) List(ctx context.Context) ([]topicimage.Info, error) {
	cur, err := r.images.Find(ctx, bson.D{}, options.Find().SetProjection(bson.D{{Key: "createdAt", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("find topic images: %w", err)
	}
	var docs []struct {
		TopicID   string    `bson:"_id"`
		CreatedAt time.Time `bson:"createdAt"`
	}
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("decode topic images: %w", err)
	}
	out := make([]topicimage.Info, len(docs))
	for i, d := range docs {
		out[i] = topicimage.Info{TopicID: d.TopicID, CreatedAt: d.CreatedAt.UTC()}
	}
	return out, nil
}

// Get returns topicimage.ErrImageNotFound when the topic has no picture.
func (r *TopicImages) Get(ctx context.Context, topicID string) (topicimage.Image, error) {
	var d topicImageDoc
	err := r.images.FindOne(ctx, bson.D{{Key: "_id", Value: topicID}}).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return topicimage.Image{}, topicimage.ErrImageNotFound
	}
	if err != nil {
		return topicimage.Image{}, fmt.Errorf("find topic image: %w", err)
	}
	return topicimage.Image{TopicID: d.TopicID, MIME: d.MIME, Data: d.Data, CreatedAt: d.CreatedAt.UTC()}, nil
}

// Save stores a picture, replacing the topic's previous one.
func (r *TopicImages) Save(ctx context.Context, img topicimage.Image) error {
	d := topicImageDoc{TopicID: img.TopicID, MIME: img.MIME, Data: img.Data, CreatedAt: img.CreatedAt.UTC()}
	if _, err := r.images.ReplaceOne(ctx, bson.D{{Key: "_id", Value: img.TopicID}}, d, options.Replace().SetUpsert(true)); err != nil {
		return fmt.Errorf("save topic image: %w", err)
	}
	return nil
}

// Delete removes a topic's picture.
func (r *TopicImages) Delete(ctx context.Context, topicID string) error {
	if _, err := r.images.DeleteOne(ctx, bson.D{{Key: "_id", Value: topicID}}); err != nil {
		return fmt.Errorf("delete topic image: %w", err)
	}
	return nil
}
