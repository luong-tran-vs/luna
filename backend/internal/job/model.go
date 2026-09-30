// Package job runs background work (audio generation, AI annotation) from a MongoDB-backed queue.
package job

import (
	"errors"
	"time"
)

// Type names the work a job does.
type Type string

const (
	TypeTTS      Type = "tts"
	TypeAnnotate Type = "annotate"
)

// Status is the lifecycle state of a job.
type Status string

const (
	StatusPending Status = "pending"
	StatusRunning Status = "running"
	StatusDone    Status = "done"
	StatusFailed  Status = "failed"
)

// MaxAttempts is how many times a job runs before it is marked failed.
const MaxAttempts = 3

// Job is one unit of background work for a lesson revision.
type Job struct {
	ID        string
	Type      Type
	LessonID  string
	Revision  int
	Status    Status
	Attempts  int
	Error     string
	RunAt     time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

type permanentError struct{ err error }

func (e permanentError) Error() string { return e.err.Error() }
func (e permanentError) Unwrap() error { return e.err }

// Permanent marks err as not worth retrying (for example a missing API key).
func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return permanentError{err: err}
}

// IsPermanent reports whether err was wrapped with Permanent.
func IsPermanent(err error) bool {
	var p permanentError
	return errors.As(err, &p)
}
