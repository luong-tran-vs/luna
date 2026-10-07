package mongo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/luongtran/luna/backend/internal/progress"
)

type dictationDoc struct {
	ID             bson.ObjectID `bson:"_id,omitempty"`
	UserID         bson.ObjectID `bson:"userId"`
	LessonID       bson.ObjectID `bson:"lessonId"`
	LessonRevision int           `bson:"lessonRevision"`
	SentenceIndex  int           `bson:"sentenceIndex"`
	Typed          string        `bson:"typed"`
	CorrectWords   int           `bson:"correctWords"`
	TotalWords     int           `bson:"totalWords"`
	CheckedAt      time.Time     `bson:"checkedAt"`
}

// DictationResults implements progress.DictationRepository on "dictation_results".
type DictationResults struct {
	coll *mongo.Collection
}

// NewDictationResults returns a dictation result repository on db.
func NewDictationResults(db *mongo.Database) *DictationResults {
	return &DictationResults{coll: db.Collection("dictation_results")}
}

var _ progress.DictationRepository = (*DictationResults)(nil)

func dictationIDs(userID, lessonID string) (bson.ObjectID, bson.ObjectID, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return bson.ObjectID{}, bson.ObjectID{}, fmt.Errorf("dictation user id: %w", err)
	}
	lid, err := bson.ObjectIDFromHex(lessonID)
	if err != nil {
		return bson.ObjectID{}, bson.ObjectID{}, fmt.Errorf("dictation lesson id: %w", err)
	}
	return uid, lid, nil
}

// Upsert replaces the result of one sentence; the unique (userId, lessonId, sentenceIndex)
// index keeps one document per sentence.
func (r *DictationResults) Upsert(ctx context.Context, userID, lessonID string, revision int, res progress.Result) error {
	uid, lid, err := dictationIDs(userID, lessonID)
	if err != nil {
		return err
	}
	filter := bson.D{{Key: "userId", Value: uid}, {Key: "lessonId", Value: lid}, {Key: "sentenceIndex", Value: res.SentenceIndex}}
	update := bson.D{{Key: "$set", Value: bson.D{
		{Key: "lessonRevision", Value: revision},
		{Key: "typed", Value: res.Typed},
		{Key: "correctWords", Value: res.CorrectWords},
		{Key: "totalWords", Value: res.TotalWords},
		{Key: "checkedAt", Value: res.CheckedAt.UTC()},
	}}}
	if _, err := r.coll.UpdateOne(ctx, filter, update, options.UpdateOne().SetUpsert(true)); err != nil {
		return fmt.Errorf("upsert dictation result: %w", err)
	}
	return nil
}

// List returns the user's results for the lesson.
func (r *DictationResults) List(ctx context.Context, userID, lessonID string) ([]progress.StoredResult, error) {
	uid, lid, err := dictationIDs(userID, lessonID)
	if err != nil {
		return nil, err
	}
	cur, err := r.coll.Find(ctx, bson.D{{Key: "userId", Value: uid}, {Key: "lessonId", Value: lid}})
	if err != nil {
		return nil, fmt.Errorf("find dictation results: %w", err)
	}
	var docs []dictationDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("read dictation results: %w", err)
	}
	out := make([]progress.StoredResult, len(docs))
	for i, d := range docs {
		out[i] = progress.StoredResult{
			Result: progress.Result{
				SentenceIndex: d.SentenceIndex, Typed: d.Typed, CorrectWords: d.CorrectWords,
				TotalWords: d.TotalWords, CheckedAt: d.CheckedAt,
			},
			Revision: d.LessonRevision,
		}
	}
	return out, nil
}

// Totals sums the user's results over every lesson, checked at or after since (nil = all).
func (r *DictationResults) Totals(ctx context.Context, userID string, since *time.Time) (progress.DictationTotals, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return progress.DictationTotals{}, fmt.Errorf("dictation user id: %w", err)
	}
	cur, err := r.coll.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: withSince(bson.D{{Key: "userId", Value: uid}}, "checkedAt", since)}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "sentences", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "correct", Value: bson.D{{Key: "$sum", Value: "$correctWords"}}},
			{Key: "total", Value: bson.D{{Key: "$sum", Value: "$totalWords"}}},
			{Key: "lessons", Value: bson.D{{Key: "$addToSet", Value: "$lessonId"}}},
		}}},
	})
	if err != nil {
		return progress.DictationTotals{}, fmt.Errorf("aggregate dictation totals: %w", err)
	}
	var rows []struct {
		Sentences int             `bson:"sentences"`
		Correct   int             `bson:"correct"`
		Total     int             `bson:"total"`
		Lessons   []bson.ObjectID `bson:"lessons"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return progress.DictationTotals{}, fmt.Errorf("read dictation totals: %w", err)
	}
	if len(rows) == 0 {
		return progress.DictationTotals{}, nil
	}
	return progress.DictationTotals{
		Sentences: rows[0].Sentences, CorrectWords: rows[0].Correct, TotalWords: rows[0].Total, Lessons: len(rows[0].Lessons),
	}, nil
}

// Delete removes the user's results for the lesson, of every revision.
func (r *DictationResults) Delete(ctx context.Context, userID, lessonID string) error {
	uid, lid, err := dictationIDs(userID, lessonID)
	if err != nil {
		return err
	}
	if _, err := r.coll.DeleteMany(ctx, bson.D{{Key: "userId", Value: uid}, {Key: "lessonId", Value: lid}}); err != nil {
		return fmt.Errorf("delete dictation results: %w", err)
	}
	return nil
}
