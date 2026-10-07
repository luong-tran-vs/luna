package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/luongtran/luna/backend/internal/topic"
)

// wordDoc is a topic word (F18). Words stored before they had a level are plain strings; they
// decode as a word for every level.
type wordDoc struct {
	Text  string `bson:"text"`
	Level string `bson:"level"`
}

// UnmarshalBSONValue accepts {text, level} or a plain string.
func (w *wordDoc) UnmarshalBSONValue(typ byte, data []byte) error {
	raw := bson.RawValue{Type: bson.Type(typ), Value: data}
	if s, ok := raw.StringValueOK(); ok {
		*w = wordDoc{Text: s}
		return nil
	}
	var d struct {
		Text  string `bson:"text"`
		Level string `bson:"level"`
	}
	if err := raw.Unmarshal(&d); err != nil {
		return fmt.Errorf("topic word: %w", err)
	}
	*w = wordDoc(d)
	return nil
}

func toWordDocs(words []topic.Word) []wordDoc {
	out := make([]wordDoc, len(words))
	for i, w := range words {
		out[i] = wordDoc(w)
	}
	return out
}

func fromWordDocs(docs []wordDoc) []topic.Word {
	out := make([]topic.Word, len(docs))
	for i, d := range docs {
		out[i] = topic.Word(d)
	}
	return out
}

type topicDoc struct {
	ID          bson.ObjectID              `bson:"_id,omitempty"`
	Name        string                     `bson:"name"`
	NameKey     string                     `bson:"nameKey"`
	Description string                     `bson:"description"`
	Roadmaps    map[string][]bson.ObjectID `bson:"roadmaps"`
	Words       []wordDoc                  `bson:"words"`
	WordsSeeded bool                       `bson:"wordsSeeded"`
	CreatedAt   time.Time                  `bson:"createdAt"`
	UpdatedAt   time.Time                  `bson:"updatedAt"`
}

func (d topicDoc) toTopic() topic.Topic {
	t := topic.Topic{
		ID: d.ID.Hex(), Name: d.Name, Description: d.Description, Roadmaps: hexRoadmaps(d.Roadmaps),
		CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt, Words: fromWordDocs(d.Words), WordsSeeded: d.WordsSeeded,
	}
	return t
}

func hexRoadmaps(in map[string][]bson.ObjectID) map[string][]string {
	out := make(map[string][]string, len(in))
	for level, oids := range in {
		ids := make([]string, len(oids))
		for i, oid := range oids {
			ids[i] = oid.Hex()
		}
		out[level] = ids
	}
	return out
}

func oidRoadmaps(in map[string][]string) (map[string][]bson.ObjectID, error) {
	out := make(map[string][]bson.ObjectID, len(in))
	for level, ids := range in {
		oids, err := toOIDs(ids)
		if err != nil {
			return nil, err
		}
		out[level] = oids
	}
	return out, nil
}

func toOIDs(ids []string) ([]bson.ObjectID, error) {
	oids := make([]bson.ObjectID, len(ids))
	for i, id := range ids {
		oid, err := bson.ObjectIDFromHex(id)
		if err != nil {
			return nil, fmt.Errorf("id %q: %w", id, err)
		}
		oids[i] = oid
	}
	return oids, nil
}

// roadmapField is the path of the roadmap of level inside a topic document.
func roadmapField(level string) (string, error) {
	if !topic.ValidLevel(level) {
		return "", fmt.Errorf("roadmap level %q", level)
	}
	return "roadmaps." + level, nil
}

// Topics implements topic.Repository on the "topics" collection; the roadmaps are the
// roadmaps.<level> arrays of each topic.
type Topics struct {
	coll *mongo.Collection
}

// NewTopics returns a topic repository on db.
func NewTopics(db *mongo.Database) *Topics {
	return &Topics{coll: db.Collection("topics")}
}

var _ topic.Repository = (*Topics)(nil)

// Create inserts t; the unique nameKey index turns duplicates into ErrNameTaken.
func (r *Topics) Create(ctx context.Context, t topic.Topic) (topic.Topic, error) {
	roadmaps, err := oidRoadmaps(t.Roadmaps)
	if err != nil {
		return topic.Topic{}, err
	}
	d := topicDoc{
		ID: bson.NewObjectID(), Name: t.Name, NameKey: topic.NameKey(t.Name), Description: t.Description,
		Roadmaps: roadmaps, Words: []wordDoc{}, CreatedAt: t.CreatedAt.UTC(), UpdatedAt: t.UpdatedAt.UTC(),
	}
	if _, err := r.coll.InsertOne(ctx, d); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return topic.Topic{}, topic.ErrNameTaken
		}
		return topic.Topic{}, fmt.Errorf("insert topic: %w", err)
	}
	return d.toTopic(), nil
}

func (r *Topics) filter(id string) (bson.D, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, topic.ErrNotFound
	}
	return bson.D{{Key: "_id", Value: oid}}, nil
}

// Get returns topic.ErrNotFound for unknown or malformed ids.
func (r *Topics) Get(ctx context.Context, id string) (topic.Topic, error) {
	f, err := r.filter(id)
	if err != nil {
		return topic.Topic{}, err
	}
	var d topicDoc
	if err := r.coll.FindOne(ctx, f).Decode(&d); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return topic.Topic{}, topic.ErrNotFound
		}
		return topic.Topic{}, fmt.Errorf("find topic: %w", err)
	}
	return d.toTopic(), nil
}

