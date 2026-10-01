package lesson

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

func newAskAPI(t *testing.T) (*http.ServeMux, *askEnv) {
	t.Helper()
	e := newAskEnv(t)
	mux := http.NewServeMux()
	NewReadingHandler(e.r, slog.New(slog.DiscardHandler)).Register(mux, httpx.RequireAuth(resolver), allowAll)
	return mux, e
}

func postAsk(t *testing.T, mux http.Handler, lessonID, token, body string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/lessons/"+lessonID+"/ask", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.AddCookie(&http.Cookie{Name: httpx.SessionCookieName, Value: token})
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

func TestAskEndpoint(t *testing.T) {
	t.Parallel()
	mux, e := newAskAPI(t)

	code, out := postAsk(t, mux, e.lesson.ID, "learner", `{"text":"Gave up","sentenceIndex":1}`)
	if code != http.StatusOK || out["cached"] != false {
		t.Fatalf("ask: %d %v", code, out)
	}
	res := out["result"].(map[string]any)
	meanings := res["meanings"].([]any)
	if res["source"] != "ai" || res["text"] != "gave up" || res["lemma"] != "give up" || res["note"] != "Ở đây là bỏ hút thuốc." ||
		len(meanings) != 1 || meanings[0].(map[string]any)["text"] != "bỏ (thói quen)" || meanings[0].(map[string]any)["pos"] != "" {
		t.Fatalf("result = %v", res)
	}
	if code, out = postAsk(t, mux, e.lesson.ID, "learner", `{"text":"gave up","sentenceIndex":1}`); code != http.StatusOK || out["cached"] != true {
		t.Fatalf("second ask: %d %v", code, out)
	}

	// Lookup returns the stored explanation, with its note; dictionary results have note "".
	rec := get(t, mux, "/api/lessons/"+e.lesson.ID+"/lookup?q=gave+up&sentence=1", "learner")
	if !strings.Contains(rec.Body.String(), `"note":"Ở đây là bỏ hút thuốc."`) {
		t.Fatalf("lookup: %s", rec.Body)
	}
	rec = get(t, mux, "/api/lessons/"+e.lesson.ID+"/lookup?q=park&sentence=0", "learner")
	if !strings.Contains(rec.Body.String(), `"source":"dictionary"`) || !strings.Contains(rec.Body.String(), `"note":""`) {
		t.Fatalf("dictionary lookup: %s", rec.Body)
	}
}

func TestAskEndpointRequests(t *testing.T) {
	t.Parallel()
	mux, e := newAskAPI(t)
	for _, c := range []struct {
		name, lesson, token, body string
		code                      int
		errCode, field            string
	}{
		{"no session", e.lesson.ID, "", `{"text":"gave up","sentenceIndex":1}`, http.StatusUnauthorized, "", ""},
		{"broken JSON", e.lesson.ID, "learner", `{"text":`, http.StatusBadRequest, "invalid_body", ""},
		{"empty text", e.lesson.ID, "learner", `{"text":" ","sentenceIndex":1}`, http.StatusBadRequest, "validation_failed", "text"},
		{"not in sentence", e.lesson.ID, "learner", `{"text":"park","sentenceIndex":1}`, http.StatusBadRequest, "validation_failed", "text"},
		{"no sentence", e.lesson.ID, "learner", `{"text":"gave up","sentenceIndex":7}`, http.StatusBadRequest, "validation_failed", "sentenceIndex"},
		{"missing lesson", "000000000000000000000000", "learner", `{"text":"x","sentenceIndex":0}`, http.StatusNotFound, "not_found", ""},
	} {
		code, out := postAsk(t, mux, c.lesson, c.token, c.body)
		if code != c.code || (c.errCode != "" && out["error"] != c.errCode) {
			t.Errorf("%s: %d %v", c.name, code, out)
			continue
		}
		if c.field != "" {
			if fields, _ := out["fields"].(map[string]any); fields[c.field] == nil {
				t.Errorf("%s: fields %v", c.name, out["fields"])
			}
		}
	}
	if calls, _ := e.ai.explains(); calls != 0 {
		t.Fatalf("AI called %d times", calls)
	}
}

func TestAskEndpointAIErrors(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name    string
		res     ai.Explanation
		err     error
		code    int
		errCode string
	}{
		{"not configured", ai.Explanation{}, ai.ErrNotConfigured, http.StatusServiceUnavailable, "ai_not_configured"},
		{"invalid key", ai.Explanation{}, ai.ErrInvalidKey, http.StatusServiceUnavailable, "ai_not_configured"},
		{"quota", ai.Explanation{}, ai.ErrQuota, http.StatusTooManyRequests, "ai_quota"},
		{"failed", ai.Explanation{}, errors.New("boom"), http.StatusBadGateway, "ai_failed"},
		{"deadline", ai.Explanation{}, context.DeadlineExceeded, http.StatusBadGateway, "ai_failed"},
		{"no meaning", ai.Explanation{Lemma: "give up"}, nil, http.StatusBadGateway, "ai_failed"},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			mux, e := newAskAPI(t)
			e.ai.explanation, e.ai.explainErr = c.res, c.err
			code, out := postAsk(t, mux, e.lesson.ID, "learner", `{"text":"gave up","sentenceIndex":1}`)
			if code != c.code || out["error"] != c.errCode || out["message"] == "" {
				t.Fatalf("%d %v", code, out)
			}
			if e.asks.count() != 0 {
				t.Fatal("a failed answer was stored")
			}
		})
	}
}

func TestAskRouteIsGuarded(t *testing.T) {
	t.Parallel()
	e := newAskEnv(t)
	mux := http.NewServeMux()
	deny := func(http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusForbidden) })
	}
	NewReadingHandler(e.r, slog.New(slog.DiscardHandler)).Register(mux, httpx.RequireAuth(resolver), deny)
	if code, _ := postAsk(t, mux, e.lesson.ID, "learner", `{"text":"gave up","sentenceIndex":1}`); code != http.StatusForbidden {
		t.Fatalf("status %d", code)
	}
	if calls, _ := e.ai.explains(); calls != 0 {
		t.Fatal("AI called behind a closed guard")
	}
}
