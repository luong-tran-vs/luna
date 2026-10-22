package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/luongtran/luna/backend/internal/ai"
)

var simplifySchema = map[string]any{
	"type": "OBJECT",
	"properties": map[string]any{
		"texts": map[string]any{
			"type": "ARRAY",
			"items": map[string]any{
				"type": "OBJECT",
				"properties": map[string]any{
					"index": map[string]any{"type": "INTEGER"},
					"text":  map[string]any{"type": "STRING"},
				},
				"required": []string{"index", "text"},
			},
		},
	},
	"required": []string{"texts"},
}

// SimplifyWords sends one low-temperature request that replaces the words above the level in the
// generated lessons (added 2026-10-22).
func (c *Client) SimplifyWords(ctx context.Context, req ai.SimplifyRequest) ([]ai.SimplifiedText, error) {
	text, err := c.generate(ctx, "lesson_simplify", simplifyPrompt(req), simplifySchema, 0.2,
		slog.String("level", req.Level), slog.Int("texts", len(req.Texts)))
	if err != nil {
		return nil, err
	}
	var out struct {
		Texts []ai.SimplifiedText `json:"texts"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return nil, fmt.Errorf("gemini: decode simplify: %w", err)
	}
	return out.Texts, nil
}

func simplifyPrompt(req ai.SimplifyRequest) string {
	var b strings.Builder
	fmt.Fprintf(&b, `You are an experienced English teacher editing lessons for Vietnamese learners at CEFR level %s.
Each text below uses a few words that are too hard for this level. In each text, replace only the
listed words with simpler, common words of level %s that keep the meaning; change the few words
around them when needed so the sentence stays natural and correct. Keep every other word, every
sentence, the paragraphs, the line breaks, the names and, in a dialogue, the "Name: line" form.
Return every text whole, with its index.
`, req.Level, req.Level)
	for _, t := range req.Texts {
		fmt.Fprintf(&b, "\nText %d. Words to replace: %s\n%s\n", t.Index, strings.Join(t.Words, ", "), t.Text)
	}
	return b.String()
}
