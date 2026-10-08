package ai

import (
	"context"
	"time"
)

// Usage is one request sent to an AI provider, for the admin's usage page: when, which model and
// operation, how it ended and the tokens the provider counted (zero when it gave none).
type Usage struct {
	At    time.Time
	Model string
	// Op names the operation ("annotate", "practice", "image"...).
	Op string
	// Status is the HTTP status, 0 when the request did not get an answer.
	Status        int
	PromptTokens  int
	OutputTokens  int
	ThoughtTokens int
	TotalTokens   int
	Duration      time.Duration
}

// UsageRecorder keeps the requests sent to a provider. It must not block for long or fail the
// request: a lost record is only a gap in the statistics.
type UsageRecorder interface {
	RecordUsage(ctx context.Context, u Usage)
}
