package mongo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/luongtran/luna/backend/internal/export"
	"github.com/luongtran/luna/backend/internal/toeic"
)

// The TOEIC content collections (F27); not learner data.
const (
	collToeicGroups = "toeic_groups"
	collToeicImages = "toeic_images"
	collToeicTests  = "toeic_tests"
)

// toBSON turns a value into a sub-document with its JSON field names, so the domain types need no
// bson tags.
func toBSON(v any) (bson.D, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("encode toeic body: %w", err)
	}
	var d bson.D
	if err := bson.UnmarshalExtJSON(b, false, &d); err != nil {
		return nil, fmt.Errorf("convert toeic body: %w", err)
	}
	return d, nil
}

// fromBSON reads a sub-document written by toBSON into v.
func fromBSON(raw bson.Raw, v any) error {
	if len(raw) == 0 {
		return nil
	}
	b, err := bson.MarshalExtJSON(raw, false, false)
	if err != nil {
		return fmt.Errorf("convert toeic body: %w", err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("decode toeic body: %w", err)
	}
	return nil
}

// --- toeic_groups ---

type toeicGroupDoc struct {
	ID        bson.ObjectID `bson:"_id,omitempty"`
	Key       string        `bson:"key"`
	Part      int           `bson:"part"`
	Level     int           `bson:"level"`
	Status    string        `bson:"status"`
	Source    string        `bson:"source"`
	Version   int           `bson:"version"`
	Content   bson.Raw      `bson:"content"`
	CreatedAt time.Time     `bson:"createdAt"`
	UpdatedAt time.Time     `bson:"updatedAt"`
}

func (d toeicGroupDoc) group() (toeic.Group, error) {
	g := toeic.Group{ID: d.ID.Hex(), Key: d.Key, Part: toeic.Part(d.Part), Level: d.Level, Status: toeic.Status(d.Status),
		Source: d.Source, Version: d.Version, CreatedAt: d.CreatedAt.UTC(), UpdatedAt: d.UpdatedAt.UTC()}
	var c toeic.Content
	if err := fromBSON(d.Content, &c); err != nil {
		return toeic.Group{}, err
	}
	g.SetContent(c)
	return g, nil
}

// ToeicGroups implements toeic.GroupRepository on "toeic_groups"; key is unique.
type ToeicGroups struct{ coll *mongo.Collection }

// NewToeicGroups returns the TOEIC groups repository on db.
func NewToeicGroups(db *mongo.Database) *ToeicGroups {
	return &ToeicGroups{coll: db.Collection(collToeicGroups)}
}

var _ toeic.GroupRepository = (*ToeicGroups)(nil)

func (r *ToeicGroups) find(ctx context.Context, filter bson.D) ([]toeic.Group, error) {
	cur, err := r.coll.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "part", Value: 1}, {Key: "key", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("find toeic groups: %w", err)
	}
	var docs []toeicGroupDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("read toeic groups: %w", err)
	}
	out := make([]toeic.Group, 0, len(docs))
	for _, d := range docs {
		g, err := d.group()
		if err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, nil
}

// List returns the groups matching f.
func (r *ToeicGroups) List(ctx context.Context, f toeic.GroupFilter) ([]toeic.Group, error) {
	filter := bson.D{}
	if f.Part != 0 {
		filter = append(filter, bson.E{Key: "part", Value: int(f.Part)})
	}
	if f.Status != "" {
		filter = append(filter, bson.E{Key: "status", Value: string(f.Status)})
	}
	return r.find(ctx, filter)
}

// ByKeys returns the groups of the keys that exist.
func (r *ToeicGroups) ByKeys(ctx context.Context, keys []string) ([]toeic.Group, error) {
	if len(keys) == 0 {
		return nil, nil
	}
	return r.find(ctx, bson.D{{Key: "key", Value: bson.D{{Key: "$in", Value: keys}}}})
}

// Published returns the published groups of a part (all parts when 0).
func (r *ToeicGroups) Published(ctx context.Context, part toeic.Part) ([]toeic.Group, error) {
	return r.List(ctx, toeic.GroupFilter{Part: part, Status: toeic.StatusPublished})
}

