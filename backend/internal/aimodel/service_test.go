package aimodel_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/aimodel"
)

type memRepo struct {
	c     aimodel.Choice
	reads int
	err   error
}

func (r *memRepo) Get(context.Context) (aimodel.Choice, error) { r.reads++; return r.c, r.err }
func (r *memRepo) Save(_ context.Context, c aimodel.Choice) error {
	if r.err != nil {
		return r.err
	}
	r.c = c
	return nil
}

type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

func newService(repo *memRepo, tester aimodel.Tester) (*aimodel.Service, *clock) {
	clk := &clock{t: time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC)}
	return aimodel.NewService(repo, "gemini-3.5-flash-lite", true, tester, clk.now, slog.New(slog.DiscardHandler)), clk
}

func TestModelDefaultsAndCaches(t *testing.T) {
	t.Parallel()
	repo := &memRepo{}
	svc, clk := newService(repo, nil)
	if m := svc.ModelFor(t.Context(), "explain"); m != "gemini-3.5-flash-lite" {
		t.Fatalf("model %q, want the default", m)
	}
	// A choice saved elsewhere (the other server) shows once the cache is old.
	repo.c = aimodel.Choice{Model: "gemini-3.6-flash"}
	if m := svc.ModelFor(t.Context(), "explain"); m != "gemini-3.5-flash-lite" || repo.reads != 1 {
		t.Fatalf("model %q after %d reads, want the cached default", m, repo.reads)
	}
	clk.t = clk.t.Add(31 * time.Second)
	if m := svc.ModelFor(t.Context(), "explain"); m != "gemini-3.6-flash" {
		t.Fatalf("model %q, want the saved one", m)
	}
	// A failed read keeps the last model known.
	repo.err = errors.New("down")
	clk.t = clk.t.Add(31 * time.Second)
	if m := svc.ModelFor(t.Context(), "explain"); m != "gemini-3.6-flash" {
		t.Fatalf("model %q, want the last known", m)
	}
}

func TestSet(t *testing.T) {
	t.Parallel()
	repo := &memRepo{}
	svc, _ := newService(repo, nil)
	if _, err := svc.Set(t.Context(), "gpt-5", nil); !errors.Is(err, aimodel.ErrUnknownModel) {
		t.Fatalf("err = %v, want ErrUnknownModel", err)
	}
	s, err := svc.Set(t.Context(), "gemini-3.8-flash", nil)
	if err != nil {
		t.Fatal(err)
	}
	if s.Model != "gemini-3.8-flash" || s.Saved != "gemini-3.8-flash" || s.Default != "gemini-3.5-flash-lite" {
		t.Errorf("state %+v", s)
	}
	// The new model is used at once, without waiting for the cache.
	if m := svc.ModelFor(t.Context(), "explain"); m != "gemini-3.8-flash" {
		t.Errorf("model %q", m)
	}
	s, err = svc.Set(t.Context(), "", nil)
	if err != nil || s.Model != "gemini-3.5-flash-lite" || s.Saved != "" {
		t.Errorf("reset: %+v, %v", s, err)
	}
}

type fakeTester struct {
	model string
	err   error
}

func (f *fakeTester) TestModel(_ context.Context, model string) error {
	f.model = model
	return f.err
}

func TestTest(t *testing.T) {
	t.Parallel()
	svc, _ := newService(&memRepo{}, nil)
	if _, err := svc.Test(t.Context(), "gemini-3.6-flash"); !errors.Is(err, aimodel.ErrNoProvider) {
		t.Fatalf("err = %v, want ErrNoProvider", err)
	}
	tester := &fakeTester{}
	svc, _ = newService(&memRepo{}, tester)
	if _, err := svc.Test(t.Context(), "nope"); !errors.Is(err, aimodel.ErrUnknownModel) {
		t.Fatalf("err = %v, want ErrUnknownModel", err)
	}
	if _, err := svc.Test(t.Context(), "gemini-3.7-flash"); err != nil || tester.model != "gemini-3.7-flash" {
		t.Errorf("err %v, tested %q", err, tester.model)
	}
}

func TestModelByOp(t *testing.T) {
	t.Parallel()
	repo := &memRepo{}
	svc, _ := newService(repo, nil)
	if _, err := svc.Set(t.Context(), "gemini-3.6-flash", map[string]string{"nope": "gemini-3.5-flash"}); !errors.Is(err, aimodel.ErrUnknownOp) {
		t.Fatalf("unknown op: %v", err)
	}
	if _, err := svc.Set(t.Context(), "", map[string]string{"talk": "gpt-5"}); !errors.Is(err, aimodel.ErrUnknownModel) {
		t.Fatalf("unknown model: %v", err)
	}
	s, err := svc.Set(t.Context(), "", map[string]string{"explain": "openai/gpt-oss-120b", "talk": "openai/gpt-oss-120b", "grade": ""})
	if err != nil {
		t.Fatal(err)
	}
	// An op mapped to "" is not kept: it follows the model of every op.
	if len(s.ByOp) != 2 || s.ByOp["explain"] != "openai/gpt-oss-120b" || s.Model != "gemini-3.5-flash-lite" {
		t.Errorf("state %+v", s)
	}
	for op, want := range map[string]string{
		"explain": "openai/gpt-oss-120b", "talk": "openai/gpt-oss-120b",
		"grade": "gemini-3.5-flash-lite", "curriculum_draft": "gemini-3.5-flash-lite",
	} {
		if got := svc.ModelFor(t.Context(), op); got != want {
			t.Errorf("%s: %q, want %q", op, got, want)
		}
	}
	if _, err := svc.Set(t.Context(), "gemini-3.6-flash", s.ByOp); err != nil {
		t.Fatal(err)
	}
	if got := svc.ModelFor(t.Context(), "grade"); got != "gemini-3.6-flash" {
		t.Errorf("grade follows the model of every op: %q", got)
	}

	// Without the Groq key its models are not offered: the ops fall back to the model of every op.
	noGroq := aimodel.NewService(repo, "gemini-3.5-flash-lite", false, nil, time.Now, slog.New(slog.DiscardHandler))
	if got := noGroq.ModelFor(t.Context(), "explain"); got != "gemini-3.6-flash" {
		t.Errorf("without groq: %q", got)
	}
}
