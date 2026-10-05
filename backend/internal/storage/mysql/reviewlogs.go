package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/luongtran/luna/backend/internal/vocab"
)

// reviewLogsSchema is the DDL of the review history. state_before is the card's schedule before the
// review, as JSON with the keys due, stability, difficulty, elapsedDays, scheduledDays, reps, lapses,
// state, lastReview (the names Mongo uses).
func reviewLogsSchema() []string {
	return []string{`CREATE TABLE IF NOT EXISTS review_logs (
  id CHAR(24) NOT NULL PRIMARY KEY,
  user_id CHAR(24) NOT NULL,
  card_id CHAR(24) NOT NULL,
  rating INT NOT NULL,
  mode VARCHAR(32) NOT NULL,
  context VARCHAR(32) NOT NULL,
  reviewed_at DATETIME(6) NOT NULL,
  state_before JSON NOT NULL,
  KEY review_logs_user_card (user_id, card_id),
  KEY review_logs_user_reviewed (user_id, reviewed_at),
  KEY review_logs_user_context_reviewed (user_id, context, reviewed_at)
) ` + tableOptions}
}

// reviewLogsState is the JSON form of vocab.Schedule in state_before.
type reviewLogsState struct {
	Due           *time.Time `json:"due,omitempty"`
	Stability     float64    `json:"stability"`
	Difficulty    float64    `json:"difficulty"`
	ElapsedDays   uint64     `json:"elapsedDays"`
	ScheduledDays uint64     `json:"scheduledDays"`
	Reps          uint64     `json:"reps"`
	Lapses        uint64     `json:"lapses"`
	State         int        `json:"state"`
	LastReview    *time.Time `json:"lastReview,omitempty"`
}

func reviewLogsTimePtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	u := t.UTC()
	return &u
}

// ReviewLogs implements vocab.ReviewLogRepository on the "review_logs" table.
type ReviewLogs struct {
	db *sql.DB
}

// NewReviewLogs returns the ReviewLogs repository.
func NewReviewLogs(db *sql.DB) *ReviewLogs { return &ReviewLogs{db: db} }

var _ vocab.ReviewLogRepository = (*ReviewLogs)(nil)

// Add stores one review.
func (r *ReviewLogs) Add(ctx context.Context, l vocab.ReviewLog) error {
	b := l.Before
	state, err := toJSON(reviewLogsState{
		Due: reviewLogsTimePtr(b.Due), Stability: b.Stability, Difficulty: b.Difficulty, ElapsedDays: b.ElapsedDays,
		ScheduledDays: b.ScheduledDays, Reps: b.Reps, Lapses: b.Lapses, State: int(b.State), LastReview: reviewLogsTimePtr(b.LastReview),
	})
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO review_logs (id, user_id, card_id, rating, mode, context, reviewed_at, state_before)
 VALUES (?,?,?,?,?,?,?,?)`, newID(), l.UserID, l.CardID, int(l.Rating), string(l.Mode), string(l.Context), utc(l.ReviewedAt), state)
	if err != nil {
		return fmt.Errorf("mysql insert review log: %w", err)
	}
	return nil
}

// DeleteByCard removes every log of one of the user's cards.
func (r *ReviewLogs) DeleteByCard(ctx context.Context, userID, cardID string) (int, error) {
	res, err := r.db.ExecContext(ctx, "DELETE FROM review_logs WHERE user_id = ? AND card_id = ?", userID, cardID)
	if err != nil {
		return 0, fmt.Errorf("mysql delete review logs: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("mysql rows affected: %w", err)
	}
	return int(n), nil
}

// CountSince counts the user's reviews in context c at or after since.
func (r *ReviewLogs) CountSince(ctx context.Context, userID string, c vocab.Context, since time.Time) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM review_logs WHERE user_id = ? AND context = ? AND reviewed_at >= ?",
		userID, string(c), utc(since)).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("mysql count review logs: %w", err)
	}
	return n, nil
}
