package mongo

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/luongtran/luna/backend/internal/topic"
)

// AddExtraTopics creates, once, the topics added on 2026-10-07 (topic.LoadExtraTopics) that no
// topic has the name of yet, with their words. A topic deleted later is not created again. Each
// insert skips a name taken meanwhile, so a run interrupted half way is finished by the next start.
func AddExtraTopics(ctx context.Context, db *mongo.Database, log *slog.Logger) error {
	migrations := db.Collection("migrations")
	err := migrations.FindOne(ctx, bson.D{{Key: "_id", Value: topic.ExtraTopicsID}}).Err()
	if err == nil {
		return nil
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return fmt.Errorf("read migrations: %w", err)
	}

	now := time.Now().UTC()
	extra, err := topic.LoadExtraTopics(now)
	if err != nil {
		return err
	}
	coll := db.Collection("topics")
	cur, err := coll.Find(ctx, bson.D{}, options.Find().SetProjection(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		return fmt.Errorf("find topics: %w", err)
	}
	var docs []struct {
		Name string `bson:"name"`
	}
	if err := cur.All(ctx, &docs); err != nil {
		return fmt.Errorf("read topics: %w", err)
	}
	names := make([]string, len(docs))
	for i, d := range docs {
		names[i] = d.Name
	}

	added := 0
	for _, t := range topic.PlanExtraTopics(names, extra) {
		d := topicDoc{
			ID: bson.NewObjectID(), Name: t.Name, NameKey: topic.NameKey(t.Name), Description: t.Description,
			Roadmaps: map[string][]bson.ObjectID{}, Words: toWordDocs(t.Words), WordsSeeded: true,
			CreatedAt: now, UpdatedAt: now,
		}
		if _, err := coll.InsertOne(ctx, d); err != nil {
			if mongo.IsDuplicateKeyError(err) {
				continue
			}
			return fmt.Errorf("insert topic %s: %w", t.Name, err)
		}
		added++
	}
	if _, err := migrations.InsertOne(ctx, bson.D{{Key: "_id", Value: topic.ExtraTopicsID}, {Key: "doneAt", Value: now}}); err != nil {
		return fmt.Errorf("mark migration: %w", err)
	}
	log.InfoContext(ctx, "extra topics added", slog.Int("topics", added))
	return nil
}
