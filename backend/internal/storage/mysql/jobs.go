package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/luongtran/luna/backend/internal/job"
)

// Jobs implements job.Repository on the "jobs" table, a queue several processes may drain at once.
type Jobs struct {
	db *sql.DB
}

// NewJobs returns the Jobs repository.
func NewJobs(db *sql.DB) *Jobs { return &Jobs{db: db} }

var _ job.Repository = (*Jobs)(nil)

// jobsSchema is the DDL of this domain (see migrate.go). lesson_id and target_id are NULL when the
// job has none (a grade job has only a target).
func jobsSchema() []string {
	return []string{`CREATE TABLE IF NOT EXISTS jobs (
	id         CHAR(24)     NOT NULL PRIMARY KEY,
	type       VARCHAR(32)  NOT NULL,
	lesson_id  CHAR(24)     NULL,
	target_id  CHAR(24)     NULL,
	revision   INT          NOT NULL DEFAULT 0,
	status     VARCHAR(16)  NOT NULL,
	attempts   INT          NOT NULL DEFAULT 0,
	error      TEXT         NOT NULL,
	run_at     DATETIME(6)  NOT NULL,
	created_at DATETIME(6)  NOT NULL,
	updated_at DATETIME(6)  NOT NULL,
	KEY jobs_status_run_at (status, run_at),
	KEY jobs_lesson_id (lesson_id)
) ` + tableOptions}
}

const jobsCols = "id, type, lesson_id, target_id, revision, status, attempts, error, run_at, created_at, updated_at"

// jobsNullID stores an empty id as NULL, after checking that a set one has the shape of an id.
func jobsNullID(what, id string) (sql.NullString, error) {
	if id == "" {
		return sql.NullString{}, nil
	}
	if !sessionsIsID(id) {
		return sql.NullString{}, fmt.Errorf("mysql job %s: %q is not a valid id", what, id)
	}
	return sql.NullString{String: id, Valid: true}, nil
}

// Enqueue inserts a pending job (or one with the status j carries).
func (r *Jobs) Enqueue(ctx context.Context, j job.Job) error {
	lesson, err := jobsNullID("lesson id", j.LessonID)
	if err != nil {
		return err
	}
	target, err := jobsNullID("target id", j.TargetID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	status := string(job.StatusPending)
	if j.Status != "" {
		status = string(j.Status)
	}
	runAt := j.RunAt.UTC()
	if runAt.IsZero() {
		runAt = now // a zero time is not storable; no run time means "due now"
	}
	_, err = r.db.ExecContext(ctx,
		"INSERT INTO jobs ("+jobsCols+") VALUES (?, ?, ?, ?, ?, ?, 0, '', ?, ?, ?)",
		newID(), string(j.Type), lesson, target, j.Revision, status, runAt, now, now)
	if err != nil {
		return fmt.Errorf("mysql insert job: %w", err)
	}
	return nil
}

// ClaimNext atomically takes the oldest due pending job. FOR UPDATE SKIP LOCKED lets concurrent
// claimers (several workers or processes) each take a different row and never wait on or double-take one.
func (r *Jobs) ClaimNext(ctx context.Context, now time.Time) (job.Job, bool, error) {
	var (
		out   job.Job
		found bool
	)
	now = now.UTC()
	err := inTx(ctx, r.db, func(tx *sql.Tx) error {
		var id string
		err := tx.QueryRowContext(ctx,
			"SELECT id FROM jobs WHERE status = ? AND run_at <= ? ORDER BY run_at, created_at, id LIMIT 1 FOR UPDATE SKIP LOCKED",
			string(job.StatusPending), now).Scan(&id)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("mysql select job: %w", err)
		}
		if _, err := tx.ExecContext(ctx,
			"UPDATE jobs SET status = ?, attempts = attempts + 1, updated_at = ? WHERE id = ?",
			string(job.StatusRunning), now, id); err != nil {
			return fmt.Errorf("mysql claim job: %w", err)
		}
		out, err = jobsScan(tx.QueryRowContext(ctx, "SELECT "+jobsCols+" FROM jobs WHERE id = ?", id))
		if err != nil {
			return err
		}
		found = true
		return nil
	})
	if err != nil {
		return job.Job{}, false, err
	}
	return out, found, nil
}