// List returns every topic; the service sorts them.
func (r *Topics) List(ctx context.Context) ([]topic.Topic, error) {
	cur, err := r.coll.Find(ctx, bson.D{})
	if err != nil {
		return nil, fmt.Errorf("find topics: %w", err)
	}
	var docs []topicDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("read topics: %w", err)
	}
	out := make([]topic.Topic, len(docs))
	for i, d := range docs {
		out[i] = d.toTopic()
	}
	return out, nil
}

// Update changes name and description.
func (r *Topics) Update(ctx context.Context, id string, in topic.Input) (topic.Topic, error) {
	f, err := r.filter(id)
	if err != nil {
		return topic.Topic{}, err
	}
	var d topicDoc
	err = r.coll.FindOneAndUpdate(ctx, f, bson.D{{Key: "$set", Value: bson.D{
		{Key: "name", Value: in.Name},
		{Key: "nameKey", Value: topic.NameKey(in.Name)},
		{Key: "description", Value: in.Description},
		{Key: "updatedAt", Value: time.Now().UTC()},
	}}}, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&d)
	switch {
	case mongo.IsDuplicateKeyError(err):
		return topic.Topic{}, topic.ErrNameTaken
	case errors.Is(err, mongo.ErrNoDocuments):
		return topic.Topic{}, topic.ErrNotFound
	case err != nil:
		return topic.Topic{}, fmt.Errorf("update topic: %w", err)
	}
	return d.toTopic(), nil
}

// Delete removes a topic.
func (r *Topics) Delete(ctx context.Context, id string) error {
	f, err := r.filter(id)
	if err != nil {
		return err
	}
	if _, err := r.coll.DeleteOne(ctx, f); err != nil {
		return fmt.Errorf("delete topic: %w", err)
	}
	return nil
}

// SetLessons replaces the roadmap of level.
func (r *Topics) SetLessons(ctx context.Context, id, level string, lessonIDs []string) error {
	f, err := r.filter(id)
	if err != nil {
		return err
	}
	field, err := roadmapField(level)
	if err != nil {
		return err
	}
	oids, err := toOIDs(lessonIDs)
	if err != nil {
		return err
	}
	res, err := r.coll.UpdateOne(ctx, f, bson.D{{Key: "$set", Value: bson.D{
		{Key: field, Value: oids}, {Key: "updatedAt", Value: time.Now().UTC()},
	}}})
	if err != nil {
		return fmt.Errorf("set roadmap: %w", err)
	}
	if res.MatchedCount == 0 {
		return topic.ErrNotFound
	}
	return nil
}

// RemoveLesson pulls a lesson out of the roadmap of level.
func (r *Topics) RemoveLesson(ctx context.Context, id, level, lessonID string) (bool, error) {
	f, err := r.filter(id)
	if err != nil {
		return false, nil //nolint:nilerr // a malformed topic id has no roadmap
	}
	field, err := roadmapField(level)
	if err != nil {
		return false, nil //nolint:nilerr // an unknown level has no roadmap
	}
	lid, err := bson.ObjectIDFromHex(lessonID)
	if err != nil {
		return false, nil //nolint:nilerr // a malformed lesson id is in no roadmap
	}
	res, err := r.coll.UpdateOne(ctx, append(f, bson.E{Key: field, Value: lid}),
		bson.D{{Key: "$pull", Value: bson.D{{Key: field, Value: lid}}}})
	if err != nil {
		return false, fmt.Errorf("remove from roadmap: %w", err)
	}
	return res.ModifiedCount == 1, nil
}

// AppendLesson pushes a lesson at the end of the roadmap of level unless it is already there.
func (r *Topics) AppendLesson(ctx context.Context, id, level, lessonID string) error {
	f, err := r.filter(id)
	if err != nil {
		return err
	}
	field, err := roadmapField(level)
	if err != nil {
		return err
	}
	lid, err := bson.ObjectIDFromHex(lessonID)
	if err != nil {
		return fmt.Errorf("lesson id: %w", err)
	}
	_, err = r.coll.UpdateOne(ctx, append(f, bson.E{Key: field, Value: bson.D{{Key: "$ne", Value: lid}}}),
		bson.D{{Key: "$push", Value: bson.D{{Key: field, Value: lid}}}})
	if err != nil {
		return fmt.Errorf("append to roadmap: %w", err)
	}
	return nil
}

// SetWords replaces the topic's words and marks them seeded, so the startup seed never
// overwrites an admin edit (F18).
func (r *Topics) SetWords(ctx context.Context, id string, words []topic.Word) (topic.Topic, error) {
	f, err := r.filter(id)
	if err != nil {
		return topic.Topic{}, err
	}
	var d topicDoc
	err = r.coll.FindOneAndUpdate(ctx, f, bson.D{{Key: "$set", Value: bson.D{
		{Key: "words", Value: toWordDocs(words)},
		{Key: "wordsSeeded", Value: true},
		{Key: "updatedAt", Value: time.Now().UTC()},
	}}}, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&d)
	switch {
	case errors.Is(err, mongo.ErrNoDocuments):
		return topic.Topic{}, topic.ErrNotFound
	case err != nil:
		return topic.Topic{}, fmt.Errorf("set topic words: %w", err)
	}
	return d.toTopic(), nil
}
