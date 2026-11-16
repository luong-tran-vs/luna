package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/luongtran/luna/backend/internal/export"
	"github.com/luongtran/luna/backend/internal/games"
)

// collGrammarSpotItems keeps the extra "Bắn lỗi" sentences (F26); not learner data.
const collGrammarSpotItems = "grammar_spot_items"

// --- game_bests ---

type gameBestDoc struct {
	UserID bson.ObjectID `bson:"userId"`
	Game   string        `bson:"game"`
	Score  int           `bson:"score"`
	At     time.Time     `bson:"at"`
}

// GameBests implements games.BestRepository on "game_bests"; the unique (userId, game) index keeps
// one document per game.
type GameBests struct {
	coll *mongo.Collection
}

// NewGameBests returns the personal bests repository on db.
func NewGameBests(db *mongo.Database) *GameBests {
	return &GameBests{coll: db.Collection(export.CollGameBests)}
}

var _ games.BestRepository = (*GameBests)(nil)

// List returns every best of the user.
func (r *GameBests) List(ctx context.Context, userID string) ([]games.Best, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, fmt.Errorf("game best user id: %w", err)
	}
	cur, err := r.coll.Find(ctx, bson.D{{Key: "userId", Value: uid}})
	if err != nil {
		return nil, fmt.Errorf("find game bests: %w", err)
	}
	var docs []gameBestDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("read game bests: %w", err)
	}
	out := make([]games.Best, len(docs))
	for i, d := range docs {
		out[i] = games.Best{UserID: userID, Game: games.Game(d.Game), Score: d.Score, At: d.At.UTC()}
	}
	return out, nil
}

// Raise stores score when it beats the best: an update that only matches a lower score, or an
// insert when the game has no best yet. A concurrent insert of the same best loses on the unique
// index and is retried once as an update.
func (r *GameBests) Raise(ctx context.Context, userID string, game games.Game, score int, at time.Time) (bool, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return false, fmt.Errorf("game best user id: %w", err)
	}
	for attempt := 0; attempt < 2; attempt++ {
		res, err := r.coll.UpdateOne(ctx,
			bson.D{{Key: "userId", Value: uid}, {Key: "game", Value: string(game)}, {Key: "score", Value: bson.D{{Key: "$lt", Value: score}}}},
			bson.D{{Key: "$set", Value: bson.D{{Key: "score", Value: score}, {Key: "at", Value: at.UTC()}}}})
		if err != nil {
			return false, fmt.Errorf("raise game best: %w", err)
		}
		if res.MatchedCount > 0 {
			return true, nil
		}
		n, err := r.coll.CountDocuments(ctx, bson.D{{Key: "userId", Value: uid}, {Key: "game", Value: string(game)}})
		if err != nil {
			return false, fmt.Errorf("find game best: %w", err)
		}
		if n > 0 {
			return false, nil
		}
		_, err = r.coll.InsertOne(ctx, gameBestDoc{UserID: uid, Game: string(game), Score: score, At: at.UTC()})
		if err == nil {
			return true, nil
		}
		if !mongo.IsDuplicateKeyError(err) {
			return false, fmt.Errorf("insert game best: %w", err)
		}
	}
	return false, nil
}

// --- daily_challenges ---

type dailyAnswerDoc struct {
	Kind  string `bson:"kind"`
	Right bool   `bson:"right"`
}

type dailyChallengeDoc struct {
	UserID     bson.ObjectID    `bson:"userId"`
	Day        string           `bson:"day"`
	Score      int              `bson:"score"`
	Answers    []dailyAnswerDoc `bson:"answers"`
	FinishedAt time.Time        `bson:"finishedAt"`
}

// DailyChallenges implements games.DailyRepository on "daily_challenges"; the unique (userId, day)
// index keeps only the first result of a day.
type DailyChallenges struct {
	coll *mongo.Collection
}

// NewDailyChallenges returns the daily challenge repository on db.
func NewDailyChallenges(db *mongo.Database) *DailyChallenges {
	return &DailyChallenges{coll: db.Collection(export.CollDailyChallenges)}
}

var _ games.DailyRepository = (*DailyChallenges)(nil)

func (d dailyChallengeDoc) result(userID string) games.DailyResult {
	out := games.DailyResult{UserID: userID, Day: d.Day, Score: d.Score, FinishedAt: d.FinishedAt.UTC(), Answers: make([]games.DailyAnswer, len(d.Answers))}
	for i, a := range d.Answers {
		out.Answers[i] = games.DailyAnswer{Kind: a.Kind, Right: a.Right}
	}
	return out
}

