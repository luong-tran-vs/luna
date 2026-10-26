package llm

import (
	"context"
	"log/slog"
	"strings"
)

// OpenAI (added 2026-10-26) runs ChatGPT's models (GPT-6 Sol and Luna) behind the same chat API as
// Groq (groq.go), so the client sends a text request there when the model of the request is one of
// OpenAI's; pictures stay with Gemini.

const defaultOpenAIURL = "https://api.openai.com/v1"

// IsOpenAIModel reports whether model is one of OpenAI's: their ids start with "gpt-"
// ("gpt-6-sol"); Groq's name the maker first ("openai/gpt-oss-120b").
func IsOpenAIModel(model string) bool {
	return strings.HasPrefix(model, "gpt-")
}

// SetOpenAI lets the client send the requests of an OpenAI model with key; an empty key leaves
// OpenAI off.
func (c *Client) SetOpenAI(key string) {
	c.openaiKey = key
	if c.OpenAIURL == "" {
		c.OpenAIURL = defaultOpenAIURL
	}
}

// openaiGenerate sends one chat request to OpenAI and returns the text of the answer. The
// temperature is left out: GPT-6 is a reasoning model and refuses any but its own.
func (c *Client) openaiGenerate(ctx context.Context, model, op, prompt string, schema map[string]any,
	attrs []slog.Attr,
) (string, error) {
	api := chatAPI{name: "openai", url: c.OpenAIURL, key: c.openaiKey, noTemperature: true}
	return c.chatGenerate(ctx, api, model, op, prompt, schema, 0, attrs)
}
