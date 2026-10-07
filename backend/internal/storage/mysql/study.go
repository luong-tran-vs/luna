package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/luongtran/luna/backend/internal/progress"
)

// --- goals ---

// Goals is the progress.GoalRepository on "goals": one row per (user, topic).
type Goals struct {
	db *sql.DB
}

var _ progress.GoalRepository = (*Goals)(nil)

// NewGoals returns the Goals repository.
func NewGoals(db *sql.DB) *Goals { return &Goals{db: db} }

// LessonProgress is the progress.ProgressRepository on "lesson_progress": one row per (user, lesson),
// the steps done as one column each.
type LessonProgress struct {
	db *sql.DB
}

var _ progress.ProgressRepository = (*LessonProgress)(nil)

// NewLessonProgress returns the LessonProgress repository.
func NewLessonProgress(db *sql.DB) *LessonProgress { return &LessonProgress{db: db} }

// StudyDays is the progress.DayRepository on "study_days".
type StudyDays struct {
	db *sql.DB
}

var _ progress.DayRepository = (*StudyDays)(nil)

// NewStudyDays returns the StudyDays repository.
func NewStudyDays(db *sql.DB) *StudyDays { return &StudyDays{db: db} }

// studySchema is the DDL of this domain (see migrate.go).
func studySchema() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS goals (
  id             CHAR(24)     NOT NULL PRIMARY KEY,
  user_id        CHAR(24)     NOT NULL,
  topic_id       CHAR(24)     NOT NULL,
  level          VARCHAR(16)  NOT NULL,
  status         VARCHAR(16)  NOT NULL,
  effective_from VARCHAR(10)  NOT NULL,
  started_at     DATETIME(6)  NOT NULL,
  UNIQUE KEY goals_user_topic (user_id, topic_id),
  KEY goals_user_status (user_id, status)
) ` + tableOptions,
		`CREATE TABLE IF NOT EXISTS lesson_progress (
  user_id        CHAR(24)     NOT NULL,
  lesson_id      CHAR(24)     NOT NULL,
  topic_id       CHAR(24)     NOT NULL DEFAULT '',
  day_key        VARCHAR(10)  NOT NULL DEFAULT '',
  step_read      TINYINT(1)   NOT NULL DEFAULT 0,
  step_listen    TINYINT(1)   NOT NULL DEFAULT 0,
  step_write     TINYINT(1)   NOT NULL DEFAULT 0,
  current_step   VARCHAR(16)  NOT NULL DEFAULT '',
  sentence_index INT          NOT NULL DEFAULT 0,
  started_at     DATETIME(6)  NOT NULL,
  completed_at   DATETIME(6)  NULL,
  PRIMARY KEY (user_id, lesson_id),
  KEY lesson_progress_completed (user_id, completed_at)
) ` + tableOptions,
		`CREATE TABLE IF NOT EXISTS study_days (
  user_id   CHAR(24)    NOT NULL,
  day_key   VARCHAR(10) NOT NULL,
  completed TINYINT(1)  NOT NULL DEFAULT 1,
  PRIMARY KEY (user_id, day_key)
) ` + tableOptions,
	}
}

// stepTimesSchema adds when each step of a lesson was done, for the stats by period. Rows saved
// before have NULL times. One column per statement, so a rerun skips the ones already added (the
// migration ignores "duplicate column name").
func stepTimesSchema() []string {
	return []string{
		`ALTER TABLE lesson_progress ADD COLUMN step_read_at DATETIME(6) NULL`,
		`ALTER TABLE lesson_progress ADD COLUMN step_listen_at DATETIME(6) NULL`,
		`ALTER TABLE lesson_progress ADD COLUMN step_write_at DATETIME(6) NULL`,
	}
}

// studyValidID reports whether id has the shape of an id made by newID.
func studyValidID(id string) bool {
	if len(id) != 24 {
		return false
	}
	for _, c := range id {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

const studyGoalCols = "id, user_id, topic_id, level, status, effective_from, started_at"

func studyScanGoal(sc interface{ Scan(...any) error }) (progress.Goal, error) {
	var g progress.Goal
	var status string
	if err := sc.Scan(&g.ID, &g.UserID, &g.TopicID, &g.Level, &status, &g.EffectiveFrom, &g.StartedAt); err != nil {
		return progress.Goal{}, err
	}
	g.Status = progress.GoalStatus(status)
	g.StartedAt = utc(g.StartedAt)
	return g, nil
}

// List returns the user's goals, oldest first.
func (r *Goals) List(ctx context.Context, userID string) ([]progress.Goal, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT "+studyGoalCols+" FROM goals WHERE user_id = ? ORDER BY started_at, id", userID)
	if err != nil {
		return nil, fmt.Errorf("mysql list goals: %w", err)
	}
	defer rows.Close()
	out := []progress.Goal{}
	for rows.Next() {
		g, err := studyScanGoal(rows)
		if err != nil {
			return nil, fmt.Errorf("mysql scan goal: %w", err)
		}
		out = append(out, g)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql list goals: %w", err)
	}
	return out, nil
}

// Activate pauses the user's other active goals and makes topicID's goal active (creating it).
func (r *Goals) Activate(ctx context.Context, userID, topicID, level, effectiveFrom string, now time.Time) (progress.Goal, error) {
	if !studyValidID(topicID) {
		return progress.Goal{}, progress.ErrTopicNotFound
	}
	var g progress.Goal
	err := inTx(ctx, r.db, func(tx *sql.Tx) error {
		// Serialise concurrent activations of one user on the rows they already have.
		if _, err := tx.ExecContext(ctx, "SELECT id FROM goals WHERE user_id = ? FOR UPDATE", userID); err != nil {
			return fmt.Errorf("mysql lock goals: %w", err)
		}
		if _, err := tx.ExecContext(ctx, "UPDATE goals SET status = ? WHERE user_id = ? AND status = ? AND topic_id <> ?",
			string(progress.GoalPaused), userID, string(progress.GoalActive), topicID); err != nil {
			return fmt.Errorf("mysql pause goals: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO goals (`+studyGoalCols+`) VALUES (?, ?, ?, ?, ?, ?, ?) AS new
ON DUPLICATE KEY UPDATE status = new.status, level = new.level, effective_from = new.effective_from`,
			newID(), userID, topicID, level, string(progress.GoalActive), effectiveFrom, utc(now)); err != nil {
			return fmt.Errorf("mysql activate goal: %w", err)
		}
		var err error
		g, err = studyScanGoal(tx.QueryRowContext(ctx, "SELECT "+studyGoalCols+" FROM goals WHERE user_id = ? AND topic_id = ?", userID, topicID))
		if err != nil {
			return fmt.Errorf("mysql read goal: %w", err)
		}
		return nil
	})
	if err != nil {
		return progress.Goal{}, err
	}
	return g, nil
}