// Get returns the user's result of day.
func (r *DailyChallenges) Get(ctx context.Context, userID, day string) (games.DailyResult, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return games.DailyResult{}, games.ErrNotFound
	}
	var d dailyChallengeDoc
	err = r.coll.FindOne(ctx, bson.D{{Key: "userId", Value: uid}, {Key: "day", Value: day}}).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return games.DailyResult{}, games.ErrNotFound
	}
	if err != nil {
		return games.DailyResult{}, fmt.Errorf("find daily challenge: %w", err)
	}
	return d.result(userID), nil
}

// Save stores the first result of a day.
func (r *DailyChallenges) Save(ctx context.Context, res games.DailyResult) error {
	uid, err := bson.ObjectIDFromHex(res.UserID)
	if err != nil {
		return fmt.Errorf("daily challenge user id: %w", err)
	}
	doc := dailyChallengeDoc{UserID: uid, Day: res.Day, Score: res.Score, FinishedAt: res.FinishedAt.UTC(), Answers: make([]dailyAnswerDoc, len(res.Answers))}
	for i, a := range res.Answers {
		doc.Answers[i] = dailyAnswerDoc{Kind: a.Kind, Right: a.Right}
	}
	_, err = r.coll.InsertOne(ctx, doc)
	if mongo.IsDuplicateKeyError(err) {
		return games.ErrExists
	}
	if err != nil {
		return fmt.Errorf("insert daily challenge: %w", err)
	}
	return nil
}

// List returns the user's results, newest day first.
func (r *DailyChallenges) List(ctx context.Context, userID string) ([]games.DailyResult, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, fmt.Errorf("daily challenge user id: %w", err)
	}
	cur, err := r.coll.Find(ctx, bson.D{{Key: "userId", Value: uid}}, options.Find().SetSort(bson.D{{Key: "day", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("find daily challenges: %w", err)
	}
	var docs []dailyChallengeDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("read daily challenges: %w", err)
	}
	out := make([]games.DailyResult, len(docs))
	for i, d := range docs {
		out[i] = d.result(userID)
	}
	return out, nil
}

// --- grammar_spot_items ---

type spotItemDoc struct {
	ID          bson.ObjectID `bson:"_id"`
	PointID     string        `bson:"pointId"`
	Level       string        `bson:"level"`
	Sentence    string        `bson:"sentence"`
	Wrong       string        `bson:"wrong"`
	Right       string        `bson:"right"`
	Distractors []string      `bson:"distractors"`
	RuleVi      string        `bson:"ruleVi"`
	Status      string        `bson:"status"`
	CreatedAt   time.Time     `bson:"createdAt"`
	UpdatedAt   time.Time     `bson:"updatedAt"`
}

func (d spotItemDoc) item() games.SpotItem {
	return games.SpotItem{
		ID: d.ID.Hex(), PointID: d.PointID, Level: d.Level, Sentence: d.Sentence, Wrong: d.Wrong, Right: d.Right,
		Distractors: d.Distractors, RuleVi: d.RuleVi, Status: games.SpotStatus(d.Status),
		CreatedAt: d.CreatedAt.UTC(), UpdatedAt: d.UpdatedAt.UTC(),
	}
}

func spotDoc(id bson.ObjectID, it games.SpotItem) spotItemDoc {
	return spotItemDoc{
		ID: id, PointID: it.PointID, Level: it.Level, Sentence: it.Sentence, Wrong: it.Wrong, Right: it.Right,
		Distractors: it.Distractors, RuleVi: it.RuleVi, Status: string(it.Status),
		CreatedAt: it.CreatedAt.UTC(), UpdatedAt: it.UpdatedAt.UTC(),
	}
}

// GrammarSpotItems implements games.SpotRepository on "grammar_spot_items".
type GrammarSpotItems struct {
	coll *mongo.Collection
}

// NewGrammarSpotItems returns the extra "Bắn lỗi" sentences repository on db.
func NewGrammarSpotItems(db *mongo.Database) *GrammarSpotItems {
	return &GrammarSpotItems{coll: db.Collection(collGrammarSpotItems)}
}

var _ games.SpotRepository = (*GrammarSpotItems)(nil)

func (r *GrammarSpotItems) find(ctx context.Context, filter bson.D) ([]games.SpotItem, error) {
	cur, err := r.coll.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}, {Key: "_id", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("find spot items: %w", err)
	}
	var docs []spotItemDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("read spot items: %w", err)
	}
	out := make([]games.SpotItem, len(docs))
	for i, d := range docs {
		out[i] = d.item()
	}
	return out, nil
}