// Upsert creates the new keys and replaces the content of the known ones, keeping their status and
// creation time and raising their version.
func (r *ToeicGroups) Upsert(ctx context.Context, gs []toeic.Group) (created, replaced []string, err error) {
	for _, g := range gs {
		content, err := toBSON(g.Content())
		if err != nil {
			return created, replaced, err
		}
		res, err := r.coll.UpdateOne(ctx, bson.D{{Key: "key", Value: g.Key}}, bson.D{
			{Key: "$set", Value: bson.D{{Key: "part", Value: int(g.Part)}, {Key: "level", Value: g.Level}, {Key: "source", Value: g.Source},
				{Key: "content", Value: content}, {Key: "updatedAt", Value: g.UpdatedAt.UTC()}}},
			{Key: "$inc", Value: bson.D{{Key: "version", Value: 1}}},
			{Key: "$setOnInsert", Value: bson.D{{Key: "status", Value: string(g.Status)}, {Key: "createdAt", Value: g.CreatedAt.UTC()}}},
		}, options.UpdateOne().SetUpsert(true))
		if err != nil {
			return created, replaced, fmt.Errorf("upsert toeic group %s: %w", g.Key, err)
		}
		if res.UpsertedCount > 0 {
			created = append(created, g.Key)
		} else {
			replaced = append(replaced, g.Key)
		}
	}
	return created, replaced, nil
}

// SetStatus changes the status of a group.
func (r *ToeicGroups) SetStatus(ctx context.Context, key string, s toeic.Status) error {
	res, err := r.coll.UpdateOne(ctx, bson.D{{Key: "key", Value: key}}, bson.D{{Key: "$set", Value: bson.D{{Key: "status", Value: string(s)}}}})
	if err != nil {
		return fmt.Errorf("set toeic group status: %w", err)
	}
	if res.MatchedCount == 0 {
		return toeic.ErrNotFound
	}
	return nil
}

// Delete removes the groups of the keys.
func (r *ToeicGroups) Delete(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	if _, err := r.coll.DeleteMany(ctx, bson.D{{Key: "key", Value: bson.D{{Key: "$in", Value: keys}}}}); err != nil {
		return fmt.Errorf("delete toeic groups: %w", err)
	}
	return nil
}

// --- toeic_images ---

type toeicImageDoc struct {
	ID        string    `bson:"_id"`
	GroupKey  string    `bson:"groupKey"`
	Slot      string    `bson:"slot"`
	MIME      string    `bson:"mime"`
	Data      []byte    `bson:"data"`
	Origin    string    `bson:"origin"`
	CreatedAt time.Time `bson:"createdAt"`
}

// ToeicImages implements toeic.ImageRepository on "toeic_images", beside the other pictures.
type ToeicImages struct{ coll *mongo.Collection }

// NewToeicImages returns the TOEIC pictures repository on db.
func NewToeicImages(db *mongo.Database) *ToeicImages {
	return &ToeicImages{coll: db.Collection(collToeicImages)}
}

var _ toeic.ImageRepository = (*ToeicImages)(nil)

func imageID(key, slot string) string { return key + "/" + slot }

// Get returns a picture.
func (r *ToeicImages) Get(ctx context.Context, key, slot string) (toeic.Image, error) {
	var d toeicImageDoc
	err := r.coll.FindOne(ctx, bson.D{{Key: "_id", Value: imageID(key, slot)}}).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return toeic.Image{}, toeic.ErrNotFound
	}
	if err != nil {
		return toeic.Image{}, fmt.Errorf("find toeic image: %w", err)
	}
	return toeic.Image{GroupKey: d.GroupKey, Slot: d.Slot, MIME: d.MIME, Data: d.Data, Origin: d.Origin, CreatedAt: d.CreatedAt.UTC()}, nil
}

