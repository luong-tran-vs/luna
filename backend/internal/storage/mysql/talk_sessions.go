package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/luongtran/luna/backend/internal/talk"
)

// TalkSessions implements talk.Repository on "talk_sessions" (added 2026-10-20): one row per
// conversation with the AI; what the session is about, its turns and review are one JSON body.
type TalkSessions struct {
	db *sql.DB
}

var _ talk.Repository = (*TalkSessions)(nil)

// NewTalkSessions returns the TalkSessions repository.
func NewTalkSessions(db *sql.DB) *TalkSessions { return &TalkSessions{db: db} }

// talkSessionsSchema is the DDL of this domain (see migrate.go).
func talkSessionsSchema() []string {
	return []string{`CREATE TABLE IF NOT EXISTS talk_sessions (
  id         CHAR(24)    NOT NULL PRIMARY KEY,
  user_id    CHAR(24)    NOT NULL,
  topic_id   VARCHAR(64) NOT NULL,
  status     VARCHAR(16) NOT NULL,
  body       JSON        NOT NULL,
  created_at DATETIME(6) NOT NULL,
  updated_at DATETIME(6) NOT NULL,
  ended_at   DATETIME(6) NULL,
  KEY idx_talk_sessions_user (user_id, created_at)
) ` + tableOptions}
}

type talkTurnJSON struct {
	Role     string    `json:"role"`
	Text     string    `json:"text"`
	TextVi   string    `json:"textVi,omitempty"`
	Spoken   bool      `json:"spoken,omitempty"`
	Reaction string    `json:"reaction,omitempty"`
	At       time.Time `json:"at"`
}

type talkGoodJSON struct {
	Said   string `json:"said"`
	NoteVi string `json:"noteVi"`
}

type talkHintJSON struct {
	Text string `json:"text"`
	Vi   string `json:"vi"`
}

type talkFixJSON struct {
	Said   string `json:"said"`
	Better string `json:"better"`
	NoteVi string `json:"noteVi"`
}

type talkReviewJSON struct {
	OverallVi   string         `json:"overallVi"`
	Good        []talkGoodJSON `json:"good"`
	Fixes       []talkFixJSON  `json:"fixes"`
	WordsUsed   []string       `json:"wordsUsed"`
	WordsUnused []string       `json:"wordsUnused"`
}

// talkBodyJSON is the body column.
type talkBodyJSON struct {
	TopicName  string          `json:"topicName"`
	Level      string          `json:"level"`
	Words      []string        `json:"words"`
	Grammar    []string        `json:"grammar"`
	ScenarioVi string          `json:"scenarioVi"`
	Turns      []talkTurnJSON  `json:"turns"`
	Hints      []talkHintJSON  `json:"hints,omitempty"`
	Review     *talkReviewJSON `json:"review,omitempty"`
}

func talkBody(s talk.Session) ([]byte, error) {
	b := talkBodyJSON{
		TopicName: s.TopicName, Level: s.Level, Words: s.Words, Grammar: s.Grammar, ScenarioVi: s.ScenarioVi,
		Turns: make([]talkTurnJSON, len(s.Turns)), Hints: make([]talkHintJSON, len(s.Hints)),
	}
	for i, t := range s.Turns {
		b.Turns[i] = talkTurnJSON{
			Role: string(t.Role), Text: t.Text, TextVi: t.TextVi, Spoken: t.Spoken, Reaction: t.Reaction, At: utc(t.At),
		}
	}
	for i, h := range s.Hints {
		b.Hints[i] = talkHintJSON(h)
	}
	if r := s.Review; r != nil {
		rj := &talkReviewJSON{
			OverallVi: r.OverallVi, Good: make([]talkGoodJSON, len(r.Good)), Fixes: make([]talkFixJSON, len(r.Fixes)),
			WordsUsed: r.WordsUsed, WordsUnused: r.WordsUnused,
		}
		for i, g := range r.Good {
			rj.Good[i] = talkGoodJSON(g)
		}
		for i, f := range r.Fixes {
			rj.Fixes[i] = talkFixJSON(f)
		}
		b.Review = rj
	}
	return toJSON(b)
}

