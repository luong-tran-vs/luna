package gemini

import (
	"bytes"
	"context"
	"encoding/base64"
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

// maxImageResponse bounds an image response: a base64 PNG of 1024×1024 is about 2 MB.
const maxImageResponse = 16 << 20

// defaultImageStyle is used when the admin gave no style (F23).
const defaultImageStyle = "a simple, friendly flat illustration with soft colors on a plain light background"

var _ ai.ImageProvider = (*Client)(nil)

type imagePart struct {
	Text       string `json:"text,omitempty"`
	InlineData *struct {
		MimeType string `json:"mimeType"`
		Data     string `json:"data"`
	} `json:"inlineData,omitempty"`
}

type imageResponse struct {
	Candidates []struct {
		Content struct {
			Parts []imagePart `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

// GenerateImage sends one generateContent request to an image model (set the client's model to
// one, e.g. gemini-2.5-flash-image) and returns the first picture. Only the word length is logged.
func (c *Client) GenerateImage(ctx context.Context, req ai.ImageRequest) (ai.Image, error) {
	if c.apiKey == "" {
		return ai.Image{}, ai.ErrNotConfigured
	}
	body, err := json.Marshal(request{
		Contents: []content{{Role: "user", Parts: []part{{Text: imagePrompt(req)}}}},
		GenerationConfig: map[string]any{
			"responseModalities": []string{"IMAGE"},
			"imageConfig":        map[string]any{"aspectRatio": "4:3"},
		},
	})
	if err != nil {
		return ai.Image{}, fmt.Errorf("gemini: encode image request: %w", err)
	}
	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent", c.BaseURL, c.model)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return ai.Image{}, fmt.Errorf("gemini: build image request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", c.apiKey)

	start := time.Now()
	attrs := []slog.Attr{slog.Int("word_chars", len(req.Word))}
	resp, err := c.client.Do(httpReq)
	if err != nil {
		c.logRequest(ctx, "image", start, 0, attrs)
		c.recordUsage(ctx, "image", start, 0, nil)
		return ai.Image{}, fmt.Errorf("gemini: image request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // body fully read below
	c.logRequest(ctx, "image", start, resp.StatusCode, attrs)

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxImageResponse))
	c.recordUsage(ctx, "image", start, resp.StatusCode, raw)
	if err != nil {
		return ai.Image{}, fmt.Errorf("gemini: read image response: %w", err)
	}
	if err := statusError(resp.StatusCode, raw); err != nil {
		return ai.Image{}, err
	}
	var r imageResponse
	if err := json.Unmarshal(raw, &r); err != nil {
		return ai.Image{}, fmt.Errorf("gemini: decode image response: %w", err)
	}
	for _, cand := range r.Candidates {
		for _, p := range cand.Content.Parts {
			if p.InlineData == nil || !strings.HasPrefix(p.InlineData.MimeType, "image/") {
				continue
			}
			data, err := base64.StdEncoding.DecodeString(p.InlineData.Data)
			if err != nil {
				return ai.Image{}, fmt.Errorf("gemini: decode image data: %w", err)
			}
			return ai.Image{MIME: p.InlineData.MimeType, Data: data}, nil
		}
	}
	return ai.Image{}, errors.New("gemini: no image in response")
}

// imagePrompt describes one picture: the word in its sense, in the admin's style, with no text
// drawn, since a learner must recognise the thing, not read its name.
func imagePrompt(req ai.ImageRequest) string {
	style := strings.TrimSpace(req.Style)
	if style == "" {
		style = defaultImageStyle
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Draw one picture that helps a Vietnamese learner of English remember the English word %q", req.Word)
	if req.MeaningVi != "" {
		fmt.Fprintf(&b, " (Vietnamese meaning: %q)", req.MeaningVi)
	}
	b.WriteString(".\n")
	if req.Sentence != "" {
		fmt.Fprintf(&b, "The word is used like this: %q. Show this sense of the word.\n", req.Sentence)
	}
	fmt.Fprintf(&b, "Style: %s.\n", style)
	b.WriteString("Show the meaning clearly with one main subject. Do not draw any letters, words, captions or watermarks. " +
		"Keep it suitable for learners of every age.")
	return b.String()
}
