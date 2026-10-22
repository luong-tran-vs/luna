package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/luongtran/luna/backend/internal/ai"
)

var _ ai.SpeakChecker = (*Client)(nil)

var speakSchema = map[string]any{
	"type": "OBJECT",
	"properties": map[string]any{
		"answers": map[string]any{
			"type": "ARRAY",
			"items": map[string]any{
				"type": "OBJECT",
				"properties": map[string]any{
					"index":     map[string]any{"type": "INTEGER"},
					"corrected": map[string]any{"type": "STRING"},
					"noteVi":    map[string]any{"type": "STRING"},
				},
				"required": []string{"index", "corrected", "noteVi"},
			},
		},
	},
	"required": []string{"answers"},
}

// CheckSpeaking sends one generateContent request that corrects the learner's spoken answers of
// the Speaking step (added 2026-10-22).
func (c *Client) CheckSpeaking(ctx context.Context, req ai.SpeakCheckRequest) ([]ai.SpeakFeedback, error) {
	text, err := c.generate(ctx, "speak_check", speakPrompt(req), speakSchema, 0.2, slog.Int("answers", len(req.Answers)))
	if err != nil {
		return nil, err
	}
	var out struct {
		Answers []ai.SpeakFeedback `json:"answers"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return nil, fmt.Errorf("gemini: decode speaking check: %w", err)
	}
	return out.Answers, nil
}

func speakPrompt(req ai.SpeakCheckRequest) string {
	var b strings.Builder
	fmt.Fprintf(&b, `You check short spoken answers of a Vietnamese learner of English at CEFR level %s.
The answers were recognised from speech, so ignore capital letters, punctuation and filler words, and
do not correct a word that is only a likely recognition slip of the right word.
For each answer return its index, corrected: the answer with the fewest changes that make it correct
and natural (exactly the answer when it is already fine), and noteVi: one short Vietnamese sentence
naming the main mistake and the rule (for example "Sau listen cần to: listen to music."), empty when
the answer is fine. Keep the learner's ideas. Plain text only, no markdown.

`, req.Level)
	for i, a := range req.Answers {
		fmt.Fprintf(&b, "%d. Question: %s\n   Answer: %s\n", i, a.Question, a.Said)
	}
	return b.String()
}
