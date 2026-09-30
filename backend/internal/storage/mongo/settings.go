package mongo

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/luongtran/luna/backend/internal/settings"
)

// Settings implements settings.Repository on users.settings (F12).
type Settings struct {
	coll *mongo.Collection
}

// NewSettings returns a settings repository on db.
func NewSettings(db *mongo.Database) *Settings {
	return &Settings{coll: db.Collection("users")}
}

var _ settings.Repository = (*Settings)(nil)

func (d userDoc) toSettings() settings.Settings {
	return settings.Settings{
		Theme: settings.Theme(d.Settings.Theme), DailyReviewLimit: d.Settings.DailyReviewLimit, Timezone: d.timezone(),
	}
}

// Get returns the user's stored settings.
func (r *Settings) Get(ctx context.Context, userID string) (settings.Settings, error) {
	oid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return settings.Settings{}, settings.ErrNotFound
	}
	var d userDoc
	err = r.coll.FindOne(ctx, bson.D{{Key: "_id", Value: oid}},
		options.FindOne().SetProjection(bson.D{{Key: "timezone", Value: 1}, {Key: "settings", Value: 1}})).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return settings.Settings{}, settings.ErrNotFound
	}
	if err != nil {
		return settings.Settings{}, fmt.Errorf("find settings: %w", err)
	}
	return d.toSettings(), nil
}

// Update sets the non-nil fields of p.
func (r *Settings) Update(ctx context.Context, userID string, p settings.Patch) (settings.Settings, error) {
	oid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return settings.Settings{}, settings.ErrNotFound
	}
	set := bson.D{}
	if p.Theme != nil {
		set = append(set, bson.E{Key: "settings.theme", Value: string(*p.Theme)})
	}
	if p.DailyReviewLimit != nil {
		set = append(set, bson.E{Key: "settings.dailyReviewLimit", Value: *p.DailyReviewLimit})
	}
	if p.Timezone != nil {
		set = append(set, bson.E{Key: "settings.timezone", Value: *p.Timezone})
	}
	if len(set) == 0 {
		return r.Get(ctx, userID)
	}
	var d userDoc
	err = r.coll.FindOneAndUpdate(ctx, bson.D{{Key: "_id", Value: oid}}, bson.D{{Key: "$set", Value: set}},
		options.FindOneAndUpdate().SetReturnDocument(options.After).
			SetProjection(bson.D{{Key: "timezone", Value: 1}, {Key: "settings", Value: 1}})).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return settings.Settings{}, settings.ErrNotFound
	}
	if err != nil {
		return settings.Settings{}, fmt.Errorf("update settings: %w", err)
	}
	return d.toSettings(), nil
}