const talkColumns = `id, user_id, topic_id, status, body, created_at, updated_at, ended_at`

func scanTalkSession(row interface{ Scan(...any) error }) (talk.Session, error) {
	var s talk.Session
	var status string
	var raw []byte
	var ended sql.NullTime
	if err := row.Scan(&s.ID, &s.UserID, &s.TopicID, &status, &raw, &s.CreatedAt, &s.UpdatedAt, &ended); err != nil {
		return talk.Session{}, err
	}
	var b talkBodyJSON
	if err := fromJSON(raw, &b); err != nil {
		return talk.Session{}, err
	}
	s.Status, s.EndedAt = talk.Status(status), timeOf(ended)
	s.CreatedAt, s.UpdatedAt = utc(s.CreatedAt), utc(s.UpdatedAt)
	s.TopicName, s.Level, s.Words, s.Grammar, s.ScenarioVi = b.TopicName, b.Level, b.Words, b.Grammar, b.ScenarioVi
	s.Hints = make([]talk.Hint, len(b.Hints))
	for i, h := range b.Hints {
		s.Hints[i] = talk.Hint(h)
	}
	s.Turns = make([]talk.Turn, len(b.Turns))
	for i, t := range b.Turns {
		s.Turns[i] = talk.Turn{
			Role: talk.Role(t.Role), Text: t.Text, TextVi: t.TextVi, Spoken: t.Spoken, Reaction: t.Reaction, At: t.At,
		}
	}
	if r := b.Review; r != nil {
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
	return s, nil
}

// Create stores a new session.
func (r *TalkSessions) Create(ctx context.Context, s talk.Session) (talk.Session, error) {
	body, err := talkBody(s)
	if err != nil {
		return talk.Session{}, err
	}
	s.ID = newID()
	_, err = r.db.ExecContext(ctx, `INSERT INTO talk_sessions (`+talkColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.UserID, s.TopicID, string(s.Status), string(body), utc(s.CreatedAt), utc(s.UpdatedAt), nullTime(s.EndedAt))
	if err != nil {
		return talk.Session{}, fmt.Errorf("mysql create talk session: %w", err)
	}
	return s, nil
}

// Get returns one session.
func (r *TalkSessions) Get(ctx context.Context, id string) (talk.Session, error) {
	s, err := scanTalkSession(r.db.QueryRowContext(ctx, `SELECT `+talkColumns+` FROM talk_sessions WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return talk.Session{}, talk.ErrNotFound
	}
	if err != nil {
		return talk.Session{}, fmt.Errorf("mysql get talk session: %w", err)
	}
	return s, nil
}

// Save replaces the parts of a session that change while talking.
func (r *TalkSessions) Save(ctx context.Context, s talk.Session) error {
	body, err := talkBody(s)
	if err != nil {
		return err
	}
	res, err := r.db.ExecContext(ctx, `UPDATE talk_sessions SET status = ?, body = ?, updated_at = ?, ended_at = ? WHERE id = ?`,
		string(s.Status), string(body), utc(s.UpdatedAt), nullTime(s.EndedAt), s.ID)
	if err != nil {
		return fmt.Errorf("mysql save talk session: %w", err)
	}
	ok, err := changed(res)
	if err != nil {
		return err
	}
	if !ok {
		return talk.ErrNotFound
	}
	return nil
}

// List returns the user's sessions, newest first.
func (r *TalkSessions) List(ctx context.Context, userID string, limit int) ([]talk.Session, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+talkColumns+` FROM talk_sessions WHERE user_id = ? ORDER BY created_at DESC LIMIT ?`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("mysql list talk sessions: %w", err)
	}
	defer rows.Close()
	out := []talk.Session{}
	for rows.Next() {
		s, err := scanTalkSession(rows)
		if err != nil {
			return nil, fmt.Errorf("mysql scan talk session: %w", err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql list talk sessions: %w", err)
	}
	return out, nil
}