// ListByPoint returns the items of a point, oldest first.
func (r *GrammarSpotItems) ListByPoint(ctx context.Context, pointID string) ([]games.SpotItem, error) {
	return r.find(ctx, bson.D{{Key: "pointId", Value: pointID}})
}

// Approved returns the approved items of the points (all when pointIDs is nil).
func (r *GrammarSpotItems) Approved(ctx context.Context, pointIDs []string) ([]games.SpotItem, error) {
	filter := bson.D{{Key: "status", Value: string(games.SpotApproved)}}
	if pointIDs != nil {
		filter = append(filter, bson.E{Key: "pointId", Value: bson.D{{Key: "$in", Value: pointIDs}}})
	}
	return r.find(ctx, filter)
}

// Counts returns the approved and draft items per point.
func (r *GrammarSpotItems) Counts(ctx context.Context) (map[string]games.SpotCount, error) {
	cur, err := r.coll.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{{Key: "point", Value: "$pointId"}, {Key: "status", Value: "$status"}}},
			{Key: "n", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
	})
	if err != nil {
		return nil, fmt.Errorf("count spot items: %w", err)
	}
	var rows []struct {
		ID struct {
			Point  string `bson:"point"`
			Status string `bson:"status"`
		} `bson:"_id"`
		N int `bson:"n"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, fmt.Errorf("read spot item counts: %w", err)
	}
	out := map[string]games.SpotCount{}
	for _, row := range rows {
		c := out[row.ID.Point]
		if row.ID.Status == string(games.SpotApproved) {
			c.Approved += row.N
		} else {
			c.Draft += row.N
		}
		out[row.ID.Point] = c
	}
	return out, nil
}

// Create stores the items with new ids.
func (r *GrammarSpotItems) Create(ctx context.Context, items []games.SpotItem) ([]games.SpotItem, error) {
	if len(items) == 0 {
		return nil, nil
	}
	docs := make([]any, len(items))
	out := make([]games.SpotItem, len(items))
	for i, it := range items {
		id := bson.NewObjectID()
		docs[i] = spotDoc(id, it)
		it.ID = id.Hex()
		out[i] = it
	}
	if _, err := r.coll.InsertMany(ctx, docs); err != nil {
		return nil, fmt.Errorf("insert spot items: %w", err)
	}
	return out, nil
}

// Get returns the item with id.
func (r *GrammarSpotItems) Get(ctx context.Context, id string) (games.SpotItem, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return games.SpotItem{}, games.ErrNotFound
	}
	var d spotItemDoc
	err = r.coll.FindOne(ctx, bson.D{{Key: "_id", Value: oid}}).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return games.SpotItem{}, games.ErrNotFound
	}
	if err != nil {
		return games.SpotItem{}, fmt.Errorf("find spot item: %w", err)
	}
	return d.item(), nil
}

// Update replaces the item.
func (r *GrammarSpotItems) Update(ctx context.Context, it games.SpotItem) error {
	oid, err := bson.ObjectIDFromHex(it.ID)
	if err != nil {
		return games.ErrNotFound
	}
	res, err := r.coll.ReplaceOne(ctx, bson.D{{Key: "_id", Value: oid}}, spotDoc(oid, it))
	if err != nil {
		return fmt.Errorf("update spot item: %w", err)
	}
	if res.MatchedCount == 0 {
		return games.ErrNotFound
	}
	return nil
}

// Delete removes the item.
func (r *GrammarSpotItems) Delete(ctx context.Context, id string) error {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return games.ErrNotFound
	}
	res, err := r.coll.DeleteOne(ctx, bson.D{{Key: "_id", Value: oid}})
	if err != nil {
		return fmt.Errorf("delete spot item: %w", err)
	}
	if res.DeletedCount == 0 {
		return games.ErrNotFound
	}
	return nil
}

// ApproveDrafts approves every draft of the points (all when pointIDs is nil).
func (r *GrammarSpotItems) ApproveDrafts(ctx context.Context, pointIDs []string, now time.Time) (int, error) {
	filter := bson.D{{Key: "status", Value: string(games.SpotDraft)}}
	if pointIDs != nil {
		filter = append(filter, bson.E{Key: "pointId", Value: bson.D{{Key: "$in", Value: pointIDs}}})
	}
	res, err := r.coll.UpdateMany(ctx, filter, bson.D{{Key: "$set", Value: bson.D{
		{Key: "status", Value: string(games.SpotApproved)}, {Key: "updatedAt", Value: now.UTC()},
	}}})
	if err != nil {
		return 0, fmt.Errorf("approve spot drafts: %w", err)
	}
	return int(res.ModifiedCount), nil
}
