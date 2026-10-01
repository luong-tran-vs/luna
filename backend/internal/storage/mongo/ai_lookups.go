package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/luongtran/luna/backend/internal/lesson"
)

type aiLookupDoc struct {
	ID            bson.ObjectID `bson:"_id,omitempty"`
	LessonID      bson.ObjectID `bson:"lessonId"`
	Revision      int           `bson:"revision"`
	SentenceIndex int           `bson:"sentenceIndex"`
	TextLower     string        `bson:"textLower"`
	Lemma         string        `bson:"lemma"`
	MeaningVi     string        `bson:"meaningVi"`
	NoteVi        string        `bson:"noteVi"`
	CreatedAt     time.Time     `bson:"createdAt"`
}

func (d aiLookupDoc) toResult() lesson.AskResult {
	return lesson.AskResult{
		AskKey: lesson.AskKey{
			LessonID: d.LessonID.Hex(), Revision: d.Revision, SentenceIndex: d.SentenceIndex, Text: d.TextLower,
		},
		Lemma: d.Lemma, MeaningVi: d.MeaningVi, NoteVi: d.NoteVi, CreatedAt: d.CreatedAt,
	}
}

// AILookups implements lesson.AskRepository on "ai_lookups" (F9): AI explanations shared by
// every learner, one per (lessonId, revision, sentenceIndex, textLower).
type AILookups struct {
	coll *mongo.Collection
}

// NewAILookups returns an AI lookup repository on db.
func NewAILookups(db *mongo.Database) *AILookups {
	return &AILookups{coll: db.Collection("ai_lookups")}
}

var _ lesson.AskRepository = (*AILookups)(nil)

func askFilter(key lesson.AskKey) (bson.D, error) {
	lid, err := bson.ObjectIDFromHex(key.LessonID)
	if err != nil {
		return nil, lesson.ErrNotFound
	}
	return bson.D{
		{Key: "lessonId", Value: lid},
		{Key: "revision", Value: key.Revision},
		{Key: "sentenceIndex", Value: key.SentenceIndex},
		{Key: "textLower", Value: key.Text},
	}, nil
}

// Get returns the stored explanation for key.
func (r *AILookups) Get(ctx context.Context, key lesson.AskKey) (lesson.AskResult, bool, error) {
	filter, err := askFilter(key)
	if err != nil {
		return lesson.AskResult{}, false, nil //nolint:nilerr // a malformed lesson id has no stored explanation
	}
	var d aiLookupDoc
	err = r.coll.FindOne(ctx, filter).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return lesson.AskResult{}, false, nil
	}
	if err != nil {
		return lesson.AskResult{}, false, fmt.Errorf("find ai lookup: %w", err)
	}
	return d.toResult(), true, nil
}

// Put stores an explanation; when the key is already stored it returns that one.
func (r *AILookups) Put(ctx context.Context, res lesson.AskResult) (lesson.AskResult, error) {
	lid, err := bson.ObjectIDFromHex(res.LessonID)
	if err != nil {
		return lesson.AskResult{}, lesson.ErrNotFound
	}
	_, err = r.coll.InsertOne(ctx, aiLookupDoc{
		LessonID: lid, Revision: res.Revision, SentenceIndex: res.SentenceIndex, TextLower: res.Text,
		Lemma: res.Lemma, MeaningVi: res.MeaningVi, NoteVi: res.NoteVi, CreatedAt: res.CreatedAt.UTC(),
	})
	switch {
	case err == nil:
		return res, nil
	case !mongo.IsDuplicateKeyError(err):
		return lesson.AskResult{}, fmt.Errorf("insert ai lookup: %w", err)
	}
	stored, ok, err := r.Get(ctx, res.AskKey)
	if err != nil {
		return lesson.AskResult{}, err
	}
	if !ok {
		return lesson.AskResult{}, errors.New("ai lookup vanished after a duplicate key")
	}
	return stored, nil
}