// Save stores a picture, replacing the slot's previous one.
func (r *ToeicImages) Save(ctx context.Context, img toeic.Image) error {
	d := toeicImageDoc{ID: imageID(img.GroupKey, img.Slot), GroupKey: img.GroupKey, Slot: img.Slot, MIME: img.MIME, Data: img.Data,
		Origin: img.Origin, CreatedAt: img.CreatedAt.UTC()}
	if _, err := r.coll.ReplaceOne(ctx, bson.D{{Key: "_id", Value: d.ID}}, d, options.Replace().SetUpsert(true)); err != nil {
		return fmt.Errorf("save toeic image: %w", err)
	}
	return nil
}

// Delete removes a picture.
func (r *ToeicImages) Delete(ctx context.Context, key, slot string) error {
	if _, err := r.coll.DeleteOne(ctx, bson.D{{Key: "_id", Value: imageID(key, slot)}}); err != nil {
		return fmt.Errorf("delete toeic image: %w", err)
	}
	return nil
}

// Has returns the slots with a picture of each group.
func (r *ToeicImages) Has(ctx context.Context, keys []string) (map[string][]string, error) {
	out := map[string][]string{}
	if len(keys) == 0 {
		return out, nil
	}
	cur, err := r.coll.Find(ctx, bson.D{{Key: "groupKey", Value: bson.D{{Key: "$in", Value: keys}}}},
		options.Find().SetProjection(bson.D{{Key: "groupKey", Value: 1}, {Key: "slot", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("find toeic images: %w", err)
	}
	var docs []toeicImageDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("read toeic images: %w", err)
	}
	for _, d := range docs {
		out[d.GroupKey] = append(out[d.GroupKey], d.Slot)
	}
	return out, nil
}

// --- toeic_tests ---

type toeicTestDoc struct {
	ID        bson.ObjectID `bson:"_id,omitempty"`
	Key       string        `bson:"key"`
	Title     string        `bson:"title"`
	Kind      string        `bson:"kind"`
	Order     int           `bson:"order"`
	Groups    []string      `bson:"groups"`
	Status    string        `bson:"status"`
	CreatedAt time.Time     `bson:"createdAt"`
	UpdatedAt time.Time     `bson:"updatedAt"`
}

func (d toeicTestDoc) test() toeic.Test {
	return toeic.Test{ID: d.ID.Hex(), Key: d.Key, Title: d.Title, Kind: toeic.TestKind(d.Kind), Order: d.Order, Groups: d.Groups,
		Status: toeic.Status(d.Status), CreatedAt: d.CreatedAt.UTC(), UpdatedAt: d.UpdatedAt.UTC()}
}

// ToeicTests implements toeic.TestRepository on "toeic_tests"; key is unique.
type ToeicTests struct{ coll *mongo.Collection }

// NewToeicTests returns the TOEIC tests repository on db.
func NewToeicTests(db *mongo.Database) *ToeicTests {
	return &ToeicTests{coll: db.Collection(collToeicTests)}
}

var _ toeic.TestRepository = (*ToeicTests)(nil)

// List returns the tests by order.
func (r *ToeicTests) List(ctx context.Context, onlyPublished bool) ([]toeic.Test, error) {
	filter := bson.D{}
	if onlyPublished {
		filter = bson.D{{Key: "status", Value: string(toeic.StatusPublished)}}
	}
	cur, err := r.coll.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "order", Value: 1}, {Key: "key", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("find toeic tests: %w", err)
	}
	var docs []toeicTestDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("read toeic tests: %w", err)
	}
	out := make([]toeic.Test, len(docs))
	for i, d := range docs {
		out[i] = d.test()
	}
	return out, nil
}

// Get returns a test.
func (r *ToeicTests) Get(ctx context.Context, key string) (toeic.Test, error) {
	var d toeicTestDoc
	err := r.coll.FindOne(ctx, bson.D{{Key: "key", Value: key}}).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return toeic.Test{}, toeic.ErrNotFound
	}
	if err != nil {
		return toeic.Test{}, fmt.Errorf("find toeic test: %w", err)
	}
	return d.test(), nil
}

