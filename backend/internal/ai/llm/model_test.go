package llm_test

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"testing"

	"github.com/luongtran/luna/backend/internal/ai"
)

func TestModelOfPicksTheModel(t *testing.T) {
	t.Parallel()
	var paths []string
	c, _ := newClient(t, "main", func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		_, _ = w.Write([]byte(okResponse))
	})
	log := &usageLog{}
	c.Usage = log
	model := "gemini-3.6-flash"
	var ops []string
	c.ModelOf = func(_ context.Context, op string) string {
		ops = append(ops, op)
		return model
	}
	req := ai.AnnotateRequest{Sentences: []string{"We went home."}, Level: "A1"}
	if _, err := c.Annotate(t.Context(), req); err != nil {
		t.Fatal(err)
	}
	// An empty answer falls back to the client's model.
	model = ""
	if _, err := c.Annotate(t.Context(), req); err != nil {
		t.Fatal(err)
	}
	want := []string{"/v1beta/models/gemini-3.6-flash:generateContent", "/v1beta/models/gemini-test:generateContent"}
	if !slices.Equal(paths, want) {
		t.Errorf("paths %v", paths)
	}
	if !slices.Equal(ops, []string{"annotate", "annotate"}) {
		t.Errorf("ops asked %v", ops)
	}
	if log.list[0].Model != "gemini-3.6-flash" || log.list[1].Model != "gemini-test" {
		t.Errorf("recorded models %q, %q", log.list[0].Model, log.list[1].Model)
	}
}

func TestTestModel(t *testing.T) {
	t.Parallel()
	var path string
	c, _ := newClient(t, "main", func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	log := &usageLog{}
	c.Usage = log
	if err := c.TestModel(t.Context(), "gemini-3.8-flash"); !errors.Is(err, ai.ErrBusy) {
		t.Fatalf("err = %v, want ErrBusy", err)
	}
	if path != "/v1beta/models/gemini-3.8-flash:generateContent" || log.list[0].Op != "model_test" {
		t.Errorf("path %q, op %q", path, log.list[0].Op)
	}
}
