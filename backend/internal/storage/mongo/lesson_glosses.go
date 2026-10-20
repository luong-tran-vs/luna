package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/luongtran/luna/backend/internal/lesson"
)

// collLessonGlosses holds the glosses of each lesson for one revision (added 2026-10-20).
const collLessonGlosses = "lesson_glosses"

type glossDoc struct {
	Text          string `bson:"text"`
	Lemma         string `bson:"lemma"`
	MeaningVi     string `bson:"meaningVi"`
	SentenceIndex int    `bson:"sentenceIndex"`
	POS           string `bson:"pos,omitempty"`
	Phrase        string `bson:"phrase,omitempty"`
	PhraseVi      string `bson:"phraseVi,omitempty"`
}

type lessonGlossesDoc struct {
	LessonID  bson.ObjectID `bson:"_id"`
	Revision  int           `bson:"revision"`
	Glosses   []glossDoc    `bson:"glosses"`
	UpdatedAt time.Time     `bson:"updatedAt"`
}

// LessonGlosses implements lesson.GlossRepository on "lesson_glosses", one document per lesson.
type LessonGlosses struct {
	coll *mongo.Collection
}

// NewLessonGlosses returns a gloss repository on db.
func NewLessonGlosses(db *mongo.Database) *LessonGlosses {
	return &LessonGlosses{coll: db.Collection(collLessonGlosses)}
}

var _ lesson.GlossRepository = (*LessonGlosses)(nil)

// Get returns the lesson's glosses and the revision they were written for.
func (r *LessonGlosses) Get(ctx context.Context, lessonID string) (int, []lesson.Gloss, bool, error) {
	id, err := bson.ObjectIDFromHex(lessonID)
	if err != nil {
		return 0, nil, false, nil
	}
	var doc lessonGlossesDoc
	err = r.coll.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return 0, nil, false, nil
	}
	if err != nil {
		return 0, nil, false, fmt.Errorf("get glosses: %w", err)
	}
	out := make([]lesson.Gloss, len(doc.Glosses))
	for i, g := range doc.Glosses {
		out[i] = lesson.Gloss(g)
	}
	return doc.Revision, out, true, nil
}

// Save replaces the lesson's glosses.
func (r *LessonGlosses) Save(ctx context.Context, lessonID string, revision int, glosses []lesson.Gloss) error {
	id, err := bson.ObjectIDFromHex(lessonID)
	if err != nil {
		return fmt.Errorf("glosses lesson id: %w", err)
	}
	docs := make([]glossDoc, len(glosses))
	for i, g := range glosses {
		docs[i] = glossDoc(g)
	}
	doc := lessonGlossesDoc{LessonID: id, Revision: revision, Glosses: docs, UpdatedAt: time.Now().UTC()}
	if _, err := r.coll.ReplaceOne(ctx, bson.D{{Key: "_id", Value: id}}, doc, options.Replace().SetUpsert(true)); err != nil {
		return fmt.Errorf("save glosses: %w", err)
	}
	return nil
}

// Revisions gives the revision of each lesson's glosses.
func (r *LessonGlosses) Revisions(ctx context.Context) (map[string]int, error) {
	cur, err := r.coll.Find(ctx, bson.D{}, options.Find().SetProjection(bson.D{{Key: "revision", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("find gloss revisions: %w", err)
	}
	var docs []struct {
		ID       bson.ObjectID `bson:"_id"`
		Revision int           `bson:"revision"`
	}
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("read gloss revisions: %w", err)
	}
	out := make(map[string]int, len(docs))
	for _, d := range docs {
		out[d.ID.Hex()] = d.Revision
	}
	return out, nil
}
