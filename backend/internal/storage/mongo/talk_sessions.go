package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/luongtran/luna/backend/internal/talk"
)

// TalkSessions implements talk.Repository on "talk_sessions" (added 2026-10-20): one document per
// conversation with the AI, its turns and review embedded.
type TalkSessions struct {
	coll *mongo.Collection
}

// NewTalkSessions returns the TalkSessions repository on db.
func NewTalkSessions(db *mongo.Database) *TalkSessions {
	return &TalkSessions{coll: db.Collection("talk_sessions")}
}

var _ talk.Repository = (*TalkSessions)(nil)

type talkTurnDoc struct {
	Role   string    `bson:"role"`
	Text   string    `bson:"text"`
	TextVi string    `bson:"textVi,omitempty"`
	Spoken bool      `bson:"spoken,omitempty"`
	At     time.Time `bson:"at"`
}

type talkGoodDoc struct {
	Said   string `bson:"said"`
	NoteVi string `bson:"noteVi"`
}

type talkFixDoc struct {
	Said   string `bson:"said"`
	Better string `bson:"better"`
	NoteVi string `bson:"noteVi"`
}

type talkReviewDoc struct {
	OverallVi   string        `bson:"overallVi"`
	Good        []talkGoodDoc `bson:"good"`
	Fixes       []talkFixDoc  `bson:"fixes"`
	WordsUsed   []string      `bson:"wordsUsed"`
	WordsUnused []string      `bson:"wordsUnused"`
}

type talkSessionDoc struct {
	ID         bson.ObjectID  `bson:"_id,omitempty"`
	UserID     bson.ObjectID  `bson:"userId"`
	TopicID    string         `bson:"topicId"`
	TopicName  string         `bson:"topicName"`
	Level      string         `bson:"level"`
	Words      []string       `bson:"words"`
	Grammar    []string       `bson:"grammar"`
	ScenarioVi string         `bson:"scenarioVi"`
	Turns      []talkTurnDoc  `bson:"turns"`
	Hint       string         `bson:"hint,omitempty"`
	HintVi     string         `bson:"hintVi,omitempty"`
	Status     string         `bson:"status"`
	Review     *talkReviewDoc `bson:"review,omitempty"`
	CreatedAt  time.Time      `bson:"createdAt"`
	UpdatedAt  time.Time      `bson:"updatedAt"`
	EndedAt    *time.Time     `bson:"endedAt,omitempty"`
}

func toTalkTurnDocs(turns []talk.Turn) []talkTurnDoc {
	out := make([]talkTurnDoc, len(turns))
	for i, t := range turns {
		out[i] = talkTurnDoc{Role: string(t.Role), Text: t.Text, TextVi: t.TextVi, Spoken: t.Spoken, At: t.At.UTC()}
	}
	return out
}

func toTalkReviewDoc(r *talk.Review) *talkReviewDoc {
	if r == nil {
		return nil
	}
	d := &talkReviewDoc{
		OverallVi: r.OverallVi, Good: make([]talkGoodDoc, len(r.Good)), Fixes: make([]talkFixDoc, len(r.Fixes)),
		WordsUsed: r.WordsUsed, WordsUnused: r.WordsUnused,
	}
	for i, g := range r.Good {
		d.Good[i] = talkGoodDoc(g)
	}
	for i, f := range r.Fixes {
		d.Fixes[i] = talkFixDoc(f)
	}
	return d
}

func endedAtDoc(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	u := t.UTC()
	return &u
}

