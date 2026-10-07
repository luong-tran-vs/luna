package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/luongtran/luna/backend/internal/writing"
)

// Writings implements writing.Repository on the "writings" table (F8). The unique (user_id,
// lesson_id) key keeps one writing per learner and lesson. The grade is flattened into grade_*
// columns (grade_status NULL means the writing has no grade yet); the four criteria are one JSON
// array read and written whole.
type Writings struct {
	db *sql.DB
}

// NewWritings returns the Writings repository.
func NewWritings(db *sql.DB) *Writings { return &Writings{db: db} }

var _ writing.Repository = (*Writings)(nil)

// writingsSchema is the DDL of this domain (see migrate.go).
func writingsSchema() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS writings (
  id CHAR(24) NOT NULL PRIMARY KEY,
  user_id CHAR(24) NOT NULL,
  lesson_id CHAR(24) NOT NULL,
  lesson_revision INT NOT NULL DEFAULT 0,
  lesson_title VARCHAR(500) NOT NULL DEFAULT '',
  prompt TEXT NOT NULL,
  text MEDIUMTEXT NOT NULL,
  status VARCHAR(16) NOT NULL,
  grade_status VARCHAR(16) NULL,
  grade_error TEXT NULL,
  grade_criteria JSON NULL,
  grade_overall TEXT NULL,
  grade_corrected MEDIUMTEXT NULL,
  grade_graded_at DATETIME(6) NULL,
  grade_seen TINYINT(1) NOT NULL DEFAULT 0,
  created_at DATETIME(6) NOT NULL,
  updated_at DATETIME(6) NOT NULL,
  submitted_at DATETIME(6) NULL,
  UNIQUE KEY writings_user_lesson (user_id, lesson_id),
  KEY writings_user_status_submitted (user_id, status, submitted_at)
) ` + tableOptions,
	}
}

type writingsCriterion struct {
	Name      string `json:"name"`
	Score     int    `json:"score"`
	CommentVi string `json:"commentVi"`
}

const writingsColsNoText = `id, user_id, lesson_id, lesson_revision, lesson_title, prompt, '' AS text, status,
  grade_status, grade_error, grade_criteria, grade_overall, grade_corrected, grade_graded_at, grade_seen,
  created_at, updated_at, submitted_at`

const writingsCols = `id, user_id, lesson_id, lesson_revision, lesson_title, prompt, text, status,
  grade_status, grade_error, grade_criteria, grade_overall, grade_corrected, grade_graded_at, grade_seen,
  created_at, updated_at, submitted_at`

type writingsScanner interface{ Scan(dest ...any) error }

func writingsScan(s writingsScanner) (writing.Writing, error) {
	var (
		w                                   writing.Writing
		status                              string
		gStatus, gError, gOverall, gCorrect sql.NullString
		gCriteria                           []byte
		gAt, submitted                      sql.NullTime
		gSeen                               bool
	)
	if err := s.Scan(&w.ID, &w.UserID, &w.LessonID, &w.LessonRevision, &w.LessonTitle, &w.Prompt, &w.Text, &status,
		&gStatus, &gError, &gCriteria, &gOverall, &gCorrect, &gAt, &gSeen,
		&w.CreatedAt, &w.UpdatedAt, &submitted); err != nil {
		return writing.Writing{}, err
	}
	w.Status = writing.Status(status)
	w.CreatedAt, w.UpdatedAt, w.SubmittedAt = w.CreatedAt.UTC(), w.UpdatedAt.UTC(), timeOf(submitted)
	if gStatus.Valid {
		var cs []writingsCriterion
		if err := fromJSON(gCriteria, &cs); err != nil {
			return writing.Writing{}, err
		}
		g := &writing.Grade{
			Status: writing.GradeStatus(gStatus.String), Error: gError.String, Criteria: make([]writing.Criterion, len(cs)),
			OverallVi: gOverall.String, CorrectedText: gCorrect.String, GradedAt: timeOf(gAt), Seen: gSeen,
		}
		for i, c := range cs {
			g.Criteria[i] = writing.Criterion(c)
		}
		w.Grade = g
	}
	return w, nil
}

// writingsGradeArgs flattens a grade into the values of the grade_* columns, in column order
// (status, error, criteria, overall, corrected, graded_at, seen).
func writingsGradeArgs(g writing.Grade) ([]any, error) {
	cs := make([]writingsCriterion, len(g.Criteria))
	for i, c := range g.Criteria {
		cs[i] = writingsCriterion(c)
	}
	raw, err := toJSON(cs)
	if err != nil {
		return nil, err
	}
	return []any{string(g.Status), g.Error, string(raw), g.OverallVi, g.CorrectedText, nullTime(g.GradedAt), g.Seen}, nil
}

func (r *Writings) one(ctx context.Context, q string, a ...any) (writing.Writing, bool, error) {
	w, err := writingsScan(r.db.QueryRowContext(ctx, q, a...))
	if errors.Is(err, sql.ErrNoRows) {
		return writing.Writing{}, false, nil
	}
	if err != nil {
		return writing.Writing{}, false, fmt.Errorf("mysql find writing: %w", err)
	}
	return w, true, nil
}

// GetByLesson returns the learner's writing for a lesson.
func (r *Writings) GetByLesson(ctx context.Context, userID, lessonID string) (writing.Writing, bool, error) {
	return r.one(ctx, "SELECT "+writingsCols+" FROM writings WHERE user_id = ? AND lesson_id = ?", userID, lessonID)
}

// Get returns a writing by id.
func (r *Writings) Get(ctx context.Context, id string) (writing.Writing, error) {
	w, ok, err := r.one(ctx, "SELECT "+writingsCols+" FROM writings WHERE id = ?", id)
	if err != nil {
		return writing.Writing{}, err
	}
	if !ok {
		return writing.Writing{}, writing.ErrNotFound
	}
	return w, nil
}

// SaveDraft upserts the draft unless it was submitted. One statement creates the row or, while it
// is still a draft, rewrites text and updated_at; a submitted row is left alone and reported.
func (r *Writings) SaveDraft(ctx context.Context, userID, lessonID, text string, now time.Time) (writing.Writing, error) {
	now = utc(now)
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO writings (id, user_id, lesson_id, prompt, text, status, created_at, updated_at)
VALUES (?, ?, ?, '', ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
  text = IF(status = ?, text, VALUES(text)),
  updated_at = IF(status = ?, updated_at, VALUES(updated_at))`,
		newID(), userID, lessonID, text, string(writing.StatusDraft), now, now,
		string(writing.StatusSubmitted), string(writing.StatusSubmitted))
	if err != nil {
		return writing.Writing{}, fmt.Errorf("mysql save draft: %w", err)
	}
	w, ok, err := r.GetByLesson(ctx, userID, lessonID)
	if err != nil {
		return writing.Writing{}, err
	}
	if !ok {
		return writing.Writing{}, fmt.Errorf("mysql save draft: row vanished")
	}
	if w.Status == writing.StatusSubmitted {
		return writing.Writing{}, writing.ErrSubmitted
	}
	return w, nil
}

