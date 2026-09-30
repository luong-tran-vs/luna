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

type sessionDoc struct {
	ID        bson.ObjectID `bson:"_id,omitempty"`
	TokenHash string        `bson:"tokenHash"`
	UserID    bson.ObjectID `bson:"userId"`
	ExpiresAt time.Time     `bson:"expiresAt"`
	CreatedAt time.Time     `bson:"createdAt"`
}

// Sessions implements auth.SessionRepository on the "sessions" collection.
type Sessions struct {
	coll *mongo.Collection
}

// NewSessions returns a session repository on db.
func NewSessions(db *mongo.Database) *Sessions {
	return &Sessions{coll: db.Collection("sessions")}
}

var _ auth.SessionRepository = (*Sessions)(nil)

// Create inserts a session.
func (r *Sessions) Create(ctx context.Context, s auth.Session) error {
	uid, err := bson.ObjectIDFromHex(s.UserID)
	if err != nil {
		return fmt.Errorf("session user id: %w", err)
	}
	doc := sessionDoc{
		ID:        bson.NewObjectID(),
		TokenHash: s.TokenHash,
		UserID:    uid,
		ExpiresAt: s.ExpiresAt.UTC(),
		CreatedAt: s.CreatedAt.UTC(),
	}
	if _, err := r.coll.InsertOne(ctx, doc); err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

// FindByTokenHash returns auth.ErrNotFound when no session matches.
func (r *Sessions) FindByTokenHash(ctx context.Context, hash string) (auth.Session, error) {
	var doc sessionDoc
	err := r.coll.FindOne(ctx, bson.D{{Key: "tokenHash", Value: hash}}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return auth.Session{}, auth.ErrNotFound
		}
		return auth.Session{}, fmt.Errorf("find session: %w", err)
	}
	return auth.Session{
		TokenHash: doc.TokenHash,
		UserID:    doc.UserID.Hex(),
		ExpiresAt: doc.ExpiresAt,
		CreatedAt: doc.CreatedAt,
	}, nil
}

// Delete removes the session; a missing session is not an error.
func (r *Sessions) Delete(ctx context.Context, hash string) error {
	if _, err := r.coll.DeleteOne(ctx, bson.D{{Key: "tokenHash", Value: hash}}); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}
