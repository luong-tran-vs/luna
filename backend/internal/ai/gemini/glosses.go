package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/luongtran/luna/backend/internal/ai"
)

var _ ai.GlossProvider = (*Client)(nil)

// glossesSchema is one entry per content word or phrase of the lesson, with its meaning there.
var glossesSchema = map[string]any{
	"type": "OBJECT",
	"properties": map[string]any{
		"glosses": map[string]any{
			"type": "ARRAY",
			"items": map[string]any{
				"type": "OBJECT",
				"properties": map[string]any{
					"text":          map[string]any{"type": "STRING"},
					"lemma":         map[string]any{"type": "STRING"},
					"meaningVi":     map[string]any{"type": "STRING"},
					"sentenceIndex": map[string]any{"type": "INTEGER"},
					"pos": map[string]any{
						"type": "STRING",
						"enum": []string{"noun", "verb", "adjective", "adverb", "pronoun", "preposition", "conjunction", "determiner", "interjection", "phrasal verb", "phrase"},
					},
					"phrase":   map[string]any{"type": "STRING"},
					"phraseVi": map[string]any{"type": "STRING"},
				},
				"required": []string{"text", "lemma", "meaningVi", "sentenceIndex", "pos", "phrase", "phraseVi"},
			},
		},
	},
	"required": []string{"glosses"},
}

// Glosses sends one generateContent request for the meaning in context of every content word of a
// lesson (added 2026-10-20); the lesson service checks them against its sentences.
func (c *Client) Glosses(ctx context.Context, req ai.GlossRequest) ([]ai.Gloss, error) {
	text, err := c.generate(ctx, "glosses", glossesPrompt(req), glossesSchema, 0.2,
		slog.Int("sentences", len(req.Sentences)))
	if err != nil {
		return nil, err
	}
	var out struct {
		Glosses []ai.Gloss `json:"glosses"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return nil, fmt.Errorf("gemini: decode glosses: %w", err)
	}
	return out.Glosses, nil
}

func glossesPrompt(req ai.GlossRequest) string {
	var b strings.Builder
	fmt.Fprintf(&b, `You help Vietnamese learners of English at CEFR level %s read a short lesson. When they tap a word,
they first see what it means in this very sentence. Write that for every content word of the lesson.
Rules:
- One entry per noun, main verb, adjective and adverb, and per phrasal verb, collocation or idiom (one entry
  for the whole phrase, e.g. "turn on", "a lot", "take a break"). Skip articles, pronouns, auxiliary verbs
  ("be", "do", "have" as helpers), prepositions and conjunctions unless part of such a phrase.
- A word used again with the same meaning gets one entry, in the first sentence where it appears; with
  another meaning, another entry.
- text: the word or phrase exactly as written in the sentence. lemma: its dictionary form in lower case.
- sentenceIndex: the number of the sentence it comes from.
- meaningVi: its Vietnamese meaning in this sentence only, 1 to 4 words, the everyday word a Vietnamese
  speaker would use (for "relax" after school: "thư giãn"), no explanations.
- pos: its part of speech as used here.
- phrase: 3 to 7 words of the sentence around it, copied exactly ("a great way to relax");
  phraseVi: a natural Vietnamese translation of that phrase.
Sentences:
`, req.Level)
	for i, s := range req.Sentences {
		fmt.Fprintf(&b, "%d: %s\n", i, s)
	}
	return b.String()
}
