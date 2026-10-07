package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/luongtran/luna/backend/internal/vocab"
)

// cardsSchema is the DDL of the notebook (see migrate.go). seq keeps insertion order for the rows that
// share a created_at (Mongo sorts those by _id). A card without a schedule (saved before F5) has a NULL
// due; reps is never NULL, 0 meaning "missing".
func cardsSchema() []string {
	return []string{`CREATE TABLE IF NOT EXISTS cards (
  id CHAR(24) NOT NULL PRIMARY KEY,
  seq BIGINT NOT NULL AUTO_INCREMENT,
  user_id CHAR(24) NOT NULL,
  text VARCHAR(500) NOT NULL,
  lemma VARCHAR(191) NOT NULL,
  ipa VARCHAR(255) NOT NULL DEFAULT '',
  meaning_vi TEXT NOT NULL,
  context_sentence TEXT NOT NULL,
  lesson_id CHAR(24) NULL,
  source VARCHAR(32) NOT NULL DEFAULT '',
  created_at DATETIME(6) NOT NULL,
  due DATETIME(6) NULL,
  stability DOUBLE NOT NULL DEFAULT 0,
  difficulty DOUBLE NOT NULL DEFAULT 0,
  elapsed_days BIGINT UNSIGNED NOT NULL DEFAULT 0,
  scheduled_days BIGINT UNSIGNED NOT NULL DEFAULT 0,
  reps BIGINT UNSIGNED NOT NULL DEFAULT 0,
  lapses BIGINT UNSIGNED NOT NULL DEFAULT 0,
  state INT NOT NULL DEFAULT 0,
  last_review DATETIME(6) NULL,
  UNIQUE KEY cards_seq (seq),
  UNIQUE KEY cards_user_lemma (user_id, lemma),
  KEY cards_user_created (user_id, created_at),
  KEY cards_user_due (user_id, due),
  KEY cards_user_lesson_created (user_id, lesson_id, created_at)
) ` + tableOptions}
}

// Cards implements vocab.Repository on the "cards" table.
type Cards struct {
	db *sql.DB
}

// NewCards returns the Cards repository.
func NewCards(db *sql.DB) *Cards { return &Cards{db: db} }

var _ vocab.Repository = (*Cards)(nil)

const cardsCols = `id, user_id, text, lemma, ipa, meaning_vi, context_sentence, lesson_id, source, created_at,
 due, stability, difficulty, elapsed_days, scheduled_days, reps, lapses, state, last_review`

type cardsScanner interface{ Scan(dest ...any) error }

func cardsScan(r cardsScanner) (vocab.Card, error) {
	var (
		c        vocab.Card
		lesson   sql.NullString
		source   string
		due, rev sql.NullTime
		state    int
	)
	err := r.Scan(&c.ID, &c.UserID, &c.Text, &c.Lemma, &c.IPA, &c.MeaningVi, &c.ContextSentence, &lesson, &source,
		&c.CreatedAt, &due, &c.Schedule.Stability, &c.Schedule.Difficulty, &c.Schedule.ElapsedDays,
		&c.Schedule.ScheduledDays, &c.Schedule.Reps, &c.Schedule.Lapses, &state, &rev)
	if err != nil {
		return vocab.Card{}, err
	}
	c.LessonID = lesson.String
	c.Source = vocab.Source(source)
	c.CreatedAt = c.CreatedAt.UTC()
	c.Schedule.Due = timeOf(due)
	c.Schedule.LastReview = timeOf(rev)
	c.Schedule.State = vocab.State(state)
	return c, nil
}

