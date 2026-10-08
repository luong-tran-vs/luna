package gemini

import (
	"context"
	"encoding/json"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
)

// usageResponse is the token count Gemini gives with every answer.
type usageResponse struct {
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		ThoughtsTokenCount   int `json:"thoughtsTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
}

// recordUsage hands one request to the Usage recorder, if any, with the tokens of raw, the
// response body (nil when there was none).
func (c *Client) recordUsage(ctx context.Context, op string, start time.Time, status int, raw []byte) {
	if c.Usage == nil {
		return
	}
	u := ai.Usage{At: start, Model: c.model, Op: op, Status: status, Duration: time.Since(start)}
	var r usageResponse
	if len(raw) > 0 && json.Unmarshal(raw, &r) == nil {
		m := r.UsageMetadata
		u.PromptTokens, u.OutputTokens, u.ThoughtTokens, u.TotalTokens =
			m.PromptTokenCount, m.CandidatesTokenCount, m.ThoughtsTokenCount, m.TotalTokenCount
	}
	c.Usage.RecordUsage(ctx, u)
}
