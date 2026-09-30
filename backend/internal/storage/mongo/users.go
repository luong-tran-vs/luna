package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/luongtran/luna/backend/internal/auth"
)

type userDoc struct {
	ID           bson.ObjectID `bson:"_id,omitempty"`
	Email        string        `bson:"email"`
	PasswordHash string        `bson:"passwordHash"`
	Role         string        `bson:"role"`
	// Timezone is the F1 field, moved to Settings.Timezone by the users-settings-timezone migration.
	Timezone  string      `bson:"timezone,omitempty"`
	Settings  settingsDoc `bson:"settings,omitempty"`
	CreatedAt time.Time   `bson:"createdAt"`
}

// settingsDoc is users.settings (F12); missing fields mean defaults.
type settingsDoc struct {
	Theme            string `bson:"theme,omitempty"`
	DailyReviewLimit int    `bson:"dailyReviewLimit,omitempty"`
	Timezone         string `bson:"timezone,omitempty"`
}

// timezone is the stored timezone, from settings or the F1 field not migrated yet.
func (d userDoc) timezone() string {
	if d.Settings.Timezone != "" {
		return d.Settings.Timezone
	}
	return d.Timezone
}

func (d userDoc) toUser() auth.User {
	return auth.User{
		ID:           d.ID.Hex(),
		Email:        d.Email,
		PasswordHash: d.PasswordHash,
		Role:         auth.Role(d.Role),
		Timezone:     d.timezone(),
		CreatedAt:    d.CreatedAt,
	}
}

// Users implements auth.UserRepository on the "users" collection.
type Users struct {
	coll *mongo.Collection
}

// NewUsers returns a user repository on db.
func NewUsers(db *mongo.Database) *Users {
	return &Users{coll: db.Collection("users")}
}

var _ auth.UserRepository = (*Users)(nil)

// Create inserts u; a duplicate email returns auth.ErrEmailTaken.
func (r *Users) Create(ctx context.Context, u auth.User) (auth.User, error) {
	doc := userDoc{
		ID:           bson.NewObjectID(),
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Role:         string(u.Role),
		Settings:     settingsDoc{Timezone: u.Timezone},
		CreatedAt:    u.CreatedAt.UTC(),
	}
	if _, err := r.coll.InsertOne(ctx, doc); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return auth.User{}, auth.ErrEmailTaken
		}
		return auth.User{}, fmt.Errorf("insert user: %w", err)
	}
	return doc.toUser(), nil
}

// FindByEmail looks up a normalized email.
func (r *Users) FindByEmail(ctx context.Context, email string) (auth.User, error) {
	return r.findOne(ctx, bson.D{{Key: "email", Value: email}})
}

// FindByID looks up a hex ObjectID; malformed IDs are reported as not found.
func (r *Users) FindByID(ctx context.Context, id string) (auth.User, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return auth.User{}, auth.ErrNotFound
	}
	return r.findOne(ctx, bson.D{{Key: "_id", Value: oid}})
}

// Count returns the number of accounts.
func (r *Users) Count(ctx context.Context) (int64, error) {
	n, err := r.coll.CountDocuments(ctx, bson.D{})
	if err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return n, nil
}

func (r *Users) findOne(ctx context.Context, filter bson.D) (auth.User, error) {
	var doc userDoc
	if err := r.coll.FindOne(ctx, filter).Decode(&doc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return auth.User{}, auth.ErrNotFound
		}
		return auth.User{}, fmt.Errorf("find user: %w", err)
	}
	return doc.toUser(), nil
}
