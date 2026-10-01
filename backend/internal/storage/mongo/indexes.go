package mongo

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// EnsureIndexes creates the indexes the repositories rely on. It is idempotent.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("users").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("users.email index: %w", err)
	}

	_, err = db.Collection("sessions").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "tokenHash", Value: 1}}, Options: options.Index().SetUnique(true)},
		// MongoDB removes a session once its expiresAt has passed.
		{Keys: bson.D{{Key: "expiresAt", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)},
	})
	if err != nil {
		return fmt.Errorf("sessions indexes: %w", err)
	}

	_, err = db.Collection("lessons").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "createdAt", Value: -1}}},
		{Keys: bson.D{{Key: "level", Value: 1}}},
		{Keys: bson.D{{Key: "topicId", Value: 1}}},
	})
	if err != nil {
		return fmt.Errorf("lessons indexes: %w", err)
	}

	_, err = db.Collection("topics").Indexes().CreateOne(ctx, mongo.IndexModel{
		// Topic names are unique within a level, ignoring case and extra spaces (F14).
		Keys:    bson.D{{Key: "level", Value: 1}, {Key: "nameKey", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("topics index: %w", err)
	}

	_, err = db.Collection("jobs").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "status", Value: 1}, {Key: "runAt", Value: 1}}},
		{Keys: bson.D{{Key: "lessonId", Value: 1}}},
	})
	if err != nil {
		return fmt.Errorf("jobs indexes: %w", err)
	}

	_, err = db.Collection("cards").Indexes().CreateMany(ctx, []mongo.IndexModel{
		// One card per base form in each learner notebook.
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "lemma", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "createdAt", Value: 1}}},
		// Review queue (F5) and notebook filtered by lesson.
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "due", Value: 1}}},
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "lessonId", Value: 1}, {Key: "createdAt", Value: -1}}},
	})
	if err != nil {
		return fmt.Errorf("cards indexes: %w", err)
	}

	_, err = db.Collection("review_logs").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "cardId", Value: 1}}},
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "reviewedAt", Value: 1}}},
		// Daily review limit (L).
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "context", Value: 1}, {Key: "reviewedAt", Value: 1}}},
	})
	if err != nil {
		return fmt.Errorf("review_logs indexes: %w", err)
	}

	// Daily flow (L).
	_, err = db.Collection("goals").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "topicId", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "status", Value: 1}}},
	})
	if err != nil {
		return fmt.Errorf("goals indexes: %w", err)
	}
	_, err = db.Collection("lesson_progress").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "lessonId", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "completedAt", Value: -1}}},
	})
	if err != nil {
		return fmt.Errorf("lesson_progress indexes: %w", err)
	}
	_, err = db.Collection("study_days").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "userId", Value: 1}, {Key: "dayKey", Value: 1}}, Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("study_days index: %w", err)
	}

	_, err = db.Collection("dictation_results").Indexes().CreateOne(ctx, mongo.IndexModel{
		// One result per sentence: the latest check replaces the previous one.
		Keys:    bson.D{{Key: "userId", Value: 1}, {Key: "lessonId", Value: 1}, {Key: "sentenceIndex", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("dictation_results index: %w", err)
	}

	_, err = db.Collection("reading_answers").Indexes().CreateOne(ctx, mongo.IndexModel{
		// F15: one answer per question of a question-set version; a second answer is refused.
		Keys: bson.D{
			{Key: "userId", Value: 1},
			{Key: "lessonId", Value: 1},
			{Key: "quizVersion", Value: 1},
			{Key: "questionIndex", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("reading_answers index: %w", err)
	}

	// F8: one writing per learner and lesson; the list by date; the new-result count.
	_, err = db.Collection("writings").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "lessonId", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "submittedAt", Value: -1}}},
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "grade.status", Value: 1}, {Key: "grade.seen", Value: 1}}},
	})
	if err != nil {
		return fmt.Errorf("writings indexes: %w", err)
	}

	// F9: one AI explanation per lesson revision, sentence and text, shared by everyone.
	_, err = db.Collection("ai_lookups").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "lessonId", Value: 1},
			{Key: "revision", Value: 1},
			{Key: "sentenceIndex", Value: 1},
			{Key: "textLower", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("ai_lookups index: %w", err)
	}
	return nil
}

// PrepareInBackground runs the data migrations (MigrateTopics, MigrateUserSettings) and then EnsureIndexes, retrying
// every interval until both succeed or ctx ends, so the backend can start while the database is
// still down. With the database up, this finishes within moments of starting.
func PrepareInBackground(ctx context.Context, db *mongo.Database, interval time.Duration, log *slog.Logger) {
	go func() {
		for {
			err := MigrateTopics(ctx, db, log)
			if err == nil {
				err = MigrateUserSettings(ctx, db, log)
			}
			if err == nil {
				err = EnsureIndexes(ctx, db)
			}
			if err == nil {
				log.InfoContext(ctx, "mongo ready: migrations and indexes")
				return
			}
			log.WarnContext(ctx, "mongo not prepared, will retry", slog.Any("error", err))
			select {
			case <-ctx.Done():
				return
			case <-time.After(interval):
			}
		}
	}()
}
