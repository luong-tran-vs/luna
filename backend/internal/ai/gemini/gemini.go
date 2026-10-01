// Package gemini implements ai.Provider with the Gemini generateContent REST API.
package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
)

const (
	defaultBaseURL = "https://generativelanguage.googleapis.com"
	// maxResponse bounds the response body; an annotation with questions stays far below it.
	maxResponse = 2 << 20
)

// Client calls Gemini. BaseURL can be replaced in tests.
type Client struct {
	BaseURL string
	apiKey  string
	model   string
	client  *http.Client
	log     *slog.Logger
}

// New returns a Gemini provider. An empty apiKey makes every call return ai.ErrNotConfigured.
func New(apiKey, model string, client *http.Client, log *slog.Logger) *Client {
	return &Client{BaseURL: defaultBaseURL, apiKey: apiKey, model: model, client: client, log: log}
}

var _ ai.Provider = (*Client)(nil)

var stringArray = map[string]any{"type": "ARRAY", "items": map[string]any{"type": "STRING"}}

// annotateSchema forces an object with the annotations and, optionally, the questions, the
// grammar note and the writing prompt (F15), so a missing part never fails the request.
var annotateSchema = map[string]any{
	"type": "OBJECT",
	"properties": map[string]any{
		"annotations": map[string]any{
			"type": "ARRAY",
			"items": map[string]any{
				"type": "OBJECT",
				"properties": map[string]any{
					"text":          map[string]any{"type": "STRING"},
					"lemma":         map[string]any{"type": "STRING"},
					"meaningVi":     map[string]any{"type": "STRING"},
					"sentenceIndex": map[string]any{"type": "INTEGER"},
				},
				"required": []string{"text", "lemma", "meaningVi", "sentenceIndex"},
			},
		},
		"questions": map[string]any{
			"type": "ARRAY",
			"items": map[string]any{
				"type": "OBJECT",
				"properties": map[string]any{
					"prompt":        map[string]any{"type": "STRING"},
					"options":       stringArray,
					"answerIndex":   map[string]any{"type": "INTEGER"},
					"explanationVi": map[string]any{"type": "STRING"},
				},
				"required": []string{"prompt", "options", "answerIndex", "explanationVi"},
			},
		},
		"grammarNote": map[string]any{
			"type": "OBJECT",
			"properties": map[string]any{
				"title":    map[string]any{"type": "STRING"},
				"bodyVi":   map[string]any{"type": "STRING"},
				"examples": stringArray,
			},
			"required": []string{"title", "bodyVi", "examples"},
		},
		"writingPrompt": map[string]any{"type": "STRING"},
	},
	"required": []string{"annotations"},
}

// generateSchema forces a JSON array of lesson drafts.
var generateSchema = map[string]any{
	"type": "ARRAY",
	"items": map[string]any{
		"type": "OBJECT",
		"properties": map[string]any{
			"title":   map[string]any{"type": "STRING"},
			"content": map[string]any{"type": "STRING"},
		},
		"required": []string{"title", "content"},
	},
}

type part struct {
	Text string `json:"text"`
}

type content struct {
	Role  string `json:"role,omitempty"`
	Parts []part `json:"parts"`
}

type request struct {
	Contents         []content      `json:"contents"`
	GenerationConfig map[string]any `json:"generationConfig"`
}

type response struct {
	Candidates []struct {
		Content content `json:"content"`
	} `json:"candidates"`
}

// Annotate sends one generateContent request for the whole lesson: annotations, questions,
// grammar note and writing prompt.
func (c *Client) Annotate(ctx context.Context, sentences []string, level string) (ai.LessonExtras, error) {
	text, err := c.generate(ctx, "annotate", annotatePrompt(sentences, level), annotateSchema, 0.2,
		slog.Int("sentences", len(sentences)))
	if err != nil {
		return ai.LessonExtras{}, err
	}
	var out ai.LessonExtras
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return ai.LessonExtras{}, fmt.Errorf("gemini: decode annotations: %w", err)
	}
	return out, nil
}

// GenerateLessons sends one generateContent request for the whole batch. A high temperature
// keeps the lessons of a batch different from each other.
func (c *Client) GenerateLessons(ctx context.Context, req ai.GenerateRequest) ([]ai.LessonDraft, error) {
	text, err := c.generate(ctx, "generate", generatePrompt(req), generateSchema, 0.9,
		slog.Int("count", req.Count), slog.Int("words", req.Words))
	if err != nil {
		return nil, err
	}
	var out []ai.LessonDraft
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return nil, fmt.Errorf("gemini: decode drafts: %w", err)
	}
	return out, nil
}

