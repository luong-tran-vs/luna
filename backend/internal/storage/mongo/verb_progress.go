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
	// Mistakes and CleanRun: "Từ hay sai" (added 2026-11-10).
	Mistakes  []verbMistakeDoc `bson:"mistakes,omitempty"`
	CleanRun  int              `bson:"cleanRun,omitempty"`
	UpdatedAt time.Time        `bson:"updatedAt"`
}

type verbMistakeDoc struct {
	Form  string    `bson:"form"`
	Typed string    `bson:"typed"`
	At    time.Time `bson:"at"`
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
			CleanRun: d.CleanRun, UpdatedAt: d.UpdatedAt.UTC(),
		}
		// learnedAt was neither saved nor read before 2026-11-10, so no verb stayed learned on Mongo. A
		// verb in a review box was learned (as MySQL migration 031 did): since its last update.
		switch {
		case d.LearnedAt != nil:
			out[i].LearnedAt = d.LearnedAt.UTC()
		case d.Level > 0:
			out[i].LearnedAt = d.UpdatedAt.UTC()
		}
		for _, m := range d.Mistakes {
			out[i].Mistakes = append(out[i].Mistakes, verbs.Mistake{Form: m.Form, Typed: m.Typed, At: m.At.UTC()})
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
			CleanRun: p.CleanRun, UpdatedAt: p.UpdatedAt.UTC(),
		}
		if !p.LearnedAt.IsZero() {
			learned := p.LearnedAt.UTC()
			doc.LearnedAt = &learned
		}
		for _, m := range p.Mistakes {
			doc.Mistakes = append(doc.Mistakes, verbMistakeDoc{Form: m.Form, Typed: m.Typed, At: m.At.UTC()})
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
