package mongo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/luongtran/luna/backend/internal/export"
	"github.com/luongtran/luna/backend/internal/verbs"
)

// verbProgressDoc is one learner's record of one irregular verb; learnedAt and dueAt are absent for
// a verb not learned.
type verbProgressDoc struct {
	UserID     bson.ObjectID `bson:"userId"`
	Base       string        `bson:"base"`
	LearnedAt  *time.Time    `bson:"learnedAt,omitempty"`
	Level      int           `bson:"level"`
	DueAt      *time.Time    `bson:"dueAt,omitempty"`
	RightCount int           `bson:"rightCount"`
	WrongCount int           `bson:"wrongCount"`
	UpdatedAt  time.Time     `bson:"updatedAt"`
}

// VerbProgress implements verbs.Repository on "verb_progress"; the unique (userId, base) index
// keeps one document per verb.
type VerbProgress struct {
	coll *mongo.Collection
}

// NewVerbProgress returns a verb progress repository on db.
func NewVerbProgress(db *mongo.Database) *VerbProgress {
	return &VerbProgress{coll: db.Collection(export.CollVerbProgress)}
}

var _ verbs.Repository = (*VerbProgress)(nil)

// List returns every record of the user, by verb.
func (r *VerbProgress) List(ctx context.Context, userID string) ([]verbs.Progress, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, fmt.Errorf("verb progress user id: %w", err)
	}
	cur, err := r.coll.Find(ctx, bson.D{{Key: "userId", Value: uid}}, options.Find().SetSort(bson.D{{Key: "base", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("find verb progress: %w", err)
	}
	var docs []verbProgressDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("read verb progress: %w", err)
	}
	out := make([]verbs.Progress, len(docs))
	for i, d := range docs {
		out[i] = verbs.Progress{
			UserID: userID, Base: d.Base, Level: d.Level, Right: d.RightCount, Wrong: d.WrongCount,
			UpdatedAt: d.UpdatedAt.UTC(),
		}
		if d.DueAt != nil {
			out[i].DueAt = d.DueAt.UTC()
		}
	}
	return out, nil
}

// Save writes the records in one bulk request, each replacing the user's document of its verb.
func (r *VerbProgress) Save(ctx context.Context, records []verbs.Progress) error {
	if len(records) == 0 {
		return nil
	}
	models := make([]mongo.WriteModel, 0, len(records))
	for _, p := range records {
		uid, err := bson.ObjectIDFromHex(p.UserID)
		if err != nil {
			return fmt.Errorf("verb progress user id: %w", err)
		}
		doc := verbProgressDoc{
			UserID: uid, Base: p.Base, Level: p.Level, RightCount: p.Right, WrongCount: p.Wrong,
			UpdatedAt: p.UpdatedAt.UTC(),
		}
		if !p.DueAt.IsZero() {
			due := p.DueAt.UTC()
			doc.DueAt = &due
		}
		models = append(models, mongo.NewReplaceOneModel().
			SetFilter(bson.D{{Key: "userId", Value: uid}, {Key: "base", Value: p.Base}}).
			SetReplacement(doc).
			SetUpsert(true))
	}
	if _, err := r.coll.BulkWrite(ctx, models); err != nil {
		return fmt.Errorf("save verb progress: %w", err)
	}
	return nil
}