// Save creates or replaces a test, keeping its creation time.
func (r *ToeicTests) Save(ctx context.Context, t toeic.Test) error {
	groups := t.Groups
	if groups == nil {
		groups = []string{}
	}
	_, err := r.coll.UpdateOne(ctx, bson.D{{Key: "key", Value: t.Key}}, bson.D{
		{Key: "$set", Value: bson.D{{Key: "title", Value: t.Title}, {Key: "kind", Value: string(t.Kind)}, {Key: "order", Value: t.Order},
			{Key: "groups", Value: groups}, {Key: "status", Value: string(t.Status)}, {Key: "updatedAt", Value: t.UpdatedAt.UTC()}}},
		{Key: "$setOnInsert", Value: bson.D{{Key: "createdAt", Value: t.CreatedAt.UTC()}}},
	}, options.UpdateOne().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("save toeic test: %w", err)
	}
	return nil
}

// SetStatus changes the status of a test.
func (r *ToeicTests) SetStatus(ctx context.Context, key string, s toeic.Status) error {
	res, err := r.coll.UpdateOne(ctx, bson.D{{Key: "key", Value: key}}, bson.D{{Key: "$set", Value: bson.D{{Key: "status", Value: string(s)}}}})
	if err != nil {
		return fmt.Errorf("set toeic test status: %w", err)
	}
	if res.MatchedCount == 0 {
		return toeic.ErrNotFound
	}
	return nil
}

// Delete removes a test.
func (r *ToeicTests) Delete(ctx context.Context, key string) error {
	res, err := r.coll.DeleteOne(ctx, bson.D{{Key: "key", Value: key}})
	if err != nil {
		return fmt.Errorf("delete toeic test: %w", err)
	}
	if res.DeletedCount == 0 {
		return toeic.ErrNotFound
	}
	return nil
}

// --- toeic_attempts ---

type toeicSourceDoc struct {
	Kind        string `bson:"kind"`
	TestKey     string `bson:"testKey"`
	Part        int    `bson:"part"`
	Tag         string `bson:"tag"`
	FromAttempt string `bson:"fromAttempt"`
	Size        int    `bson:"size"`
	Count       int    `bson:"count"`
}

type toeicAttemptDoc struct {
	ID          bson.ObjectID  `bson:"_id,omitempty"`
	UserID      bson.ObjectID  `bson:"userId"`
	Mode        string         `bson:"mode"`
	Status      string         `bson:"status"`
	Source      toeicSourceDoc `bson:"source"`
	GroupKeys   []string       `bson:"groupKeys"`
	QuestionIDs []string       `bson:"questionIds"`
	Counted     bool           `bson:"counted"`
	State       bson.Raw       `bson:"state"`
	Result      bson.Raw       `bson:"result,omitempty"`
	StartedAt   time.Time      `bson:"startedAt"`
	SavedAt     time.Time      `bson:"savedAt"`
	SubmittedAt *time.Time     `bson:"submittedAt,omitempty"`
}

func attemptDocOf(a toeic.Attempt) (toeicAttemptDoc, error) {
	uid, err := bson.ObjectIDFromHex(a.UserID)
	if err != nil {
		return toeicAttemptDoc{}, fmt.Errorf("toeic attempt user id: %w", err)
	}
	state, err := toBSON(a.State())
	if err != nil {
		return toeicAttemptDoc{}, err
	}
	stateRaw, err := bson.Marshal(state)
	if err != nil {
		return toeicAttemptDoc{}, fmt.Errorf("encode toeic state: %w", err)
	}
	d := toeicAttemptDoc{
		UserID: uid, Mode: string(a.Mode), Status: string(a.Status), GroupKeys: a.GroupKeys, QuestionIDs: a.QuestionIDs,
		Counted: a.Counted, State: stateRaw, StartedAt: a.StartedAt.UTC(), SavedAt: a.SavedAt.UTC(),
		Source: toeicSourceDoc{Kind: string(a.Source.Kind), TestKey: a.Source.TestKey, Part: int(a.Source.Part), Tag: a.Source.Tag,
			FromAttempt: a.Source.FromAttempt, Size: a.Source.Size, Count: a.Source.Count},
	}
	if a.ID != "" {
		if d.ID, err = bson.ObjectIDFromHex(a.ID); err != nil {
			return toeicAttemptDoc{}, toeic.ErrNotFound
		}
	}
	if a.Result != nil {
		res, err := toBSON(a.Result)
		if err != nil {
			return toeicAttemptDoc{}, err
		}
		if d.Result, err = bson.Marshal(res); err != nil {
			return toeicAttemptDoc{}, fmt.Errorf("encode toeic result: %w", err)
		}
	}
	if !a.SubmittedAt.IsZero() {
		at := a.SubmittedAt.UTC()
		d.SubmittedAt = &at
	}
	return d, nil
}

