package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/luongtran/luna/backend/internal/ai"
)

var fixQuestionSchema = map[string]any{
	"type": "OBJECT",
	"properties": map[string]any{
		"prompt":        map[string]any{"type": "STRING"},
		"options":       map[string]any{"type": "ARRAY", "items": map[string]any{"type": "STRING"}},
		"answerIndex":   map[string]any{"type": "INTEGER"},
		"explanationVi": map[string]any{"type": "STRING"},
	},
	"required": []string{"prompt", "options", "answerIndex", "explanationVi"},
}

// fixSchemas forces the fields of the corrected item, one schema per area.
var fixSchemas = map[string]map[string]any{
	"sentence":    fixSchema(map[string]any{"text": map[string]any{"type": "STRING"}}),
	"annotation":  fixSchema(map[string]any{"meaningVi": map[string]any{"type": "STRING"}}),
	"question":    fixSchema(map[string]any{"question": fixQuestionSchema}),
	"translation": fixSchema(map[string]any{"vi": map[string]any{"type": "STRING"}, "en": map[string]any{"type": "STRING"}}),
}

func fixSchema(fields map[string]any) map[string]any {
	props := map[string]any{"noteVi": map[string]any{"type": "STRING"}}
	required := []string{"noteVi"}
	for k, v := range fields {
		props[k] = v
		required = append(required, k)
	}
	return map[string]any{"type": "OBJECT", "properties": props, "required": required}
}

// SuggestFix sends one low-temperature request for a corrected version of one flagged item (F22).
func (c *Client) SuggestFix(ctx context.Context, req ai.FixRequest) (ai.FixResult, error) {
	schema, ok := fixSchemas[req.Area]
	if !ok {
		return ai.FixResult{}, fmt.Errorf("gemini: suggest fix: unknown area %q", req.Area)
	}
	text, err := c.generate(ctx, "lesson_fix", fixPrompt(req), schema, 0.2,
		slog.String("level", req.Level), slog.String("area", req.Area))
	if err != nil {
		return ai.FixResult{}, err
	}
	var out ai.FixResult
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return ai.FixResult{}, fmt.Errorf("gemini: decode lesson fix: %w", err)
	}
	return out, nil
}

func fixPrompt(req ai.FixRequest) string {
	var b strings.Builder
	fmt.Fprintf(&b, `You are a careful English teacher who corrects a lesson written by another AI for Vietnamese learners at CEFR level %s. The lesson is titled "%s".
A proofreader flagged one item. Return a corrected version of that item only, changed as little as needed, keeping the level and the meaning of the lesson. If you find the item already correct, return it unchanged and say so in noteVi.
noteVi: one short Vietnamese sentence saying what you changed and why.
The lesson may be a dialogue whose lines look like "Name: sentence": keep the "Name: " at the start of such a sentence and do not change names.
Plain text only, no markdown.
Problem reported (Vietnamese): %s

Lesson sentences (JSON):
`, req.Level, req.Title, req.Problem)
	writeJSONLine(&b, nonNilSlice(req.Sentences))
	switch req.Area {
	case "sentence":
		fmt.Fprintf(&b, "Flagged item: the sentence at index %d. Return text: the corrected sentence. It must still fit between the sentences around it.\n", req.Sentence)
	case "annotation":
		b.WriteString("Flagged item: this annotation (a word of the sentence at sentenceIndex with its Vietnamese meaning). Return meaningVi: the right short Vietnamese meaning of the word as used in that sentence.\n")
		writeJSONLine(&b, req.Annotation)
	case "question":
		b.WriteString("Flagged item: this multiple-choice comprehension question about the lesson. Return question: prompt and options in English, exactly one right option, answerIndex its 0-based index, explanationVi a short Vietnamese explanation of the answer. Keep the same number of options.\n")
		writeJSONLine(&b, req.Question)
	case "translation":
		b.WriteString("Flagged item: this translation exercise (a Vietnamese sentence and its English answer, which learners build from word tiles). Return vi and en with the same meaning; en must be natural, correct English using words of the lesson where possible.\n")
		writeJSONLine(&b, req.Translation)
	}
	return b.String()
}