// Submit turns the draft (or a new writing) into a submitted one, once.
func (r *Writings) Submit(ctx context.Context, w writing.Writing) (writing.Writing, error) {
	at := utc(w.SubmittedAt)
	gargs, err := writingsGradeArgs(writing.Grade{Status: writing.GradePending, Seen: true})
	if err != nil {
		return writing.Writing{}, err
	}
	// A concurrent SaveDraft may create the row between our UPDATE and INSERT; then try again.
	for range 3 {
		upd := append([]any{w.LessonRevision, w.LessonTitle, w.Prompt, w.Text, string(writing.StatusSubmitted), at, at}, gargs...)
		upd = append(upd, w.UserID, w.LessonID, string(writing.StatusSubmitted))
		res, err := r.db.ExecContext(ctx,
			`UPDATE writings SET lesson_revision = ?, lesson_title = ?, prompt = ?, text = ?, status = ?,
  submitted_at = ?, updated_at = ?,
  grade_status = ?, grade_error = ?, grade_criteria = ?, grade_overall = ?, grade_corrected = ?, grade_graded_at = ?, grade_seen = ?
WHERE user_id = ? AND lesson_id = ? AND status <> ?`, upd...)
		if err != nil {
			return writing.Writing{}, fmt.Errorf("mysql submit writing: %w", err)
		}
		ok, err := changed(res)
		if err != nil {
			return writing.Writing{}, err
		}
		if !ok {
			ins := append([]any{newID(), w.UserID, w.LessonID, w.LessonRevision, w.LessonTitle, w.Prompt, w.Text,
				string(writing.StatusSubmitted)}, gargs...)
			ins = append(ins, at, at, at)
			_, err = r.db.ExecContext(ctx,
				`INSERT INTO writings (id, user_id, lesson_id, lesson_revision, lesson_title, prompt, text, status,
  grade_status, grade_error, grade_criteria, grade_overall, grade_corrected, grade_graded_at, grade_seen,
  created_at, updated_at, submitted_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, ins...)
			if isDuplicate(err) {
				cur, found, gerr := r.GetByLesson(ctx, w.UserID, w.LessonID)
				if gerr != nil {
					return writing.Writing{}, gerr
				}
				if found && cur.Status == writing.StatusSubmitted {
					return writing.Writing{}, writing.ErrSubmitted
				}
				continue
			}
			if err != nil {
				return writing.Writing{}, fmt.Errorf("mysql submit writing: %w", err)
			}
		}
		out, found, err := r.GetByLesson(ctx, w.UserID, w.LessonID)
		if err != nil {
			return writing.Writing{}, err
		}
		if !found {
			return writing.Writing{}, fmt.Errorf("mysql submit writing: row vanished")
		}
		return out, nil
	}
	return writing.Writing{}, fmt.Errorf("mysql submit writing: too much contention")
}

func (r *Writings) updateByID(ctx context.Context, q string, a ...any) error {
	res, err := r.db.ExecContext(ctx, q, a...)
	if err != nil {
		return fmt.Errorf("mysql update writing: %w", err)
	}
	ok, err := changed(res)
	if err != nil {
		return err
	}
	if !ok {
		return writing.ErrNotFound
	}
	return nil
}

// SetGrade replaces the grade.
func (r *Writings) SetGrade(ctx context.Context, id string, g writing.Grade) error {
	gargs, err := writingsGradeArgs(g)
	if err != nil {
		return err
	}
	a := append(gargs, utc(time.Now()), id)
	return r.updateByID(ctx,
		`UPDATE writings SET grade_status = ?, grade_error = ?, grade_criteria = ?, grade_overall = ?,
  grade_corrected = ?, grade_graded_at = ?, grade_seen = ?, updated_at = ? WHERE id = ?`, a...)
}

// MarkSeen marks the result as seen.
func (r *Writings) MarkSeen(ctx context.Context, id string) error {
	return r.updateByID(ctx, "UPDATE writings SET grade_seen = 1 WHERE id = ?", id)
}

// List returns the learner's submitted writings, newest first (the text is left out).
func (r *Writings) List(ctx context.Context, userID string) ([]writing.Writing, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT "+writingsColsNoText+" FROM writings WHERE user_id = ? AND status = ? ORDER BY submitted_at DESC, id DESC",
		userID, string(writing.StatusSubmitted))
	if err != nil {
		return nil, fmt.Errorf("mysql find writings: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := []writing.Writing{}
	for rows.Next() {
		w, err := writingsScan(rows)
		if err != nil {
			return nil, fmt.Errorf("mysql read writings: %w", err)
		}
		out = append(out, w)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql read writings: %w", err)
	}
	return out, nil
}

// Unseen counts new results (done or failed, not seen) and gradings in progress.
func (r *Writings) Unseen(ctx context.Context, userID string) (writing.UnseenCount, error) {
	var out writing.UnseenCount
	err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM writings WHERE user_id = ? AND status = ? AND grade_status = ?",
		userID, string(writing.StatusSubmitted), string(writing.GradePending)).Scan(&out.Pending)
	if err != nil {
		return writing.UnseenCount{}, fmt.Errorf("mysql count pending writings: %w", err)
	}
	const unseenWhere = " FROM writings WHERE user_id = ? AND status = ? AND grade_status IN (?, ?) AND grade_seen = 0"
	a := []any{userID, string(writing.StatusSubmitted), string(writing.GradeDone), string(writing.GradeFailed)}
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*)"+unseenWhere, a...).Scan(&out.Unseen); err != nil {
		return writing.UnseenCount{}, fmt.Errorf("mysql count unseen writings: %w", err)
	}
	if out.Unseen > 0 {
		var id, st string
		err := r.db.QueryRowContext(ctx, "SELECT id, grade_status"+unseenWhere+" ORDER BY grade_graded_at DESC, id DESC LIMIT 1", a...).Scan(&id, &st)
		if err != nil {
			return writing.UnseenCount{}, fmt.Errorf("mysql find latest result: %w", err)
		}
		out.Latest = &writing.Latest{ID: id, Status: writing.GradeStatus(st)}
	}
	return out, nil
}

// Stats counts the writings submitted at or after since (nil = all) and averages the mean score of
// the graded ones among them.
func (r *Writings) Stats(ctx context.Context, userID string, since *time.Time) (int, *float64, error) {
	where, params := sinceClause("submitted_at", since)
	rows, err := r.db.QueryContext(ctx,
		"SELECT grade_status, grade_criteria FROM writings WHERE user_id = ? AND status = ?"+where,
		append([]any{userID, string(writing.StatusSubmitted)}, params...)...)
	if err != nil {
		return 0, nil, fmt.Errorf("mysql aggregate writings: %w", err)
	}
	defer func() { _ = rows.Close() }()
	submitted, averaged := 0, 0
	sum := 0.0
	for rows.Next() {
		var st sql.NullString
		var raw []byte
		if err := rows.Scan(&st, &raw); err != nil {
			return 0, nil, fmt.Errorf("mysql read writing stats: %w", err)
		}
		submitted++
		if st.String != string(writing.GradeDone) {
			continue
		}
		var cs []writingsCriterion
		if err := fromJSON(raw, &cs); err != nil {
			return 0, nil, err
		}
		if len(cs) == 0 {
			continue
		}
		total := 0
		for _, c := range cs {
			total += c.Score
		}
		sum += float64(total) / float64(len(cs))
		averaged++
	}
	if err := rows.Err(); err != nil {
		return 0, nil, fmt.Errorf("mysql read writing stats: %w", err)
	}
	if averaged == 0 {
		return submitted, nil, nil
	}
	avg := sum / float64(averaged)
	return submitted, &avg, nil
}