func (d toeicAttemptDoc) attempt() (toeic.Attempt, error) {
	a := toeic.Attempt{
		ID: d.ID.Hex(), UserID: d.UserID.Hex(), Mode: toeic.Mode(d.Mode), Status: toeic.AttemptStatus(d.Status),
		GroupKeys: d.GroupKeys, QuestionIDs: d.QuestionIDs, Counted: d.Counted, StartedAt: d.StartedAt.UTC(), SavedAt: d.SavedAt.UTC(),
		Source: toeic.Source{Kind: toeic.SourceKind(d.Source.Kind), TestKey: d.Source.TestKey, Part: toeic.Part(d.Source.Part),
			Tag: d.Source.Tag, FromAttempt: d.Source.FromAttempt, Size: d.Source.Size, Count: d.Source.Count},
	}
	var s toeic.State
	if err := fromBSON(d.State, &s); err != nil {
		return toeic.Attempt{}, err
	}
	a.SetState(s)
	if len(d.Result) > 0 {
		var r toeic.Result
		if err := fromBSON(d.Result, &r); err != nil {
			return toeic.Attempt{}, err
		}
		a.Result = &r
	}
	if d.SubmittedAt != nil {
		a.SubmittedAt = d.SubmittedAt.UTC()
	}
	return a, nil
}

// ToeicAttempts implements toeic.AttemptRepository on "toeic_attempts".
type ToeicAttempts struct{ coll *mongo.Collection }

// NewToeicAttempts returns the TOEIC attempts repository on db.
func NewToeicAttempts(db *mongo.Database) *ToeicAttempts {
	return &ToeicAttempts{coll: db.Collection(export.CollToeicAttempts)}
}

var _ toeic.AttemptRepository = (*ToeicAttempts)(nil)

// Create stores a new attempt.
func (r *ToeicAttempts) Create(ctx context.Context, a toeic.Attempt) (toeic.Attempt, error) {
	a.ID = ""
	d, err := attemptDocOf(a)
	if err != nil {
		return toeic.Attempt{}, err
	}
	d.ID = bson.NewObjectID()
	if _, err := r.coll.InsertOne(ctx, d); err != nil {
		return toeic.Attempt{}, fmt.Errorf("insert toeic attempt: %w", err)
	}
	a.ID = d.ID.Hex()
	return a, nil
}

// Get returns the user's attempt.
func (r *ToeicAttempts) Get(ctx context.Context, userID, id string) (toeic.Attempt, error) {
	oid, err1 := bson.ObjectIDFromHex(id)
	uid, err2 := bson.ObjectIDFromHex(userID)
	if err1 != nil || err2 != nil {
		return toeic.Attempt{}, toeic.ErrNotFound
	}
	var d toeicAttemptDoc
	err := r.coll.FindOne(ctx, bson.D{{Key: "_id", Value: oid}, {Key: "userId", Value: uid}}).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return toeic.Attempt{}, toeic.ErrNotFound
	}
	if err != nil {
		return toeic.Attempt{}, fmt.Errorf("find toeic attempt: %w", err)
	}
	return d.attempt()
}

