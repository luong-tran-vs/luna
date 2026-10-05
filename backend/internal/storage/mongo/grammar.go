package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/luongtran/luna/backend/internal/grammar"
)

// --- grammar_lessons (F20) ---

type grammarExerciseDoc struct {
	ID            string   `bson:"id"`
	Kind          string   `bson:"kind"`
	PromptVi      string   `bson:"promptVi,omitempty"`
	Text          string   `bson:"text,omitempty"`
	Options       []string `bson:"options,omitempty"`
	AnswerIndex   int      `bson:"answerIndex,omitempty"`
	Answers       []string `bson:"answers,omitempty"`
	Words         []string `bson:"words,omitempty"`
	Sentence      string   `bson:"sentence,omitempty"`
	ExplanationVi string   `bson:"explanationVi,omitempty"`
}

type grammarStructureDoc struct {
	Label   string `bson:"label"`
	Pattern string `bson:"pattern"`
	Example string `bson:"example"`
}

type grammarExampleDoc struct {
	En string `bson:"en"`
	Vi string `bson:"vi"`
}

type grammarMistakeDoc struct {
	Wrong  string `bson:"wrong"`
	Right  string `bson:"right"`
	NoteVi string `bson:"noteVi"`
}

type grammarContentDoc struct {
	Objective   string                `bson:"objective"`
	Explanation []string              `bson:"explanation"`
	Usage       []string              `bson:"usage"`
	Structures  []grammarStructureDoc `bson:"structures"`
	Examples    []grammarExampleDoc   `bson:"examples"`
	Mistakes    []grammarMistakeDoc   `bson:"mistakes"`
	Practice    []grammarExerciseDoc  `bson:"practice"`
	Mastery     []grammarExerciseDoc  `bson:"mastery"`
}

type grammarCheckDoc struct {
	ExerciseID string `bson:"exerciseId"`
	Kind       string `bson:"kind"`
	NoteVi     string `bson:"noteVi"`
	Confirmed  bool   `bson:"confirmed"`
}

type grammarLessonDoc struct {
	PointID     string            `bson:"pointId"`
	Status      string            `bson:"status"`
	Edited      bool              `bson:"edited"`
	Content     grammarContentDoc `bson:"content"`
	Checks      []grammarCheckDoc `bson:"checks"`
	CheckedAt   time.Time         `bson:"checkedAt,omitempty"`
	VerifiedAt  time.Time         `bson:"verifiedAt,omitempty"`
	CreatedAt   time.Time         `bson:"createdAt"`
	UpdatedAt   time.Time         `bson:"updatedAt"`
	PublishedAt time.Time         `bson:"publishedAt,omitempty"`
}

// grammarCheckFlagDoc is the part of a check that the list needs.
type grammarCheckFlagDoc struct {
	Confirmed bool `bson:"confirmed"`
}

// unconfirmed counts the checks an admin has not confirmed yet.
func unconfirmed(cs []grammarCheckFlagDoc) int {
	n := 0
	for _, c := range cs {
		if !c.Confirmed {
			n++
		}
	}
	return n
}

// grammarSummaryDoc is a lesson without its content, for lists.
type grammarSummaryDoc struct {
	PointID     string                `bson:"pointId"`
	Status      string                `bson:"status"`
	Edited      bool                  `bson:"edited"`
	Checks      []grammarCheckFlagDoc `bson:"checks"`
	CheckedAt   time.Time             `bson:"checkedAt,omitempty"`
	VerifiedAt  time.Time             `bson:"verifiedAt,omitempty"`
	UpdatedAt   time.Time             `bson:"updatedAt"`
	PublishedAt time.Time             `bson:"publishedAt,omitempty"`
}

