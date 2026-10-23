package llm_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/ai/llm"
)

const groqOK = `{"choices":[{"message":{"role":"assistant","content":"{\"annotations\":[{\"text\":\"went\",\"lemma\":\"go\",\"meaningVi\":\"đã đi\",\"sentenceIndex\":0}]}"}}],
"usage":{"prompt_tokens":120,"completion_tokens":300,"total_tokens":420,"completion_tokens_details":{"reasoning_tokens":200}}}`

// groqClient answers Gemini requests with okResponse and Groq ones with h, with the text model
// set to model.
func groqClient(t *testing.T, model string, h http.HandlerFunc) (*llm.Client, *usageLog) {
	t.Helper()
	c, _ := newClient(t, "main", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/groq/") {
			h(w, r)
			return
		}
		_, _ = w.Write([]byte(okResponse))
	})
	c.SetGroq("groq-key")
	c.GroqURL = c.BaseURL + "/groq/v1"
	c.ModelOf = func(context.Context, string) string { return model }
	log := &usageLog{}
	c.Usage = log
	return c, log
}

func TestGroqModelGoesToGroq(t *testing.T) {
	t.Parallel()
	var body map[string]any
	var auth, path string
	c, log := groqClient(t, "openai/gpt-oss-120b", func(w http.ResponseWriter, r *http.Request) {
		auth, path = r.Header.Get("Authorization"), r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&body)
		_, _ = w.Write([]byte(groqOK))
	})
	out, err := c.Annotate(t.Context(), ai.AnnotateRequest{Sentences: []string{"We went home."}, Level: "A1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Annotations) != 1 || out.Annotations[0].Lemma != "go" {
		t.Errorf("annotations = %+v", out.Annotations)
	}
	if path != "/groq/v1/chat/completions" || auth != "Bearer groq-key" || body["model"] != "openai/gpt-oss-120b" {
		t.Errorf("path %q, auth %q, model %v", path, auth, body["model"])
	}
	// The Gemini schema reaches Groq as a JSON schema, its types in lower case.
	schema := body["response_format"].(map[string]any)["json_schema"].(map[string]any)["schema"].(map[string]any)
	ann := schema["properties"].(map[string]any)["annotations"].(map[string]any)
	if schema["type"] != "object" || ann["type"] != "array" || ann["items"].(map[string]any)["type"] != "object" {
		t.Errorf("schema = %v", schema)
	}
	u := log.list[0]
	if u.Model != "openai/gpt-oss-120b" || u.Op != "annotate" || u.Status != 200 || u.PromptTokens != 120 ||
		u.OutputTokens != 100 || u.ThoughtTokens != 200 || u.TotalTokens != 420 {
		t.Errorf("usage = %+v", u)
	}
}

func TestGroqErrors(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		status int
		want   error
	}{
		{http.StatusTooManyRequests, ai.ErrQuota},
		{498, ai.ErrBusy},
		{http.StatusServiceUnavailable, ai.ErrBusy},
		{http.StatusUnauthorized, ai.ErrInvalidKey},
	} {
		cl, _ := groqClient(t, "qwen/qwen3.8-27b", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(c.status)
		})
		if err := cl.TestModel(t.Context(), "qwen/qwen3.8-27b"); !errors.Is(err, c.want) {
			t.Errorf("status %d: err = %v, want %v", c.status, err, c.want)
		}
	}
}

func TestGroqWithoutKey(t *testing.T) {
	t.Parallel()
	c, _ := newClient(t, "main", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(okResponse)) })
	c.ModelOf = func(context.Context, string) string { return "openai/gpt-oss-20b" }
	if _, err := c.Annotate(t.Context(), ai.AnnotateRequest{Sentences: []string{"Hi."}}); !errors.Is(err, ai.ErrNotConfigured) {
		t.Errorf("err = %v, want ErrNotConfigured", err)
	}
	// A Gemini model is unchanged.
	c.ModelOf = nil
	if _, err := c.Annotate(t.Context(), ai.AnnotateRequest{Sentences: []string{"Hi."}}); err != nil {
		t.Errorf("gemini: %v", err)
	}
}