// Save stores the progress of an attempt in progress.
func (r *ToeicAttempts) Save(ctx context.Context, a toeic.Attempt) error {
	d, err := attemptDocOf(a)
	if err != nil {
		return err
	}
	res, err := r.coll.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: d.ID}, {Key: "userId", Value: d.UserID}, {Key: "status", Value: string(toeic.AttemptInProgress)}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "state", Value: d.State}, {Key: "savedAt", Value: d.SavedAt}}}})
	if err != nil {
		return fmt.Errorf("save toeic attempt: %w", err)
	}
	if res.MatchedCount == 0 {
		return toeic.ErrAttemptSubmitted
	}
	return nil
}

// Submit stores the result and the submitted status.
func (r *ToeicAttempts) Submit(ctx context.Context, a toeic.Attempt) error {
	d, err := attemptDocOf(a)
	if err != nil {
		return err
	}
	res, err := r.coll.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: d.ID}, {Key: "userId", Value: d.UserID}, {Key: "status", Value: string(toeic.AttemptInProgress)}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "state", Value: d.State}, {Key: "result", Value: d.Result}, {Key: "status", Value: d.Status},
			{Key: "counted", Value: d.Counted}, {Key: "savedAt", Value: d.SavedAt}, {Key: "submittedAt", Value: d.SubmittedAt}}}})
	if err != nil {
		return fmt.Errorf("submit toeic attempt: %w", err)
	}
	if res.MatchedCount == 0 {
		return toeic.ErrAttemptSubmitted
	}
	return nil
}

func (r *ToeicAttempts) find(ctx context.Context, filter bson.D, limit int) ([]toeic.Attempt, error) {
	opts := options.Find().SetSort(bson.D{{Key: "startedAt", Value: -1}, {Key: "_id", Value: -1}})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("find toeic attempts: %w", err)
	}
	var docs []toeicAttemptDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("read toeic attempts: %w", err)
	}
	out := make([]toeic.Attempt, 0, len(docs))
	for _, d := range docs {
		a, err := d.attempt()
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

// List returns the user's attempts matching f, newest first.
func (r *ToeicAttempts) List(ctx context.Context, userID string, f toeic.AttemptFilter) ([]toeic.Attempt, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, fmt.Errorf("toeic attempt user id: %w", err)
	}
	filter := bson.D{{Key: "userId", Value: uid}}
	if f.TestKey != "" {
		filter = append(filter, bson.E{Key: "source.testKey", Value: f.TestKey})
	}
	if f.Status != "" {
		filter = append(filter, bson.E{Key: "status", Value: string(f.Status)})
	}
	if f.Counted {
		filter = append(filter, bson.E{Key: "counted", Value: true})
	}
	return r.find(ctx, filter, f.Limit)
}

// InProgress returns the user's attempt in progress of a test.
func (r *ToeicAttempts) InProgress(ctx context.Context, userID, testKey string) (toeic.Attempt, error) {
	list, err := r.List(ctx, userID, toeic.AttemptFilter{TestKey: testKey, Status: toeic.AttemptInProgress, Limit: 1})
	if err != nil {
		return toeic.Attempt{}, err
	}
	if len(list) == 0 {
		return toeic.Attempt{}, toeic.ErrNotFound
	}
	return list[0], nil
}

