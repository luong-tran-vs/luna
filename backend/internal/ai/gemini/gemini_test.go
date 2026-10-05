package gemini_test

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
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

	got, err := c.Annotate(t.Context(), ai.AnnotateRequest{Sentences: []string{"We went home.", "It was late."}, Level: "B1"})
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

	got, err := c.Annotate(t.Context(), ai.AnnotateRequest{Sentences: []string{"We went home."}, Level: "A2"})
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
			_, err := c.Annotate(t.Context(), ai.AnnotateRequest{Sentences: []string{"Hi."}, Level: "A1"})
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
	if _, err := c.Annotate(t.Context(), ai.AnnotateRequest{Sentences: []string{"Hi."}, Level: "A1"}); !errors.Is(err, ai.ErrNotConfigured) {
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

func TestGradeWriting(t *testing.T) {
	t.Parallel()

	inner, _ := json.Marshal(ai.Grade{
		Task: ai.Criterion{Score: 4, CommentVi: "Đúng đề."}, Grammar: ai.Criterion{Score: 3, CommentVi: "Sai thì."},
		Vocabulary: ai.Criterion{Score: 4, CommentVi: "Đủ từ."}, Coherence: ai.Criterion{Score: 4, CommentVi: "Mạch lạc."},
		OverallVi: "Khá tốt.", CorrectedText: "My family has four members.",
	})
	resp, _ := json.Marshal(map[string]any{"candidates": []any{map[string]any{
		"content": map[string]any{"parts": []any{map[string]any{"text": string(inner)}}},
	}}})
	var body map[string]any
	c, calls := newClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		_, _ = w.Write(resp)
	})

	got, err := c.GradeWriting(t.Context(), ai.GradeRequest{
		Level: "A1", LessonText: "Tom has a big family.", Prompt: "Write about your family.", Text: "My family have four people.",
	})
	if err != nil {
		t.Fatalf("GradeWriting: %v", err)
	}
	if calls.Load() != 1 || got.Grammar.Score != 3 || got.CorrectedText != "My family has four members." {
		t.Fatalf("calls %d got %+v", calls.Load(), got)
	}

	cfg := body["generationConfig"].(map[string]any)
	if cfg["temperature"] != 0.3 {
		t.Errorf("temperature = %v", cfg["temperature"])
	}
	schema := cfg["responseSchema"].(map[string]any)
	props := schema["properties"].(map[string]any)
	for _, k := range []string{"task", "grammar", "vocabulary", "coherence"} {
		c, ok := props[k].(map[string]any)
		if !ok || c["type"] != "OBJECT" || c["properties"].(map[string]any)["score"] == nil {
			t.Errorf("criterion %s = %v", k, props[k])
		}
	}
	if req, _ := json.Marshal(schema["required"]); string(req) != `["task","grammar","vocabulary","coherence","overallVi","correctedText"]` {
		t.Errorf("required = %s", req)
	}
	prompt, _ := json.Marshal(body["contents"])
	for _, s := range []string{"A1", "Write about your family.", "Tom has a big family.", "My family have four people.", "1 to 5"} {
		if !strings.Contains(string(prompt), s) {
			t.Errorf("prompt missing %q", s)
		}
	}
}

func TestGradeWritingErrors(t *testing.T) {
	t.Parallel()
	c, _ := newClient(t, "k", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTooManyRequests) })
	if _, err := c.GradeWriting(t.Context(), ai.GradeRequest{Text: "x"}); !errors.Is(err, ai.ErrQuota) {
		t.Fatalf("429: %v", err)
	}
	noKey, calls := newClient(t, "", func(http.ResponseWriter, *http.Request) {})
	if _, err := noKey.GradeWriting(t.Context(), ai.GradeRequest{Text: "x"}); !errors.Is(err, ai.ErrNotConfigured) || calls.Load() != 0 {
		t.Fatalf("no key: %v, calls %d", err, calls.Load())
	}
}

