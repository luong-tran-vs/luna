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

const defaultBaseURL = "https://generativelanguage.googleapis.com"

// Client calls Gemini. BaseURL can be replaced in tests.
type Client struct {
	BaseURL string
	apiKey  string
	model   string
	client  *http.Client
	log     *slog.Logger
}

// New returns a Gemini provider. An empty apiKey makes Annotate return ai.ErrNotConfigured.
func New(apiKey, model string, client *http.Client, log *slog.Logger) *Client {
	return &Client{BaseURL: defaultBaseURL, apiKey: apiKey, model: model, client: client, log: log}
}

var _ ai.Provider = (*Client)(nil)

// responseSchema forces a JSON array of annotations.
var responseSchema = map[string]any{
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

// Annotate sends one generateContent request for the whole lesson.
func (c *Client) Annotate(ctx context.Context, sentences []string, level string) ([]ai.Annotation, error) {
	if c.apiKey == "" {
		return nil, ai.ErrNotConfigured
	}

	body, err := json.Marshal(request{
		Contents: []content{{Role: "user", Parts: []part{{Text: prompt(sentences, level)}}}},
		GenerationConfig: map[string]any{
			"temperature":      0.2,
			"responseMimeType": "application/json",
			"responseSchema":   responseSchema,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("gemini: encode request: %w", err)
	}
	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent", c.BaseURL, c.model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("gemini: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", c.apiKey)

	start := time.Now()
	resp, err := c.client.Do(req)
	if err != nil {
		c.logRequest(ctx, len(sentences), start, 0)
		return nil, fmt.Errorf("gemini: request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // body fully read below
	c.logRequest(ctx, len(sentences), start, resp.StatusCode)

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("gemini: read response: %w", err)
	}
	if err := statusError(resp.StatusCode, raw); err != nil {
		return nil, err
	}

	var r response
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("gemini: decode response: %w", err)
	}
	if len(r.Candidates) == 0 || len(r.Candidates[0].Content.Parts) == 0 {
		return nil, errors.New("gemini: empty response")
	}
	var out []ai.Annotation
	if err := json.Unmarshal([]byte(r.Candidates[0].Content.Parts[0].Text), &out); err != nil {
		return nil, fmt.Errorf("gemini: decode annotations: %w", err)
	}
	return out, nil
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
func (c *Client) logRequest(ctx context.Context, sentences int, start time.Time, status int) {
	c.log.InfoContext(ctx, "ai request",
		slog.String("provider", "gemini"),
		slog.String("model", c.model),
		slog.Int("sentences", sentences),
		slog.Int("status", status),
		slog.Int64("duration_ms", time.Since(start).Milliseconds()),
	)
}

func prompt(sentences []string, level string) string {
	var b strings.Builder
	fmt.Fprintf(&b, `You help Vietnamese learners of English at CEFR level %s.
From the numbered lesson sentences below, pick 8 to 25 words or phrases worth learning at this level
(phrasal verbs, collocations and idioms count as one item, e.g. "give up").
For each item return:
- text: copied exactly as it appears in the sentence (same spelling and inflection),
- lemma: the dictionary form (e.g. "went" -> "go", "gave up" -> "give up"),
- meaningVi: a short Vietnamese meaning that fits this context,
- sentenceIndex: the number of the sentence containing it.
Do not invent text that is not in the sentences.

Sentences:
`, level)
	for i, s := range sentences {
		fmt.Fprintf(&b, "%d: %s\n", i, s)
	}
	return b.String()
}
