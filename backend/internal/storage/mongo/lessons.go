package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/luongtran/luna/backend/internal/job"
	"github.com/luongtran/luna/backend/internal/lesson"
)

type sentenceDoc struct {
	Index     int    `bson:"index"`
	Text      string `bson:"text"`
	AudioPath string `bson:"audioPath"`
}

type annotationDoc struct {
	Text          string `bson:"text"`
	Lemma         string `bson:"lemma"`
	MeaningVi     string `bson:"meaningVi"`
	SentenceIndex int    `bson:"sentenceIndex"`
	EditedByAdmin bool   `bson:"editedByAdmin"`
}

type questionDoc struct {
	Prompt        string   `bson:"prompt"`
	Options       []string `bson:"options"`
	AnswerIndex   int      `bson:"answerIndex"`
	ExplanationVi string   `bson:"explanationVi"`
}

type grammarNoteDoc struct {
	Title    string   `bson:"title"`
	BodyVi   string   `bson:"bodyVi"`
	Examples []string `bson:"examples"`
}

// extrasSet is the $set of a lesson's questions, grammar note and writing prompt (F15).
func extrasSet(x lesson.Extras) bson.D {
	qs := make([]questionDoc, len(x.Questions))
	for i, q := range x.Questions {
		qs[i] = questionDoc(q)
	}
	var note *grammarNoteDoc
	if x.GrammarNote != nil {
		n := grammarNoteDoc(*x.GrammarNote)
		note = &n
	}
	return bson.D{
		{Key: "questions", Value: qs},
		{Key: "grammarNote", Value: note},
		{Key: "writingPrompt", Value: x.WritingPrompt},
	}
}

func (d lessonDoc) extras() lesson.Extras {
	x := lesson.Extras{Questions: make([]lesson.Question, len(d.Questions)), WritingPrompt: d.WritingPrompt}
	for i, q := range d.Questions {
		x.Questions[i] = lesson.Question(q)
	}
	if d.GrammarNote != nil {
		n := lesson.GrammarNote(*d.GrammarNote)
		x.GrammarNote = &n
	}
	return x
}

type lessonDoc struct {
	ID               bson.ObjectID   `bson:"_id,omitempty"`
	Title            string          `bson:"title"`
	Content          string          `bson:"content"`
	Level            string          `bson:"level"`
	TopicID          bson.ObjectID   `bson:"topicId"`
	Source           string          `bson:"source"`
	License          string          `bson:"license"`
	Revision         int             `bson:"revision"`
	Sentences        []sentenceDoc   `bson:"sentences"`
	AudioStatus      string          `bson:"audioStatus"`
	AudioError       string          `bson:"audioError"`
	AnnotationStatus string          `bson:"annotationStatus"`
	AnnotationError  string          `bson:"annotationError"`
	Annotations      []annotationDoc `bson:"annotations"`
	Questions        []questionDoc   `bson:"questions"`
	GrammarNote      *grammarNoteDoc `bson:"grammarNote"`
	WritingPrompt    string          `bson:"writingPrompt"`
	ExtrasEdited     bool            `bson:"extrasEditedByAdmin"`
	QuizVersion      int             `bson:"quizVersion"`
	CreatedAt        time.Time       `bson:"createdAt"`
	UpdatedAt        time.Time       `bson:"updatedAt"`
}

func fromLesson(l lesson.Lesson) lessonDoc {
	d := lessonDoc{
		Title: l.Title, Content: l.Content, Level: string(l.Level), TopicID: oidOrZero(l.TopicID),
		Source: l.Source, License: l.License, Revision: l.Revision,
		Sentences:   make([]sentenceDoc, len(l.Sentences)),
		AudioStatus: string(l.AudioStatus), AudioError: l.AudioError,
		AnnotationStatus: string(l.AnnotationStatus), AnnotationError: l.AnnotationError,
		Annotations: fromAnnotations(l.Annotations),
		CreatedAt:   l.CreatedAt.UTC(), UpdatedAt: l.UpdatedAt.UTC(),
	}
	for i, s := range l.Sentences {
		d.Sentences[i] = sentenceDoc(s)
	}
	return d
}

func fromAnnotations(anns []lesson.Annotation) []annotationDoc {
	out := make([]annotationDoc, len(anns))
	for i, a := range anns {
		out[i] = annotationDoc(a)
	}
	return out
}

