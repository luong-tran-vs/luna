package mongo

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/luongtran/luna/backend/internal/vocab"
)

// ScheduleDoc holds the FSRS fields inline in a card; they are absent on cards saved before F5.
type ScheduleDoc struct {
	Due           time.Time `bson:"due,omitempty"`
	Stability     float64   `bson:"stability,omitempty"`
	Difficulty    float64   `bson:"difficulty,omitempty"`
	ElapsedDays   int64     `bson:"elapsedDays,omitempty"`
	ScheduledDays int64     `bson:"scheduledDays,omitempty"`
	Reps          int64     `bson:"reps,omitempty"`
	Lapses        int64     `bson:"lapses,omitempty"`
	State         int       `bson:"state,omitempty"`
	LastReview    time.Time `bson:"lastReview,omitempty"`
}

func fromSchedule(s vocab.Schedule) ScheduleDoc {
	return ScheduleDoc{
		Due: s.Due.UTC(), Stability: s.Stability, Difficulty: s.Difficulty,
		ElapsedDays: int64(s.ElapsedDays), ScheduledDays: int64(s.ScheduledDays), //nolint:gosec // day counts are small
		Reps: int64(s.Reps), Lapses: int64(s.Lapses), //nolint:gosec // review counts are small
		State: int(s.State), LastReview: s.LastReview.UTC(),
	}
}

func (d ScheduleDoc) toSchedule() vocab.Schedule {
	return vocab.Schedule{
		Due: d.Due, Stability: d.Stability, Difficulty: d.Difficulty,
		ElapsedDays: uint64(d.ElapsedDays), ScheduledDays: uint64(d.ScheduledDays), //nolint:gosec // stored from uint64
		Reps: uint64(d.Reps), Lapses: uint64(d.Lapses), //nolint:gosec // stored from uint64
		State: vocab.State(d.State), LastReview: d.LastReview,
	}
}

// set is the $set document writing every schedule field (zero values included).
func (d ScheduleDoc) set() bson.D {
	return bson.D{
		{Key: "due", Value: d.Due},
		{Key: "stability", Value: d.Stability},
		{Key: "difficulty", Value: d.Difficulty},
		{Key: "elapsedDays", Value: d.ElapsedDays},
		{Key: "scheduledDays", Value: d.ScheduledDays},
		{Key: "reps", Value: d.Reps},
		{Key: "lapses", Value: d.Lapses},
		{Key: "state", Value: d.State},
		{Key: "lastReview", Value: d.LastReview},
	}
}

type cardDoc struct {
	ID              bson.ObjectID `bson:"_id,omitempty"`
	UserID          bson.ObjectID `bson:"userId"`
	Text            string        `bson:"text"`
	Lemma           string        `bson:"lemma"`
	IPA             string        `bson:"ipa"`
	MeaningVi       string        `bson:"meaningVi"`
	ContextSentence string        `bson:"contextSentence"`
	// LessonID is absent on cards added by hand.
	LessonID    bson.ObjectID `bson:"lessonId,omitempty"`
	Source      string        `bson:"source"`
	CreatedAt   time.Time     `bson:"createdAt"`
	ScheduleDoc `bson:",inline"`
}

func (d cardDoc) toCard() vocab.Card {
	c := vocab.Card{
		ID: d.ID.Hex(), UserID: d.UserID.Hex(), Text: d.Text, Lemma: d.Lemma, IPA: d.IPA,
		MeaningVi: d.MeaningVi, ContextSentence: d.ContextSentence,
		Source: vocab.Source(d.Source), CreatedAt: d.CreatedAt, Schedule: d.toSchedule(),
	}
	if !d.LessonID.IsZero() {
		c.LessonID = d.LessonID.Hex()
	}
	return c
}

// Cards implements vocab.Repository on the "cards" collection.
type Cards struct {
	coll *mongo.Collection
}

// NewCards returns a card repository on db.
func NewCards(db *mongo.Database) *Cards {
	return &Cards{coll: db.Collection("cards")}
}

var _ vocab.Repository = (*Cards)(nil)

