package gemini_test

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/ai/gemini"
)

const okResponse = `{"candidates":[{"content":{"parts":[{"text":"[{\"text\":\"went\",\"lemma\":\"go\",\"meaningVi\":\"đã đi\",\"sentenceIndex\":0}]"}]}}]}`

func newClient(t *testing.T, key string, h http.HandlerFunc) (*gemini.Client, *atomic.Int32) {
	t.Helper()
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		h(w, r)
	}))
	t.Cleanup(srv.Close)
	c := gemini.New(key, "gemini-test", srv.Client(), slog.New(slog.DiscardHandler))
	c.BaseURL = srv.URL
	return c, &calls
}

func TestAnnotate(t *testing.T) {
	t.Parallel()

	var body map[string]any
	c, _ := newClient(t, "secret-key", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1beta/models/gemini-test:generateContent" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Header.Get("x-goog-api-key") != "secret-key" {
			t.Errorf("api key header missing")
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode: %v", err)
		}
		_, _ = w.Write([]byte(okResponse))
	})

	got, err := c.Annotate(t.Context(), []string{"We went home.", "It was late."}, "B1")
	if err != nil {
		t.Fatalf("Annotate: %v", err)
	}
	want := []ai.Annotation{{Text: "went", Lemma: "go", MeaningVi: "đã đi", SentenceIndex: 0}}
	if len(got) != 1 || got[0] != want[0] {
		t.Fatalf("got %+v, want %+v", got, want)
	}

	cfg := body["generationConfig"].(map[string]any)
	if cfg["responseMimeType"] != "application/json" || cfg["responseSchema"] == nil {
		t.Errorf("generationConfig = %v", cfg)
	}
	prompt, _ := json.Marshal(body["contents"])
	for _, s := range []string{"B1", "0: We went home.", "1: It was late."} {
		if !strings.Contains(string(prompt), s) {
			t.Errorf("prompt missing %q", s)
		}
	}
}

func TestAnnotateErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		status  int
		body    string
		wantErr error
	}{
		{name: "quota", status: http.StatusTooManyRequests, body: `{}`, wantErr: ai.ErrQuota},
		{name: "bad key", status: http.StatusForbidden, body: `{}`, wantErr: ai.ErrInvalidKey},
		{name: "invalid key 400", status: http.StatusBadRequest, body: `{"error":{"status":"INVALID_ARGUMENT","message":"API key not valid"}}`, wantErr: ai.ErrInvalidKey},
		{name: "server error", status: http.StatusInternalServerError, body: `{}`},
		{name: "no candidates", status: http.StatusOK, body: `{"candidates":[]}`},
		{name: "malformed json text", status: http.StatusOK, body: `{"candidates":[{"content":{"parts":[{"text":"not json"}]}}]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c, _ := newClient(t, "k", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			})
			_, err := c.Annotate(t.Context(), []string{"Hi."}, "A1")
			if err == nil {
				t.Fatal("error = nil")
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestAnnotateWithoutKeyMakesNoRequest(t *testing.T) {
	t.Parallel()

	c, calls := newClient(t, "", func(http.ResponseWriter, *http.Request) {})
	if _, err := c.Annotate(t.Context(), []string{"Hi."}, "A1"); !errors.Is(err, ai.ErrNotConfigured) {
		t.Fatalf("error = %v, want ErrNotConfigured", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("made %d HTTP calls without a key", calls.Load())
	}
}