func TestExplain(t *testing.T) {
	t.Parallel()

	inner, _ := json.Marshal(ai.Explanation{Lemma: "make up for", MeaningVi: "bù lại", NoteVi: "Bù cho việc đến muộn."})
	resp, _ := json.Marshal(map[string]any{"candidates": []any{map[string]any{
		"content": map[string]any{"parts": []any{map[string]any{"text": string(inner)}}},
	}}})
	var body map[string]any
	c, calls := newClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		_, _ = w.Write(resp)
	})

	got, err := c.Explain(t.Context(), ai.ExplainRequest{
		Text: "made up for", Sentence: "She made up for the lost time.", Level: "B1",
	})
	if err != nil {
		t.Fatalf("Explain: %v", err)
	}
	if calls.Load() != 1 || got.Lemma != "make up for" || got.MeaningVi != "bù lại" || got.NoteVi == "" {
		t.Fatalf("calls %d got %+v", calls.Load(), got)
	}
	cfg := body["generationConfig"].(map[string]any)
	if cfg["temperature"] != 0.2 {
		t.Errorf("temperature = %v", cfg["temperature"])
	}
	schema := cfg["responseSchema"].(map[string]any)
	if req, _ := json.Marshal(schema["required"]); schema["type"] != "OBJECT" || string(req) != `["lemma","meaningVi","noteVi"]` {
		t.Errorf("schema = %v", schema)
	}
	prompt, _ := json.Marshal(body["contents"])
	for _, s := range []string{"B1", "She made up for the lost time.", "made up for"} {
		if !strings.Contains(string(prompt), s) {
			t.Errorf("prompt missing %q", s)
		}
	}
}

func TestExplainErrors(t *testing.T) {
	t.Parallel()
	c, _ := newClient(t, "k", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTooManyRequests) })
	if _, err := c.Explain(t.Context(), ai.ExplainRequest{Text: "x"}); !errors.Is(err, ai.ErrQuota) {
		t.Fatalf("429: %v", err)
	}
	noKey, calls := newClient(t, "", func(http.ResponseWriter, *http.Request) {})
	if _, err := noKey.Explain(t.Context(), ai.ExplainRequest{Text: "x"}); !errors.Is(err, ai.ErrNotConfigured) || calls.Load() != 0 {
		t.Fatalf("no key: %v, calls %d", err, calls.Load())
	}
}

func TestPractice(t *testing.T) {
	t.Parallel()

	want := ai.Practice{
		ObjectiveVi: "Bạn có thể chào hỏi.",
		Examples:    []ai.Example{{Lemma: "meet", Sentence: "Nice to meet you."}},
		Dialogue: ai.Dialogue{Speakers: []string{"Minh", "Anna"}, Turns: []ai.Turn{
			{Speaker: 0, Text: "Hi, I'm Minh.", MeaningVi: "Chào, mình là Minh."},
		}},
		GrammarTipVi: "Dùng I'm để giới thiệu.",
		Translations: []ai.Translation{{Vi: "Rất vui được gặp bạn.", En: "Nice to meet you.", Distractors: []string{"see"}}},
	}
	inner, _ := json.Marshal(want)
	resp, _ := json.Marshal(map[string]any{"candidates": []any{map[string]any{
		"content": map[string]any{"parts": []any{map[string]any{"text": string(inner)}}},
	}}})
	var body map[string]any
	c, calls := newClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1beta/models/gemini-test:generateContent" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		_, _ = w.Write(resp)
	})

	got, err := c.Practice(t.Context(), ai.PracticeRequest{
		Level: "A2", Title: "Greetings",
		Sentences: []string{"Hello, I am Lan.", "Nice to meet you."},
		Words:     []ai.PracticeWord{{Lemma: "meet", Text: "meet", MeaningVi: "gặp"}},
	})
	if err != nil {
		t.Fatalf("Practice: %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls = %d", calls.Load())
	}
	if got.ObjectiveVi != want.ObjectiveVi || len(got.Examples) != 1 || len(got.Dialogue.Turns) != 1 ||
		got.Dialogue.Speakers[1] != "Anna" || got.Translations[0].Distractors[0] != "see" {
		t.Fatalf("got %+v", got)
	}
	cfg := body["generationConfig"].(map[string]any)
	if cfg["temperature"] != 0.5 || cfg["responseMimeType"] != "application/json" {
		t.Errorf("config = %v", cfg)
	}
	schema := cfg["responseSchema"].(map[string]any)
	req, _ := json.Marshal(schema["required"])
	if schema["type"] != "OBJECT" ||
		string(req) != `["objectiveVi","examples","dialogue","grammarTipVi","translations"]` {
		t.Errorf("schema = %v", schema)
	}
	raw, _ := json.Marshal(schema)
	if !strings.Contains(string(raw), `"speaker":{"type":"INTEGER"}`) {
		t.Errorf("speaker must be an INTEGER: %s", raw)
	}
	prompt, _ := json.Marshal(body["contents"])
	for _, s := range []string{"A2", "Greetings", "0: Hello, I am Lan.", "1: Nice to meet you.", "meet | meet | gặp"} {
		if !strings.Contains(string(prompt), s) {
			t.Errorf("prompt missing %q", s)
		}
	}
}

func TestPracticeErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status int
		want   error
	}{
		{http.StatusTooManyRequests, ai.ErrQuota},
		{http.StatusUnauthorized, ai.ErrInvalidKey},
		{http.StatusForbidden, ai.ErrInvalidKey},
	}
	for _, tt := range tests {
		c, _ := newClient(t, "k", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(tt.status) })
		if _, err := c.Practice(t.Context(), ai.PracticeRequest{}); !errors.Is(err, tt.want) {
			t.Errorf("status %d: err = %v, want %v", tt.status, err, tt.want)
		}
	}
	bad, _ := newClient(t, "k", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"not json"}]}}]}`))
	})
	if _, err := bad.Practice(t.Context(), ai.PracticeRequest{}); err == nil {
		t.Error("invalid JSON must fail")
	}
}

func TestPracticeWithoutKeyMakesNoRequest(t *testing.T) {
	t.Parallel()
	c, calls := newClient(t, "", func(http.ResponseWriter, *http.Request) {})
	if _, err := c.Practice(t.Context(), ai.PracticeRequest{}); !errors.Is(err, ai.ErrNotConfigured) || calls.Load() != 0 {
		t.Fatalf("err %v, calls %d", err, calls.Load())
	}
}

// capturePrompt returns a client that records the prompt of each request and answers body.
func capturePrompt(t *testing.T, body string) (*gemini.Client, *string) {
	t.Helper()
	var prompt string
	c, _ := newClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Contents []struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"contents"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		prompt = req.Contents[0].Parts[0].Text
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":` + jsonQuote(body) + `}]}}]}`))
	})
	return c, &prompt
}

func jsonQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func TestGenerateLessonsTargetWords(t *testing.T) {
	t.Parallel()

	c, prompt := capturePrompt(t, "[]")
	req := ai.GenerateRequest{
		Level: "A1", TopicName: "Gia đình", Count: 3, Words: 150, Kind: ai.KindReading,
		TargetWords: [][]string{{"Family", "take a shower"}, {}, {"Uncle"}},
	}
	if _, err := c.GenerateLessons(t.Context(), req); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Lesson 1 must use every one of these words or phrases", "Family, take a shower", "Lesson 3 must use", "Uncle"} {
		if !strings.Contains(*prompt, want) {
			t.Errorf("prompt missing %q:\n%s", want, *prompt)
		}
	}
	if strings.Contains(*prompt, "Lesson 2 must use") {
		t.Errorf("empty group must not be listed:\n%s", *prompt)
	}

	if _, err := c.GenerateLessons(t.Context(), ai.GenerateRequest{Level: "A1", TopicName: "x", Count: 1, Words: 100}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(*prompt, "must use") {
		t.Errorf("no target words, but:\n%s", *prompt)
	}
}

