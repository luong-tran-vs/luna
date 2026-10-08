package lesson

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"
)

// AskLimit is how many different words or phrases a learner may ask the AI about in one lesson
// (F9). Asking again about one already asked is free; admins have no limit.
const AskLimit = 2

// ErrAskLimit means the learner already asked the AI about AskLimit other words of the lesson.
var ErrAskLimit = errors.New("lesson: AI ask limit reached")

// AskUsageRepository remembers which words each learner asked the AI about, per lesson.
// Implementations live in internal/storage.
type AskUsageRepository interface {
	// Asked lists the (normalized) texts the user asked the AI about in the lesson, oldest first.
	Asked(ctx context.Context, userID, lessonID string) ([]string, error)
	// Add records that the user asked about text in the lesson; recording it again is not an error.
	Add(ctx context.Context, userID, lessonID, text string, at time.Time) error
}

// AskQuota is what a learner may still ask the AI in a lesson. Limit 0 means no limit.
type AskQuota struct {
	Limit int
	Asked []string
}

// WithAskLimit limits the learners' AI asks per lesson (AskLimit); without it they are unlimited.
func (r *Reader) WithAskLimit(usage AskUsageRepository) *Reader {
	r.usage = usage
	return r
}

// AskQuota returns the user's quota of AI asks in the lesson; unlimited for admins or without
// the limit.
func (r *Reader) AskQuota(ctx context.Context, userID, lessonID string, unlimited bool) (AskQuota, error) {
	if r.usage == nil || unlimited {
		return AskQuota{Asked: []string{}}, nil
	}
	asked, err := r.usage.Asked(ctx, userID, lessonID)
	if err != nil {
		return AskQuota{}, fmt.Errorf("lesson: asked words: %w", err)
	}
	if asked == nil {
		asked = []string{}
	}
	return AskQuota{Limit: AskLimit, Asked: asked}, nil
}

// checkAskQuota returns ErrAskLimit when text would be one word too many.
func checkAskQuota(q AskQuota, text string) error {
	if q.Limit > 0 && !slices.Contains(q.Asked, text) && len(q.Asked) >= q.Limit {
		return ErrAskLimit
	}
	return nil
}

// recordAsk counts text against the user's quota once it was answered.
func (r *Reader) recordAsk(ctx context.Context, q AskQuota, userID, lessonID, text string) (AskQuota, error) {
	if q.Limit == 0 || slices.Contains(q.Asked, text) {
		return q, nil
	}
	if err := r.usage.Add(ctx, userID, lessonID, text, time.Now().UTC()); err != nil {
		return q, fmt.Errorf("lesson: record ask: %w", err)
	}
	q.Asked = append(slices.Clone(q.Asked), text)
	return q, nil
}

// AskAs is Ask for one user: a word not asked before in this lesson counts against the user's
// quota (ErrAskLimit once it is used up), asking again about one already asked is free, and
// admins (unlimited) are not counted. It returns the quota after the ask.
func (r *Reader) AskAs(ctx context.Context, userID, lessonID, q string, sentence int, unlimited bool) (LookupResult, bool, AskQuota, error) {
	quota, err := r.AskQuota(ctx, userID, lessonID, unlimited)
	if err != nil {
		return LookupResult{}, false, AskQuota{}, err
	}
	text := normalize(q)
	if err := checkAskQuota(quota, text); err != nil {
		return LookupResult{}, false, quota, err
	}
	res, cached, err := r.Ask(ctx, lessonID, q, sentence)
	if err != nil {
		return LookupResult{}, false, quota, err
	}
	quota, err = r.recordAsk(ctx, quota, userID, lessonID, text)
	return res, cached, quota, err
}
