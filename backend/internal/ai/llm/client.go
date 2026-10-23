// Package llm writes the app's texts with a language model: one prompt and one JSON schema per
// feature (lessons.go, curriculum.go, talk.go…), sent to Gemini (gemini.go) or Groq (groq.go) by
// the model the admin chose for that feature (package aimodel). Pictures are drawn by Gemini only
// (image.go). The package was named gemini until Groq was added (2026-10-23).
package llm

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
)

// maxResponse bounds the response body; an annotation with questions stays far below it.
const maxResponse = 2 << 20

// Client writes text with Gemini or Groq, by the model of each request, and draws pictures with
// Gemini. BaseURL (Gemini) and GroqURL can be replaced in tests.
type Client struct {
	BaseURL string
	apiKey  string
	model   string
	// backups are tried in order when the keys before them get a 429 (gemini.go).
	mu        sync.Mutex
	backups   []string
	restUntil map[int]time.Time
	client    *http.Client
	log       *slog.Logger
	// Usage, if set, keeps every request with its tokens (the admin AI usage page).
	Usage ai.UsageRecorder
	// ModelOf, if set, gives the model of each request by its op (the admin AI model page, per op
	// since 2026-10-23); model is the default when it returns "".
	ModelOf func(ctx context.Context, op string) string
	// GroqURL and groqKey send the requests of a Groq model (groq.go); no key leaves Groq off.
	GroqURL string
	groqKey string
}

// New returns a client for the Gemini key apiKey and model, the default model of every request
// (SetGroq adds Groq). An empty apiKey makes every call return ai.ErrNotConfigured.
func New(apiKey, model string, client *http.Client, log *slog.Logger) *Client {
	return &Client{BaseURL: defaultBaseURL, apiKey: apiKey, model: model, client: client, log: log}
}

var _ ai.Provider = (*Client)(nil)

// generate sends a JSON-mode request to the model of op, Gemini's or Groq's, and returns the
// text of the answer.
func (c *Client) generate(ctx context.Context, op, prompt string, schema map[string]any, temperature float64,
	attrs ...slog.Attr,
) (string, error) {
	if c.apiKey == "" {
		return "", ai.ErrNotConfigured
	}
	model := c.modelFor(ctx, op)
	if IsGroqModel(model) {
		return c.groqGenerate(ctx, model, op, prompt, schema, temperature, attrs)
	}
	return c.geminiGenerate(ctx, model, op, prompt, schema, temperature, attrs)
}

// statusError turns an HTTP status of either provider into the errors of package ai.
func statusError(status int, body []byte) error {
	switch {
	case status == http.StatusOK:
		return nil
	case status == http.StatusTooManyRequests:
		return ai.ErrQuota
	case status == http.StatusServiceUnavailable:
		return ai.ErrBusy
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return ai.ErrInvalidKey
	case status == http.StatusBadRequest && strings.Contains(string(body), "API key"):
		return ai.ErrInvalidKey
	default:
		return fmt.Errorf("llm: status %d", status)
	}
}

// logRequest records one line per call so requests can be counted. It never logs the key
// or the lesson text.
func (c *Client) logRequest(ctx context.Context, model, op string, start time.Time, status int, extra []slog.Attr) {
	attrs := []slog.Attr{
		slog.String("provider", provider(model)),
		slog.String("model", model),
		slog.String("op", op),
		slog.Int("status", status),
		slog.Int64("duration_ms", time.Since(start).Milliseconds()),
	}
	c.log.LogAttrs(ctx, slog.LevelInfo, "ai request", append(attrs, extra...)...)
}

// provider names the provider of model, for the logs.
func provider(model string) string {
	if IsGroqModel(model) {
		return "groq"
	}
	return "gemini"
}