func TestAnnotateFocusWords(t *testing.T) {
	t.Parallel()

	c, prompt := capturePrompt(t, `{"annotations":[]}`)
	req := ai.AnnotateRequest{Sentences: []string{"My family took a shower."}, Level: "A1", FocusWords: []string{"Family", "take a shower"}}
	if _, err := c.Annotate(t.Context(), req); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Always include each of these topic words", "count toward the 12 items: Family, take a shower", "0: My family took a shower."} {
		if !strings.Contains(*prompt, want) {
			t.Errorf("prompt missing %q:\n%s", want, *prompt)
		}
	}
	req.FocusWords = nil
	if _, err := c.Annotate(t.Context(), req); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(*prompt, "topic words") {
		t.Errorf("no focus words, but:\n%s", *prompt)
	}
}

func TestSuggestWords(t *testing.T) {
	t.Parallel()

	inner, _ := json.Marshal(map[string]any{"words": []string{"ladybug", "cicada"}})
	resp, _ := json.Marshal(map[string]any{"candidates": []any{map[string]any{
		"content": map[string]any{"parts": []any{map[string]any{"text": string(inner)}}},
	}}})
	var body map[string]any
	c, calls := newClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		_, _ = w.Write(resp)
	})

	got, err := c.SuggestWords(t.Context(), ai.SuggestWordsRequest{
		Level: "A1", TopicName: "Côn trùng", Existing: []string{"Ant", "Bee"}, Count: 2,
	})
	if err != nil {
		t.Fatalf("SuggestWords: %v", err)
	}
	if calls.Load() != 1 || !slices.Equal(got, []string{"ladybug", "cicada"}) {
		t.Fatalf("calls %d got %v", calls.Load(), got)
	}
	prompt, _ := json.Marshal(body["contents"])
	for _, s := range []string{"A1", "Côn trùng", "List 2 new", "Ant, Bee"} {
		if !strings.Contains(string(prompt), s) {
			t.Errorf("prompt missing %q", s)
		}
	}
	noKey, noCalls := newClient(t, "", func(http.ResponseWriter, *http.Request) {})
	if _, err := noKey.SuggestWords(t.Context(), ai.SuggestWordsRequest{Count: 1}); !errors.Is(err, ai.ErrNotConfigured) || noCalls.Load() != 0 {
		t.Fatalf("no key: %v, calls %d", err, noCalls.Load())
	}
}

func TestGrammarLesson(t *testing.T) {
	t.Parallel()

	lesson := `{"objective":"Bạn có thể nói về mình","explanation":["a"],"usage":["b"],"structures":[{"label":"l","pattern":"p","example":"e"}],` +
		`"examples":[{"en":"I am.","vi":"Tôi là."}],"mistakes":[{"wrong":"w","right":"r","noteVi":"n"}],` +
		`"practice":[{"kind":"choice","promptVi":"Chọn","text":"I ___ a boy.","options":["am","is","are","be"],"answerIndex":0,"explanationVi":"x"}],` +
		`"mastery":[{"kind":"reorder","promptVi":"Sắp xếp","text":"Tôi là học sinh","sentence":"I am a student","words":["I","am","a","student","is"],"explanationVi":"y"}]}`
	wrapped, err := json.Marshal(map[string]any{"candidates": []any{map[string]any{"content": map[string]any{
		"parts": []any{map[string]any{"text": lesson}},
	}}}})
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	c, calls := newClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode: %v", err)
		}
		_, _ = w.Write(wrapped)
	})

	got, err := c.GrammarLesson(t.Context(), ai.GrammarLessonRequest{Level: "A1", TitleEn: "Verb to be", TitleVi: "to be", Pattern: "S + be"})
	if err != nil {
		t.Fatalf("GrammarLesson: %v", err)
	}
	if calls.Load() != 1 {
		t.Errorf("requests = %d, want 1", calls.Load())
	}
	if got.Objective == "" || len(got.Structures) != 1 || got.Examples[0].Vi != "Tôi là." || got.Mistakes[0].NoteVi != "n" {
		t.Errorf("decoded %+v", got)
	}
	if p := got.Practice[0]; p.Kind != "choice" || p.AnswerIndex != 0 || len(p.Options) != 4 {
		t.Errorf("practice[0] = %+v", p)
	}
	if m := got.Mastery[0]; m.Sentence != "I am a student" || len(m.Words) != 5 {
		t.Errorf("mastery[0] = %+v", m)
	}
	cfg, _ := body["generationConfig"].(map[string]any)
	if cfg["responseMimeType"] != "application/json" || cfg["responseSchema"] == nil {
		t.Errorf("generationConfig = %v", cfg)
	}

	if _, err := gemini.New("", "m", http.DefaultClient, slog.New(slog.DiscardHandler)).GrammarLesson(t.Context(), ai.GrammarLessonRequest{}); !errors.Is(err, ai.ErrNotConfigured) {
		t.Errorf("no key: %v", err)
	}
	if _, err := (ai.Disabled{}).GrammarLesson(t.Context(), ai.GrammarLessonRequest{}); !errors.Is(err, ai.ErrNotConfigured) {
		t.Errorf("disabled: %v", err)
	}
}