func (d lessonDoc) toLesson() lesson.Lesson {
	l := lesson.Lesson{
		ID: d.ID.Hex(), Title: d.Title, Content: d.Content, Level: lesson.Level(d.Level), TopicID: hexOrEmpty(d.TopicID),
		Source: d.Source, License: d.License, Revision: d.Revision,
		Sentences:   make([]lesson.Sentence, len(d.Sentences)),
		AudioStatus: lesson.Status(d.AudioStatus), AudioError: d.AudioError,
		AnnotationStatus: lesson.Status(d.AnnotationStatus), AnnotationError: d.AnnotationError,
		Annotations: make([]lesson.Annotation, len(d.Annotations)),
		CreatedAt:   d.CreatedAt, UpdatedAt: d.UpdatedAt,
	}
	l.Extras, l.ExtrasEditedByAdmin, l.QuizVersion = d.extras(), d.ExtrasEdited, d.QuizVersion
	for i, s := range d.Sentences {
		l.Sentences[i] = lesson.Sentence(s)
	}
	for i, a := range d.Annotations {
		l.Annotations[i] = lesson.Annotation(a)
	}
	return l
}

type summaryDoc struct {
	ID               bson.ObjectID `bson:"_id"`
	Title            string        `bson:"title"`
	Level            string        `bson:"level"`
	TopicID          bson.ObjectID `bson:"topicId"`
	AudioStatus      string        `bson:"audioStatus"`
	AnnotationStatus string        `bson:"annotationStatus"`
	CreatedAt        time.Time     `bson:"createdAt"`
}

var summaryProjection = bson.D{
	{Key: "title", Value: 1},
	{Key: "level", Value: 1},
	{Key: "topicId", Value: 1},
	{Key: "audioStatus", Value: 1},
	{Key: "annotationStatus", Value: 1},
	{Key: "createdAt", Value: 1},
}

func (d summaryDoc) toSummary() lesson.Summary {
	return lesson.Summary{
		ID: d.ID.Hex(), Title: d.Title, Level: lesson.Level(d.Level), TopicID: hexOrEmpty(d.TopicID),
		AudioStatus: lesson.Status(d.AudioStatus), AnnotationStatus: lesson.Status(d.AnnotationStatus),
		CreatedAt: d.CreatedAt,
	}
}

// Lessons implements lesson.Repository on the "lessons" collection.
type Lessons struct {
	coll *mongo.Collection
}

// NewLessons returns a lesson repository on db.
func NewLessons(db *mongo.Database) *Lessons {
	return &Lessons{coll: db.Collection("lessons")}
}

var _ lesson.Repository = (*Lessons)(nil)

// Create inserts l.
func (r *Lessons) Create(ctx context.Context, l lesson.Lesson) (lesson.Lesson, error) {
	d := fromLesson(l)
	d.ID = bson.NewObjectID()
	if _, err := r.coll.InsertOne(ctx, d); err != nil {
		return lesson.Lesson{}, fmt.Errorf("insert lesson: %w", err)
	}
	return d.toLesson(), nil
}

// Get returns lesson.ErrNotFound for unknown or malformed ids.
func (r *Lessons) Get(ctx context.Context, id string) (lesson.Lesson, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return lesson.Lesson{}, lesson.ErrNotFound
	}
	var d lessonDoc
	if err := r.coll.FindOne(ctx, bson.D{{Key: "_id", Value: oid}}).Decode(&d); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return lesson.Lesson{}, lesson.ErrNotFound
		}
		return lesson.Lesson{}, fmt.Errorf("find lesson: %w", err)
	}
	return d.toLesson(), nil
}

// List returns summaries newest first, filtered by level and topic id.
func (r *Lessons) List(ctx context.Context, f lesson.Filter) ([]lesson.Summary, error) {
	filter := bson.D{}
	if f.Level != "" {
		filter = append(filter, bson.E{Key: "level", Value: string(f.Level)})
	}
	if f.TopicID != "" {
		filter = append(filter, bson.E{Key: "topicId", Value: oidOrZero(f.TopicID)})
	}
	opts := options.Find().SetProjection(summaryProjection).SetSort(bson.D{{Key: "createdAt", Value: -1}})
	return r.findSummaries(ctx, filter, opts)
}

// Summaries returns the existing lessons among ids.
func (r *Lessons) Summaries(ctx context.Context, ids []string) ([]lesson.Summary, error) {
	oids := make([]bson.ObjectID, 0, len(ids))
	for _, id := range ids {
		if oid, err := bson.ObjectIDFromHex(id); err == nil {
			oids = append(oids, oid)
		}
	}
	if len(oids) == 0 {
		return nil, nil
	}
	filter := bson.D{{Key: "_id", Value: bson.D{{Key: "$in", Value: oids}}}}
	return r.findSummaries(ctx, filter, options.Find().SetProjection(summaryProjection))
}