// Complete marks a job done.
func (r *Jobs) Complete(ctx context.Context, id string) error {
	return r.jobsSet(ctx, id, "status = ?, error = ''", string(job.StatusDone))
}

// Retry puts a job back to pending, runnable from runAt.
func (r *Jobs) Retry(ctx context.Context, id string, runAt time.Time, errMsg string) error {
	return r.jobsSet(ctx, id, "status = ?, run_at = ?, error = ?", string(job.StatusPending), runAt.UTC(), errMsg)
}

// Fail marks a job failed.
func (r *Jobs) Fail(ctx context.Context, id, errMsg string) error {
	return r.jobsSet(ctx, id, "status = ?, error = ?", string(job.StatusFailed), errMsg)
}

// DeletePending drops queued (not running) jobs of a lesson.
func (r *Jobs) DeletePending(ctx context.Context, lessonID string) error {
	if !sessionsIsID(lessonID) {
		return fmt.Errorf("mysql job lesson id: %q is not a valid id", lessonID)
	}
	if _, err := r.db.ExecContext(ctx, "DELETE FROM jobs WHERE lesson_id = ? AND status = ?",
		lessonID, string(job.StatusPending)); err != nil {
		return fmt.Errorf("mysql delete pending jobs: %w", err)
	}
	return nil
}

// DeleteForLesson drops every job of a lesson.
func (r *Jobs) DeleteForLesson(ctx context.Context, lessonID string) error {
	if !sessionsIsID(lessonID) {
		return fmt.Errorf("mysql job lesson id: %q is not a valid id", lessonID)
	}
	if _, err := r.db.ExecContext(ctx, "DELETE FROM jobs WHERE lesson_id = ?", lessonID); err != nil {
		return fmt.Errorf("mysql delete jobs: %w", err)
	}
	return nil
}

// ResetRunning returns jobs left running by a previous process to pending.
func (r *Jobs) ResetRunning(ctx context.Context) (int64, error) {
	res, err := r.db.ExecContext(ctx, "UPDATE jobs SET status = ?, updated_at = ? WHERE status = ?",
		string(job.StatusPending), time.Now().UTC(), string(job.StatusRunning))
	if err != nil {
		return 0, fmt.Errorf("mysql reset running jobs: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("mysql rows affected: %w", err)
	}
	return n, nil
}

// jobsSet updates the given assignments of one job and stamps updated_at; an unknown id changes nothing.
func (r *Jobs) jobsSet(ctx context.Context, id, assignments string, vals ...any) error {
	if !sessionsIsID(id) {
		return fmt.Errorf("mysql job id: %q is not a valid id", id)
	}
	vals = append(vals, time.Now().UTC(), id)
	if _, err := r.db.ExecContext(ctx, "UPDATE jobs SET "+assignments+", updated_at = ? WHERE id = ?", vals...); err != nil {
		return fmt.Errorf("mysql update job: %w", err)
	}
	return nil
}

func jobsScan(row *sql.Row) (job.Job, error) {
	var (
		j                      job.Job
		typ, status            string
		lesson, target         sql.NullString
		runAt, created, update time.Time
	)
	if err := row.Scan(&j.ID, &typ, &lesson, &target, &j.Revision, &status, &j.Attempts, &j.Error, &runAt, &created, &update); err != nil {
		return job.Job{}, fmt.Errorf("mysql read job: %w", err)
	}
	j.Type, j.Status = job.Type(typ), job.Status(status)
	j.LessonID, j.TargetID = lesson.String, target.String
	j.RunAt, j.CreatedAt, j.UpdatedAt = runAt.UTC(), created.UTC(), update.UTC()
	return j, nil
}