// Create inserts c; the unique (userId, lemma) index turns duplicates into vocab.ErrExists.
func (r *Cards) Create(ctx context.Context, c vocab.Card) (vocab.Card, error) {
	uid, err := bson.ObjectIDFromHex(c.UserID)
	if err != nil {
		return vocab.Card{}, fmt.Errorf("card user id: %w", err)
	}
	d := cardDoc{
		ID: bson.NewObjectID(), UserID: uid, Text: c.Text, Lemma: c.Lemma, IPA: c.IPA,
		MeaningVi: c.MeaningVi, ContextSentence: c.ContextSentence,
		Source: string(c.Source), CreatedAt: c.CreatedAt.UTC(), ScheduleDoc: fromSchedule(c.Schedule),
	}
	if c.LessonID != "" {
		if d.LessonID, err = bson.ObjectIDFromHex(c.LessonID); err != nil {
			return vocab.Card{}, fmt.Errorf("card lesson id: %w", err)
		}
	}
	if _, err := r.coll.InsertOne(ctx, d); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return vocab.Card{}, vocab.ErrExists
		}
		return vocab.Card{}, fmt.Errorf("insert card: %w", err)
	}
	return d.toCard(), nil
}

// ids parses the user and card ids; a malformed card id is simply not found.
func ids(userID, cardID string) (uid, cid bson.ObjectID, err error) {
	if uid, err = bson.ObjectIDFromHex(userID); err != nil {
		return uid, cid, fmt.Errorf("card user id: %w", err)
	}
	if cid, err = bson.ObjectIDFromHex(cardID); err != nil {
		return uid, cid, vocab.ErrNotFound
	}
	return uid, cid, nil
}

func (r *Cards) findOne(ctx context.Context, filter bson.D) (vocab.Card, error) {
	var d cardDoc
	err := r.coll.FindOne(ctx, filter).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return vocab.Card{}, vocab.ErrNotFound
	}
	if err != nil {
		return vocab.Card{}, fmt.Errorf("find card: %w", err)
	}
	return d.toCard(), nil
}

func (r *Cards) findMany(ctx context.Context, filter bson.D, opts *options.FindOptionsBuilder) ([]vocab.Card, error) {
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("find cards: %w", err)
	}
	var docs []cardDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("read cards: %w", err)
	}
	out := make([]vocab.Card, len(docs))
	for i, d := range docs {
		out[i] = d.toCard()
	}
	return out, nil
}

// FindByLemma returns the user's card with lemma.
func (r *Cards) FindByLemma(ctx context.Context, userID, lemma string) (vocab.Card, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return vocab.Card{}, vocab.ErrNotFound
	}
	return r.findOne(ctx, bson.D{{Key: "userId", Value: uid}, {Key: "lemma", Value: lemma}})
}

// Words lists the user's lemmas and texts, oldest first.
func (r *Cards) Words(ctx context.Context, userID string) ([]vocab.WordRef, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, fmt.Errorf("card user id: %w", err)
	}
	cards, err := r.findMany(ctx, bson.D{{Key: "userId", Value: uid}},
		options.Find().SetProjection(bson.D{{Key: "lemma", Value: 1}, {Key: "text", Value: 1}}).SetSort(bson.D{{Key: "createdAt", Value: 1}}))
	if err != nil {
		return nil, err
	}
	out := make([]vocab.WordRef, len(cards))
	for i, c := range cards {
		out[i] = vocab.WordRef{Lemma: c.Lemma, Text: c.Text}
	}
	return out, nil
}

// Get returns the user's card with id.
func (r *Cards) Get(ctx context.Context, userID, id string) (vocab.Card, error) {
	uid, cid, err := ids(userID, id)
	if err != nil {
		return vocab.Card{}, err
	}
	return r.findOne(ctx, bson.D{{Key: "_id", Value: cid}, {Key: "userId", Value: uid}})
}

