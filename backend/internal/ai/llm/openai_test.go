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

// openaiClient answers Gemini requests with okResponse and OpenAI ones with h, with the text model
// set to model.
func openaiClient(t *testing.T, model string, h http.HandlerFunc) (*llm.Client, *usageLog) {
	t.Helper()
	c, _ := newClient(t, "main", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/openai/") {
			h(w, r)
			return
		}
		_, _ = w.Write([]byte(okResponse))
	})
	c.SetOpenAI("openai-key")
	c.OpenAIURL = c.BaseURL + "/openai/v1"
	c.ModelOf = func(context.Context, string) string { return model }
	log := &usageLog{}
	c.Usage = log
	return c, log
}

func TestOpenAIModelGoesToOpenAI(t *testing.T) {
	t.Parallel()
	var body map[string]any
	var auth, path string
	c, log := openaiClient(t, "gpt-6-sol", func(w http.ResponseWriter, r *http.Request) {
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
	if path != "/openai/v1/chat/completions" || auth != "Bearer openai-key" || body["model"] != "gpt-6-sol" {
		t.Errorf("path %q, auth %q, model %v", path, auth, body["model"])
	}
	// GPT-6 takes only its own temperature.
	if _, ok := body["temperature"]; ok {
		t.Errorf("temperature sent: %v", body["temperature"])
	}
	if body["response_format"] == nil {
		t.Error("no response_format")
	}
	if u := log.list[0]; u.Model != "gpt-6-sol" || u.Op != "annotate" || u.TotalTokens != 420 {
		t.Errorf("usage = %+v", u)
	}
}

func TestOpenAIErrors(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		status int
		want   error
	}{
		{http.StatusTooManyRequests, ai.ErrQuota},
		{http.StatusServiceUnavailable, ai.ErrBusy},
		{http.StatusUnauthorized, ai.ErrInvalidKey},
	} {
		cl, _ := openaiClient(t, "gpt-6-luna", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(c.status)
		})
		if err := cl.TestModel(t.Context(), "gpt-6-luna"); !errors.Is(err, c.want) {
			t.Errorf("status %d: err = %v, want %v", c.status, err, c.want)
		}
	}
}

func TestOpenAIWithoutKey(t *testing.T) {
	t.Parallel()
	c, _ := newClient(t, "main", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(okResponse)) })
	c.ModelOf = func(context.Context, string) string { return "gpt-6-luna" }
	if _, err := c.Annotate(t.Context(), ai.AnnotateRequest{Sentences: []string{"Hi."}}); !errors.Is(err, ai.ErrNotConfigured) {
		t.Errorf("err = %v, want ErrNotConfigured", err)
	}
}