// strs returns s, or an empty non-nil slice, so a list never reads back as nil.
func strs(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func fromGrammarExercises(in []grammar.Exercise) []grammarExerciseDoc {
	out := make([]grammarExerciseDoc, len(in))
	for i, e := range in {
		out[i] = grammarExerciseDoc{
			ID: e.ID, Kind: string(e.Kind), PromptVi: e.PromptVi, Text: e.Text, Options: e.Options,
			AnswerIndex: e.AnswerIndex, Answers: e.Answers, Words: e.Words, Sentence: e.Sentence,
			ExplanationVi: e.ExplanationVi,
		}
	}
	return out
}

func (d grammarExerciseDoc) toExercise() grammar.Exercise {
	return grammar.Exercise{
		ID: d.ID, Kind: grammar.Kind(d.Kind), PromptVi: d.PromptVi, Text: d.Text, Options: d.Options,
		AnswerIndex: d.AnswerIndex, Answers: d.Answers, Words: d.Words, Sentence: d.Sentence,
		ExplanationVi: d.ExplanationVi,
	}
}

func fromGrammarLesson(l grammar.Lesson) grammarLessonDoc {
	c := l.Content
	cd := grammarContentDoc{
		Objective: c.Objective, Explanation: strs(c.Explanation), Usage: strs(c.Usage),
		Structures: make([]grammarStructureDoc, len(c.Structures)),
		Examples:   make([]grammarExampleDoc, len(c.Examples)),
		Mistakes:   make([]grammarMistakeDoc, len(c.Mistakes)),
		Practice:   fromGrammarExercises(c.Practice),
		Mastery:    fromGrammarExercises(c.Mastery),
	}
	for i, s := range c.Structures {
		cd.Structures[i] = grammarStructureDoc(s)
	}
	for i, e := range c.Examples {
		cd.Examples[i] = grammarExampleDoc(e)
	}
	for i, m := range c.Mistakes {
		cd.Mistakes[i] = grammarMistakeDoc(m)
	}
	d := grammarLessonDoc{
		PointID: l.PointID, Status: string(l.Status), Edited: l.Edited, Content: cd,
		Checks:    make([]grammarCheckDoc, len(l.Checks)),
		CreatedAt: l.CreatedAt.UTC(), UpdatedAt: l.UpdatedAt.UTC(),
	}
	for i, c := range l.Checks {
		d.Checks[i] = grammarCheckDoc{ExerciseID: c.ExerciseID, Kind: string(c.Kind), NoteVi: c.NoteVi, Confirmed: c.Confirmed}
	}
	if !l.PublishedAt.IsZero() {
		d.PublishedAt = l.PublishedAt.UTC()
	}
	if !l.CheckedAt.IsZero() {
		d.CheckedAt = l.CheckedAt.UTC()
	}
	if !l.VerifiedAt.IsZero() {
		d.VerifiedAt = l.VerifiedAt.UTC()
	}
	return d
}

func (d grammarLessonDoc) toLesson() grammar.Lesson {
	c := d.Content
	content := grammar.Content{
		Objective: c.Objective, Explanation: strs(c.Explanation), Usage: strs(c.Usage),
		Structures: make([]grammar.Structure, len(c.Structures)),
		Examples:   make([]grammar.Example, len(c.Examples)),
		Mistakes:   make([]grammar.Mistake, len(c.Mistakes)),
		Practice:   make([]grammar.Exercise, len(c.Practice)),
		Mastery:    make([]grammar.Exercise, len(c.Mastery)),
	}
	for i, s := range c.Structures {
		content.Structures[i] = grammar.Structure(s)
	}
	for i, e := range c.Examples {
		content.Examples[i] = grammar.Example(e)
	}
	for i, m := range c.Mistakes {
		content.Mistakes[i] = grammar.Mistake(m)
	}
	for i, e := range c.Practice {
		content.Practice[i] = e.toExercise()
	}
	for i, e := range c.Mastery {
		content.Mastery[i] = e.toExercise()
	}
	checks := make([]grammar.Check, len(d.Checks))
	for i, c := range d.Checks {
		checks[i] = grammar.Check{ExerciseID: c.ExerciseID, Kind: grammar.CheckKind(c.Kind), NoteVi: c.NoteVi, Confirmed: c.Confirmed}
	}
	return grammar.Lesson{
		PointID: d.PointID, Status: grammar.Status(d.Status), Content: content, Edited: d.Edited,
		Checks: checks, CheckedAt: zeroOrUTC(d.CheckedAt), VerifiedAt: zeroOrUTC(d.VerifiedAt),
		CreatedAt: d.CreatedAt.UTC(), UpdatedAt: d.UpdatedAt.UTC(), PublishedAt: zeroOrUTC(d.PublishedAt),
	}
}

func zeroOrUTC(t time.Time) time.Time {
	if t.IsZero() {
		return time.Time{}
	}
	return t.UTC()
}

// GrammarLessons implements grammar.LessonRepository on "grammar_lessons", one document per point.
type GrammarLessons struct {
	coll *mongo.Collection
}

// NewGrammarLessons returns the grammar lesson repository.
func NewGrammarLessons(db *mongo.Database) *GrammarLessons {
	return &GrammarLessons{coll: db.Collection("grammar_lessons")}
}

var _ grammar.LessonRepository = (*GrammarLessons)(nil)

// Get returns the lesson of a point.
func (r *GrammarLessons) Get(ctx context.Context, pointID string) (grammar.Lesson, error) {
	var d grammarLessonDoc
	err := r.coll.FindOne(ctx, bson.D{{Key: "pointId", Value: pointID}}).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return grammar.Lesson{}, grammar.ErrNotFound
	}
	if err != nil {
		return grammar.Lesson{}, fmt.Errorf("find grammar lesson: %w", err)
	}
	return d.toLesson(), nil
}

// Save inserts or replaces the whole lesson.
func (r *GrammarLessons) Save(ctx context.Context, l grammar.Lesson) error {
	_, err := r.coll.ReplaceOne(ctx, bson.D{{Key: "pointId", Value: l.PointID}}, fromGrammarLesson(l),
		options.Replace().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("save grammar lesson: %w", err)
	}
	return nil
}

