package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/luongtran/luna/backend/internal/ai"
)

// maxSpotItems bounds one request for "Bắn lỗi" sentences.
const maxSpotItems = 10

var spotSchema = map[string]any{
	"type": "OBJECT",
	"properties": map[string]any{
		"items": map[string]any{
			"type": "ARRAY",
			"items": map[string]any{
				"type": "OBJECT",
				"properties": map[string]any{
					"sentence":    map[string]any{"type": "STRING"},
					"wrong":       map[string]any{"type": "STRING"},
					"right":       map[string]any{"type": "STRING"},
					"distractors": map[string]any{"type": "ARRAY", "items": map[string]any{"type": "STRING"}},
					"ruleVi":      map[string]any{"type": "STRING"},
				},
				"required": []string{"sentence", "wrong", "right", "distractors", "ruleVi"},
			},
		},
	},
	"required": []string{"items"},
}

// GrammarSpotItems sends one low-temperature request for the sentences of the "Bắn lỗi" game of one
// grammar point (F26). The service checks every sentence before keeping it.
func (c *Client) GrammarSpotItems(ctx context.Context, req ai.GrammarSpotRequest) ([]ai.GrammarSpotItem, error) {
	text, err := c.generate(ctx, "grammar_spot", spotPrompt(req), spotSchema, 0.3,
		slog.String("level", req.Level), slog.String("point", req.TitleEn))
	if err != nil {
		return nil, err
	}
	var out struct {
		Items []ai.GrammarSpotItem `json:"items"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return nil, fmt.Errorf("llm: decode grammar spot items: %w", err)
	}
	return out.Items, nil
}

func spotPrompt(req ai.GrammarSpotRequest) string {
	count := req.Count
	if count <= 0 || count > maxSpotItems {
		count = maxSpotItems
	}
	var b strings.Builder
	fmt.Fprintf(&b, `You are an English teacher writing a "find the mistake" game for Vietnamese learners at CEFR level %s.
Grammar point: %s. Pattern: %s.
Write up to %d different short sentences (3 to 15 words) at that level, in everyday contexts. Each sentence must contain exactly ONE mistake, and the mistake must be about this grammar point. Everything else in the sentence must be correct.
For each sentence return:
- sentence: the sentence WITH the mistake.
- wrong: the wrong part, 1 to 3 consecutive words copied exactly from sentence; it must appear only once in sentence.
- right: the words that replace wrong to make the sentence correct.
- distractors: exactly 2 other replacements for wrong that are still wrong in this sentence (typical learner mistakes of the same grammar point), different from each other, from right and from wrong.
- ruleVi: the rule in one short Vietnamese sentence.
Only one replacement may make the sentence correct: right. Check every distractor by putting it in place of wrong: if the sentence becomes grammatical and makes sense, choose another distractor (prefer a wrong form of the same word, e.g. "go / going / went" for "goes").
The mistake must be a clear grammar error that a teacher would mark wrong, not a sentence that is only a little odd or illogical. Avoid sentences where more than one word could fit the place of the mistake. If the grammar point cannot give such clear mistakes, write fewer sentences.
Do not use names of real people. Plain text only, no markdown.
`, req.Level, req.TitleEn, req.Pattern, count)
	if len(req.Examples) > 0 {
		b.WriteString("Example sentences of the point (JSON):\n")
		writeJSONLine(&b, req.Examples)
	}
	if len(req.Existing) > 0 {
		b.WriteString("Sentences the game already has; do not repeat them (JSON):\n")
		writeJSONLine(&b, req.Existing)
	}
	return b.String()
}
