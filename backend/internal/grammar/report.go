package grammar

import (
	"context"
	"time"
)

// A learner can report an exercise they think is wrong (F21). Reports are how mistakes the AI check
// missed get found; the admin sees the reported exercises and resolves them.

// ReportReason is why the learner reported the exercise.
type ReportReason string

const (
	ReasonWrongAnswer ReportReason = "wrong_answer"
	ReasonAmbiguous   ReportReason = "ambiguous"
	ReasonTypo        ReportReason = "typo"
	ReasonOther       ReportReason = "other"
)

// ReportStatus is where a report stands.
type ReportStatus string

const (
	ReportOpen     ReportStatus = "open"
	ReportResolved ReportStatus = "resolved"
)

// Report is one learner's report on one exercise. A learner has at most one report per exercise.
type Report struct {
	ID         string
	UserID     string
	PointID    string
	ExerciseID string
	Reason     ReportReason
	// Note is an optional free-text explanation, at most 300 characters.
	Note       string
	Status     ReportStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time
	ResolvedAt time.Time
}

// ReportRepository stores exercise reports. Implementations live in internal/storage.
type ReportRepository interface {
	// Upsert stores r for (r.UserID, r.PointID, r.ExerciseID): a new report is inserted with an id; an existing
	// one keeps its id and CreatedAt, takes the new Reason and Note, and becomes open again.
	Upsert(ctx context.Context, r Report) error
	// ListOpen returns every open report, newest first.
	ListOpen(ctx context.Context) ([]Report, error)
	// Resolve marks the open reports on (pointID, exerciseID) resolved at the given time and returns how many.
	Resolve(ctx context.Context, pointID, exerciseID string, at time.Time) (int, error)
}
