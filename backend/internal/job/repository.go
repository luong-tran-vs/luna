package job

import (
	"context"
	"time"
)

// Repository stores jobs. Implementations live in internal/storage.
type Repository interface {
	Enqueue(ctx context.Context, j Job) error
	// ClaimNext atomically moves the oldest due pending job to running, increments its
	// attempts and returns it; ok is false when nothing is due.
	ClaimNext(ctx context.Context, now time.Time) (j Job, ok bool, err error)
	Complete(ctx context.Context, id string) error
	// Retry puts a job back to pending, runnable from runAt.
	Retry(ctx context.Context, id string, runAt time.Time, errMsg string) error
	Fail(ctx context.Context, id string, errMsg string) error
	DeletePending(ctx context.Context, lessonID string) error
	DeleteForLesson(ctx context.Context, lessonID string) error
	// ResetRunning returns jobs left running by a previous process to pending.
	ResetRunning(ctx context.Context) (int64, error)
}
