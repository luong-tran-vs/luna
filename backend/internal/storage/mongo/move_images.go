package mongo

import (
	"context"
	"fmt"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// imageCollections are the collections of the word pictures (F23), each with the fields that
// identify one of its documents.
var imageCollections = []struct {
	name string
	keys []string
}{
	{name: "lesson_image_settings", keys: []string{"_id"}},
	{name: "word_images", keys: []string{"lessonId", "lemma"}},
}

// MoveImages moves the word pictures from db to imagesDB when they are different databases
// (MONGO_IMAGES_URI or MONGO_IMAGES_DATABASE set), so they stop counting against the main
// cluster's storage. A document already in imagesDB is kept, being newer than the one moved, and
// a source collection is dropped only once all its documents are there: an interrupted move is
// finished by the next start.
func MoveImages(ctx context.Context, db, imagesDB *mongo.Database, log *slog.Logger) error {
	if db == imagesDB {
		return nil
	}
	for _, c := range imageCollections {
		src, dst := db.Collection(c.name), imagesDB.Collection(c.name)
		n, err := src.EstimatedDocumentCount(ctx)
		if err != nil {
			return fmt.Errorf("count %s: %w", c.name, err)
		}
		if n == 0 {
			continue
		}
		cur, err := src.Find(ctx, bson.D{})
		if err != nil {
			return fmt.Errorf("read %s: %w", c.name, err)
		}
		moved := 0
		for cur.Next(ctx) {
			var doc bson.D
			if err := cur.Decode(&doc); err != nil {
				_ = cur.Close(ctx)
				return fmt.Errorf("decode %s: %w", c.name, err)
			}
			filter, err := keyFilter(doc, c.keys)
			if err != nil {
				_ = cur.Close(ctx)
				return fmt.Errorf("%s: %w", c.name, err)
			}
			_, err = dst.UpdateOne(ctx, filter, bson.D{{Key: "$setOnInsert", Value: doc}}, options.UpdateOne().SetUpsert(true))
			if err != nil {
				_ = cur.Close(ctx)
				return fmt.Errorf("copy %s: %w", c.name, err)
			}
			moved++
		}
		if err := cur.Err(); err != nil {
			_ = cur.Close(ctx)
			return fmt.Errorf("read %s: %w", c.name, err)
		}
		_ = cur.Close(ctx)
		if err := src.Drop(ctx); err != nil {
			return fmt.Errorf("drop %s: %w", c.name, err)
		}
		log.InfoContext(ctx, "word pictures moved to their own database",
			slog.String("collection", c.name), slog.Int("documents", moved))
	}
	return nil
}

// keyFilter matches the document of doc's identifying fields.
func keyFilter(doc bson.D, keys []string) (bson.D, error) {
	filter := make(bson.D, 0, len(keys))
	for _, k := range keys {
		found := false
		for _, e := range doc {
			if e.Key == k {
				filter = append(filter, bson.E{Key: k, Value: e.Value})
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("document without %s", k)
		}
	}
	return filter, nil
}