// --- lesson_progress ---

const studyProgressCols = "user_id, lesson_id, topic_id, day_key, step_read, step_listen, step_write, current_step, sentence_index, started_at, completed_at, " +
	"step_read_at, step_listen_at, step_write_at"

func studyScanProgress(sc interface{ Scan(...any) error }) (progress.LessonProgress, error) {
	var p progress.LessonProgress
	var read, listen, write bool
	var step string
	var completed, readAt, listenAt, writeAt sql.NullTime
	if err := sc.Scan(&p.UserID, &p.LessonID, &p.TopicID, &p.DayKey, &read, &listen, &write, &step, &p.SentenceIndex, &p.StartedAt, &completed,
		&readAt, &listenAt, &writeAt); err != nil {
		return progress.LessonProgress{}, err
	}
	p.Done = map[progress.Step]bool{}
	for s, done := range map[progress.Step]bool{progress.StepRead: read, progress.StepListen: listen, progress.StepWrite: write} {
		if done {
			p.Done[s] = true
		}
	}
	for s, at := range map[progress.Step]sql.NullTime{progress.StepRead: readAt, progress.StepListen: listenAt, progress.StepWrite: writeAt} {
		if at.Valid {
			if p.DoneAt == nil {
				p.DoneAt = map[progress.Step]time.Time{}
			}
			p.DoneAt[s] = timeOf(at)
		}
	}
	p.CurrentStep = progress.Step(step)
	p.StartedAt = utc(p.StartedAt)
	p.CompletedAt = timeOf(completed)
	return p, nil
}

// Get returns the user's progress on a lesson.
func (r *LessonProgress) Get(ctx context.Context, userID, lessonID string) (progress.LessonProgress, bool, error) {
	p, err := studyScanProgress(r.db.QueryRowContext(ctx,
		"SELECT "+studyProgressCols+" FROM lesson_progress WHERE user_id = ? AND lesson_id = ?", userID, lessonID))
	if errors.Is(err, sql.ErrNoRows) {
		return progress.LessonProgress{}, false, nil
	}
	if err != nil {
		return progress.LessonProgress{}, false, fmt.Errorf("mysql get progress: %w", err)
	}
	return p, true, nil
}

// Completed returns the completed lessons, newest first.
func (r *LessonProgress) Completed(ctx context.Context, userID string) ([]progress.LessonProgress, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT "+studyProgressCols+
		" FROM lesson_progress WHERE user_id = ? AND completed_at IS NOT NULL ORDER BY completed_at DESC, lesson_id", userID)
	if err != nil {
		return nil, fmt.Errorf("mysql completed lessons: %w", err)
	}
	defer rows.Close()
	out := []progress.LessonProgress{}
	for rows.Next() {
		p, err := studyScanProgress(rows)
		if err != nil {
			return nil, fmt.Errorf("mysql scan progress: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql completed lessons: %w", err)
	}
	return out, nil
}