// generate sends a JSON-mode request and returns the text of the first candidate.
func (c *Client) generate(ctx context.Context, op, prompt string, schema map[string]any, temperature float64,
	attrs ...slog.Attr,
) (string, error) {
	if c.apiKey == "" {
		return "", ai.ErrNotConfigured
	}

	body, err := json.Marshal(request{
		Contents: []content{{Role: "user", Parts: []part{{Text: prompt}}}},
		GenerationConfig: map[string]any{
			"temperature":      temperature,
			"responseMimeType": "application/json",
			"responseSchema":   schema,
		},
	})
	if err != nil {
		return "", fmt.Errorf("gemini: encode request: %w", err)
	}
	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent", c.BaseURL, c.model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("gemini: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", c.apiKey)

	start := time.Now()
	resp, err := c.client.Do(req)
	if err != nil {
		c.logRequest(ctx, op, start, 0, attrs)
		return "", fmt.Errorf("gemini: request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // body fully read below
	c.logRequest(ctx, op, start, resp.StatusCode, attrs)

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponse))
	if err != nil {
		return "", fmt.Errorf("gemini: read response: %w", err)
	}
	if err := statusError(resp.StatusCode, raw); err != nil {
		return "", err
	}

	var r response
	if err := json.Unmarshal(raw, &r); err != nil {
		return "", fmt.Errorf("gemini: decode response: %w", err)
	}
	if len(r.Candidates) == 0 || len(r.Candidates[0].Content.Parts) == 0 {
		return "", errors.New("gemini: empty response")
	}
	return r.Candidates[0].Content.Parts[0].Text, nil
}

func statusError(status int, body []byte) error {
	switch {
	case status == http.StatusOK:
		return nil
	case status == http.StatusTooManyRequests:
		return ai.ErrQuota
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return ai.ErrInvalidKey
	case status == http.StatusBadRequest && strings.Contains(string(body), "API key"):
		return ai.ErrInvalidKey
	default:
		return fmt.Errorf("gemini: status %d", status)
	}
}

// logRequest records one line per call so requests can be counted. It never logs the key
// or the lesson text.
func (c *Client) logRequest(ctx context.Context, op string, start time.Time, status int, extra []slog.Attr) {
	attrs := []slog.Attr{
		slog.String("provider", "gemini"),
		slog.String("model", c.model),
		slog.String("op", op),
		slog.Int("status", status),
		slog.Int64("duration_ms", time.Since(start).Milliseconds()),
	}
	c.log.LogAttrs(ctx, slog.LevelInfo, "ai request", append(attrs, extra...)...)
}

func annotatePrompt(sentences []string, level string) string {
	var b strings.Builder
	fmt.Fprintf(&b, `You help Vietnamese learners of English at CEFR level %[1]s.
Read the numbered lesson sentences below and return a JSON object with:

1. annotations: pick 8 to 25 words or phrases worth learning at this level
(phrasal verbs, collocations and idioms count as one item, e.g. "give up").
For each item return:
- text: copied exactly as it appears in the sentence (same spelling and inflection),
- lemma: the dictionary form (e.g. "went" -> "go", "gave up" -> "give up"),
- meaningVi: a short Vietnamese meaning that fits this context,
- sentenceIndex: the number of the sentence containing it.
Do not invent text that is not in the sentences.

2. questions: 3 to 5 multiple-choice questions in English that check understanding of the lesson,
at CEFR %[1]s. Each has a prompt, exactly 4 different options, answerIndex (0 to 3) of the only
correct option, and explanationVi: a short Vietnamese explanation pointing to the lesson.

3. grammarNote: one notable grammar point used in the lesson, for this level:
- title: a short Vietnamese title (e.g. "Thì quá khứ đơn"),
- bodyVi: a short explanation in Vietnamese (at most 120 words),
- examples: 1 to 3 sentences or phrases copied exactly from the lesson that use this point.

4. writingPrompt: one short English writing task related to the lesson, suitable for CEFR %[1]s.

Sentences:
`, level)
	for i, s := range sentences {
		fmt.Fprintf(&b, "%d: %s\n", i, s)
	}
	return b.String()
}

// generatePrompt asks for 0.9–1.1× the target length so drafts stay inside the ±20% the
// lesson service accepts.
func generatePrompt(req ai.GenerateRequest) string {
	low, high := req.Words*9/10, req.Words*11/10
	var b strings.Builder
	fmt.Fprintf(&b, `You write English lessons for Vietnamese learners at CEFR level %s.
Topic: %s.
Write exactly %d different lessons about this topic. Each lesson must have about %d words
(between %d and %d words; count every word of the content).
Use only vocabulary and grammar suitable for CEFR %s.
Each lesson needs a short title (at most 8 words). Titles must be different from each other,
and each lesson must tell a different story or situation.
`, req.Level, req.TopicName, req.Count, req.Words, low, high, req.Level)
	if req.Kind == ai.KindDialogue {
		b.WriteString(`Form: a dialogue between two or three people with first names.
Write one turn per line in the form "Name: what they say". No empty lines, no narration,
no stage directions.
`)
	} else {
		b.WriteString("Form: a reading text in plain paragraphs (no dialogue lines).\n")
	}
	if req.Idea != "" {
		fmt.Fprintf(&b, "Use this idea as inspiration: %s\n", req.Idea)
	}
	if len(req.ExistingTitles) > 0 {
		b.WriteString("The topic already has these lessons; do not repeat their titles or stories:\n")
		for _, t := range req.ExistingTitles {
			fmt.Fprintf(&b, "- %s\n", t)
		}
	}
	b.WriteString("Return plain text only, no markdown, no headings inside the content.\n")
	return b.String()
}
