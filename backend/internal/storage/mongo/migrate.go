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

const topicsMigrationID = "f14-topics"

type legacyLessonDoc struct {
	ID        bson.ObjectID `bson:"_id"`
	Level     string        `bson:"level"`
	Topic     string        `bson:"topic"`
	TopicID   bson.ObjectID `bson:"topicId"`
	CreatedAt time.Time     `bson:"createdAt"`
}

// MigrateTopics moves F2 data to topics (F14) once: topics from the lessons' (level, topic)
// pairs, every lesson linked to a topic, the global roadmap split per topic, then the roadmap
// collection dropped. The plan comes from topic.PlanMigration; writes are idempotent (upserts
// by (level, nameKey), a marker written last), so a run interrupted at any point is completed by
// the next start.
func MigrateTopics(ctx context.Context, db *mongo.Database, log *slog.Logger) error {
	migrations := db.Collection("migrations")
	err := migrations.FindOne(ctx, bson.D{{Key: "_id", Value: topicsMigrationID}}).Err()
	if err == nil {
		return nil
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return fmt.Errorf("read migrations: %w", err)
	}

	lessons, err := readLegacyLessons(ctx, db)
	if err != nil {
		return err
	}
	roadmap, hasRoadmap, err := readLegacyRoadmap(ctx, db)
	if err != nil {
		return err
	}
	coll := db.Collection("topics")
	existing, err := readLegacyTopics(ctx, coll)
	if err != nil {
		return err
	}

	plan := topic.PlanMigration(lessons, roadmap, existing)
	ids, err := applyTopics(ctx, coll, plan, hasRoadmap)
	if err != nil {
		return err
	}
	levels := map[topic.Key]string{}
	for _, pt := range plan.Topics {
		levels[pt.Key] = pt.Level
	}
	lessonColl := db.Collection("lessons")
	for lessonID, key := range plan.Assign {
		lid, _ := bson.ObjectIDFromHex(lessonID)
		_, err := lessonColl.UpdateOne(ctx, bson.D{{Key: "_id", Value: lid}}, bson.D{
			{Key: "$set", Value: bson.D{{Key: "topicId", Value: ids[key]}, {Key: "level", Value: levels[key]}}},
			{Key: "$unset", Value: bson.D{{Key: "topic", Value: ""}}},
		})
		if err != nil {
			return fmt.Errorf("assign lesson topic: %w", err)
		}
	}

	if _, err := migrations.InsertOne(ctx, bson.D{{Key: "_id", Value: topicsMigrationID}, {Key: "doneAt", Value: time.Now().UTC()}}); err != nil {
		return fmt.Errorf("mark migration: %w", err)
	}
	// Leftovers after the marker are harmless: the next start skips the migration.
	if err := db.Collection("roadmap").Drop(ctx); err != nil {
		log.WarnContext(ctx, "drop old roadmap failed", slog.Any("error", err))
	}
	if err := lessonColl.Indexes().DropOne(ctx, "topic_1"); err != nil {
		log.DebugContext(ctx, "old lesson topic index not dropped", slog.Any("error", err))
	}
	log.InfoContext(ctx, "topics migration done",
		slog.Int("topics", len(plan.Topics)), slog.Int("lessons", len(plan.Assign)), slog.Bool("roadmap", hasRoadmap))
	return nil
}

func readLegacyLessons(ctx context.Context, db *mongo.Database) ([]topic.LegacyLesson, error) {
	cur, err := db.Collection("lessons").Find(ctx, bson.D{}, options.Find().SetProjection(bson.D{
		{Key: "level", Value: 1}, {Key: "topic", Value: 1}, {Key: "topicId", Value: 1}, {Key: "createdAt", Value: 1},
	}))
	if err != nil {
		return nil, fmt.Errorf("find lessons: %w", err)
	}
	var docs []legacyLessonDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("read lessons: %w", err)
	}
	out := make([]topic.LegacyLesson, len(docs))
	for i, d := range docs {
		out[i] = topic.LegacyLesson{
			ID: d.ID.Hex(), Level: d.Level, TopicName: d.Topic, TopicID: hexOrEmpty(d.TopicID), CreatedAt: d.CreatedAt,
		}
	}
	return out, nil
}

// readLegacyTopics reads the topics as F14 stored them before they were shared by every level.
func readLegacyTopics(ctx context.Context, coll *mongo.Collection) ([]topic.LegacyTopic, error) {
	cur, err := coll.Find(ctx, bson.D{}, options.Find().SetProjection(bson.D{{Key: "name", Value: 1}, {Key: "level", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("find topics: %w", err)
	}
	var docs []struct {
		ID    bson.ObjectID `bson:"_id"`
		Name  string        `bson:"name"`
		Level string        `bson:"level"`
	}
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("read topics: %w", err)
	}
	out := make([]topic.LegacyTopic, len(docs))
	for i, d := range docs {
		out[i] = topic.LegacyTopic{ID: d.ID.Hex(), Name: d.Name, Level: d.Level}
	}
	return out, nil
}

func readLegacyRoadmap(ctx context.Context, db *mongo.Database) ([]string, bool, error) {
	var d struct {
		LessonIDs []bson.ObjectID `bson:"lessonIds"`
	}
	err := db.Collection("roadmap").FindOne(ctx, bson.D{{Key: "_id", Value: "main"}}).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("read old roadmap: %w", err)
	}
	ids := make([]string, len(d.LessonIDs))
	for i, oid := range d.LessonIDs {
		ids[i] = oid.Hex()
	}
	return ids, true, nil
}

// applyTopics makes sure every planned topic exists and returns their ids. Roadmaps are only
// written while the old roadmap still exists, so a re-run never empties a split roadmap.
func applyTopics(ctx context.Context, coll *mongo.Collection, plan topic.Plan, hasRoadmap bool) (map[topic.Key]bson.ObjectID, error) {
	ids := map[topic.Key]bson.ObjectID{}
	now := time.Now().UTC()
	for _, pt := range plan.Topics {
		filter := bson.D{{Key: "level", Value: pt.Level}, {Key: "nameKey", Value: pt.NameKey}}
		_, err := coll.UpdateOne(ctx, filter, bson.D{{Key: "$setOnInsert", Value: bson.D{
			{Key: "name", Value: pt.Name},
			{Key: "description", Value: ""},
			{Key: "lessonIds", Value: bson.A{}},
			{Key: "createdAt", Value: now},
			{Key: "updatedAt", Value: now},
		}}}, options.UpdateOne().SetUpsert(true))
		if err != nil {
			return nil, fmt.Errorf("upsert topic %s %s: %w", pt.Level, pt.Name, err)
		}
		var d topicDoc
		if err := coll.FindOne(ctx, filter).Decode(&d); err != nil {
			return nil, fmt.Errorf("read topic %s %s: %w", pt.Level, pt.Name, err)
		}
		ids[pt.Key] = d.ID
		if hasRoadmap {
			oids, err := toOIDs(pt.LessonIDs)
			if err != nil {
				return nil, err
			}
			if _, err := coll.UpdateOne(ctx, bson.D{{Key: "_id", Value: d.ID}},
				bson.D{{Key: "$set", Value: bson.D{{Key: "lessonIds", Value: oids}}}}); err != nil {
				return nil, fmt.Errorf("set topic roadmap: %w", err)
			}
		}
	}
	return ids, nil
}
