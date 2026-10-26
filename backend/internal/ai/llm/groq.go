package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
)

// Groq (added 2026-10-23) runs open models (gpt-oss, Qwen) much faster than Gemini, behind an
// OpenAI-style chat API. The client sends a text request to Groq when the model of the request is
// one of Groq's, so every prompt and schema of this package serves both providers; pictures stay
// with Gemini.

const defaultGroqURL = "https://api.groq.com/openai/v1"

// statusGroqCapacity is Groq's answer when a model is over capacity, as Gemini's 503.
const statusGroqCapacity = 498

// IsGroqModel reports whether model is one of Groq's: their ids name the maker ("openai/gpt-oss-120b"),
// Gemini's never have a slash.
func IsGroqModel(model string) bool {
	return strings.Contains(model, "/")
}

// SetGroq lets the client send the requests of a Groq model with key; an empty key leaves Groq off.
func (c *Client) SetGroq(key string) {
	c.groqKey = key
	if c.GroqURL == "" {
		c.GroqURL = defaultGroqURL
	}
}

type groqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type groqResponse struct {
	Choices []struct {
		Message groqMessage `json:"message"`
	} `json:"choices"`
}

// groqSchema turns a Gemini response schema into a JSON schema: the types are written in lower
// case ("OBJECT" → "object"); property names are left alone.
func groqSchema(s map[string]any) map[string]any {
	out := make(map[string]any, len(s))
	for k, v := range s {
		switch k {
		case "type":
			if t, ok := v.(string); ok {
				v = strings.ToLower(t)
			}
		case "items":
			if m, ok := v.(map[string]any); ok {
				v = groqSchema(m)
			}
		case "properties":
			if props, ok := v.(map[string]any); ok {
				conv := make(map[string]any, len(props))
				for name, p := range props {
					if m, ok := p.(map[string]any); ok {
						p = groqSchema(m)
					}
					conv[name] = p
				}
				v = conv
			}
		}
		out[k] = v
	}
	return out
}

// chatAPI is a provider with an OpenAI-style chat API: Groq, or OpenAI itself (openai.go, added
// 2026-10-26).
type chatAPI struct {
	name string
	url  string
	key  string
	// noTemperature leaves the temperature out: OpenAI's reasoning models take only their own.
	noTemperature bool
}

// groqGenerate sends one chat request to Groq and returns the text of the answer.
func (c *Client) groqGenerate(ctx context.Context, model, op, prompt string, schema map[string]any,
	temperature float64, attrs []slog.Attr,
) (string, error) {
	api := chatAPI{name: "groq", url: c.GroqURL, key: c.groqKey}
	return c.chatGenerate(ctx, api, model, op, prompt, schema, temperature, attrs)
}

// chatGenerate sends one chat request to api and returns the text of the answer; schema, if set,
// makes the answer a JSON object of that shape.
func (c *Client) chatGenerate(ctx context.Context, api chatAPI, model, op, prompt string, schema map[string]any,
	temperature float64, attrs []slog.Attr,
) (string, error) {
	if api.key == "" {
		return "", ai.ErrNotConfigured
	}
	body := map[string]any{
		"model":    model,
		"messages": []groqMessage{{Role: "user", Content: prompt}},
	}
	if !api.noTemperature {
		body["temperature"] = temperature
	}
	if schema != nil {
		// Not strict: the schemas leave some properties optional, which strict mode forbids.
		body["response_format"] = map[string]any{
			"type":        "json_schema",
			"json_schema": map[string]any{"name": "answer", "schema": groqSchema(schema)},
		}
	}
	data, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("%s: encode request: %w", api.name, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, api.url+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("%s: build request: %w", api.name, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+api.key)

	start := time.Now()
	resp, err := c.client.Do(req)
	if err != nil {
		c.logRequest(ctx, model, op, start, 0, attrs)
		c.recordUsage(ctx, model, op, 0, start, 0, nil)
		return "", fmt.Errorf("%s: request: %w", api.name, err)
	}
	defer resp.Body.Close() //nolint:errcheck // body fully read below
	c.logRequest(ctx, model, op, start, resp.StatusCode, attrs)
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponse))
	c.recordUsage(ctx, model, op, 0, start, resp.StatusCode, raw)
	if err != nil {
		return "", fmt.Errorf("%s: read response: %w", api.name, err)
	}
	if resp.StatusCode == statusGroqCapacity {
		return "", ai.ErrBusy
	}
	if err := statusError(resp.StatusCode, raw); err != nil {
		return "", err
	}
	var r groqResponse
	if err := json.Unmarshal(raw, &r); err != nil {
		return "", fmt.Errorf("%s: decode response: %w", api.name, err)
	}
	if len(r.Choices) == 0 || r.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("%s: empty response", api.name)
	}
	return r.Choices[0].Message.Content, nil
}