// AnsweredQuestions returns the questions of the user's submitted attempts.
func (r *ToeicAttempts) AnsweredQuestions(ctx context.Context, userID string) (map[string]bool, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, fmt.Errorf("toeic attempt user id: %w", err)
	}
	cur, err := r.coll.Find(ctx, bson.D{{Key: "userId", Value: uid}, {Key: "status", Value: string(toeic.AttemptSubmitted)}},
		options.Find().SetProjection(bson.D{{Key: "questionIds", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("find toeic answered: %w", err)
	}
	var docs []struct {
		QuestionIDs []string `bson:"questionIds"`
	}
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("read toeic answered: %w", err)
	}
	out := map[string]bool{}
	for _, d := range docs {
		for _, id := range d.QuestionIDs {
			out[id] = true
		}
	}
	return out, nil
}

// WithGroup returns every submitted attempt having the group.
func (r *ToeicAttempts) WithGroup(ctx context.Context, groupKey string) ([]toeic.Attempt, error) {
	return r.find(ctx, bson.D{{Key: "groupKeys", Value: groupKey}, {Key: "status", Value: string(toeic.AttemptSubmitted)}}, 0)
}

// CountWithTest returns how many attempts are of the test.
func (r *ToeicAttempts) CountWithTest(ctx context.Context, testKey string) (int, error) {
	n, err := r.coll.CountDocuments(ctx, bson.D{{Key: "source.testKey", Value: testKey}})
	if err != nil {
		return 0, fmt.Errorf("count toeic attempts of test: %w", err)
	}
	return int(n), nil
}

// UpdateResult replaces the result of a submitted attempt.
func (r *ToeicAttempts) UpdateResult(ctx context.Context, id string, res toeic.Result) error {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return toeic.ErrNotFound
	}
	d, err := toBSON(res)
	if err != nil {
		return err
	}
	out, err := r.coll.UpdateOne(ctx, bson.D{{Key: "_id", Value: oid}}, bson.D{{Key: "$set", Value: bson.D{{Key: "result", Value: d}}}})
	if err != nil {
		return fmt.Errorf("update toeic result: %w", err)
	}
	if out.MatchedCount == 0 {
		return toeic.ErrNotFound
	}
	return nil
}

// --- toeic_mistakes ---

type toeicMistakeDoc struct {
	UserID      bson.ObjectID `bson:"userId"`
	QuestionID  string        `bson:"questionId"`
	GroupKey    string        `bson:"groupKey"`
	Part        int           `bson:"part"`
	Tags        []string      `bson:"tags"`
	Streak      int           `bson:"streak"`
	LastWrongAt time.Time     `bson:"lastWrongAt"`
	AddedAt     time.Time     `bson:"addedAt"`
}

// ToeicMistakes implements toeic.MistakeRepository on "toeic_mistakes"; (userId, questionId) is unique.
type ToeicMistakes struct{ coll *mongo.Collection }

// NewToeicMistakes returns the TOEIC mistakes repository on db.
func NewToeicMistakes(db *mongo.Database) *ToeicMistakes {
	return &ToeicMistakes{coll: db.Collection(export.CollToeicMistakes)}
}

var _ toeic.MistakeRepository = (*ToeicMistakes)(nil)

func (d toeicMistakeDoc) mistake() toeic.Mistake {
	return toeic.Mistake{UserID: d.UserID.Hex(), QuestionID: d.QuestionID, GroupKey: d.GroupKey, Part: toeic.Part(d.Part), Tags: d.Tags,
		Streak: d.Streak, LastWrongAt: d.LastWrongAt.UTC(), AddedAt: d.AddedAt.UTC()}
}

// List returns the user's entries, oldest wrong first.
func (r *ToeicMistakes) List(ctx context.Context, userID string, part toeic.Part) ([]toeic.Mistake, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, fmt.Errorf("toeic mistake user id: %w", err)
	}
	filter := bson.D{{Key: "userId", Value: uid}}
	if part != 0 {
		filter = append(filter, bson.E{Key: "part", Value: int(part)})
	}
	cur, err := r.coll.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "lastWrongAt", Value: 1}, {Key: "questionId", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("find toeic mistakes: %w", err)
	}
	var docs []toeicMistakeDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("read toeic mistakes: %w", err)
	}
	out := make([]toeic.Mistake, len(docs))
	for i, d := range docs {
		out[i] = d.mistake()
	}
	return out, nil
}

// Get returns an entry.
func (r *ToeicMistakes) Get(ctx context.Context, userID, qid string) (toeic.Mistake, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return toeic.Mistake{}, toeic.ErrNotFound
	}
	var d toeicMistakeDoc
	err = r.coll.FindOne(ctx, bson.D{{Key: "userId", Value: uid}, {Key: "questionId", Value: qid}}).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return toeic.Mistake{}, toeic.ErrNotFound
	}
	if err != nil {
		return toeic.Mistake{}, fmt.Errorf("find toeic mistake: %w", err)
	}
	return d.mistake(), nil
}

