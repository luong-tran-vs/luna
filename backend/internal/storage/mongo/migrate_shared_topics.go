package mongo

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/luongtran/luna/backend/internal/topic"
)

const sharedTopicsMigrationID = "f14-shared-topics"

// mergeTopicDoc reads a topic in either shape: legacy (level, lessonIds) or shared (roadmaps).
type mergeTopicDoc struct {
	ID          bson.ObjectID              `bson:"_id"`
	Name        string                     `bson:"name"`
	Description string                     `bson:"description"`
	Level       string                     `bson:"level"`
	LessonIDs   []bson.ObjectID            `bson:"lessonIds"`
	Roadmaps    map[string][]bson.ObjectID `bson:"roadmaps"`
	Words       []wordDoc                  `bson:"words"`
	WordsSeeded bool                       `bson:"wordsSeeded"`
	CreatedAt   time.Time                  `bson:"createdAt"`
}

func (d mergeTopicDoc) toSource() topic.MergeSource {
	ids := make([]string, len(d.LessonIDs))
	for i, oid := range d.LessonIDs {
		ids[i] = oid.Hex()
	}
	return topic.MergeSource{
		ID: d.ID.Hex(), Name: d.Name, Description: d.Description, Level: d.Level, LessonIDs: ids,
		Roadmaps: hexRoadmaps(d.Roadmaps), Words: fromWordDocs(d.Words), WordsSeeded: d.WordsSeeded, CreatedAt: d.CreatedAt,
	}
}

// MigrateSharedTopics turns the topics of one level into topics shared by every level
// (2026-10-07), once: topics of the same name are merged (topic.PlanMerge), each old roadmap
// becoming the roadmap of its level, and the lessons, goals and lesson progress of the merged
// topics move to the kept one. Every write can be repeated and the plan of a half-merged state
// finishes the job, so a run interrupted at any point is completed by the next start.
func MigrateSharedTopics(ctx context.Context, db *mongo.Database, log *slog.Logger) error {
	migrations := db.Collection("migrations")
	err := migrations.FindOne(ctx, bson.D{{Key: "_id", Value: sharedTopicsMigrationID}}).Err()
	if err == nil {
		return nil
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return fmt.Errorf("read migrations: %w", err)
	}

	// The old unique keys would refuse merged data: one name per level, one goal per topic.
	for coll, name := range map[string]string{"topics": "level_1_nameKey_1", "goals": "userId_1_topicId_1"} {
		if err := db.Collection(coll).Indexes().DropOne(ctx, name); err != nil {
			log.DebugContext(ctx, "old index not dropped", slog.String("index", name), slog.Any("error", err))
		}
	}

	topics := db.Collection("topics")
	cur, err := topics.Find(ctx, bson.D{})
	if err != nil {
		return fmt.Errorf("find topics: %w", err)
	}
	var docs []mergeTopicDoc
	if err := cur.All(ctx, &docs); err != nil {
		return fmt.Errorf("read topics: %w", err)
	}
	sources := make([]topic.MergeSource, len(docs))
	for i, d := range docs {
		sources[i] = d.toSource()
	}

	merged, removed := 0, 0
	now := time.Now().UTC()
	for _, g := range topic.PlanMerge(sources) {
		if !g.Changed {
			continue
		}
		if err := applyMerge(ctx, db, g, now); err != nil {
			return err
		}
		merged++
		removed += len(g.Remove)
	}

	if _, err := migrations.InsertOne(ctx, bson.D{{Key: "_id", Value: sharedTopicsMigrationID}, {Key: "doneAt", Value: now}}); err != nil {
		return fmt.Errorf("mark migration: %w", err)
	}
	log.InfoContext(ctx, "shared topics migration done", slog.Int("topics", merged), slog.Int("merged away", removed))
	return nil
}

// applyMerge writes the kept topic, moves what pointed at the merged ones, then deletes them.
func applyMerge(ctx context.Context, db *mongo.Database, g topic.MergeGroup, now time.Time) error {
	keep, err := bson.ObjectIDFromHex(g.Keep.ID)
	if err != nil {
		return fmt.Errorf("topic id: %w", err)
	}
	roadmaps, err := oidRoadmaps(g.Keep.Roadmaps)
	if err != nil {
		return err
	}
	_, err = db.Collection("topics").UpdateOne(ctx, bson.D{{Key: "_id", Value: keep}}, bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "name", Value: g.Keep.Name},
			{Key: "nameKey", Value: topic.NameKey(g.Keep.Name)},
			{Key: "description", Value: g.Keep.Description},
			{Key: "roadmaps", Value: roadmaps},
			{Key: "words", Value: toWordDocs(g.Keep.Words)},
			{Key: "wordsSeeded", Value: g.Keep.WordsSeeded},
			{Key: "createdAt", Value: g.Keep.CreatedAt.UTC()},
			{Key: "updatedAt", Value: now},
		}},
		{Key: "$unset", Value: bson.D{{Key: "level", Value: ""}, {Key: "lessonIds", Value: ""}}},
	})
	if err != nil {
		return fmt.Errorf("write merged topic %s: %w", g.Keep.Name, err)
	}
	if len(g.Remove) == 0 {
		return nil
	}
	old, err := toOIDs(g.Remove)
	if err != nil {
		return err
	}
	inOld := bson.D{{Key: "topicId", Value: bson.D{{Key: "$in", Value: old}}}}
	moveTo := bson.D{{Key: "$set", Value: bson.D{{Key: "topicId", Value: keep}}}}
	for _, coll := range []string{"lessons", "goals", "lesson_progress"} {
		if _, err := db.Collection(coll).UpdateMany(ctx, inOld, moveTo); err != nil {
			return fmt.Errorf("move %s to merged topic: %w", coll, err)
		}
	}
	if _, err := db.Collection("topics").DeleteMany(ctx, bson.D{{Key: "_id", Value: bson.D{{Key: "$in", Value: old}}}}); err != nil {
		return fmt.Errorf("delete merged topics: %w", err)
	}
	return nil
}
