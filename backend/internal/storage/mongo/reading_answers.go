package mongo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/luongtran/luna/backend/internal/lesson"
)

type readingAnswerDoc struct {
	ID            bson.ObjectID `bson:"_id,omitempty"`
	UserID        bson.ObjectID `bson:"userId"`
	LessonID      bson.ObjectID `bson:"lessonId"`
	QuizVersion   int           `bson:"quizVersion"`
	QuestionIndex int           `bson:"questionIndex"`
	Choice        int           `bson:"choice"`
	Correct       bool          `bson:"correct"`
	AnsweredAt    time.Time     `bson:"answeredAt"`
}

func (d readingAnswerDoc) toAnswer() lesson.Answer {
	return lesson.Answer{
		UserID: d.UserID.Hex(), LessonID: d.LessonID.Hex(), QuizVersion: d.QuizVersion, QuestionIndex: d.QuestionIndex,
		Choice: d.Choice, Correct: d.Correct, AnsweredAt: d.AnsweredAt,
	}
}

// ReadingAnswers implements lesson.AnswerRepository on "reading_answers" (F15). The unique
// (userId, lessonId, quizVersion, questionIndex) index keeps one answer per question: the latest.
type ReadingAnswers struct {
	coll *mongo.Collection
}

// NewReadingAnswers returns a reading answer repository on db.
func NewReadingAnswers(db *mongo.Database) *ReadingAnswers {
	return &ReadingAnswers{coll: db.Collection("reading_answers")}
}

var _ lesson.AnswerRepository = (*ReadingAnswers)(nil)

func answerIDs(userID, lessonID string) (bson.ObjectID, bson.ObjectID, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return bson.ObjectID{}, bson.ObjectID{}, fmt.Errorf("answer user id: %w", err)
	}
	lid, err := bson.ObjectIDFromHex(lessonID)
	if err != nil {
		return bson.ObjectID{}, bson.ObjectID{}, fmt.Errorf("answer lesson id: %w", err)
	}
	return uid, lid, nil
}

// answerUpsert returns the filter and update that store a as the answer to its question.
func answerUpsert(uid, lid bson.ObjectID, a lesson.Answer) (filter, update bson.D) {
	filter = bson.D{
		{Key: "userId", Value: uid},
		{Key: "lessonId", Value: lid},
		{Key: "quizVersion", Value: a.QuizVersion},
		{Key: "questionIndex", Value: a.QuestionIndex},
	}
	update = bson.D{{Key: "$set", Value: bson.D{
		{Key: "choice", Value: a.Choice},
		{Key: "correct", Value: a.Correct},
		{Key: "answeredAt", Value: a.AnsweredAt.UTC()},
	}}}
	return filter, update
}

// Upsert stores a, replacing the learner's earlier answer to the same question.
func (r *ReadingAnswers) Upsert(ctx context.Context, a lesson.Answer) error {
	uid, lid, err := answerIDs(a.UserID, a.LessonID)
	if err != nil {
		return err
	}
	filter, update := answerUpsert(uid, lid, a)
	_, err = r.coll.UpdateOne(ctx, filter, update, options.UpdateOne().SetUpsert(true))
	if mongo.IsDuplicateKeyError(err) {
		// Two upserts raced to insert the same question; the retry updates the winner's document.
		_, err = r.coll.UpdateOne(ctx, filter, update, options.UpdateOne().SetUpsert(true))
	}
	if err != nil {
		return fmt.Errorf("upsert reading answer: %w", err)
	}
	return nil
}

// Delete removes a learner's answers to one question-set version of a lesson.
func (r *ReadingAnswers) Delete(ctx context.Context, userID, lessonID string, version int) error {
	uid, lid, err := answerIDs(userID, lessonID)
	if err != nil {
		return err
	}
	_, err = r.coll.DeleteMany(ctx,
		bson.D{{Key: "userId", Value: uid}, {Key: "lessonId", Value: lid}, {Key: "quizVersion", Value: version}})
	if err != nil {
		return fmt.Errorf("delete reading answers: %w", err)
	}
	return nil
}

// List returns a learner's answers to one question-set version of a lesson, by question.
func (r *ReadingAnswers) List(ctx context.Context, userID, lessonID string, version int) ([]lesson.Answer, error) {
	uid, lid, err := answerIDs(userID, lessonID)
	if err != nil {
		return nil, err
	}
	cur, err := r.coll.Find(ctx,
		bson.D{{Key: "userId", Value: uid}, {Key: "lessonId", Value: lid}, {Key: "quizVersion", Value: version}},
		options.Find().SetSort(bson.D{{Key: "questionIndex", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("find reading answers: %w", err)
	}
	var docs []readingAnswerDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("read reading answers: %w", err)
	}
	out := make([]lesson.Answer, len(docs))
	for i, d := range docs {
		out[i] = d.toAnswer()
	}
	return out, nil
}

// Totals counts the learner's answers given at or after since (nil = all) and the correct ones.
func (r *ReadingAnswers) Totals(ctx context.Context, userID string, since *time.Time) (answered, correct int, err error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return 0, 0, fmt.Errorf("answer user id: %w", err)
	}
	cur, err := r.coll.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: withSince(bson.D{{Key: "userId", Value: uid}}, "answeredAt", since)}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "answered", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "correct", Value: bson.D{{Key: "$sum", Value: bson.D{{Key: "$cond", Value: bson.A{"$correct", 1, 0}}}}}},
		}}},
	})
	if err != nil {
		return 0, 0, fmt.Errorf("aggregate reading answers: %w", err)
	}
	var rows []struct {
		Answered int `bson:"answered"`
		Correct  int `bson:"correct"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return 0, 0, fmt.Errorf("read reading answer totals: %w", err)
	}
	if len(rows) == 0 {
		return 0, 0, nil
	}
	return rows[0].Answered, rows[0].Correct, nil
}
