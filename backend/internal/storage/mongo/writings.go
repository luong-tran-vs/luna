package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/luongtran/luna/backend/internal/writing"
)

type criterionDoc struct {
	Name      string `bson:"name"`
	Score     int    `bson:"score"`
	CommentVi string `bson:"commentVi"`
}

type gradeDoc struct {
	Status        string         `bson:"status"`
	Error         string         `bson:"error"`
	Criteria      []criterionDoc `bson:"criteria"`
	OverallVi     string         `bson:"overallVi"`
	CorrectedText string         `bson:"correctedText"`
	GradedAt      time.Time      `bson:"gradedAt,omitempty"`
	Seen          bool           `bson:"seen"`
}

type writingDoc struct {
	ID             bson.ObjectID `bson:"_id,omitempty"`
	UserID         bson.ObjectID `bson:"userId"`
	LessonID       bson.ObjectID `bson:"lessonId"`
	LessonRevision int           `bson:"lessonRevision"`
	LessonTitle    string        `bson:"lessonTitle"`
	Prompt         string        `bson:"prompt"`
	Text           string        `bson:"text"`
	Status         string        `bson:"status"`
	Grade          *gradeDoc     `bson:"grade"`
	CreatedAt      time.Time     `bson:"createdAt"`
	UpdatedAt      time.Time     `bson:"updatedAt"`
	SubmittedAt    time.Time     `bson:"submittedAt,omitempty"`
	Gradings       int           `bson:"gradings,omitempty"`
}

func fromGrade(g writing.Grade) gradeDoc {
	d := gradeDoc{
		Status: string(g.Status), Error: g.Error, Criteria: make([]criterionDoc, len(g.Criteria)),
		OverallVi: g.OverallVi, CorrectedText: g.CorrectedText, GradedAt: g.GradedAt.UTC(), Seen: g.Seen,
	}
	for i, c := range g.Criteria {
		d.Criteria[i] = criterionDoc(c)
	}
	return d
}

func (d writingDoc) toWriting() writing.Writing {
	w := writing.Writing{
		ID: d.ID.Hex(), UserID: d.UserID.Hex(), LessonID: d.LessonID.Hex(), LessonRevision: d.LessonRevision,
		LessonTitle: d.LessonTitle, Prompt: d.Prompt, Text: d.Text, Status: writing.Status(d.Status),
		CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt, SubmittedAt: d.SubmittedAt, Gradings: d.Gradings,
	}
	if g := d.Grade; g != nil {
		w.Grade = &writing.Grade{
			Status: writing.GradeStatus(g.Status), Error: g.Error, Criteria: make([]writing.Criterion, len(g.Criteria)),
			OverallVi: g.OverallVi, CorrectedText: g.CorrectedText, GradedAt: g.GradedAt, Seen: g.Seen,
		}
		for i, c := range g.Criteria {
			w.Grade.Criteria[i] = writing.Criterion(c)
		}
	}
	return w
}

// Writings implements writing.Repository on "writings" (F8). The unique (userId, lessonId)
// index keeps one writing per learner and lesson.
type Writings struct {
	coll *mongo.Collection
}

// NewWritings returns a writing repository on db.
func NewWritings(db *mongo.Database) *Writings {
	return &Writings{coll: db.Collection("writings")}
}

var _ writing.Repository = (*Writings)(nil)

func writingIDs(userID, lessonID string) (bson.ObjectID, bson.ObjectID, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return bson.ObjectID{}, bson.ObjectID{}, fmt.Errorf("writing user id: %w", err)
	}
	lid, err := bson.ObjectIDFromHex(lessonID)
	if err != nil {
		return bson.ObjectID{}, bson.ObjectID{}, writing.ErrNotFound
	}
	return uid, lid, nil
}

func (r *Writings) findOne(ctx context.Context, filter bson.D) (writing.Writing, bool, error) {
	var d writingDoc
	err := r.coll.FindOne(ctx, filter).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return writing.Writing{}, false, nil
	}
	if err != nil {
		return writing.Writing{}, false, fmt.Errorf("find writing: %w", err)
	}
	return d.toWriting(), true, nil
}

// GetByLesson returns the learner's writing for a lesson.
func (r *Writings) GetByLesson(ctx context.Context, userID, lessonID string) (writing.Writing, bool, error) {
	uid, lid, err := writingIDs(userID, lessonID)
	if errors.Is(err, writing.ErrNotFound) {
		return writing.Writing{}, false, nil
	}
	if err != nil {
		return writing.Writing{}, false, err
	}
	return r.findOne(ctx, bson.D{{Key: "userId", Value: uid}, {Key: "lessonId", Value: lid}})
}