// List returns one page of the notebook, newest first.
func (r *Cards) List(ctx context.Context, userID string, q vocab.ListQuery, limit, skip int) ([]vocab.Card, bool, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, false, fmt.Errorf("card user id: %w", err)
	}
	filter := bson.D{{Key: "userId", Value: uid}}
	switch q.LessonID {
	case "":
	case vocab.ManualLesson:
		filter = append(filter, bson.E{Key: "lessonId", Value: bson.D{{Key: "$exists", Value: false}}})
	default:
		lid, err := bson.ObjectIDFromHex(q.LessonID)
		if err != nil {
			return nil, false, nil //nolint:nilerr // a malformed lesson id has no cards
		}
		filter = append(filter, bson.E{Key: "lessonId", Value: lid})
	}
	if q.Q != "" {
		re := bson.Regex{Pattern: regexp.QuoteMeta(q.Q), Options: "i"}
		filter = append(filter, bson.E{Key: "$or", Value: bson.A{
			bson.D{{Key: "text", Value: re}}, bson.D{{Key: "lemma", Value: re}},
		}})
	}
	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}, {Key: "_id", Value: -1}}).
		SetSkip(int64(skip)).SetLimit(int64(limit + 1))
	cards, err := r.findMany(ctx, filter, opts)
	if err != nil {
		return nil, false, err
	}
	if len(cards) > limit {
		return cards[:limit], true, nil
	}
	return cards, false, nil
}

