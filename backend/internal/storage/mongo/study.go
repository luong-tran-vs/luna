package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/luongtran/luna/backend/internal/progress"
)

// --- goals ---

type goalDoc struct {
	ID            bson.ObjectID `bson:"_id,omitempty"`
	UserID        bson.ObjectID `bson:"userId"`
	TopicID       bson.ObjectID `bson:"topicId"`
	Level         string        `bson:"level"`
	Status        string        `bson:"status"`
	EffectiveFrom string        `bson:"effectiveFrom"`
	StartedAt     time.Time     `bson:"startedAt"`
}

func (d goalDoc) toGoal() progress.Goal {
	return progress.Goal{
		ID: d.ID.Hex(), UserID: d.UserID.Hex(), TopicID: d.TopicID.Hex(), Level: d.Level,
		Status: progress.GoalStatus(d.Status), EffectiveFrom: d.EffectiveFrom, StartedAt: d.StartedAt,
	}
}

// Goals implements progress.GoalRepository on "goals".
type Goals struct {
	coll *mongo.Collection
}

// NewGoals returns a goal repository on db.
func NewGoals(db *mongo.Database) *Goals { return &Goals{coll: db.Collection("goals")} }

var _ progress.GoalRepository = (*Goals)(nil)

// List returns the user's goals.
func (r *Goals) List(ctx context.Context, userID string) ([]progress.Goal, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, fmt.Errorf("goal user id: %w", err)
	}
	cur, err := r.coll.Find(ctx, bson.D{{Key: "userId", Value: uid}}, options.Find().SetSort(bson.D{{Key: "startedAt", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("find goals: %w", err)
	}
	var docs []goalDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("read goals: %w", err)
	}
	out := make([]progress.Goal, len(docs))
	for i, d := range docs {
		out[i] = d.toGoal()
	}
	return out, nil
}

// Activate pauses the other active goal and makes topicID's goal active (upsert).
func (r *Goals) Activate(ctx context.Context, userID, topicID, level, effectiveFrom string, now time.Time) (progress.Goal, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return progress.Goal{}, fmt.Errorf("goal user id: %w", err)
	}
	tid, err := bson.ObjectIDFromHex(topicID)
	if err != nil {
		return progress.Goal{}, progress.ErrTopicNotFound
	}
	_, err = r.coll.UpdateMany(ctx, bson.D{
		{Key: "userId", Value: uid},
		{Key: "status", Value: string(progress.GoalActive)},
		{Key: "topicId", Value: bson.D{{Key: "$ne", Value: tid}}},
	}, bson.D{{Key: "$set", Value: bson.D{{Key: "status", Value: string(progress.GoalPaused)}}}})
	if err != nil {
		return progress.Goal{}, fmt.Errorf("pause goals: %w", err)
	}
	var d goalDoc
	err = r.coll.FindOneAndUpdate(ctx, bson.D{{Key: "userId", Value: uid}, {Key: "topicId", Value: tid}}, bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "status", Value: string(progress.GoalActive)}, {Key: "level", Value: level}, {Key: "effectiveFrom", Value: effectiveFrom},
		}},
		{Key: "$setOnInsert", Value: bson.D{{Key: "startedAt", Value: now.UTC()}}},
	}, options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)).Decode(&d)
	if err != nil {
		return progress.Goal{}, fmt.Errorf("activate goal: %w", err)
	}
	return d.toGoal(), nil
}

// --- lesson_progress ---

type progressDoc struct {
	UserID        bson.ObjectID   `bson:"userId"`
	LessonID      bson.ObjectID   `bson:"lessonId"`
	TopicID       bson.ObjectID   `bson:"topicId,omitempty"`
	DayKey        string          `bson:"dayKey"`
	Steps         map[string]bool `bson:"steps"`
	CurrentStep   string          `bson:"currentStep"`
	SentenceIndex int             `bson:"sentenceIndex"`
	StartedAt     time.Time       `bson:"startedAt"`
	CompletedAt   time.Time       `bson:"completedAt,omitempty"`
}

func (d progressDoc) toProgress() progress.LessonProgress {
	p := progress.LessonProgress{
		UserID: d.UserID.Hex(), LessonID: d.LessonID.Hex(), TopicID: hexOrEmpty(d.TopicID), DayKey: d.DayKey,
		Done: map[progress.Step]bool{}, CurrentStep: progress.Step(d.CurrentStep), SentenceIndex: d.SentenceIndex,
		StartedAt: d.StartedAt, CompletedAt: d.CompletedAt,
	}
	for s, done := range d.Steps {
		if done {
			p.Done[progress.Step(s)] = true
		}
	}
	return p
}

// LessonProgress implements progress.ProgressRepository on "lesson_progress".
type LessonProgress struct {
	coll *mongo.Collection
}