func cardsQuery(ctx context.Context, db *sql.DB, query string, a ...any) ([]vocab.Card, error) {
	rows, err := db.QueryContext(ctx, query, a...)
	if err != nil {
		return nil, fmt.Errorf("mysql query cards: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []vocab.Card
	for rows.Next() {
		c, err := cardsScan(rows)
		if err != nil {
			return nil, fmt.Errorf("mysql scan card: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql read cards: %w", err)
	}
	return out, nil
}

func cardsOne(ctx context.Context, db *sql.DB, where string, a ...any) (vocab.Card, error) {
	c, err := cardsScan(db.QueryRowContext(ctx, "SELECT "+cardsCols+" FROM cards WHERE "+where, a...))
	if errors.Is(err, sql.ErrNoRows) {
		return vocab.Card{}, vocab.ErrNotFound
	}
	if err != nil {
		return vocab.Card{}, fmt.Errorf("mysql find card: %w", err)
	}
	return c, nil
}

func cardsNullable(s string) sql.NullString { return sql.NullString{String: s, Valid: s != ""} }

// Create inserts c; the unique (user_id, lemma) key turns duplicates into vocab.ErrExists.
func (r *Cards) Create(ctx context.Context, c vocab.Card) (vocab.Card, error) {
	c.ID = newID()
	c.CreatedAt = c.CreatedAt.UTC()
	s := c.Schedule
	_, err := r.db.ExecContext(ctx, `INSERT INTO cards (id, user_id, text, lemma, ipa, meaning_vi, context_sentence,
 lesson_id, source, created_at, due, stability, difficulty, elapsed_days, scheduled_days, reps, lapses, state, last_review)
 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		c.ID, c.UserID, c.Text, c.Lemma, c.IPA, c.MeaningVi, c.ContextSentence, cardsNullable(c.LessonID), string(c.Source),
		c.CreatedAt, nullTime(s.Due), s.Stability, s.Difficulty, s.ElapsedDays, s.ScheduledDays, s.Reps, s.Lapses,
		int(s.State), nullTime(s.LastReview))
	if err != nil {
		if isDuplicate(err) {
			return vocab.Card{}, vocab.ErrExists
		}
		return vocab.Card{}, fmt.Errorf("mysql insert card: %w", err)
	}
	c.Schedule.Due = timeOf(nullTime(s.Due))
	c.Schedule.LastReview = timeOf(nullTime(s.LastReview))
	return c, nil
}

// FindByLemma returns the user's card with lemma.
func (r *Cards) FindByLemma(ctx context.Context, userID, lemma string) (vocab.Card, error) {
	return cardsOne(ctx, r.db, "user_id = ? AND lemma = ?", userID, lemma)
}

// Words lists the user's lemmas and texts, oldest first.
func (r *Cards) Words(ctx context.Context, userID string) ([]vocab.WordRef, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT lemma, text FROM cards WHERE user_id = ? ORDER BY created_at, seq", userID)
	if err != nil {
		return nil, fmt.Errorf("mysql list words: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []vocab.WordRef
	for rows.Next() {
		var w vocab.WordRef
		if err := rows.Scan(&w.Lemma, &w.Text); err != nil {
			return nil, fmt.Errorf("mysql scan word: %w", err)
		}
		out = append(out, w)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql read words: %w", err)
	}
	return out, nil
}

// Get returns the user's card with id.
func (r *Cards) Get(ctx context.Context, userID, id string) (vocab.Card, error) {
	return cardsOne(ctx, r.db, "id = ? AND user_id = ?", id, userID)
}

// cardsLikeEscaper makes a string literal inside LIKE (escape character '!').
var cardsLikeEscaper = strings.NewReplacer("!", "!!", "%", "!%", "_", "!_")

// List returns one page of the notebook, newest first.
func (r *Cards) List(ctx context.Context, userID string, q vocab.ListQuery, limit, skip int) ([]vocab.Card, bool, error) {
	where := "user_id = ?"
	a := []any{userID}
	switch q.LessonID {
	case "":
	case vocab.ManualLesson:
		where += " AND lesson_id IS NULL"
	default:
		where += " AND lesson_id = ?"
		a = append(a, q.LessonID)
	}
	if q.Q != "" {
		pat := "%" + cardsLikeEscaper.Replace(strings.ToLower(q.Q)) + "%"
		where += " AND (LOWER(text) LIKE ? ESCAPE '!' OR LOWER(lemma) LIKE ? ESCAPE '!')"
		a = append(a, pat, pat)
	}
	a = append(a, max(limit, 0)+1, max(skip, 0))
	cards, err := cardsQuery(ctx, r.db, "SELECT "+cardsCols+" FROM cards WHERE "+where+
		" ORDER BY created_at DESC, seq DESC LIMIT ? OFFSET ?", a...)
	if err != nil {
		return nil, false, err
	}
	if len(cards) > limit {
		return cards[:max(limit, 0)], true, nil
	}
	return cards, false, nil
}

// LessonCounts counts the user's cards per lesson; "" is cards added by hand.
func (r *Cards) LessonCounts(ctx context.Context, userID string) (map[string]int, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT lesson_id, COUNT(*) FROM cards WHERE user_id = ? GROUP BY lesson_id", userID)
	if err != nil {
		return nil, fmt.Errorf("mysql count cards by lesson: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := map[string]int{}
	for rows.Next() {
		var lesson sql.NullString
		var n int
		if err := rows.Scan(&lesson, &n); err != nil {
			return nil, fmt.Errorf("mysql scan lesson count: %w", err)
		}
		out[lesson.String] += n
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql read lesson counts: %w", err)
	}
	return out, nil
}

// UpdateDetails sets meaning, IPA and example only; the schedule is untouched.
func (r *Cards) UpdateDetails(ctx context.Context, userID, id string, d vocab.Details) (vocab.Card, error) {
	var sets []string
	var a []any
	if d.MeaningVi != nil {
		sets, a = append(sets, "meaning_vi = ?"), append(a, *d.MeaningVi)
	}
	if d.IPA != nil {
		sets, a = append(sets, "ipa = ?"), append(a, *d.IPA)
	}
	if d.ContextSentence != nil {
		sets, a = append(sets, "context_sentence = ?"), append(a, *d.ContextSentence)
	}
	if len(sets) > 0 {
		res, err := r.db.ExecContext(ctx, "UPDATE cards SET "+strings.Join(sets, ", ")+" WHERE id = ? AND user_id = ?",
			append(a, id, userID)...)
		if err != nil {
			return vocab.Card{}, fmt.Errorf("mysql update card: %w", err)
		}
		ok, err := changed(res)
		if err != nil {
			return vocab.Card{}, err
		}
		if !ok {
			return vocab.Card{}, vocab.ErrNotFound
		}
	}
	return r.Get(ctx, userID, id)
}

// UpdateSchedule writes s only if the card still has expectedReps reviews (missing = 0).
func (r *Cards) UpdateSchedule(ctx context.Context, userID, id string, expectedReps uint64, s vocab.Schedule) (bool, error) {
	res, err := r.db.ExecContext(ctx, `UPDATE cards SET due = ?, stability = ?, difficulty = ?, elapsed_days = ?,
 scheduled_days = ?, reps = ?, lapses = ?, state = ?, last_review = ? WHERE id = ? AND user_id = ? AND reps = ?`,
		nullTime(s.Due), s.Stability, s.Difficulty, s.ElapsedDays, s.ScheduledDays, s.Reps, s.Lapses, int(s.State),
		nullTime(s.LastReview), id, userID, expectedReps)
	if err != nil {
		return false, fmt.Errorf("mysql update schedule: %w", err)
	}
	return changed(res)
}

// Due returns the most overdue cards first (cards without a schedule sort first).
func (r *Cards) Due(ctx context.Context, userID string, now, startOfToday time.Time, limit int) ([]vocab.Card, int, error) {
	const where = "user_id = ? AND (due <= ? OR (due IS NULL AND created_at < ?))"
	a := []any{userID, utc(now), utc(startOfToday)}
	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM cards WHERE "+where, a...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("mysql count due cards: %w", err)
	}
	cards, err := cardsQuery(ctx, r.db, "SELECT "+cardsCols+" FROM cards WHERE "+where+
		" ORDER BY due, created_at, seq LIMIT ?", append(a, max(limit, 0))...)
	if err != nil {
		return nil, 0, err
	}
	return cards, total, nil
}

// NextDue is the earliest due time after now; cards without a schedule saved since startOfToday are due
// the next day.
func (r *Cards) NextDue(ctx context.Context, userID string, now, startOfToday time.Time) (time.Time, bool, error) {
	var next sql.NullTime
	if err := r.db.QueryRowContext(ctx, "SELECT MIN(due) FROM cards WHERE user_id = ? AND due > ?", userID, utc(now)).Scan(&next); err != nil {
		return time.Time{}, false, fmt.Errorf("mysql find next due: %w", err)
	}
	best := timeOf(next)
	var legacy bool
	if err := r.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM cards WHERE user_id = ? AND due IS NULL AND created_at >= ?)",
		userID, utc(startOfToday)).Scan(&legacy); err != nil {
		return time.Time{}, false, fmt.Errorf("mysql find new cards: %w", err)
	}
	if tomorrow := startOfToday.AddDate(0, 0, 1).UTC(); legacy && tomorrow.After(now) && (best.IsZero() || tomorrow.Before(best)) {
		best = tomorrow
	}
	return best, !best.IsZero(), nil
}

// CountDue counts cards due before `before`, and cards without a schedule saved before createdBefore.
func (r *Cards) CountDue(ctx context.Context, userID string, before, createdBefore time.Time) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM cards WHERE user_id = ? AND (due < ? OR (due IS NULL AND created_at < ?))",
		userID, utc(before), utc(createdBefore)).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("mysql count due cards: %w", err)
	}
	return n, nil
}

// Count is the number of the user's cards created at or after since (nil = every card).
func (r *Cards) Count(ctx context.Context, userID string, since *time.Time) (int, error) {
	var n int
	where, params := sinceClause("created_at", since)
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM cards WHERE user_id = ?"+where, append([]any{userID}, params...)...).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("mysql count cards: %w", err)
	}
	return n, nil
}

// Delete removes the user's card.
func (r *Cards) Delete(ctx context.Context, userID, id string) (bool, error) {
	res, err := r.db.ExecContext(ctx, "DELETE FROM cards WHERE id = ? AND user_id = ?", id, userID)
	if err != nil {
		return false, fmt.Errorf("mysql delete card: %w", err)
	}
	return changed(res)
}
