package llm

// Gemini (generativelanguage.googleapis.com): the generateContent requests, with the backup keys.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
)

const defaultBaseURL = "https://generativelanguage.googleapis.com"

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

// geminiGenerate sends a JSON-mode generateContent request and returns the text of the first
// candidate.
func (c *Client) geminiGenerate(ctx context.Context, model, op, prompt string, schema map[string]any,
	temperature float64, attrs []slog.Attr,
) (string, error) {
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
	raw, err := c.post(ctx, model, op, body, maxResponse, attrs)
	if err != nil {
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

// The client may hold backup keys (added 2026-10-23), each of another Google project, since the
// free tier limits each project: a request the main key gets a 429 for is sent again with the next
// key. A key that got a 429 is tried last for keyRest, so a key out of requests for the day does
// not cost an extra request every time.

// keyRest is how long a key that ran out of requests is tried after the others.
const keyRest = time.Minute

// AddBackupKey adds a key tried when the ones before it are out of requests; an empty key is
// ignored.
func (c *Client) AddBackupKey(key string) {
	if key == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.backups = append(c.backups, key)
}

func (c *Client) keys() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string{c.apiKey}, c.backups...)
}

// keyOrder lists the key indexes to try: the rested ones in order, then those still resting.
func (c *Client) keyOrder(n int, now time.Time) []int {
	c.mu.Lock()
	defer c.mu.Unlock()
	var ready, resting []int
	for i := range n {
		if now.Before(c.restUntil[i]) {
			resting = append(resting, i)
		} else {
			ready = append(ready, i)
		}
	}
	return append(ready, resting...)
}

func (c *Client) rest(i int, until time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.restUntil == nil {
		c.restUntil = map[int]time.Time{}
	}
	c.restUntil[i] = until
}

// post sends body to model, with the next key on a 429, and returns the response
// body of a 200. Every attempt is logged and recorded.
func (c *Client) post(ctx context.Context, model, op string, body []byte, limit int64, attrs []slog.Attr) ([]byte, error) {
	keys := c.keys()
	if keys[0] == "" {
		return nil, ai.ErrNotConfigured
	}
	var err error
	for _, i := range c.keyOrder(len(keys), time.Now()) {
		var raw []byte
		raw, err = c.postWith(ctx, model, op, i, keys[i], body, limit, attrs)
		if !errors.Is(err, ai.ErrQuota) {
			return raw, err
		}
		c.rest(i, time.Now().Add(keyRest))
	}
	return nil, err
}

// postWith sends one request with key number i.
func (c *Client) postWith(ctx context.Context, model, op string, i int, key string, body []byte, limit int64,
	attrs []slog.Attr,
) ([]byte, error) {
	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent", c.BaseURL, model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("gemini: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", key)
	if i > 0 {
		attrs = append(attrs, slog.Int("key", i))
	}

	start := time.Now()
	resp, err := c.client.Do(req)
	if err != nil {
		c.logRequest(ctx, model, op, start, 0, attrs)
		c.recordUsage(ctx, model, op, i, start, 0, nil)
		return nil, fmt.Errorf("gemini: request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // body fully read below
	c.logRequest(ctx, model, op, start, resp.StatusCode, attrs)

	raw, err := io.ReadAll(io.LimitReader(resp.Body, limit))
	c.recordUsage(ctx, model, op, i, start, resp.StatusCode, raw)
	if err != nil {
		return nil, fmt.Errorf("gemini: read response: %w", err)
	}
	if err := statusError(resp.StatusCode, raw); err != nil {
		return nil, err
	}
	return raw, nil
}
