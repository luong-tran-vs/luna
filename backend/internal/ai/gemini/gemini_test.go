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

const okResponse = `{"candidates":[{"content":{"parts":[{"text":"{\"annotations\":[{\"text\":\"went\",\"lemma\":\"go\",\"meaningVi\":\"đã đi\",\"sentenceIndex\":0}]}"}]}}]}`

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
	want := ai.Annotation{Text: "went", Lemma: "go", MeaningVi: "đã đi", SentenceIndex: 0}
	if len(got.Annotations) != 1 || got.Annotations[0] != want {
		t.Fatalf("got %+v, want [%+v]", got.Annotations, want)
	}
	if got.Questions != nil || got.GrammarNote != nil || got.WritingPrompt != "" {
		t.Errorf("missing parts should stay empty: %+v", got)
	}

	cfg := body["generationConfig"].(map[string]any)
	if cfg["responseMimeType"] != "application/json" || cfg["responseSchema"] == nil {
		t.Errorf("generationConfig = %v", cfg)
	}
	schema := cfg["responseSchema"].(map[string]any)
	props := schema["properties"].(map[string]any)
	if schema["type"] != "OBJECT" || props["annotations"] == nil || props["questions"] == nil ||
		props["grammarNote"] == nil || props["writingPrompt"] == nil {
		t.Errorf("responseSchema = %v", schema)
	}
	if req, _ := json.Marshal(schema["required"]); string(req) != `["annotations"]` {
		t.Errorf("required = %s, want only annotations", req)
	}
	prompt, _ := json.Marshal(body["contents"])
	for _, s := range []string{
		"B1", "0: We went home.", "1: It was late.", "3 to 5 multiple-choice", "exactly 4",
		"grammarNote", "copied exactly from the lesson", "writingPrompt",
	} {
		if !strings.Contains(string(prompt), s) {
			t.Errorf("prompt missing %q", s)
		}
	}
}

func TestAnnotateDecodesExtras(t *testing.T) {
	t.Parallel()

	inner, _ := json.Marshal(ai.LessonExtras{
		Annotations: []ai.Annotation{{Text: "went", Lemma: "go", MeaningVi: "đã đi"}},
		Questions: []ai.Question{{
			Prompt: "Where did we go?", Options: []string{"Home", "School", "Work", "Park"}, AnswerIndex: 0,
			ExplanationVi: "Câu 1: We went home.",
		}},
		GrammarNote:   &ai.GrammarNote{Title: "Quá khứ đơn", BodyVi: "Dùng cho việc đã xong.", Examples: []string{"We went home."}},
		WritingPrompt: "Write about your evening.",
	})
	resp, _ := json.Marshal(map[string]any{"candidates": []any{map[string]any{
		"content": map[string]any{"parts": []any{map[string]any{"text": string(inner)}}},
	}}})
	c, calls := newClient(t, "k", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(resp) })

	got, err := c.Annotate(t.Context(), []string{"We went home."}, "A2")
	if err != nil {
		t.Fatalf("Annotate: %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("HTTP calls = %d, want 1", calls.Load())
	}
	if len(got.Questions) != 1 || got.Questions[0].Options[3] != "Park" || got.GrammarNote == nil ||
		got.GrammarNote.Examples[0] != "We went home." || got.WritingPrompt != "Write about your evening." {
		t.Fatalf("got %+v", got)
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

const draftsResponse = `{"candidates":[{"content":{"parts":[{"text":"[{\"title\":\"At the Park\",\"content\":\"Anna: Hi!\\nBen: Hello.\"}]"}]}}]}`

func TestGenerateLessons(t *testing.T) {
	t.Parallel()

	var body map[string]any
	c, calls := newClient(t, "secret-key", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1beta/models/gemini-test:generateContent" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Header.Get("x-goog-api-key") != "secret-key" {
			t.Errorf("api key header missing")
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode: %v", err)
		}
		_, _ = w.Write([]byte(draftsResponse))
	})

	got, err := c.GenerateLessons(t.Context(), ai.GenerateRequest{
		Level: "A2", TopicName: "Family", Count: 3, Words: 160, Kind: ai.KindDialogue,
		Idea: "a birthday party", ExistingTitles: []string{"My Brother Tom", "Sunday Lunch"},
	})
	if err != nil {
		t.Fatalf("GenerateLessons: %v", err)
	}
	want := ai.LessonDraft{Title: "At the Park", Content: "Anna: Hi!\nBen: Hello."}
	if len(got) != 1 || got[0] != want {
		t.Fatalf("got %+v, want [%+v]", got, want)
	}
	if calls.Load() != 1 {
		t.Fatalf("HTTP calls = %d, want 1 for the whole batch", calls.Load())
	}

	cfg := body["generationConfig"].(map[string]any)
	if cfg["responseMimeType"] != "application/json" {
		t.Errorf("responseMimeType = %v", cfg["responseMimeType"])
	}
	if cfg["temperature"] != 0.9 {
		t.Errorf("temperature = %v, want 0.9", cfg["temperature"])
	}
	schema := cfg["responseSchema"].(map[string]any)
	items := schema["items"].(map[string]any)
	props := items["properties"].(map[string]any)
	if schema["type"] != "ARRAY" || items["type"] != "OBJECT" || props["title"] == nil || props["content"] == nil {
		t.Errorf("responseSchema = %v", schema)
	}
	if req, _ := json.Marshal(items["required"]); string(req) != `["title","content"]` {
		t.Errorf("required = %s", req)
	}

	raw, _ := json.Marshal(body["contents"])
	var contents []struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	}
	_ = json.Unmarshal(raw, &contents)
	prompt := contents[0].Parts[0].Text
	for _, s := range []string{"A2", "Family", "3", "160", "My Brother Tom", "Sunday Lunch", "a birthday party", "Name: "} {
		if !strings.Contains(prompt, s) {
			t.Errorf("prompt missing %q:\n%s", s, prompt)
		}
	}
}

