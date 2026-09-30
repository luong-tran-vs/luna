package mongo

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

const settingsMigrationID = "f12-users-settings-timezone"

// MigrateUserSettings moves the F1 users.timezone field to users.settings.timezone (F12) once.
// The update is idempotent and the marker is written last, so an interrupted run is completed by
// the next start; until then the repositories read the old field as a fallback.
func MigrateUserSettings(ctx context.Context, db *mongo.Database, log *slog.Logger) error {
	migrations := db.Collection("migrations")
	err := migrations.FindOne(ctx, bson.D{{Key: "_id", Value: settingsMigrationID}}).Err()
	if err == nil {
		return nil
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return fmt.Errorf("read migrations: %w", err)
	}
	res, err := db.Collection("users").UpdateMany(ctx,
		bson.D{
			{Key: "timezone", Value: bson.D{{Key: "$exists", Value: true}}},
			{Key: "settings.timezone", Value: bson.D{{Key: "$exists", Value: false}}},
		},
		mongo.Pipeline{
			{{Key: "$set", Value: bson.D{{Key: "settings.timezone", Value: "$timezone"}}}},
			{{Key: "$unset", Value: "timezone"}},
		})
	if err != nil {
		return fmt.Errorf("move user timezones: %w", err)
	}
	// Users already migrated by an interrupted run may still carry the old field.
	if _, err := db.Collection("users").UpdateMany(ctx,
		bson.D{{Key: "timezone", Value: bson.D{{Key: "$exists", Value: true}}}},
		bson.D{{Key: "$unset", Value: bson.D{{Key: "timezone", Value: ""}}}}); err != nil {
		return fmt.Errorf("drop old user timezones: %w", err)
	}
	if _, err := migrations.InsertOne(ctx, bson.D{{Key: "_id", Value: settingsMigrationID}, {Key: "doneAt", Value: time.Now().UTC()}}); err != nil {
		return fmt.Errorf("mark migration: %w", err)
	}
	log.InfoContext(ctx, "settings migration done", slog.Int64("users", res.ModifiedCount))
	return nil
}