// Upsert writes the whole progress of a lesson. Like the MongoDB repository, a zero CompletedAt
// or step time leaves an existing one alone.
func (r *LessonProgress) Upsert(ctx context.Context, p progress.LessonProgress) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO lesson_progress (`+studyProgressCols+`)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) AS new
ON DUPLICATE KEY UPDATE topic_id = new.topic_id, day_key = new.day_key, step_read = new.step_read,
  step_listen = new.step_listen, step_write = new.step_write, current_step = new.current_step,
  sentence_index = new.sentence_index, started_at = new.started_at,
  completed_at = COALESCE(new.completed_at, lesson_progress.completed_at),
  step_read_at = COALESCE(new.step_read_at, lesson_progress.step_read_at),
  step_listen_at = COALESCE(new.step_listen_at, lesson_progress.step_listen_at),
  step_write_at = COALESCE(new.step_write_at, lesson_progress.step_write_at)`,
		p.UserID, p.LessonID, p.TopicID, p.DayKey, p.Done[progress.StepRead], p.Done[progress.StepListen], p.Done[progress.StepWrite],
		string(p.CurrentStep), p.SentenceIndex, utc(p.StartedAt), nullTime(p.CompletedAt),
		nullTime(p.DoneAt[progress.StepRead]), nullTime(p.DoneAt[progress.StepListen]), nullTime(p.DoneAt[progress.StepWrite]))
	if err != nil {
		return fmt.Errorf("mysql upsert progress: %w", err)
	}
	return nil
}

// SetPosition saves the current step and sentence of a lesson already started (no row, no change).
func (r *LessonProgress) SetPosition(ctx context.Context, userID, lessonID string, step progress.Step, sentence int) error {
	_, err := r.db.ExecContext(ctx, "UPDATE lesson_progress SET current_step = ?, sentence_index = ? WHERE user_id = ? AND lesson_id = ?",
		string(step), sentence, userID, lessonID)
	if err != nil {
		return fmt.Errorf("mysql set position: %w", err)
	}
	return nil
}

// StepCounts counts the user's lessons among lessonIDs (nil = every lesson) by steps done. With
// since set it counts steps done and lessons completed at or after since; steps done before the
// step times existed are NULL and never count then.
func (r *LessonProgress) StepCounts(ctx context.Context, userID string, lessonIDs []string, since *time.Time) (progress.StepCounts, error) {
	query := `SELECT CAST(COALESCE(SUM(step_read), 0) AS SIGNED), CAST(COALESCE(SUM(step_listen), 0) AS SIGNED),
  CAST(COALESCE(SUM(step_write), 0) AS SIGNED), CAST(COALESCE(SUM(completed_at IS NOT NULL), 0) AS SIGNED)
FROM lesson_progress WHERE user_id = ?`
	params := []any{userID}
	if since != nil {
		// A NULL comparison adds nothing to the sum.
		query = `SELECT CAST(COALESCE(SUM(step_read_at >= ?), 0) AS SIGNED), CAST(COALESCE(SUM(step_listen_at >= ?), 0) AS SIGNED),
  CAST(COALESCE(SUM(step_write_at >= ?), 0) AS SIGNED), CAST(COALESCE(SUM(completed_at >= ?), 0) AS SIGNED)
FROM lesson_progress WHERE user_id = ?`
		at := utc(*since)
		params = []any{at, at, at, at, userID}
	}
	if lessonIDs != nil {
		if len(lessonIDs) == 0 {
			return progress.StepCounts{}, nil
		}
		query += " AND lesson_id IN (" + placeholders(len(lessonIDs)) + ")"
		params = append(params, args(lessonIDs)...)
	}
	var c progress.StepCounts
	if err := r.db.QueryRowContext(ctx, query, params...).Scan(&c.Read, &c.Listen, &c.Write, &c.Completed); err != nil {
		return progress.StepCounts{}, fmt.Errorf("mysql step counts: %w", err)
	}
	return c, nil
}

// --- study_days ---

// MarkCompleted records a completed lesson on dayKey (the streak counts such days).
func (r *StudyDays) MarkCompleted(ctx context.Context, userID, dayKey string) error {
	_, err := r.db.ExecContext(ctx, "INSERT INTO study_days (user_id, day_key, completed) VALUES (?, ?, 1) ON DUPLICATE KEY UPDATE completed = 1",
		userID, dayKey)
	if err != nil {
		return fmt.Errorf("mysql mark study day: %w", err)
	}
	return nil
}

// CompletedKeys lists the user's days with a completed lesson, oldest first.
func (r *StudyDays) CompletedKeys(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT day_key FROM study_days WHERE user_id = ? AND completed = 1 ORDER BY day_key", userID)
	if err != nil {
		return nil, fmt.Errorf("mysql study days: %w", err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, fmt.Errorf("mysql scan study day: %w", err)
		}
		out = append(out, k)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql study days: %w", err)
	}
	return out, nil
}