func (r *Lessons) findSummaries(ctx context.Context, filter bson.D, opts *options.FindOptionsBuilder) ([]lesson.Summary, error) {
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("find lessons: %w", err)
	}
	var docs []summaryDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("read lessons: %w", err)
	}
	out := make([]lesson.Summary, len(docs))
	for i, d := range docs {
		out[i] = d.toSummary()
	}
	return out, nil
}

// UpdateInfo changes only the descriptive fields, leaving content and work results intact.
func (r *Lessons) UpdateInfo(ctx context.Context, id string, in lesson.Info) error {
	return r.updateOne(ctx, id, bson.D{{Key: "$set", Value: bson.D{
		{Key: "title", Value: in.Title},
		{Key: "level", Value: string(in.Level)},
		{Key: "topicId", Value: oidOrZero(in.TopicID)},
		{Key: "source", Value: in.Source},
		{Key: "license", Value: in.License},
		{Key: "updatedAt", Value: time.Now().UTC()},
	}}})
}

// ReplaceContent overwrites every field except the id and creation time.
func (r *Lessons) ReplaceContent(ctx context.Context, l lesson.Lesson) error {
	d := fromLesson(l)
	return r.updateOne(ctx, l.ID, bson.D{{Key: "$set", Value: append(bson.D{
		{Key: "title", Value: d.Title},
		{Key: "content", Value: d.Content},
		{Key: "level", Value: d.Level},
		{Key: "topicId", Value: d.TopicID},
		{Key: "source", Value: d.Source},
		{Key: "license", Value: d.License},
		{Key: "revision", Value: d.Revision},
		{Key: "sentences", Value: d.Sentences},
		{Key: "audioStatus", Value: d.AudioStatus},
		{Key: "audioError", Value: d.AudioError},
		{Key: "annotationStatus", Value: d.AnnotationStatus},
		{Key: "annotationError", Value: d.AnnotationError},
		{Key: "annotations", Value: d.Annotations},
		{Key: "extrasEditedByAdmin", Value: l.ExtrasEditedByAdmin},
		{Key: "quizVersion", Value: l.QuizVersion},
		{Key: "updatedAt", Value: d.UpdatedAt},
	}, extrasSet(l.Extras)...)}})
}

// SetStatus changes one work status if the lesson is still at revision.
func (r *Lessons) SetStatus(ctx context.Context, id string, revision int, t job.Type, st lesson.Status, errMsg string) (bool, error) {
	status, errField := "annotationStatus", "annotationError"
	if t == job.TypeTTS {
		status, errField = "audioStatus", "audioError"
	}
	return r.updateAtRevision(ctx, id, revision, bson.D{
		{Key: status, Value: string(st)}, {Key: errField, Value: errMsg},
	})
}

// SaveAudio stores sentence audio paths and marks audio done if still at revision.
func (r *Lessons) SaveAudio(ctx context.Context, id string, revision int, paths []string) (bool, error) {
	set := bson.D{{Key: "audioStatus", Value: string(lesson.StatusDone)}, {Key: "audioError", Value: ""}}
	for i, p := range paths {
		set = append(set, bson.E{Key: fmt.Sprintf("sentences.%d.audioPath", i), Value: p})
	}
	return r.updateAtRevision(ctx, id, revision, set)
}

// SaveAnnotations stores AI annotations and extras, marks them done and bumps quizVersion if
// still at revision.
func (r *Lessons) SaveAnnotations(ctx context.Context, id string, revision int, anns []lesson.Annotation,
	x lesson.Extras,
) (bool, error) {
	set := append(bson.D{
		{Key: "annotations", Value: fromAnnotations(anns)},
		{Key: "annotationStatus", Value: string(lesson.StatusDone)},
		{Key: "annotationError", Value: ""},
		{Key: "extrasEditedByAdmin", Value: false},
	}, extrasSet(x)...)
	return r.updateAtRevisionInc(ctx, id, revision, set, bson.D{{Key: "quizVersion", Value: 1}})
}

