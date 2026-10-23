package llm_test

import (
	"errors"
	"net/http"
	"slices"
	"testing"

	"github.com/luongtran/luna/backend/internal/ai"
)

func TestBackupKeyOnQuota(t *testing.T) {
	t.Parallel()
	var used []string
	c, _ := newClient(t, "main", func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("x-goog-api-key")
		used = append(used, key)
		if key == "main" {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(okResponse))
	})
	log := &usageLog{}
	c.Usage = log
	c.AddBackupKey("")
	c.AddBackupKey("backup")
	req := ai.AnnotateRequest{Sentences: []string{"We went home."}, Level: "A1"}
	if _, err := c.Annotate(t.Context(), req); err != nil {
		t.Fatalf("first: %v", err)
	}
	// The main key rests after its 429: the next request goes to the backup first.
	if _, err := c.Annotate(t.Context(), req); err != nil {
		t.Fatalf("second: %v", err)
	}
	if !slices.Equal(used, []string{"main", "backup", "backup"}) {
		t.Errorf("keys used %v", used)
	}
	var keys []int
	for _, u := range log.list {
		keys = append(keys, u.Key)
	}
	if !slices.Equal(keys, []int{0, 1, 1}) || log.list[0].Status != http.StatusTooManyRequests {
		t.Errorf("recorded keys %v, first status %d", keys, log.list[0].Status)
	}
}

func TestEveryKeyOutOfRequests(t *testing.T) {
	t.Parallel()
	c, calls := newClient(t, "main", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	})
	c.AddBackupKey("backup")
	if _, err := c.Annotate(t.Context(), ai.AnnotateRequest{Sentences: []string{"Hi."}}); !errors.Is(err, ai.ErrQuota) {
		t.Fatalf("err = %v", err)
	}
	if calls.Load() != 2 {
		t.Errorf("%d requests, want one per key", calls.Load())
	}
}

func TestOtherErrorsDoNotSwitchKey(t *testing.T) {
	t.Parallel()
	c, calls := newClient(t, "main", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	c.AddBackupKey("backup")
	if _, err := c.Annotate(t.Context(), ai.AnnotateRequest{Sentences: []string{"Hi."}}); err == nil {
		t.Fatal("want an error")
	}
	if calls.Load() != 1 {
		t.Errorf("%d requests, want 1", calls.Load())
	}
}
