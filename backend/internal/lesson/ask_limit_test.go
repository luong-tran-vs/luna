package lesson

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeAskUsage keeps the asked texts per user and lesson.
type fakeAskUsage struct {
	mu   sync.Mutex
	rows map[string][]string
}

func newFakeAskUsage() *fakeAskUsage { return &fakeAskUsage{rows: map[string][]string{}} }

func (f *fakeAskUsage) Asked(_ context.Context, userID, lessonID string) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.rows[userID+"|"+lessonID]), nil
}

func (f *fakeAskUsage) Add(_ context.Context, userID, lessonID, text string, _ time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	k := userID + "|" + lessonID
	if !slices.Contains(f.rows[k], text) {
		f.rows[k] = append(f.rows[k], text)
	}
	return nil
}

func TestAskAsLimitsNewWordsPerLesson(t *testing.T) {
	t.Parallel()
	e := newAskEnv(t)
	usage := newFakeAskUsage()
	e.r.WithAskLimit(usage)
	ask := func(user, text string, sentence int, unlimited bool) (AskQuota, error) {
		_, _, q, err := e.r.AskAs(t.Context(), user, e.lesson.ID, text, sentence, unlimited)
		return q, err
	}

	q, err := e.r.AskQuota(t.Context(), "u1", e.lesson.ID, false)
	if err != nil || q.Limit != AskLimit || len(q.Asked) != 0 {
		t.Fatalf("start = %+v, %v", q, err)
	}
	if q, err = ask("u1", "Gave up", 1, false); err != nil || !slices.Equal(q.Asked, []string{"gave up"}) {
		t.Fatalf("first = %+v, %v", q, err)
	}
	if q, err = ask("u1", "park", 0, false); err != nil || len(q.Asked) != 2 {
		t.Fatalf("second = %+v, %v", q, err)
	}
	calls, _ := e.ai.explains()
	if q, err = ask("u1", "studies", 2, false); !errors.Is(err, ErrAskLimit) || len(q.Asked) != 2 {
		t.Fatalf("third = %+v, %v", q, err)
	}
	if after, _ := e.ai.explains(); after != calls {
		t.Fatal("a refused ask reached the AI")
	}
	// Asking again about an asked word is free.
	if _, err = ask("u1", "gave up", 1, false); err != nil {
		t.Fatalf("again: %v", err)
	}
	// Another learner and admins have their own (or no) limit.
	if _, err = ask("u2", "studies", 2, false); err != nil {
		t.Fatalf("other learner: %v", err)
	}
	for _, w := range []struct {
		text     string
		sentence int
	}{{"we", 0}, {"park", 0}, {"home", 3}} {
		if _, err = ask("admin", w.text, w.sentence, true); err != nil {
			t.Fatalf("admin %q: %v", w.text, err)
		}
	}
	if q, _ = e.r.AskQuota(t.Context(), "admin", e.lesson.ID, true); q.Limit != 0 {
		t.Fatalf("admin quota = %+v", q)
	}
	if rows, _ := usage.Asked(t.Context(), "admin", e.lesson.ID); len(rows) != 0 {
		t.Fatalf("admin counted: %v", rows)
	}
}

func TestAskLimitEndpoint(t *testing.T) {
	t.Parallel()
	mux, e := newAskAPI(t)
	e.r.WithAskLimit(newFakeAskUsage())

	code, out := postAsk(t, mux, e.lesson.ID, "learner", `{"text":"gave up","sentenceIndex":1}`)
	if q, _ := out["quota"].(map[string]any); code != http.StatusOK || q["limit"] != float64(2) || len(q["asked"].([]any)) != 1 {
		t.Fatalf("first: %d %v", code, out)
	}
	_, _ = postAsk(t, mux, e.lesson.ID, "learner", `{"text":"park","sentenceIndex":0}`)
	code, out = postAsk(t, mux, e.lesson.ID, "learner", `{"text":"studies","sentenceIndex":2}`)
	if code != http.StatusTooManyRequests || out["error"] != "ask_limit" ||
		out["message"] != "Mỗi bài chỉ được hỏi AI 2 từ. Bạn vẫn hỏi lại được các từ đã hỏi." {
		t.Fatalf("third: %d %v", code, out)
	}

	rec := get(t, mux, "/api/lessons/"+e.lesson.ID, "learner")
	if !strings.Contains(rec.Body.String(), `"askQuota":{"limit":2,"asked":["gave up","park"]}`) {
		t.Fatalf("reading view: %s", rec.Body)
	}
	if rec := get(t, mux, "/api/lessons/"+e.lesson.ID, "admin"); !strings.Contains(rec.Body.String(), `"askQuota":{"limit":0,"asked":[]}`) {
		t.Fatalf("admin view: %s", rec.Body)
	}
}