// NewLessonProgress returns a lesson progress repository on db.
func NewLessonProgress(db *mongo.Database) *LessonProgress {
	return &LessonProgress{coll: db.Collection("lesson_progress")}
}

var _ progress.ProgressRepository = (*LessonProgress)(nil)

func userLesson(userID, lessonID string) (bson.D, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, fmt.Errorf("progress user id: %w", err)
	}
	return bson.D{{Key: "userId", Value: uid}, {Key: "lessonId", Value: oidOrZero(lessonID)}}, nil
}

// Get returns the user's progress on a lesson.
func (r *LessonProgress) Get(ctx context.Context, userID, lessonID string) (progress.LessonProgress, bool, error) {
	f, err := userLesson(userID, lessonID)
	if err != nil {
		return progress.LessonProgress{}, false, err
	}
	var d progressDoc
	err = r.coll.FindOne(ctx, f).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return progress.LessonProgress{}, false, nil
	}
	if err != nil {
		return progress.LessonProgress{}, false, fmt.Errorf("find progress: %w", err)
	}
	return d.toProgress(), true, nil
}

// Completed returns the completed lessons, newest first.
func (r *LessonProgress) Completed(ctx context.Context, userID string) ([]progress.LessonProgress, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, fmt.Errorf("progress user id: %w", err)
	}
	cur, err := r.coll.Find(ctx, bson.D{{Key: "userId", Value: uid}, {Key: "completedAt", Value: bson.D{{Key: "$exists", Value: true}}}},
		options.Find().SetSort(bson.D{{Key: "completedAt", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("find completed lessons: %w", err)
	}
	var docs []progressDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("read completed lessons: %w", err)
	}
	out := make([]progress.LessonProgress, len(docs))
	for i, d := range docs {
		out[i] = d.toProgress()
	}
	return out, nil
}

// Upsert writes the whole progress of a lesson.
func (r *LessonProgress) Upsert(ctx context.Context, p progress.LessonProgress) error {
	f, err := userLesson(p.UserID, p.LessonID)
	if err != nil {
		return err
	}
	steps := map[string]bool{}
	for _, s := range progress.Steps {
		steps[string(s)] = p.Done[s]
	}
	set := bson.D{
		{Key: "topicId", Value: oidOrZero(p.TopicID)},
		{Key: "dayKey", Value: p.DayKey},
		{Key: "steps", Value: steps},
		{Key: "currentStep", Value: string(p.CurrentStep)},
		{Key: "sentenceIndex", Value: p.SentenceIndex},
		{Key: "startedAt", Value: p.StartedAt.UTC()},
	}
	if !p.CompletedAt.IsZero() {
		set = append(set, bson.E{Key: "completedAt", Value: p.CompletedAt.UTC()})
	}
	update := bson.D{{Key: "$set", Value: set}}
	if _, err := r.coll.UpdateOne(ctx, f, update, options.UpdateOne().SetUpsert(true)); err != nil {
		return fmt.Errorf("upsert progress: %w", err)
	}
	return nil
}

// SetPosition saves the current step and sentence.
func (r *LessonProgress) SetPosition(ctx context.Context, userID, lessonID string, step progress.Step, sentence int) error {
	f, err := userLesson(userID, lessonID)
	if err != nil {
		return err
	}
	_, err = r.coll.UpdateOne(ctx, f, bson.D{{Key: "$set", Value: bson.D{
		{Key: "currentStep", Value: string(step)}, {Key: "sentenceIndex", Value: sentence},
	}}})
	if err != nil {
		return fmt.Errorf("set position: %w", err)
	}
	return nil
}

// --- study_days ---

type dayDoc struct {
	DayKey        string        `bson:"dayKey"`
	LessonID      bson.ObjectID `bson:"lessonId"`
	ReviewedCount int           `bson:"reviewedCount"`
	Completed     bool          `bson:"completed"`
}

// StudyDays implements progress.DayRepository on "study_days".
type StudyDays struct {
	coll *mongo.Collection
}

// NewStudyDays returns a study day repository on db.
func NewStudyDays(db *mongo.Database) *StudyDays {
	return &StudyDays{coll: db.Collection("study_days")}
}

var _ progress.DayRepository = (*StudyDays)(nil)

func userDay(userID, dayKey string) (bson.D, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, fmt.Errorf("day user id: %w", err)
	}
	return bson.D{{Key: "userId", Value: uid}, {Key: "dayKey", Value: dayKey}}, nil
}

// Get returns nil when the day has no lesson.
func (r *StudyDays) Get(ctx context.Context, userID, dayKey string) (*progress.StudyDay, error) {
	f, err := userDay(userID, dayKey)
	if err != nil {
		return nil, err
	}
	var d dayDoc
	err = r.coll.FindOne(ctx, f).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find study day: %w", err)
	}
	return &progress.StudyDay{DayKey: d.DayKey, LessonID: hexOrEmpty(d.LessonID), ReviewedCount: d.ReviewedCount, Completed: d.Completed}, nil
}

