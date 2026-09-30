package job

import (
	"context"
	"log/slog"
	"time"
)

// Handler does the work of one job. Returning an error retries the job (with backoff) until
// MaxAttempts, unless the error is wrapped with Permanent.
type Handler func(ctx context.Context, j Job) error

// FailedFunc is called once when a job gives up, so the owner can record the failure.
type FailedFunc func(ctx context.Context, j Job, err error)

// backoff is the delay before attempt n+1 after attempt n failed.
var backoff = map[int]time.Duration{1: 30 * time.Second, 2: 2 * time.Minute}

// Worker runs jobs one at a time. One worker is enough: TTS is CPU-bound and AI calls are rare.
type Worker struct {
	repo     Repository
	handlers map[Type]Handler
	onFailed FailedFunc
	now      func() time.Time
	log      *slog.Logger
	wake     chan struct{}
	idle     time.Duration
}

// NewWorker returns a worker dispatching jobs to handlers by type.
func NewWorker(repo Repository, handlers map[Type]Handler, onFailed FailedFunc, now func() time.Time, log *slog.Logger) *Worker {
	return &Worker{
		repo: repo, handlers: handlers, onFailed: onFailed, now: now, log: log,
		wake: make(chan struct{}, 1),
		idle: 2 * time.Second,
	}
}

// Notify wakes an idle worker so a newly enqueued job starts at once.
func (w *Worker) Notify() {
	select {
	case w.wake <- struct{}{}:
	default:
	}
}

// Run resets jobs left running by a previous process, then processes jobs until ctx ends.
func (w *Worker) Run(ctx context.Context) {
	if n, err := w.repo.ResetRunning(ctx); err != nil {
		w.log.WarnContext(ctx, "reset running jobs failed", slog.Any("error", err))
	} else if n > 0 {
		w.log.InfoContext(ctx, "resumed interrupted jobs", slog.Int64("count", n))
	}

	for ctx.Err() == nil {
		if w.RunOnce(ctx) {
			continue
		}
		select {
		case <-ctx.Done():
		case <-w.wake:
		case <-time.After(w.idle):
		}
	}
}

// RunOnce claims and processes at most one due job; it reports whether a job was found.
func (w *Worker) RunOnce(ctx context.Context) bool {
	j, ok, err := w.repo.ClaimNext(ctx, w.now())
	if err != nil {
		if ctx.Err() == nil {
			w.log.WarnContext(ctx, "claim job failed", slog.Any("error", err))
		}
		return false
	}
	if !ok {
		return false
	}
	w.process(ctx, j)
	return true
}

func (w *Worker) process(ctx context.Context, j Job) {
	attrs := []any{
		slog.String("job_id", j.ID), slog.String("type", string(j.Type)),
		slog.String("lesson_id", j.LessonID), slog.Int("attempt", j.Attempts),
	}

	h, ok := w.handlers[j.Type]
	var err error
	if !ok {
		err = Permanent(errUnknownType(j.Type))
	} else {
		err = h(ctx, j)
	}

	// Shutting down: leave the job running; ResetRunning picks it up on the next start.
	if err != nil && ctx.Err() != nil {
		return
	}
	// Record the outcome even if ctx is cancelled right now.
	rctx := context.WithoutCancel(ctx)

	switch {
	case err == nil:
		w.log.InfoContext(ctx, "job done", attrs...)
		w.logIfErr(ctx, w.repo.Complete(rctx, j.ID))
	case IsPermanent(err) || j.Attempts >= MaxAttempts:
		w.log.WarnContext(ctx, "job failed", append(attrs, slog.Any("error", err))...)
		w.logIfErr(ctx, w.repo.Fail(rctx, j.ID, err.Error()))
		if w.onFailed != nil {
			w.onFailed(rctx, j, err)
		}
	default:
		runAt := w.now().Add(backoff[j.Attempts])
		w.log.InfoContext(ctx, "job will retry", append(attrs, slog.Any("error", err), slog.Time("run_at", runAt))...)
		w.logIfErr(ctx, w.repo.Retry(rctx, j.ID, runAt, err.Error()))
	}
}

func (w *Worker) logIfErr(ctx context.Context, err error) {
	if err != nil {
		w.log.ErrorContext(ctx, "update job failed", slog.Any("error", err))
	}
}

type errUnknownType Type

func (e errUnknownType) Error() string { return "job: no handler for type " + string(e) }