func TestGenerateLessonsReadingPrompt(t *testing.T) {
	t.Parallel()

	var prompt string
	c, _ := newClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Contents []struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"contents"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		prompt = body.Contents[0].Parts[0].Text
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"[]"}]}}]}`))
	})
	if _, err := c.GenerateLessons(t.Context(), ai.GenerateRequest{Level: "B1", TopicName: "Work", Count: 1, Words: 220, Kind: ai.KindReading}); err != nil {
		t.Fatalf("GenerateLessons: %v", err)
	}
	if strings.Contains(prompt, "Name: ") {
		t.Errorf("reading prompt has the dialogue rule:\n%s", prompt)
	}
	if strings.Contains(prompt, "inspiration") {
		t.Errorf("prompt mentions an idea that was not given:\n%s", prompt)
	}
}

func TestGenerateLessonsErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		status  int
		body    string
		wantErr error
	}{
		{name: "quota", status: http.StatusTooManyRequests, body: `{}`, wantErr: ai.ErrQuota},
		{name: "bad key", status: http.StatusUnauthorized, body: `{}`, wantErr: ai.ErrInvalidKey},
		{name: "server error", status: http.StatusInternalServerError, body: `{}`},
		{name: "malformed json text", status: http.StatusOK, body: `{"candidates":[{"content":{"parts":[{"text":"[{\"title\":"}]}}]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c, _ := newClient(t, "k", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			})
			_, err := c.GenerateLessons(t.Context(), ai.GenerateRequest{Level: "A1", TopicName: "Family", Count: 1, Words: 120, Kind: ai.KindReading})
			if err == nil {
				t.Fatal("error = nil")
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestGenerateLessonsWithoutKeyMakesNoRequest(t *testing.T) {
	t.Parallel()

	c, calls := newClient(t, "", func(http.ResponseWriter, *http.Request) {})
	if _, err := c.GenerateLessons(t.Context(), ai.GenerateRequest{Count: 1}); !errors.Is(err, ai.ErrNotConfigured) {
		t.Fatalf("error = %v, want ErrNotConfigured", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("made %d HTTP calls without a key", calls.Load())
	}
}
