package mongo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/luongtran/luna/backend/internal/vocab"
)

// collLessonResults holds the latest lesson review round of each lesson, per user.
const collLessonResults = "lesson_results"

type lessonResultDoc struct {
	UserID     bson.ObjectID `bson:"userId"`
	LessonID   bson.ObjectID `bson:"lessonId"`
	Correct    int           `bson:"correct"`
	Total      int           `bson:"total"`
	Wrong      []string      `bson:"wrong"`
	ReviewedAt time.Time     `bson:"reviewedAt"`
}

// LessonResults implements vocab.LessonResultRepository on "lesson_results".
type LessonResults struct {
	coll *mongo.Collection
}

// NewLessonResults returns a lesson result repository on db.
func NewLessonResults(db *mongo.Database) *LessonResults {
	return &LessonResults{coll: db.Collection(collLessonResults)}
}

var _ vocab.LessonResultRepository = (*LessonResults)(nil)

// Save replaces the user's result for the lesson; the unique (userId, lessonId) index keeps one.
func (r *LessonResults) Save(ctx context.Context, res vocab.LessonResult) error {
	uid, err := bson.ObjectIDFromHex(res.UserID)
	if err != nil {
		return fmt.Errorf("lesson result user id: %w", err)
	}
	lid, err := bson.ObjectIDFromHex(res.LessonID)
	if err != nil {
		return fmt.Errorf("lesson result lesson id: %w", err)
	}
	wrong := res.Wrong
	if wrong == nil {
		wrong = []string{}
	}
	filter := bson.D{{Key: "userId", Value: uid}, {Key: "lessonId", Value: lid}}
	update := bson.D{{Key: "$set", Value: bson.D{
		{Key: "correct", Value: res.Correct},
		{Key: "total", Value: res.Total},
		{Key: "wrong", Value: wrong},
		{Key: "reviewedAt", Value: res.ReviewedAt.UTC()},
	}}}
	if _, err := r.coll.UpdateOne(ctx, filter, update, options.UpdateOne().SetUpsert(true)); err != nil {
		return fmt.Errorf("save lesson result: %w", err)
	}
	return nil
}

// List returns every result of the user.
func (r *LessonResults) List(ctx context.Context, userID string) ([]vocab.LessonResult, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, fmt.Errorf("lesson result user id: %w", err)
	}
	cur, err := r.coll.Find(ctx, bson.D{{Key: "userId", Value: uid}})
	if err != nil {
		return nil, fmt.Errorf("find lesson results: %w", err)
	}
	var docs []lessonResultDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("read lesson results: %w", err)
	}
	out := make([]vocab.LessonResult, len(docs))
	for i, d := range docs {
		out[i] = vocab.LessonResult{
			UserID: userID, LessonID: d.LessonID.Hex(), Correct: d.Correct, Total: d.Total, Wrong: d.Wrong,
			ReviewedAt: d.ReviewedAt.UTC(),
		}
	}
	return out, nil
}