// List returns a summary of every lesson, without content.
func (r *GrammarLessons) List(ctx context.Context) ([]grammar.LessonSummary, error) {
	cur, err := r.coll.Find(ctx, bson.D{}, options.Find().SetProjection(bson.D{{Key: "content", Value: 0}}))
	if err != nil {
		return nil, fmt.Errorf("find grammar lessons: %w", err)
	}
	var docs []grammarSummaryDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("read grammar lessons: %w", err)
	}
	out := make([]grammar.LessonSummary, len(docs))
	for i, d := range docs {
		out[i] = grammar.LessonSummary{
			PointID: d.PointID, Status: grammar.Status(d.Status), Edited: d.Edited,
			Flags: unconfirmed(d.Checks), Checked: !d.CheckedAt.IsZero(), Verified: !d.VerifiedAt.IsZero(),
			UpdatedAt: d.UpdatedAt.UTC(), PublishedAt: zeroOrUTC(d.PublishedAt),
		}
	}
	return out, nil
}

// --- grammar_progress (F20) ---

type grammarProgressDoc struct {
	UserID           bson.ObjectID `bson:"userId"`
	PointID          string        `bson:"pointId"`
	Status           string        `bson:"status"`
	PracticeAttempts int           `bson:"practiceAttempts"`
	LastPractice     int           `bson:"lastPractice"`
	MasteryAttempts  int           `bson:"masteryAttempts"`
	BestMastery      int           `bson:"bestMastery"`
	MasteredAt       time.Time     `bson:"masteredAt,omitempty"`
	Weak             []string      `bson:"weak"`
	UpdatedAt        time.Time     `bson:"updatedAt"`
}

func (d grammarProgressDoc) toProgress() grammar.Progress {
	return grammar.Progress{
		UserID: d.UserID.Hex(), PointID: d.PointID, Status: grammar.ProgressStatus(d.Status),
		PracticeAttempts: d.PracticeAttempts, LastPractice: d.LastPractice,
		MasteryAttempts: d.MasteryAttempts, BestMastery: d.BestMastery,
		MasteredAt: zeroOrUTC(d.MasteredAt), Weak: strs(d.Weak), UpdatedAt: d.UpdatedAt.UTC(),
	}
}

// GrammarProgress implements grammar.ProgressRepository on "grammar_progress", one document per
// learner and point.
type GrammarProgress struct {
	coll *mongo.Collection
}

// NewGrammarProgress returns the grammar progress repository.
func NewGrammarProgress(db *mongo.Database) *GrammarProgress {
	return &GrammarProgress{coll: db.Collection("grammar_progress")}
}

var _ grammar.ProgressRepository = (*GrammarProgress)(nil)

// Get returns the learner's record on a point.
func (r *GrammarProgress) Get(ctx context.Context, userID, pointID string) (grammar.Progress, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return grammar.Progress{}, grammar.ErrNotFound
	}
	var d grammarProgressDoc
	err = r.coll.FindOne(ctx, bson.D{{Key: "userId", Value: uid}, {Key: "pointId", Value: pointID}}).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return grammar.Progress{}, grammar.ErrNotFound
	}
	if err != nil {
		return grammar.Progress{}, fmt.Errorf("find grammar progress: %w", err)
	}
	return d.toProgress(), nil
}

// List returns every record of the learner, ordered by point.
func (r *GrammarProgress) List(ctx context.Context, userID string) ([]grammar.Progress, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return []grammar.Progress{}, nil
	}
	cur, err := r.coll.Find(ctx, bson.D{{Key: "userId", Value: uid}}, options.Find().SetSort(bson.D{{Key: "pointId", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("find grammar progress: %w", err)
	}
	var docs []grammarProgressDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("read grammar progress: %w", err)
	}
	out := make([]grammar.Progress, len(docs))
	for i, d := range docs {
		out[i] = d.toProgress()
	}
	return out, nil
}

// Save inserts or replaces the record of the learner and point.
func (r *GrammarProgress) Save(ctx context.Context, p grammar.Progress) error {
	uid, err := bson.ObjectIDFromHex(p.UserID)
	if err != nil {
		return fmt.Errorf("grammar progress user id: %w", err)
	}
	d := grammarProgressDoc{
		UserID: uid, PointID: p.PointID, Status: string(p.Status),
		PracticeAttempts: p.PracticeAttempts, LastPractice: p.LastPractice,
		MasteryAttempts: p.MasteryAttempts, BestMastery: p.BestMastery,
		Weak: strs(p.Weak), UpdatedAt: p.UpdatedAt.UTC(),
	}
	if !p.MasteredAt.IsZero() {
		d.MasteredAt = p.MasteredAt.UTC()
	}
	// $set (not a replace) keeps the document's _id, which the export sorts by.
	_, err = r.coll.UpdateOne(ctx, bson.D{{Key: "userId", Value: uid}, {Key: "pointId", Value: p.PointID}},
		bson.D{{Key: "$set", Value: d}}, options.UpdateOne().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("save grammar progress: %w", err)
	}
	return nil
}