// ReplaceExtras stores admin-edited extras and marks them edited; bumpQuiz bumps quizVersion.
func (r *Lessons) ReplaceExtras(ctx context.Context, id string, x lesson.Extras, bumpQuiz bool) error {
	set := append(extrasSet(x),
		bson.E{Key: "extrasEditedByAdmin", Value: true},
		bson.E{Key: "updatedAt", Value: time.Now().UTC()},
	)
	update := bson.D{{Key: "$set", Value: set}}
	if bumpQuiz {
		update = append(update, bson.E{Key: "$inc", Value: bson.D{{Key: "quizVersion", Value: 1}}})
	}
	return r.updateOne(ctx, id, update)
}

// ReplaceAnnotations stores admin-edited annotations and marks them done.
func (r *Lessons) ReplaceAnnotations(ctx context.Context, id string, anns []lesson.Annotation) error {
	return r.updateOne(ctx, id, bson.D{{Key: "$set", Value: bson.D{
		{Key: "annotations", Value: fromAnnotations(anns)},
		{Key: "annotationStatus", Value: string(lesson.StatusDone)},
		{Key: "annotationError", Value: ""},
		{Key: "updatedAt", Value: time.Now().UTC()},
	}}})
}

// Delete removes a lesson; deleting a missing lesson is not an error.
func (r *Lessons) Delete(ctx context.Context, id string) error {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return lesson.ErrNotFound
	}
	if _, err := r.coll.DeleteOne(ctx, bson.D{{Key: "_id", Value: oid}}); err != nil {
		return fmt.Errorf("delete lesson: %w", err)
	}
	return nil
}

func (r *Lessons) updateOne(ctx context.Context, id string, update bson.D) error {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return lesson.ErrNotFound
	}
	res, err := r.coll.UpdateOne(ctx, bson.D{{Key: "_id", Value: oid}}, update)
	if err != nil {
		return fmt.Errorf("update lesson: %w", err)
	}
	if res.MatchedCount == 0 {
		return lesson.ErrNotFound
	}
	return nil
}

// updateAtRevision applies set only while the lesson is still at revision.
func (r *Lessons) updateAtRevision(ctx context.Context, id string, revision int, set bson.D) (bool, error) {
	return r.updateAtRevisionInc(ctx, id, revision, set, nil)
}

// updateAtRevisionInc is updateAtRevision with an optional $inc.
func (r *Lessons) updateAtRevisionInc(ctx context.Context, id string, revision int, set, inc bson.D) (bool, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return false, lesson.ErrNotFound
	}
	set = append(set, bson.E{Key: "updatedAt", Value: time.Now().UTC()})
	update := bson.D{{Key: "$set", Value: set}}
	if len(inc) > 0 {
		update = append(update, bson.E{Key: "$inc", Value: inc})
	}
	res, err := r.coll.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: oid}, {Key: "revision", Value: revision}}, update)
	if err != nil {
		return false, fmt.Errorf("update lesson: %w", err)
	}
	return res.MatchedCount == 1, nil
}

// oidOrZero parses a hex id; malformed ids become the zero id, which matches nothing.
func oidOrZero(id string) bson.ObjectID {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return bson.ObjectID{}
	}
	return oid
}

func hexOrEmpty(oid bson.ObjectID) string {
	if oid.IsZero() {
		return ""
	}
	return oid.Hex()
}

// --- topic.Lessons (F14), adapted in main ---

// CountByTopic counts lessons per topic id.
func (r *Lessons) CountByTopic(ctx context.Context) (map[string]int, error) {
	cur, err := r.coll.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$group", Value: bson.D{{Key: "_id", Value: "$topicId"}, {Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}}}}},
	})
	if err != nil {
		return nil, fmt.Errorf("count lessons: %w", err)
	}
	var rows []struct {
		ID    bson.ObjectID `bson:"_id"`
		Count int           `bson:"count"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, fmt.Errorf("read lesson counts: %w", err)
	}
	out := make(map[string]int, len(rows))
	for _, row := range rows {
		out[hexOrEmpty(row.ID)] += row.Count
	}
	return out, nil
}

// TopicOf returns the topic id of each existing lesson among ids.
func (r *Lessons) TopicOf(ctx context.Context, ids []string) (map[string]string, error) {
	sums, err := r.Summaries(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(sums))
	for _, s := range sums {
		out[s.ID] = s.TopicID
	}
	return out, nil
}

// SetLevelByTopic sets the level of every lesson of a topic.
func (r *Lessons) SetLevelByTopic(ctx context.Context, topicID, level string) error {
	_, err := r.coll.UpdateMany(ctx, bson.D{{Key: "topicId", Value: oidOrZero(topicID)}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "level", Value: level}}}})
	if err != nil {
		return fmt.Errorf("update lesson levels: %w", err)
	}
	return nil
}
