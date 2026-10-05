package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/luongtran/luna/backend/internal/ai"
)

var reviewIssueArray = map[string]any{
	"type": "ARRAY",
	"items": map[string]any{
		"type": "OBJECT",
		"properties": map[string]any{
			"index":  map[string]any{"type": "INTEGER"},
			"noteVi": map[string]any{"type": "STRING"},
		},
		"required": []string{"index", "noteVi"},
	},
}

// lessonReviewSchema forces the four lists of a lesson review (F22).
var lessonReviewSchema = map[string]any{
	"type": "OBJECT",
	"properties": map[string]any{
		"sentences":    reviewIssueArray,
		"annotations":  reviewIssueArray,
		"translations": reviewIssueArray,
		"answers": map[string]any{
			"type": "ARRAY",
			"items": map[string]any{
				"type": "OBJECT",
				"properties": map[string]any{
					"index":       map[string]any{"type": "INTEGER"},
					"choiceIndex": map[string]any{"type": "INTEGER"},
					"ambiguous":   map[string]any{"type": "BOOLEAN"},
					"noteVi":      map[string]any{"type": "STRING"},
				},
				"required": []string{"index", "ambiguous"},
			},
		},
	},
	"required": []string{"sentences", "annotations", "translations", "answers"},
}

// ReviewLesson sends one low-temperature request that reads over the sentences, annotations,
// comprehension questions and translations of a lesson (F22). The lesson service turns the answer
// into flags.
func (c *Client) ReviewLesson(ctx context.Context, req ai.ReviewRequest) (ai.ReviewResult, error) {
	text, err := c.generate(ctx, "lesson_review", reviewPrompt(req), lessonReviewSchema, 0.1,
		slog.String("level", req.Level), slog.Int("sentences", len(req.Sentences)), slog.Int("questions", len(req.Questions)))
	if err != nil {
		return ai.ReviewResult{}, err
	}
	var out ai.ReviewResult
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return ai.ReviewResult{}, fmt.Errorf("gemini: decode lesson review: %w", err)
	}
	return out, nil
}

func reviewPrompt(req ai.ReviewRequest) string {
	var b strings.Builder
	fmt.Fprintf(&b, `You are a careful English teacher who proofreads a lesson written by another AI for Vietnamese learners at CEFR level %s. The lesson is titled "%s".
You have no answer key. Do not guess what the author intended: decide what is really correct. Report only what you are sure about; when in doubt, report nothing. Write every note in one short Vietnamese sentence.
The lesson may be a dialogue whose lines look like "Name: sentence". Do not correct the spelling of names.
Return a JSON object with:
- sentences: the sentences that are clearly wrong, each with index (the sentence index below) and noteVi. Report a sentence only for a clear grammar error or text that no native speaker would write; do not report a style choice or a sentence that is a little above or below the level.
- annotations: the annotations whose Vietnamese meaning is wrong for the word as used in its sentence (sentenceIndex), each with index (the annotation index below) and noteVi.
- translations: the translation pairs whose Vietnamese and English do not mean the same, or whose English has a grammar error, each with index and noteVi.
- answers: exactly one object per question, in the same order, each with index (copy it), choiceIndex (the 0-based index of the one right option), and ambiguous: true when, in your honest opinion, more than one option is right, or none is, or the question is unclear or has an error; otherwise false. When ambiguous is true, noteVi says why. Do not mark a question ambiguous just to be safe.
Use empty arrays for the lists with nothing to report. Plain text only, no markdown.

Sentences (JSON):
`, req.Level, req.Title)
	numbered := make([]struct {
		Index int    `json:"index"`
		Text  string `json:"text"`
	}, len(req.Sentences))
	for i, s := range req.Sentences {
		numbered[i].Index, numbered[i].Text = i, s
	}
	writeJSONLine(&b, numbered)
	b.WriteString("Annotations (JSON):\n")
	writeJSONLine(&b, nonNilSlice(req.Annotations))
	b.WriteString("Questions (JSON):\n")
	writeJSONLine(&b, nonNilSlice(req.Questions))
	b.WriteString("Translations (JSON):\n")
	writeJSONLine(&b, nonNilSlice(req.Translations))
	return b.String()
}

func nonNilSlice[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

func writeJSONLine(b *strings.Builder, v any) {
	raw, _ := json.Marshal(v)
	b.Write(raw)
	b.WriteString("\n")
}
