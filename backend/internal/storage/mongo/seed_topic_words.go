package mongo

import (
	"context"
	"fmt"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/luongtran/luna/backend/internal/topic"
)

// SeedTopicWords gives every topic not seeded yet its starting vocabulary (F18): the seed list
// of the same name, or an empty list. Each write only applies while the topic is still not
// seeded, so a restart or a concurrent run never overwrites a list, even one an admin emptied.
func SeedTopicWords(ctx context.Context, db *mongo.Database, seed topic.Seed, log *slog.Logger) error {
	coll := db.Collection("topics")
	notSeeded := bson.D{{Key: "wordsSeeded", Value: bson.D{{Key: "$ne", Value: true}}}}
	cur, err := coll.Find(ctx, notSeeded, options.Find().SetProjection(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		return fmt.Errorf("find topics to seed: %w", err)
	}
	var docs []topicDoc
	if err := cur.All(ctx, &docs); err != nil {
		return fmt.Errorf("decode topics to seed: %w", err)
	}
	if len(docs) == 0 {
		return nil
	}
	targets := make([]topic.SeedTarget, len(docs))
	ids := make(map[string]bson.ObjectID, len(docs))
	for i, d := range docs {
		targets[i] = topic.SeedTarget{ID: d.ID.Hex(), Name: d.Name}
		ids[d.ID.Hex()] = d.ID
	}

	matched, unmatched := 0, 0
	for _, w := range topic.PlanSeed(targets, seed) {
		filter := append(bson.D{{Key: "_id", Value: ids[w.ID]}}, notSeeded...)
		res, err := coll.UpdateOne(ctx, filter, bson.D{{Key: "$set", Value: bson.D{
			{Key: "words", Value: w.Words},
			{Key: "wordsSeeded", Value: true},
		}}})
		if err != nil {
			return fmt.Errorf("seed topic words: %w", err)
		}
		switch {
		case res.ModifiedCount == 0: // seeded meanwhile
		case w.Matched:
			matched++
		default:
			unmatched++
		}
	}
	log.InfoContext(ctx, "topic words seeded", slog.Int("matched", matched), slog.Int("unmatched", unmatched))
	return nil
}

// seedTopicWords runs SeedTopicWords with the embedded seed.
func seedTopicWords(ctx context.Context, db *mongo.Database, log *slog.Logger) error {
	seed, err := topic.LoadSeed()
	if err != nil {
		return err
	}
	return SeedTopicWords(ctx, db, seed, log)
}
