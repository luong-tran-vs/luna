package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/luongtran/luna/backend/internal/ai"
)

var _ ai.CurriculumWriter = (*Client)(nil)

var outlineSchema = map[string]any{
	"type": "OBJECT",
	"properties": map[string]any{
		"topics": map[string]any{
			"type": "ARRAY",
			"items": map[string]any{
				"type": "OBJECT",
				"properties": map[string]any{
					"name":    map[string]any{"type": "STRING"},
					"canDoVi": map[string]any{"type": "STRING"},
					"grammar": map[string]any{"type": "ARRAY", "items": map[string]any{"type": "STRING"}},
				},
				"required": []string{"name", "canDoVi", "grammar"},
			},
		},
	},
	"required": []string{"topics"},
}

var slotsSchema = map[string]any{
	"type": "OBJECT",
	"properties": map[string]any{
		"lessons": map[string]any{
			"type": "ARRAY",
			"items": map[string]any{
				"type": "OBJECT",
				"properties": map[string]any{
					"title":   map[string]any{"type": "STRING"},
					"idea":    map[string]any{"type": "STRING"},
					"grammar": map[string]any{"type": "ARRAY", "items": map[string]any{"type": "STRING"}},
				},
				"required": []string{"title", "idea", "grammar"},
			},
		},
	},
	"required": []string{"lessons"},
}

// OutlineCurriculum asks for the topics of a level's course in one request (added 2026-10-23).
func (c *Client) OutlineCurriculum(ctx context.Context, req ai.OutlineRequest) ([]ai.OutlineTopic, error) {
	text, err := c.generate(ctx, "curriculum_draft", outlinePrompt(req), outlineSchema, 0.5,
		slog.String("level", req.Level), slog.Int("retry", len(req.PreviousErrors)))
	if err != nil {
		return nil, err
	}
	var out struct {
		Topics []ai.OutlineTopic `json:"topics"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return nil, fmt.Errorf("llm: decode curriculum outline: %w", err)
	}
	return out.Topics, nil
}

// WriteTopicSlots asks for the lessons of one topic of a course in one request (added 2026-10-23).
func (c *Client) WriteTopicSlots(ctx context.Context, req ai.SlotsRequest) ([]ai.SlotDraft, error) {
	text, err := c.generate(ctx, "curriculum_draft_topic", slotsPrompt(req), slotsSchema, 0.7,
		slog.String("level", req.Level), slog.String("topic", req.Topic.Name))
	if err != nil {
		return nil, err
	}
	var out struct {
		Lessons []ai.SlotDraft `json:"lessons"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return nil, fmt.Errorf("llm: decode curriculum lessons: %w", err)
	}
	return out.Lessons, nil
}

func writePoints(b *strings.Builder, points []ai.CurriculumPoint) {
	for _, p := range points {
		fmt.Fprintf(b, "- %s: %s (%s)\n", p.ID, p.Title, p.Pattern)
	}
}

func outlinePrompt(req ai.OutlineRequest) string {
	var b strings.Builder
	fmt.Fprintf(&b, `You are an experienced English teacher writing the course of CEFR level %[1]s for Vietnamese adult
learners. The course is a list of %[2]d to %[3]d topics in teaching order. Each topic teaches one to three
grammar points of the list below, so that the points come in a sensible order (easier ones first) and
every point of the list is taught by at least one topic. A later topic reuses the words and grammar of
the earlier ones. Choose topics learners at this level can talk and read about, and that differ from
the topics of the lower levels' courses.
`, req.Level, req.MinTopics, req.MaxTopics)
	writeProfile(&b, req.Level)
	fmt.Fprintf(&b, `
For each topic give:
- "name": the topic's name in Vietnamese. Prefer a name of the existing topics below, written exactly
  as listed; give a new name only when none fits.
- "canDoVi": in Vietnamese, what the learner can do after the topic, written to follow "Sau chủ đề này
  bạn có thể " (for example "kể về một chuyến đi đã qua và hỏi bạn về kỳ nghỉ (past simple)."), one sentence.
- "grammar": the ids of the grammar points the topic teaches, from the list only.

Grammar points of %[1]s in the order they are best taught (id: name (pattern)):
`, req.Level)
	writePoints(&b, req.Points)
	if len(req.Suggestions) > 0 {
		b.WriteString("\nSuggested topics for this level, with why they fit (a starting point, not a must):\n")
		for _, s := range req.Suggestions {
			fmt.Fprintf(&b, "- %s: %s\n", s.Name, s.Why)
		}
	}
	if len(req.Topics) > 0 {
		fmt.Fprintf(&b, "\nExisting topics: %s\n", strings.Join(req.Topics, "; "))
	}
	if len(req.Earlier) > 0 {
		fmt.Fprintf(&b, "\nTopics of the lower levels' courses: %s\n", strings.Join(req.Earlier, "; "))
	}
	if len(req.PreviousErrors) > 0 {
		fmt.Fprintf(&b, "\nYour previous answer could not be used: %s. Fix these.\n", strings.Join(req.PreviousErrors, "; "))
	}
	return b.String()
}

func slotsPrompt(req ai.SlotsRequest) string {
	var b strings.Builder
	fmt.Fprintf(&b, `You are an experienced English teacher writing the lessons of one topic of a CEFR level %[1]s course
for Vietnamese adult learners. Each lesson is a short text that an AI will write later from your
title and idea.

Topic: %[2]s
After the topic the learner can: %[3]s
Grammar points of the topic (id: name (pattern)):
`, req.Level, req.Topic.Name, req.Topic.CanDoVi)
	writePoints(&b, req.Points)
	writeProfile(&b, req.Level)
	fmt.Fprintf(&b, `
Plan only stories that can be told with the grammar of this level and the levels below: no story
that needs a tense the learners have not met (at A1, no past events). Set them in situations Vietnamese
adults meet, each lesson in a different situation. Do not name a kind of text (email, blog post): the
text will be a dialogue or a reading, chosen when it is written.
`)
	if req.Main > 0 {
		fmt.Fprintf(&b, `
Write %[1]d lessons: the first %[2]d are the main lessons, which together teach every grammar point of
the topic; the other %[3]d are extra lessons to practise the same points in new situations.`, req.Main+req.Extra, req.Main, req.Extra)
	} else {
		// The topic keeps its lessons; only extra ones are added (their titles are in Avoid).
		fmt.Fprintf(&b, `
The topic already has its lessons. Write %d more extra lessons to practise its grammar points in new
situations, different from the lessons it has.`, req.Extra)
	}
	b.WriteString(` For each lesson give:
- "title": a short, catchy English title (2 to 6 words), different from every other title.
- "idea": in English, one or two sentences on what happens in the text: who, where, what they do or
  talk about, so that the grammar point is used naturally several times.
- "grammar": the ids of the grammar points the lesson uses, from the topic's list; the first is the
  point the lesson is built on.
`)
	if len(req.Avoid) > 0 {
		fmt.Fprintf(&b, "\nTitles already used elsewhere, do not reuse them: %s\n", strings.Join(req.Avoid, "; "))
	}
	return b.String()
}
