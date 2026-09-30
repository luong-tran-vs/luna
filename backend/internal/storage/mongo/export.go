package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/luongtran/luna/backend/internal/export"
)

// Export implements export.Repository: read-only access to one learner's documents (F13).
type Export struct {
	db *mongo.Database
}

// NewExport returns an export repository on db.
func NewExport(db *mongo.Database) *Export {
	return &Export{db: db}
}

var _ export.Repository = (*Export)(nil)

// Account reads only the exported account fields; the password hash is never loaded.
func (r *Export) Account(ctx context.Context, userID string) (export.Account, error) {
	oid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return export.Account{}, export.ErrNotFound
	}
	var d struct {
		Email     string    `bson:"email"`
		Role      string    `bson:"role"`
		CreatedAt time.Time `bson:"createdAt"`
	}
	err = r.db.Collection("users").FindOne(ctx, bson.D{{Key: "_id", Value: oid}},
		options.FindOne().SetProjection(bson.D{{Key: "email", Value: 1}, {Key: "role", Value: 1}, {Key: "createdAt", Value: 1}})).
		Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return export.Account{}, export.ErrNotFound
	}
	if err != nil {
		return export.Account{}, fmt.Errorf("find account: %w", err)
	}
	return export.Account{ID: userID, Email: d.Email, Role: d.Role, CreatedAt: d.CreatedAt}, nil
}

// UserDocs returns the user's documents of an allowed collection, oldest first.
func (r *Export) UserDocs(ctx context.Context, collection, userID string) ([]export.Doc, error) {
	if !export.Allowed(collection) {
		return nil, export.ErrCollection
	}
	oid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, export.ErrNotFound
	}
	return r.find(ctx, collection, bson.D{{Key: "userId", Value: oid}})
}

// Lessons returns the lessons among ids that still exist.
func (r *Export) Lessons(ctx context.Context, ids []string) ([]export.Doc, error) {
	oids := make(bson.A, 0, len(ids))
	for _, id := range ids {
		if oid, err := bson.ObjectIDFromHex(id); err == nil {
			oids = append(oids, oid)
		}
	}
	if len(oids) == 0 {
		return []export.Doc{}, nil
	}
	return r.find(ctx, "lessons", bson.D{{Key: "_id", Value: bson.D{{Key: "$in", Value: oids}}}})
}

func (r *Export) find(ctx context.Context, collection string, filter bson.D) ([]export.Doc, error) {
	cur, err := r.db.Collection(collection).Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("find %s: %w", collection, err)
	}
	var raw []bson.M
	if err := cur.All(ctx, &raw); err != nil {
		return nil, fmt.Errorf("read %s: %w", collection, err)
	}
	out := make([]export.Doc, len(raw))
	for i, d := range raw {
		out[i] = normalizeDoc(d)
	}
	return out, nil
}

// normalizeDoc converts a stored document to JSON-friendly values: ObjectIDs as hex strings,
// dates as RFC 3339 strings, nested documents as maps.
func normalizeDoc(d bson.M) map[string]any {
	out := make(map[string]any, len(d))
	for k, v := range d {
		out[k] = normalize(v)
	}
	return out
}

func normalize(v any) any {
	switch x := v.(type) {
	case bson.ObjectID:
		return x.Hex()
	case bson.DateTime:
		return x.Time().UTC().Format(time.RFC3339Nano)
	case time.Time:
		return x.UTC().Format(time.RFC3339Nano)
	case bson.M:
		return normalizeDoc(x)
	case bson.D:
		out := make(map[string]any, len(x))
		for _, e := range x {
			out[e.Key] = normalize(e.Value)
		}
		return out
	case bson.A:
		return normalizeList(x)
	case []any:
		return normalizeList(x)
	default:
		return v
	}
}

func normalizeList(list []any) []any {
	out := make([]any, len(list))
	for i, v := range list {
		out[i] = normalize(v)
	}
	return out
}
