package gemini_test

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"testing"

	"github.com/luongtran/luna/backend/internal/ai"
)

type usageLog struct {
	mu   sync.Mutex
	list []ai.Usage
}

func (l *usageLog) RecordUsage(_ context.Context, u ai.Usage) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.list = append(l.list, u)
}

func TestRecordsUsage(t *testing.T) {
	t.Parallel()
	inner, _ := json.Marshal(map[string]any{"words": []string{"ant"}})
	ok, _ := json.Marshal(map[string]any{
		"candidates": []any{map[string]any{"content": map[string]any{"parts": []any{map[string]any{"text": string(inner)}}}}},
		"usageMetadata": map[string]int{
			"promptTokenCount": 120, "candidatesTokenCount": 30, "thoughtsTokenCount": 5, "totalTokenCount": 155,
		},
	})
	quota := false
	c, _ := newClient(t, "k", func(w http.ResponseWriter, _ *http.Request) {
		if quota {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write(ok)
	})
	log := &usageLog{}
	c.Usage = log

	if _, err := c.SuggestWords(t.Context(), ai.SuggestWordsRequest{Count: 1}); err != nil {
		t.Fatal(err)
	}
	quota = true
	_, _ = c.SuggestWords(t.Context(), ai.SuggestWordsRequest{Count: 1})

	if len(log.list) != 2 {
		t.Fatalf("records = %+v", log.list)
	}
	u := log.list[0]
	if u.Op != "suggest_words" || u.Status != 200 || u.PromptTokens != 120 || u.OutputTokens != 30 ||
		u.ThoughtTokens != 5 || u.TotalTokens != 155 || u.Model == "" || u.At.IsZero() {
		t.Fatalf("first = %+v", u)
	}
	if u := log.list[1]; u.Status != http.StatusTooManyRequests || u.TotalTokens != 0 {
		t.Fatalf("quota = %+v", u)
	}
}