// Start records the day's lesson unless the day already has one.
func (r *StudyDays) Start(ctx context.Context, userID, dayKey, lessonID string) error {
	f, err := userDay(userID, dayKey)
	if err != nil {
		return err
	}
	_, err = r.coll.UpdateOne(ctx, f, bson.D{{Key: "$setOnInsert", Value: bson.D{
		{Key: "lessonId", Value: oidOrZero(lessonID)}, {Key: "reviewedCount", Value: 0}, {Key: "completed", Value: false},
	}}}, options.UpdateOne().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("start study day: %w", err)
	}
	return nil
}

// Update sets the reviewed count and completion of a day.
func (r *StudyDays) Update(ctx context.Context, userID, dayKey string, reviewed int, completed bool) error {
	f, err := userDay(userID, dayKey)
	if err != nil {
		return err
	}
	_, err = r.coll.UpdateOne(ctx, f, bson.D{{Key: "$set", Value: bson.D{
		{Key: "reviewedCount", Value: reviewed}, {Key: "completed", Value: completed},
	}}})
	if err != nil {
		return fmt.Errorf("update study day: %w", err)
	}
	return nil
}

// CompletedKeys lists the user's days with a completed lesson.
func (r *StudyDays) CompletedKeys(ctx context.Context, userID string) ([]string, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, fmt.Errorf("day user id: %w", err)
	}
	cur, err := r.coll.Find(ctx, bson.D{{Key: "userId", Value: uid}, {Key: "completed", Value: true}},
		options.Find().SetProjection(bson.D{{Key: "dayKey", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("find study days: %w", err)
	}
	var docs []dayDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("read study days: %w", err)
	}
	out := make([]string, len(docs))
	for i, d := range docs {
		out[i] = d.DayKey
	}
	return out, nil
}

// StepCounts counts the user's lessons among lessonIDs (nil = every lesson) with the read step
// done, the listen step done, and completed.
func (r *LessonProgress) StepCounts(ctx context.Context, userID string, lessonIDs []string) (progress.StepCounts, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return progress.StepCounts{}, fmt.Errorf("progress user id: %w", err)
	}
	match := bson.D{{Key: "userId", Value: uid}}
	if lessonIDs != nil {
		if len(lessonIDs) == 0 {
			return progress.StepCounts{}, nil
		}
		oids := make(bson.A, len(lessonIDs))
		for i, id := range lessonIDs {
			oids[i] = oidOrZero(id)
		}
		match = append(match, bson.E{Key: "lessonId", Value: bson.D{{Key: "$in", Value: oids}}})
	}
	count := func(cond bson.D) bson.D {
		return bson.D{{Key: "$sum", Value: bson.D{{Key: "$cond", Value: bson.A{cond, 1, 0}}}}}
	}
	cur, err := r.coll.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: match}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "read", Value: count(bson.D{{Key: "$eq", Value: bson.A{"$steps.read", true}}})},
			{Key: "listen", Value: count(bson.D{{Key: "$eq", Value: bson.A{"$steps.listen", true}}})},
			{Key: "write", Value: count(bson.D{{Key: "$eq", Value: bson.A{"$steps.write", true}}})},
			// A missing completedAt sorts below null; a date above it.
			{Key: "completed", Value: count(bson.D{{Key: "$gt", Value: bson.A{"$completedAt", nil}}})},
		}}},
	})
	if err != nil {
		return progress.StepCounts{}, fmt.Errorf("aggregate step counts: %w", err)
	}
	var rows []struct {
		Read      int `bson:"read"`
		Listen    int `bson:"listen"`
		Write     int `bson:"write"`
		Completed int `bson:"completed"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return progress.StepCounts{}, fmt.Errorf("read step counts: %w", err)
	}
	if len(rows) == 0 {
		return progress.StepCounts{}, nil
	}
	return progress.StepCounts{Read: rows[0].Read, Listen: rows[0].Listen, Write: rows[0].Write, Completed: rows[0].Completed}, nil
}

// LatestKey is the user's latest study day, "" when none.
func (r *StudyDays) LatestKey(ctx context.Context, userID string) (string, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return "", fmt.Errorf("day user id: %w", err)
	}
	var d dayDoc
	err = r.coll.FindOne(ctx, bson.D{{Key: "userId", Value: uid}},
		options.FindOne().SetSort(bson.D{{Key: "dayKey", Value: -1}}).SetProjection(bson.D{{Key: "dayKey", Value: 1}})).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("find latest study day: %w", err)
	}
	return d.DayKey, nil
}
