package llm

import (
	"context"
	"encoding/json"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
)

// usageResponse is the token count Gemini (usageMetadata) or Groq (usage) gives with every answer.
type usageResponse struct {
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		ThoughtsTokenCount   int `json:"thoughtsTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
	// Usage is Groq's count (groq.go); its completion tokens include the reasoning ones.
	Usage struct {
		PromptTokens            int `json:"prompt_tokens"`
		CompletionTokens        int `json:"completion_tokens"`
		TotalTokens             int `json:"total_tokens"`
		CompletionTokensDetails struct {
			ReasoningTokens int `json:"reasoning_tokens"`
		} `json:"completion_tokens_details"`
	} `json:"usage"`
}

// recordUsage hands one request to the Usage recorder, if any, with the tokens of raw, the
// response body (nil when there was none); key is the number of the key used, 0 for the main one.
func (c *Client) recordUsage(ctx context.Context, model, op string, key int, start time.Time, status int, raw []byte) {
	if c.Usage == nil {
		return
	}
	u := ai.Usage{At: start, Model: model, Op: op, Key: key, Status: status, Duration: time.Since(start)}
	var r usageResponse
	if len(raw) > 0 && json.Unmarshal(raw, &r) == nil {
		m := r.UsageMetadata
		u.PromptTokens, u.OutputTokens, u.ThoughtTokens, u.TotalTokens =
			m.PromptTokenCount, m.CandidatesTokenCount, m.ThoughtsTokenCount, m.TotalTokenCount
		if g := r.Usage; g.TotalTokens > 0 {
			reasoning := g.CompletionTokensDetails.ReasoningTokens
			u.PromptTokens, u.OutputTokens, u.ThoughtTokens, u.TotalTokens =
				g.PromptTokens, g.CompletionTokens-reasoning, reasoning, g.TotalTokens
		}
	}
	c.Usage.RecordUsage(ctx, u)
}
