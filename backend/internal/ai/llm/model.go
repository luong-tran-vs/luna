package llm

import (
	"context"
	"encoding/json"
	"fmt"
)

// modelFor returns the model of a request of op: the one ModelOf gives, else the client's.
func (c *Client) modelFor(ctx context.Context, op string) string {
	if c.ModelOf != nil {
		if m := c.ModelOf(ctx, op); m != "" {
			return m
		}
	}
	return c.model
}

// TestModel sends one short request to model, to see whether it answers (the admin AI model page,
// added 2026-10-23). It is recorded as the op "model_test".
func (c *Client) TestModel(ctx context.Context, model string) error {
	if IsOpenAIModel(model) {
		_, err := c.openaiGenerate(ctx, model, "model_test", "Reply with the single word OK.", nil, nil)
		return err
	}
	if IsGroqModel(model) {
		_, err := c.groqGenerate(ctx, model, "model_test", "Reply with the single word OK.", nil, 0, nil)
		return err
	}
	body, err := json.Marshal(request{
		Contents: []content{{Role: "user", Parts: []part{{Text: "Reply with the single word OK."}}}},
	})
	if err != nil {
		return fmt.Errorf("gemini: encode request: %w", err)
	}
	_, err = c.post(ctx, model, "model_test", body, maxResponse, nil)
	return err
}
