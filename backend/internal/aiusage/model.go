// Package aiusage keeps the requests sent to the AI provider and sums them up for the admin: per
// minute (the free tier limits requests and tokens per minute), per day and per operation.
package aiusage

import (
	"context"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
)

// Keep is how long requests are kept.
const Keep = 30 * 24 * time.Hour

// Repository stores the requests. Implementations live in internal/storage.
type Repository interface {
	Add(ctx context.Context, u ai.Usage) error
	// Since returns the requests made at or after from, oldest first.
	Since(ctx context.Context, from time.Time) ([]ai.Usage, error)
	// DeleteBefore removes the requests made before t.
	DeleteBefore(ctx context.Context, t time.Time) error
}

// Outcome sorts a request by its HTTP status.
type Outcome string

const (
	OutcomeOK    Outcome = "ok"
	OutcomeQuota Outcome = "quota"
	OutcomeError Outcome = "error"
)

// OutcomeOf is the outcome of a request with HTTP status s (0: no answer).
func OutcomeOf(s int) Outcome {
	switch {
	case s == 200:
		return OutcomeOK
	case s == 429:
		return OutcomeQuota
	default:
		return OutcomeError
	}
}

// Totals counts requests and tokens.
type Totals struct {
	Requests     int
	Errors       int
	Quota        int
	PromptTokens int
	OutputTokens int
	TotalTokens  int
}

func (t *Totals) add(u ai.Usage) {
	t.Requests++
	switch OutcomeOf(u.Status) {
	case OutcomeQuota:
		t.Quota++
	case OutcomeError:
		t.Errors++
	}
	t.PromptTokens += u.PromptTokens
	t.OutputTokens += u.OutputTokens
	t.TotalTokens += u.TotalTokens
}

// Minute is one minute of the last hour.
type Minute struct {
	Start time.Time
	Totals
}

// ModelTotals are the totals of one model.
type ModelTotals struct {
	Model string
	Totals
}

// OpTotals are the totals of one operation.
type OpTotals struct {
	Op string
	Totals
}

// Summary is what the admin page shows.
type Summary struct {
	Now time.Time
	// LastMinute covers the 60 seconds before Now, per model (the limits are per model).
	LastMinute []ModelTotals
	// PeakMinute is the busiest minute of the last hour, all models together.
	PeakMinute Totals
	// Minutes are the 60 minutes before Now, oldest first, all models together.
	Minutes []Minute
	// DayStart is midnight Pacific time, when Google resets the daily limits.
	DayStart time.Time
	// Today is since DayStart, per model.
	Today []ModelTotals
	// Week is the last 7 days, per operation, most requests first.
	Week []OpTotals
	// Recent are the last requests, newest first.
	Recent []ai.Usage
}
