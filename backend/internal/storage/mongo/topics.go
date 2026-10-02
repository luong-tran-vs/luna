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

type topicDoc struct {
	ID          bson.ObjectID   `bson:"_id,omitempty"`
	Name        string          `bson:"name"`
	NameKey     string          `bson:"nameKey"`
	Level       string          `bson:"level"`
	Description string          `bson:"description"`
	LessonIDs   []bson.ObjectID `bson:"lessonIds"`
	Words       []string        `bson:"words"`
	WordsSeeded bool            `bson:"wordsSeeded"`
	CreatedAt   time.Time       `bson:"createdAt"`
	UpdatedAt   time.Time       `bson:"updatedAt"`
}

func (d topicDoc) toTopic() topic.Topic {
	t := topic.Topic{
		ID: d.ID.Hex(), Name: d.Name, Level: d.Level, Description: d.Description,
		LessonIDs: make([]string, len(d.LessonIDs)), CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
		Words: d.Words, WordsSeeded: d.WordsSeeded,
	}
	for i, oid := range d.LessonIDs {
		t.LessonIDs[i] = oid.Hex()
	}
	return t
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

// Topics implements topic.Repository on the "topics" collection; the roadmap is the
// lessonIds array of each topic.
type Topics struct {
	coll *mongo.Collection
}

// NewTopics returns a topic repository on db.
func NewTopics(db *mongo.Database) *Topics {
	return &Topics{coll: db.Collection("topics")}
}

var _ topic.Repository = (*Topics)(nil)

// Create inserts t; the unique (level, nameKey) index turns duplicates into ErrNameTaken.
func (r *Topics) Create(ctx context.Context, t topic.Topic) (topic.Topic, error) {
	oids, err := toOIDs(t.LessonIDs)
	if err != nil {
		return topic.Topic{}, err
	}
	d := topicDoc{
		ID: bson.NewObjectID(), Name: t.Name, NameKey: topic.NameKey(t.Name), Level: t.Level,
		Description: t.Description, LessonIDs: oids, CreatedAt: t.CreatedAt.UTC(), UpdatedAt: t.UpdatedAt.UTC(),
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

// List returns the topics of level ("" = all); the service sorts them.
func (r *Topics) List(ctx context.Context, level string) ([]topic.Topic, error) {
	filter := bson.D{}
	if level != "" {
		filter = append(filter, bson.E{Key: "level", Value: level})
	}
	cur, err := r.coll.Find(ctx, filter)
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

// Update changes name, level and description.
func (r *Topics) Update(ctx context.Context, id string, in topic.Input) (topic.Topic, error) {
	f, err := r.filter(id)
	if err != nil {
		return topic.Topic{}, err
	}
	var d topicDoc
	err = r.coll.FindOneAndUpdate(ctx, f, bson.D{{Key: "$set", Value: bson.D{
		{Key: "name", Value: in.Name},
		{Key: "nameKey", Value: topic.NameKey(in.Name)},
		{Key: "level", Value: in.Level},
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

// SetLessons replaces the roadmap.
func (r *Topics) SetLessons(ctx context.Context, id string, lessonIDs []string) error {
	f, err := r.filter(id)
	if err != nil {
		return err
	}
	oids, err := toOIDs(lessonIDs)
	if err != nil {
		return err
	}
	res, err := r.coll.UpdateOne(ctx, f, bson.D{{Key: "$set", Value: bson.D{
		{Key: "lessonIds", Value: oids}, {Key: "updatedAt", Value: time.Now().UTC()},
	}}})
	if err != nil {
		return fmt.Errorf("set roadmap: %w", err)
	}
	if res.MatchedCount == 0 {
		return topic.ErrNotFound
	}
	return nil
}

// RemoveLesson pulls a lesson out of the roadmap.
func (r *Topics) RemoveLesson(ctx context.Context, id, lessonID string) (bool, error) {
	f, err := r.filter(id)
	if err != nil {
		return false, nil //nolint:nilerr // a malformed topic id has no roadmap
	}
	lid, err := bson.ObjectIDFromHex(lessonID)
	if err != nil {
		return false, nil //nolint:nilerr // a malformed lesson id is in no roadmap
	}
	res, err := r.coll.UpdateOne(ctx, append(f, bson.E{Key: "lessonIds", Value: lid}),
		bson.D{{Key: "$pull", Value: bson.D{{Key: "lessonIds", Value: lid}}}})
	if err != nil {
		return false, fmt.Errorf("remove from roadmap: %w", err)
	}
	return res.ModifiedCount == 1, nil
}

// AppendLesson pushes a lesson at the end of the roadmap unless it is already there.
func (r *Topics) AppendLesson(ctx context.Context, id, lessonID string) error {
	f, err := r.filter(id)
	if err != nil {
		return err
	}
	lid, err := bson.ObjectIDFromHex(lessonID)
	if err != nil {
		return fmt.Errorf("lesson id: %w", err)
	}
	_, err = r.coll.UpdateOne(ctx, append(f, bson.E{Key: "lessonIds", Value: bson.D{{Key: "$ne", Value: lid}}}),
		bson.D{{Key: "$push", Value: bson.D{{Key: "lessonIds", Value: lid}}}})
	if err != nil {
		return fmt.Errorf("append to roadmap: %w", err)
	}
	return nil
}

// SetWords replaces the topic's words and marks them seeded, so the startup seed never
// overwrites an admin edit (F18).
func (r *Topics) SetWords(ctx context.Context, id string, words []string) (topic.Topic, error) {
	f, err := r.filter(id)
	if err != nil {
		return topic.Topic{}, err
	}
	if words == nil {
		words = []string{}
	}
	var d topicDoc
	err = r.coll.FindOneAndUpdate(ctx, f, bson.D{{Key: "$set", Value: bson.D{
		{Key: "words", Value: words},
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
