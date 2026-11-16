package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/luongtran/luna/backend/internal/games"
)

// gamesSchema is the DDL of the games (F26, migration 045): the personal bests, the daily challenge
// results (answers as JSON) and the extra "Bắn lỗi" sentences (distractors as JSON).
func gamesSchema() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS game_bests (
  user_id CHAR(24)    NOT NULL,
  game    VARCHAR(20) NOT NULL,
  score   INT         NOT NULL,
  at      DATETIME(6) NOT NULL,
  PRIMARY KEY (user_id, game)
) ` + tableOptions,
		`CREATE TABLE IF NOT EXISTS daily_challenges (
  user_id     CHAR(24)    NOT NULL,
  day         CHAR(10)    NOT NULL,
  score       INT         NOT NULL,
  answers     TEXT        NOT NULL,
  finished_at DATETIME(6) NOT NULL,
  PRIMARY KEY (user_id, day)
) ` + tableOptions,
		`CREATE TABLE IF NOT EXISTS grammar_spot_items (
  id          CHAR(24)     NOT NULL PRIMARY KEY,
  point_id    VARCHAR(80)  NOT NULL,
  level       VARCHAR(4)   NOT NULL,
  sentence    TEXT         NOT NULL,
  wrong       VARCHAR(200) NOT NULL,
  right_text  VARCHAR(200) NOT NULL,
  distractors TEXT         NOT NULL,
  rule_vi     TEXT         NOT NULL,
  status      VARCHAR(10)  NOT NULL,
  created_at  DATETIME(6)  NOT NULL,
  updated_at  DATETIME(6)  NOT NULL,
  KEY idx_spot_point_status (point_id, status)
) ` + tableOptions,
	}
}

// --- game_bests ---

// GameBests implements games.BestRepository on "game_bests".
type GameBests struct {
	db *sql.DB
}

// NewGameBests returns the GameBests repository.
func NewGameBests(db *sql.DB) *GameBests { return &GameBests{db: db} }

var _ games.BestRepository = (*GameBests)(nil)

// List returns every best of the user.
func (r *GameBests) List(ctx context.Context, userID string) ([]games.Best, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT game, score, at FROM game_bests WHERE user_id = ? ORDER BY game`, userID)
	if err != nil {
		return nil, fmt.Errorf("mysql list game bests: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []games.Best
	for rows.Next() {
		b := games.Best{UserID: userID}
		var game string
		if err := rows.Scan(&game, &b.Score, &b.At); err != nil {
			return nil, fmt.Errorf("mysql scan game best: %w", err)
		}
		b.Game, b.At = games.Game(game), utc(b.At)
		out = append(out, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql list game bests: %w", err)
	}
	return out, nil
}

// Raise stores score when it beats the best; the time only changes with the score.
func (r *GameBests) Raise(ctx context.Context, userID string, game games.Game, score int, at time.Time) (bool, error) {
	// The DSN counts matched rows (clientFoundRows), so the update matches only a lower best, and the
	// insert, ignored when a best exists, only a first one.
	res, err := r.db.ExecContext(ctx, `UPDATE game_bests SET score = ?, at = ? WHERE user_id = ? AND game = ? AND score < ?`,
		score, utc(at), userID, string(game), score)
	if err != nil {
		return false, fmt.Errorf("mysql raise game best: %w", err)
	}
	if ok, err := changed(res); err != nil || ok {
		return ok, err
	}
	res, err = r.db.ExecContext(ctx, `INSERT IGNORE INTO game_bests (user_id, game, score, at) VALUES (?, ?, ?, ?)`,
		userID, string(game), score, utc(at))
	if err != nil {
		return false, fmt.Errorf("mysql insert game best: %w", err)
	}
	return changed(res)
}

// --- daily_challenges ---

// DailyChallenges implements games.DailyRepository on "daily_challenges".
type DailyChallenges struct {
	db *sql.DB
}

// NewDailyChallenges returns the DailyChallenges repository.
func NewDailyChallenges(db *sql.DB) *DailyChallenges { return &DailyChallenges{db: db} }

var _ games.DailyRepository = (*DailyChallenges)(nil)

type dailyAnswerRow struct {
	Kind  string `json:"kind"`
	Right bool   `json:"right"`
}

func decodeAnswers(s string) []games.DailyAnswer {
	var rows []dailyAnswerRow
	if json.Unmarshal([]byte(s), &rows) != nil {
		return nil
	}
	out := make([]games.DailyAnswer, len(rows))
	for i, a := range rows {
		out[i] = games.DailyAnswer{Kind: a.Kind, Right: a.Right}
	}
	return out
}

// Get returns the user's result of day.
func (r *DailyChallenges) Get(ctx context.Context, userID, day string) (games.DailyResult, error) {
	out := games.DailyResult{UserID: userID, Day: day}
	var answers string
	err := r.db.QueryRowContext(ctx, `SELECT score, answers, finished_at FROM daily_challenges WHERE user_id = ? AND day = ?`, userID, day).
		Scan(&out.Score, &answers, &out.FinishedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return games.DailyResult{}, games.ErrNotFound
	}
	if err != nil {
		return games.DailyResult{}, fmt.Errorf("mysql get daily challenge: %w", err)
	}
	out.Answers, out.FinishedAt = decodeAnswers(answers), utc(out.FinishedAt)
	return out, nil
}

// Save stores the first result of a day.
func (r *DailyChallenges) Save(ctx context.Context, res games.DailyResult) error {
	rows := make([]dailyAnswerRow, len(res.Answers))
	for i, a := range res.Answers {
		rows[i] = dailyAnswerRow{Kind: a.Kind, Right: a.Right}
	}
	answers, err := json.Marshal(rows)
	if err != nil {
		return fmt.Errorf("encode daily answers: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO daily_challenges (user_id, day, score, answers, finished_at) VALUES (?, ?, ?, ?, ?)`,
		res.UserID, res.Day, res.Score, string(answers), utc(res.FinishedAt))
	if isDuplicate(err) {
		return games.ErrExists
	}
	if err != nil {
		return fmt.Errorf("mysql save daily challenge: %w", err)
	}
	return nil
}

// List returns the user's results, newest day first.
func (r *DailyChallenges) List(ctx context.Context, userID string) ([]games.DailyResult, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT day, score, answers, finished_at FROM daily_challenges WHERE user_id = ? ORDER BY day DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("mysql list daily challenges: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []games.DailyResult
	for rows.Next() {
		d := games.DailyResult{UserID: userID}
		var answers string
		if err := rows.Scan(&d.Day, &d.Score, &answers, &d.FinishedAt); err != nil {
			return nil, fmt.Errorf("mysql scan daily challenge: %w", err)
		}
		d.Answers, d.FinishedAt = decodeAnswers(answers), utc(d.FinishedAt)
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql list daily challenges: %w", err)
	}
	return out, nil
}

// --- grammar_spot_items ---

// GrammarSpotItems implements games.SpotRepository on "grammar_spot_items".
type GrammarSpotItems struct {
	db *sql.DB
}

// NewGrammarSpotItems returns the GrammarSpotItems repository.
func NewGrammarSpotItems(db *sql.DB) *GrammarSpotItems { return &GrammarSpotItems{db: db} }

var _ games.SpotRepository = (*GrammarSpotItems)(nil)

const spotColumns = `id, point_id, level, sentence, wrong, right_text, distractors, rule_vi, status, created_at, updated_at`

func (r *GrammarSpotItems) query(ctx context.Context, where string, params ...any) ([]games.SpotItem, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+spotColumns+` FROM grammar_spot_items `+where+` ORDER BY created_at, id`, params...)
	if err != nil {
		return nil, fmt.Errorf("mysql list spot items: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []games.SpotItem
	for rows.Next() {
		it, err := scanSpot(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql list spot items: %w", err)
	}
	return out, nil
}

type scanner interface{ Scan(dest ...any) error }

func scanSpot(s scanner) (games.SpotItem, error) {
	var it games.SpotItem
	var distractors, status string
	if err := s.Scan(&it.ID, &it.PointID, &it.Level, &it.Sentence, &it.Wrong, &it.Right, &distractors, &it.RuleVi, &status,
		&it.CreatedAt, &it.UpdatedAt); err != nil {
		return games.SpotItem{}, err
	}
	_ = json.Unmarshal([]byte(distractors), &it.Distractors)
	it.Status, it.CreatedAt, it.UpdatedAt = games.SpotStatus(status), utc(it.CreatedAt), utc(it.UpdatedAt)
	return it, nil
}

// ListByPoint returns the items of a point, oldest first.
func (r *GrammarSpotItems) ListByPoint(ctx context.Context, pointID string) ([]games.SpotItem, error) {
	return r.query(ctx, `WHERE point_id = ?`, pointID)
}

// Approved returns the approved items of the points (all when pointIDs is nil).
func (r *GrammarSpotItems) Approved(ctx context.Context, pointIDs []string) ([]games.SpotItem, error) {
	if pointIDs == nil {
		return r.query(ctx, `WHERE status = ?`, string(games.SpotApproved))
	}
	if len(pointIDs) == 0 {
		return nil, nil
	}
	return r.query(ctx, `WHERE status = ? AND point_id IN (`+placeholders(len(pointIDs))+`)`,
		append([]any{string(games.SpotApproved)}, args(pointIDs)...)...)
}

// Counts returns the approved and draft items per point.
func (r *GrammarSpotItems) Counts(ctx context.Context) (map[string]games.SpotCount, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT point_id, status, COUNT(*) FROM grammar_spot_items GROUP BY point_id, status`)
	if err != nil {
		return nil, fmt.Errorf("mysql count spot items: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := map[string]games.SpotCount{}
	for rows.Next() {
		var point, status string
		var n int
		if err := rows.Scan(&point, &status, &n); err != nil {
			return nil, fmt.Errorf("mysql scan spot count: %w", err)
		}
		c := out[point]
		if status == string(games.SpotApproved) {
			c.Approved += n
		} else {
			c.Draft += n
		}
		out[point] = c
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql count spot items: %w", err)
	}
	return out, nil
}

func encodeDistractors(d []string) (string, error) {
	if d == nil {
		d = []string{}
	}
	b, err := json.Marshal(d)
	if err != nil {
		return "", fmt.Errorf("encode distractors: %w", err)
	}
	return string(b), nil
}

// Create stores the items with new ids.
func (r *GrammarSpotItems) Create(ctx context.Context, items []games.SpotItem) ([]games.SpotItem, error) {
	out := make([]games.SpotItem, 0, len(items))
	for _, it := range items {
		it.ID = newID()
		d, err := encodeDistractors(it.Distractors)
		if err != nil {
			return nil, err
		}
		if _, err := r.db.ExecContext(ctx, `INSERT INTO grammar_spot_items (`+spotColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			it.ID, it.PointID, it.Level, it.Sentence, it.Wrong, it.Right, d, it.RuleVi, string(it.Status), utc(it.CreatedAt), utc(it.UpdatedAt)); err != nil {
			return nil, fmt.Errorf("mysql insert spot item: %w", err)
		}
		out = append(out, it)
	}
	return out, nil
}

// Get returns the item with id.
func (r *GrammarSpotItems) Get(ctx context.Context, id string) (games.SpotItem, error) {
	it, err := scanSpot(r.db.QueryRowContext(ctx, `SELECT `+spotColumns+` FROM grammar_spot_items WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return games.SpotItem{}, games.ErrNotFound
	}
	if err != nil {
		return games.SpotItem{}, fmt.Errorf("mysql get spot item: %w", err)
	}
	return it, nil
}

// Update replaces the item.
func (r *GrammarSpotItems) Update(ctx context.Context, it games.SpotItem) error {
	d, err := encodeDistractors(it.Distractors)
	if err != nil {
		return err
	}
	res, err := r.db.ExecContext(ctx, `UPDATE grammar_spot_items SET point_id = ?, level = ?, sentence = ?, wrong = ?, right_text = ?,
  distractors = ?, rule_vi = ?, status = ?, updated_at = ? WHERE id = ?`,
		it.PointID, it.Level, it.Sentence, it.Wrong, it.Right, d, it.RuleVi, string(it.Status), utc(it.UpdatedAt), it.ID)
	if err != nil {
		return fmt.Errorf("mysql update spot item: %w", err)
	}
	if ok, err := changed(res); err != nil || ok {
		return err
	}
	if _, err := r.Get(ctx, it.ID); err != nil {
		return err
	}
	return nil
}

// Delete removes the item.
func (r *GrammarSpotItems) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM grammar_spot_items WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("mysql delete spot item: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("mysql delete spot item: %w", err)
	}
	if n == 0 {
		return games.ErrNotFound
	}
	return nil
}