// Upsert stores an entry.
func (r *ToeicMistakes) Upsert(ctx context.Context, m toeic.Mistake) error {
	uid, err := bson.ObjectIDFromHex(m.UserID)
	if err != nil {
		return fmt.Errorf("toeic mistake user id: %w", err)
	}
	tags := m.Tags
	if tags == nil {
		tags = []string{}
	}
	d := toeicMistakeDoc{UserID: uid, QuestionID: m.QuestionID, GroupKey: m.GroupKey, Part: int(m.Part), Tags: tags, Streak: m.Streak,
		LastWrongAt: m.LastWrongAt.UTC(), AddedAt: m.AddedAt.UTC()}
	_, err = r.coll.ReplaceOne(ctx, bson.D{{Key: "userId", Value: uid}, {Key: "questionId", Value: m.QuestionID}}, d,
		options.Replace().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("save toeic mistake: %w", err)
	}
	return nil
}

// Delete removes an entry.
func (r *ToeicMistakes) Delete(ctx context.Context, userID, qid string) error {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil
	}
	if _, err := r.coll.DeleteOne(ctx, bson.D{{Key: "userId", Value: uid}, {Key: "questionId", Value: qid}}); err != nil {
		return fmt.Errorf("delete toeic mistake: %w", err)
	}
	return nil
}

// Count returns how many entries the user has.
func (r *ToeicMistakes) Count(ctx context.Context, userID string) (int, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return 0, fmt.Errorf("toeic mistake user id: %w", err)
	}
	n, err := r.coll.CountDocuments(ctx, bson.D{{Key: "userId", Value: uid}})
	if err != nil {
		return 0, fmt.Errorf("count toeic mistakes: %w", err)
	}
	return int(n), nil
}

// --- toeic_goals ---

type toeicGoalDoc struct {
	UserID    bson.ObjectID `bson:"userId"`
	Target    int           `bson:"target"`
	ExamDate  string        `bson:"examDate"`
	UpdatedAt time.Time     `bson:"updatedAt"`
}

// ToeicGoals implements toeic.GoalRepository on "toeic_goals"; userId is unique.
type ToeicGoals struct{ coll *mongo.Collection }

// NewToeicGoals returns the TOEIC goals repository on db.
func NewToeicGoals(db *mongo.Database) *ToeicGoals {
	return &ToeicGoals{coll: db.Collection(export.CollToeicGoals)}
}

var _ toeic.GoalRepository = (*ToeicGoals)(nil)

// Get returns the user's goal.
func (r *ToeicGoals) Get(ctx context.Context, userID string) (toeic.Goal, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return toeic.Goal{}, toeic.ErrNotFound
	}
	var d toeicGoalDoc
	err = r.coll.FindOne(ctx, bson.D{{Key: "userId", Value: uid}}).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return toeic.Goal{}, toeic.ErrNotFound
	}
	if err != nil {
		return toeic.Goal{}, fmt.Errorf("find toeic goal: %w", err)
	}
	return toeic.Goal{UserID: userID, Target: d.Target, ExamDate: d.ExamDate, UpdatedAt: d.UpdatedAt.UTC()}, nil
}

// Save stores the user's goal.
func (r *ToeicGoals) Save(ctx context.Context, g toeic.Goal) error {
	uid, err := bson.ObjectIDFromHex(g.UserID)
	if err != nil {
		return fmt.Errorf("toeic goal user id: %w", err)
	}
	d := toeicGoalDoc{UserID: uid, Target: g.Target, ExamDate: g.ExamDate, UpdatedAt: g.UpdatedAt.UTC()}
	if _, err := r.coll.ReplaceOne(ctx, bson.D{{Key: "userId", Value: uid}}, d, options.Replace().SetUpsert(true)); err != nil {
		return fmt.Errorf("save toeic goal: %w", err)
	}
	return nil
}

// Delete removes the user's goal.
func (r *ToeicGoals) Delete(ctx context.Context, userID string) error {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil
	}
	if _, err := r.coll.DeleteOne(ctx, bson.D{{Key: "userId", Value: uid}}); err != nil {
		return fmt.Errorf("delete toeic goal: %w", err)
	}
	return nil
}