func (d talkSessionDoc) toSession() talk.Session {
	s := talk.Session{
		ID: d.ID.Hex(), UserID: d.UserID.Hex(), TopicID: d.TopicID, TopicName: d.TopicName, Level: d.Level,
		Words: d.Words, Grammar: d.Grammar, ScenarioVi: d.ScenarioVi, Turns: make([]talk.Turn, len(d.Turns)),
		Hint: d.Hint, HintVi: d.HintVi, Status: talk.Status(d.Status), CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
	}
	for i, t := range d.Turns {
		s.Turns[i] = talk.Turn{Role: talk.Role(t.Role), Text: t.Text, TextVi: t.TextVi, Spoken: t.Spoken, At: t.At}
	}
	if d.EndedAt != nil {
		s.EndedAt = *d.EndedAt
	}
	if r := d.Review; r != nil {
		rv := &talk.Review{
			OverallVi: r.OverallVi, Good: make([]talk.Good, len(r.Good)), Fixes: make([]talk.Fix, len(r.Fixes)),
			WordsUsed: r.WordsUsed, WordsUnused: r.WordsUnused,
		}
		for i, g := range r.Good {
			rv.Good[i] = talk.Good(g)
		}
		for i, f := range r.Fixes {
			rv.Fixes[i] = talk.Fix(f)
		}
		s.Review = rv
	}
	return s
}

// Create stores a new session.
func (r *TalkSessions) Create(ctx context.Context, s talk.Session) (talk.Session, error) {
	uid, err := bson.ObjectIDFromHex(s.UserID)
	if err != nil {
		return talk.Session{}, fmt.Errorf("create talk session: bad user id %q", s.UserID)
	}
	d := talkSessionDoc{
		ID: bson.NewObjectID(), UserID: uid, TopicID: s.TopicID, TopicName: s.TopicName, Level: s.Level,
		Words: s.Words, Grammar: s.Grammar, ScenarioVi: s.ScenarioVi, Turns: toTalkTurnDocs(s.Turns),
		Hint: s.Hint, HintVi: s.HintVi, Status: string(s.Status), Review: toTalkReviewDoc(s.Review),
		CreatedAt: s.CreatedAt.UTC(), UpdatedAt: s.UpdatedAt.UTC(), EndedAt: endedAtDoc(s.EndedAt),
	}
	if _, err := r.coll.InsertOne(ctx, d); err != nil {
		return talk.Session{}, fmt.Errorf("create talk session: %w", err)
	}
	s.ID = d.ID.Hex()
	return s, nil
}

// Get returns one session.
func (r *TalkSessions) Get(ctx context.Context, id string) (talk.Session, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return talk.Session{}, talk.ErrNotFound
	}
	var d talkSessionDoc
	err = r.coll.FindOne(ctx, bson.D{{Key: "_id", Value: oid}}).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return talk.Session{}, talk.ErrNotFound
	}
	if err != nil {
		return talk.Session{}, fmt.Errorf("get talk session: %w", err)
	}
	return d.toSession(), nil
}

// Save replaces the parts of a session that change while talking.
func (r *TalkSessions) Save(ctx context.Context, s talk.Session) error {
	oid, err := bson.ObjectIDFromHex(s.ID)
	if err != nil {
		return talk.ErrNotFound
	}
	set := bson.D{
		{Key: "turns", Value: toTalkTurnDocs(s.Turns)},
		{Key: "hint", Value: s.Hint},
		{Key: "hintVi", Value: s.HintVi},
		{Key: "status", Value: string(s.Status)},
		{Key: "review", Value: toTalkReviewDoc(s.Review)},
		{Key: "updatedAt", Value: s.UpdatedAt.UTC()},
		{Key: "endedAt", Value: endedAtDoc(s.EndedAt)},
	}
	res, err := r.coll.UpdateOne(ctx, bson.D{{Key: "_id", Value: oid}}, bson.D{{Key: "$set", Value: set}})
	if err != nil {
		return fmt.Errorf("save talk session: %w", err)
	}
	if res.MatchedCount == 0 {
		return talk.ErrNotFound
	}
	return nil
}

// List returns the user's sessions, newest first.
func (r *TalkSessions) List(ctx context.Context, userID string, limit int) ([]talk.Session, error) {
	uid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return []talk.Session{}, nil
	}
	cur, err := r.coll.Find(ctx, bson.D{{Key: "userId", Value: uid}},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(int64(limit)))
	if err != nil {
		return nil, fmt.Errorf("list talk sessions: %w", err)
	}
	var docs []talkSessionDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("decode talk sessions: %w", err)
	}
	out := make([]talk.Session, len(docs))
	for i, d := range docs {
		out[i] = d.toSession()
	}
	return out, nil
}