// LessonCounts counts the user's cards per lesson; "" is cards added by hand.
func (r *Cards) LessonCounts(ctx context.Context, userID string) (map[string]int, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, fmt.Errorf("card user id: %w", err)
	}
	cur, err := r.coll.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "userId", Value: uid}}}},
		{{Key: "$group", Value: bson.D{{Key: "_id", Value: "$lessonId"}, {Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}}}}},
	})
	if err != nil {
		return nil, fmt.Errorf("count cards: %w", err)
	}
	var rows []struct {
		ID    *bson.ObjectID `bson:"_id"`
		Count int            `bson:"count"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, fmt.Errorf("read card counts: %w", err)
	}
	out := make(map[string]int, len(rows))
	for _, row := range rows {
		key := ""
		if row.ID != nil {
			key = row.ID.Hex()
		}
		out[key] += row.Count
	}
	return out, nil
}

// UpdateDetails sets meaning, IPA and example only; the schedule is untouched.
func (r *Cards) UpdateDetails(ctx context.Context, userID, id string, d vocab.Details) (vocab.Card, error) {
	uid, cid, err := ids(userID, id)
	if err != nil {
		return vocab.Card{}, err
	}
	set := bson.D{}
	if d.MeaningVi != nil {
		set = append(set, bson.E{Key: "meaningVi", Value: *d.MeaningVi})
	}
	if d.IPA != nil {
		set = append(set, bson.E{Key: "ipa", Value: *d.IPA})
	}
	if d.ContextSentence != nil {
		set = append(set, bson.E{Key: "contextSentence", Value: *d.ContextSentence})
	}
	filter := bson.D{{Key: "_id", Value: cid}, {Key: "userId", Value: uid}}
	if len(set) == 0 {
		return r.findOne(ctx, filter)
	}
	var doc cardDoc
	err = r.coll.FindOneAndUpdate(ctx, filter, bson.D{{Key: "$set", Value: set}},
		options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return vocab.Card{}, vocab.ErrNotFound
	}
	if err != nil {
		return vocab.Card{}, fmt.Errorf("update card: %w", err)
	}
	return doc.toCard(), nil
}

// UpdateSchedule writes s only if the card still has expectedReps reviews (missing = 0).
func (r *Cards) UpdateSchedule(ctx context.Context, userID, id string, expectedReps uint64, s vocab.Schedule) (bool, error) {
	uid, cid, err := ids(userID, id)
	if err != nil {
		return false, err
	}
	reps := bson.D{{Key: "reps", Value: int64(expectedReps)}} //nolint:gosec // review counts are small
	if expectedReps == 0 {
		reps = bson.D{{Key: "$or", Value: bson.A{
			reps, bson.D{{Key: "reps", Value: bson.D{{Key: "$exists", Value: false}}}},
		}}}
	}
	filter := append(bson.D{{Key: "_id", Value: cid}, {Key: "userId", Value: uid}}, reps...)
	res, err := r.coll.UpdateOne(ctx, filter, bson.D{{Key: "$set", Value: fromSchedule(s).set()}})
	if err != nil {
		return false, fmt.Errorf("update schedule: %w", err)
	}
	return res.MatchedCount == 1, nil
}

// dueFilter matches cards due at now: a stored due ≤ now, or no schedule and saved before today.
func dueFilter(uid bson.ObjectID, now, startOfToday time.Time) bson.D {
	return bson.D{{Key: "userId", Value: uid}, {Key: "$or", Value: bson.A{
		bson.D{{Key: "due", Value: bson.D{{Key: "$lte", Value: now}}}},
		bson.D{
			{Key: "due", Value: bson.D{{Key: "$exists", Value: false}}},
			{Key: "createdAt", Value: bson.D{{Key: "$lt", Value: startOfToday}}},
		},
	}}}
}

// Due returns the most overdue cards first (cards without a schedule sort first).
func (r *Cards) Due(ctx context.Context, userID string, now, startOfToday time.Time, limit int) ([]vocab.Card, int, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, 0, fmt.Errorf("card user id: %w", err)
	}
	filter := dueFilter(uid, now, startOfToday)
	total, err := r.coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("count due cards: %w", err)
	}
	cards, err := r.findMany(ctx, filter, options.Find().
		SetSort(bson.D{{Key: "due", Value: 1}, {Key: "createdAt", Value: 1}}).SetLimit(int64(limit)))
	if err != nil {
		return nil, 0, err
	}
	return cards, int(total), nil
}

// NextDue is the earliest due time after now.
func (r *Cards) NextDue(ctx context.Context, userID string, now, startOfToday time.Time) (time.Time, bool, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("card user id: %w", err)
	}
	var best time.Time
	var d cardDoc
	err = r.coll.FindOne(ctx, bson.D{{Key: "userId", Value: uid}, {Key: "due", Value: bson.D{{Key: "$gt", Value: now}}}},
		options.FindOne().SetSort(bson.D{{Key: "due", Value: 1}})).Decode(&d)
	switch {
	case err == nil:
		best = d.Due
	case !errors.Is(err, mongo.ErrNoDocuments):
		return time.Time{}, false, fmt.Errorf("find next due: %w", err)
	}
	// Cards saved today before F5 stored a schedule are due tomorrow.
	legacy, err := r.coll.CountDocuments(ctx, bson.D{
		{Key: "userId", Value: uid},
		{Key: "due", Value: bson.D{{Key: "$exists", Value: false}}},
		{Key: "createdAt", Value: bson.D{{Key: "$gte", Value: startOfToday}}},
	}, options.Count().SetLimit(1))
	if err != nil {
		return time.Time{}, false, fmt.Errorf("count new cards: %w", err)
	}
	if tomorrow := startOfToday.AddDate(0, 0, 1); legacy > 0 && (best.IsZero() || tomorrow.Before(best)) {
		best = tomorrow
	}
	return best, !best.IsZero(), nil
}

// Delete removes the user's card.
func (r *Cards) Delete(ctx context.Context, userID, id string) (bool, error) {
	uid, cid, err := ids(userID, id)
	if errors.Is(err, vocab.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	res, err := r.coll.DeleteOne(ctx, bson.D{{Key: "_id", Value: cid}, {Key: "userId", Value: uid}})
	if err != nil {
		return false, fmt.Errorf("delete card: %w", err)
	}
	return res.DeletedCount == 1, nil
}

// CountDue counts the user's cards due before `before`, and cards without a schedule saved
// before createdBefore.
func (r *Cards) CountDue(ctx context.Context, userID string, before, createdBefore time.Time) (int, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return 0, fmt.Errorf("card user id: %w", err)
	}
	n, err := r.coll.CountDocuments(ctx, bson.D{{Key: "userId", Value: uid}, {Key: "$or", Value: bson.A{
		bson.D{{Key: "due", Value: bson.D{{Key: "$lt", Value: before}}}},
		bson.D{
			{Key: "due", Value: bson.D{{Key: "$exists", Value: false}}},
			{Key: "createdAt", Value: bson.D{{Key: "$lt", Value: createdBefore}}},
		},
	}}})
	if err != nil {
		return 0, fmt.Errorf("count due cards: %w", err)
	}
	return int(n), nil
}

// Count is the number of the user's cards.
func (r *Cards) Count(ctx context.Context, userID string) (int, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return 0, fmt.Errorf("card user id: %w", err)
	}
	n, err := r.coll.CountDocuments(ctx, bson.D{{Key: "userId", Value: uid}})
	if err != nil {
		return 0, fmt.Errorf("count cards: %w", err)
	}
	return int(n), nil
}