// Get returns a writing by id.
func (r *Writings) Get(ctx context.Context, id string) (writing.Writing, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return writing.Writing{}, writing.ErrNotFound
	}
	w, ok, err := r.findOne(ctx, bson.D{{Key: "_id", Value: oid}})
	if err != nil {
		return writing.Writing{}, err
	}
	if !ok {
		return writing.Writing{}, writing.ErrNotFound
	}
	return w, nil
}

// SaveDraft upserts the draft unless it was submitted.
func (r *Writings) SaveDraft(ctx context.Context, userID, lessonID, text string, now time.Time) (writing.Writing, error) {
	uid, lid, err := writingIDs(userID, lessonID)
	if err != nil {
		return writing.Writing{}, err
	}
	filter := bson.D{
		{Key: "userId", Value: uid},
		{Key: "lessonId", Value: lid},
		{Key: "status", Value: bson.D{{Key: "$ne", Value: string(writing.StatusSubmitted)}}},
	}
	update := bson.D{
		{Key: "$set", Value: bson.D{{Key: "text", Value: text}, {Key: "updatedAt", Value: now}}},
		{Key: "$setOnInsert", Value: bson.D{
			{Key: "status", Value: string(writing.StatusDraft)}, {Key: "createdAt", Value: now}, {Key: "grade", Value: nil},
		}},
	}
	var d writingDoc
	err = r.coll.FindOneAndUpdate(ctx, filter, update,
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)).Decode(&d)
	if mongo.IsDuplicateKeyError(err) {
		// The upsert collided with the submitted writing of this lesson.
		return writing.Writing{}, writing.ErrSubmitted
	}
	if err != nil {
		return writing.Writing{}, fmt.Errorf("save draft: %w", err)
	}
	return d.toWriting(), nil
}

// Submit turns the draft (or a new writing) into a submitted one, once.
func (r *Writings) Submit(ctx context.Context, w writing.Writing) (writing.Writing, error) {
	uid, lid, err := writingIDs(w.UserID, w.LessonID)
	if err != nil {
		return writing.Writing{}, err
	}
	at := w.SubmittedAt.UTC()
	filter := bson.D{
		{Key: "userId", Value: uid},
		{Key: "lessonId", Value: lid},
		{Key: "status", Value: bson.D{{Key: "$ne", Value: string(writing.StatusSubmitted)}}},
	}
	update := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "lessonRevision", Value: w.LessonRevision},
			{Key: "lessonTitle", Value: w.LessonTitle},
			{Key: "prompt", Value: w.Prompt},
			{Key: "text", Value: w.Text},
			{Key: "status", Value: string(writing.StatusSubmitted)},
			{Key: "submittedAt", Value: at},
			{Key: "updatedAt", Value: at},
			{Key: "grade", Value: fromGrade(writing.Grade{Status: writing.GradePending, Seen: true})},
			{Key: "gradings", Value: w.Gradings},
		}},
		{Key: "$setOnInsert", Value: bson.D{{Key: "createdAt", Value: at}}},
	}
	var d writingDoc
	err = r.coll.FindOneAndUpdate(ctx, filter, update,
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)).Decode(&d)
	if mongo.IsDuplicateKeyError(err) {
		return writing.Writing{}, writing.ErrSubmitted
	}
	if err != nil {
		return writing.Writing{}, fmt.Errorf("submit writing: %w", err)
	}
	return d.toWriting(), nil
}

func (r *Writings) updateByID(ctx context.Context, id string, set bson.D) error {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return writing.ErrNotFound
	}
	res, err := r.coll.UpdateOne(ctx, bson.D{{Key: "_id", Value: oid}}, bson.D{{Key: "$set", Value: set}})
	if err != nil {
		return fmt.Errorf("update writing: %w", err)
	}
	if res.MatchedCount == 0 {
		return writing.ErrNotFound
	}
	return nil
}

// SetGrade replaces the grade.
func (r *Writings) SetGrade(ctx context.Context, id string, g writing.Grade) error {
	return r.updateByID(ctx, id, bson.D{{Key: "grade", Value: fromGrade(g)}, {Key: "updatedAt", Value: time.Now().UTC()}})
}

// MarkSeen marks the result as seen.
func (r *Writings) MarkSeen(ctx context.Context, id string) error {
	return r.updateByID(ctx, id, bson.D{{Key: "grade.seen", Value: true}})
}

func submittedFilter(uid bson.ObjectID) bson.D {
	return bson.D{{Key: "userId", Value: uid}, {Key: "status", Value: string(writing.StatusSubmitted)}}
}