func TestSolveGrammarExercises(t *testing.T) {
	t.Parallel()

	solutions := `[{"exerciseId":"p1","choiceIndex":1,"ambiguous":false},` +
		`{"exerciseId":"p2","answer":"am","ambiguous":false},` +
		`{"exerciseId":"m1","sentence":"I am a student","ambiguous":false},` +
		`{"exerciseId":"m2","ambiguous":true,"noteVi":"Hai đáp án đúng."}]`
	wrapped, err := json.Marshal(map[string]any{"candidates": []any{map[string]any{"content": map[string]any{
		"parts": []any{map[string]any{"text": solutions}},
	}}}})
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	c, calls := newClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode: %v", err)
		}
		_, _ = w.Write(wrapped)
	})

	got, err := c.SolveGrammarExercises(t.Context(), ai.SolveRequest{
		Level: "A1", TitleEn: "Verb to be", Pattern: "S + be",
		Exercises: []ai.SolveExercise{{ID: "p1", Kind: "choice", Text: "She ___ a nurse.", Options: []string{"am", "is", "are", "be"}}},
	})
	if err != nil {
		t.Fatalf("SolveGrammarExercises: %v", err)
	}
	if calls.Load() != 1 || len(got) != 4 {
		t.Fatalf("requests = %d, solutions = %v", calls.Load(), got)
	}
	if got[0].ChoiceIndex == nil || *got[0].ChoiceIndex != 1 || got[1].Answer != "am" || got[2].Sentence != "I am a student" ||
		!got[3].Ambiguous || got[3].NoteVi == "" || got[1].ChoiceIndex != nil {
		t.Errorf("decoded %+v", got)
	}
	cfg, _ := body["generationConfig"].(map[string]any)
	if cfg["responseMimeType"] != "application/json" || cfg["responseSchema"] == nil {
		t.Errorf("generationConfig = %v", cfg)
	}
	if temp, _ := cfg["temperature"].(float64); temp > 0.3 {
		t.Errorf("temperature = %v, want low", temp)
	}

	if _, err := gemini.New("", "m", http.DefaultClient, slog.New(slog.DiscardHandler)).SolveGrammarExercises(t.Context(), ai.SolveRequest{}); !errors.Is(err, ai.ErrNotConfigured) {
		t.Errorf("no key: %v", err)
	}
	if _, err := (ai.Disabled{}).SolveGrammarExercises(t.Context(), ai.SolveRequest{}); !errors.Is(err, ai.ErrNotConfigured) {
		t.Errorf("disabled: %v", err)
	}
}
