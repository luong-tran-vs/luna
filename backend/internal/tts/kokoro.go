package tts

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// maxAudioBytes bounds one sentence of mp3 (a long sentence is well under 1 MiB).
const maxAudioBytes = 10 << 20

// Kokoro calls a Kokoro-FastAPI server (OpenAI-compatible /v1/audio/speech).
type Kokoro struct {
	baseURL string
	voice   string
	client  *http.Client
}

// NewKokoro returns a client for the server at baseURL using voice (for example af_heart).
func NewKokoro(baseURL, voice string, client *http.Client) *Kokoro {
	return &Kokoro{baseURL: baseURL, voice: voice, client: client}
}

var _ Synthesizer = (*Kokoro)(nil)

type speechRequest struct {
	Model          string  `json:"model"`
	Input          string  `json:"input"`
	Voice          string  `json:"voice"`
	ResponseFormat string  `json:"response_format"`
	Speed          float64 `json:"speed"`
}

// Synthesize returns mp3 audio for text.
func (k *Kokoro) Synthesize(ctx context.Context, text string) ([]byte, error) {
	body, err := json.Marshal(speechRequest{Model: "kokoro", Input: text, Voice: k.voice, ResponseFormat: "mp3", Speed: 1.0})
	if err != nil {
		return nil, fmt.Errorf("tts: encode request: %w", err)
	}
	// baseURL comes from TTS_URL (operator configuration), not from users.
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, k.baseURL+"/v1/audio/speech", bytes.NewReader(body)) //nolint:gosec // trusted configured URL
	if err != nil {
		return nil, fmt.Errorf("tts: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := k.client.Do(req) //nolint:gosec // trusted configured URL
	if err != nil {
		return nil, fmt.Errorf("tts: request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // body fully read below

	audio, err := io.ReadAll(io.LimitReader(resp.Body, maxAudioBytes))
	if err != nil {
		return nil, fmt.Errorf("tts: read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tts: status %d", resp.StatusCode)
	}
	if len(audio) == 0 {
		return nil, errors.New("tts: empty audio")
	}
	return audio, nil
}