// List returns the learner's submitted writings, newest first.
func (r *Writings) List(ctx context.Context, userID string) ([]writing.Writing, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, fmt.Errorf("writing user id: %w", err)
	}
	cur, err := r.coll.Find(ctx, submittedFilter(uid),
		options.Find().SetSort(bson.D{{Key: "submittedAt", Value: -1}}).SetProjection(bson.D{{Key: "text", Value: 0}}))
	if err != nil {
		return nil, fmt.Errorf("find writings: %w", err)
	}
	var docs []writingDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("read writings: %w", err)
	}
	out := make([]writing.Writing, len(docs))
	for i, d := range docs {
		out[i] = d.toWriting()
	}
	return out, nil
}

// Unseen counts new results (done or failed, not seen) and gradings in progress.
func (r *Writings) Unseen(ctx context.Context, userID string) (writing.UnseenCount, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return writing.UnseenCount{}, fmt.Errorf("writing user id: %w", err)
	}
	var out writing.UnseenCount
	pending, err := r.coll.CountDocuments(ctx, append(submittedFilter(uid), bson.E{Key: "grade.status", Value: string(writing.GradePending)}))
	if err != nil {
		return writing.UnseenCount{}, fmt.Errorf("count pending writings: %w", err)
	}
	unseenFilter := append(submittedFilter(uid),
		bson.E{Key: "grade.status", Value: bson.D{{Key: "$in", Value: bson.A{string(writing.GradeDone), string(writing.GradeFailed)}}}},
		bson.E{Key: "grade.seen", Value: false})
	unseen, err := r.coll.CountDocuments(ctx, unseenFilter)
	if err != nil {
		return writing.UnseenCount{}, fmt.Errorf("count unseen writings: %w", err)
	}
	out.Pending, out.Unseen = int(pending), int(unseen)
	if unseen > 0 {
		var d writingDoc
		err := r.coll.FindOne(ctx, unseenFilter, options.FindOne().
			SetSort(bson.D{{Key: "grade.gradedAt", Value: -1}}).SetProjection(bson.D{{Key: "grade.status", Value: 1}})).Decode(&d)
		if err != nil {
			return writing.UnseenCount{}, fmt.Errorf("find latest result: %w", err)
		}
		out.Latest = &writing.Latest{ID: d.ID.Hex(), Status: writing.GradeStatus(d.Grade.Status)}
	}
	return out, nil
}

// Stats counts the writings submitted at or after since (nil = all) and averages the mean score of
// the graded ones among them.
func (r *Writings) Stats(ctx context.Context, userID string, since *time.Time) (int, *float64, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return 0, nil, fmt.Errorf("writing user id: %w", err)
	}
	cur, err := r.coll.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: withSince(submittedFilter(uid), "submittedAt", since)}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "submitted", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "graded", Value: bson.D{{Key: "$sum", Value: bson.D{{Key: "$cond", Value: bson.A{
				bson.D{{Key: "$eq", Value: bson.A{"$grade.status", string(writing.GradeDone)}}}, 1, 0,
			}}}}}},
			// $avg skips the nulls of writings that are not graded yet.
			{Key: "average", Value: bson.D{{Key: "$avg", Value: bson.D{{Key: "$cond", Value: bson.A{
				bson.D{{Key: "$eq", Value: bson.A{"$grade.status", string(writing.GradeDone)}}},
				bson.D{{Key: "$avg", Value: "$grade.criteria.score"}},
				nil,
			}}}}}},
		}}},
	})
	if err != nil {
		return 0, nil, fmt.Errorf("aggregate writings: %w", err)
	}
	var rows []struct {
		Submitted int      `bson:"submitted"`
		Graded    int      `bson:"graded"`
		Average   *float64 `bson:"average"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return 0, nil, fmt.Errorf("read writing stats: %w", err)
	}
	if len(rows) == 0 {
		return 0, nil, nil
	}
	if rows[0].Graded == 0 {
		return rows[0].Submitted, nil, nil
	}
	return rows[0].Submitted, rows[0].Average, nil
}

// Regrade stores the text to grade again and resets the grade to pending.
func (r *Writings) Regrade(ctx context.Context, id, text string, submittedAt time.Time, gradings int) error {
	return r.updateByID(ctx, id, bson.D{
		{Key: "text", Value: text},
		{Key: "submittedAt", Value: submittedAt.UTC()},
		{Key: "gradings", Value: gradings},
		{Key: "grade", Value: fromGrade(writing.Grade{Status: writing.GradePending, Seen: true})},
		{Key: "updatedAt", Value: time.Now().UTC()},
	})
}
