package progress

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

func resolver(_ context.Context, token string) (httpx.Principal, error) {
	switch token {
	case "an":
		return httpx.Principal{UserID: "u1", Role: "learner"}, nil
	case "binh":
		return httpx.Principal{UserID: "u2", Role: "admin"}, nil
	}
	return httpx.Principal{}, errors.New("no session")
}

// allowAll is a guard that lets every request through.
func allowAll(next http.Handler) http.Handler { return next }

func TestDictationRoutesAreGuarded(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService()
	mux := http.NewServeMux()
	deny := func(http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusForbidden) })
	}
	NewHandler(svc, slog.New(slog.DiscardHandler)).Register(mux, httpx.RequireAuth(resolver), deny)
	if rec := call(t, mux, http.MethodGet, dictationPath+"/summary", "an", ""); rec.Code != http.StatusForbidden {
		t.Fatalf("summary: %d", rec.Code)
	}
	if rec := call(t, mux, http.MethodPost, dictationPath, "an", `{"sentenceIndex":0,"typed":"a","correctWords":1,"totalWords":1}`); rec.Code != http.StatusForbidden {
		t.Fatalf("record: %d", rec.Code)
	}
}

func newAPI(t *testing.T) *http.ServeMux {
	t.Helper()
	svc, _ := newTestService()
	mux := http.NewServeMux()
	NewHandler(svc, slog.New(slog.DiscardHandler)).Register(mux, httpx.RequireAuth(resolver), allowAll)
	return mux
}

func call(t *testing.T, mux http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.AddCookie(&http.Cookie{Name: httpx.SessionCookieName, Value: token})
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

type summaryBody struct {
	Summary struct {
		SentenceCount int              `json:"sentenceCount"`
		CheckedCount  int              `json:"checkedCount"`
		CorrectWords  int              `json:"correctWords"`
		TotalWords    int              `json:"totalWords"`
		Rate          float64          `json:"rate"`
		Completed     bool             `json:"completed"`
		Results       []map[string]any `json:"results"`
	} `json:"summary"`
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) summaryBody {
	t.Helper()
	var b summaryBody
	if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil {
		t.Fatalf("decode %s: %v", rec.Body, err)
	}
	return b
}

const dictationPath = "/api/lessons/l1/dictation"

func TestRecordEndpoint(t *testing.T) {
	t.Parallel()
	mux := newAPI(t)

	rec := call(t, mux, http.MethodPost, dictationPath, "an", `{"sentenceIndex":0,"typed":"i dont like green apples","correctWords":4,"totalWords":5}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("post: %d %s", rec.Code, rec.Body)
	}
	b := decode(t, rec)
	if b.Summary.CheckedCount != 1 || b.Summary.SentenceCount != 3 || b.Summary.Rate != 0.8 || len(b.Summary.Results) != 1 {
		t.Fatalf("summary = %+v", b.Summary)
	}
	r := b.Summary.Results[0]
	if r["typed"] != "i dont like green apples" || r["checkedAt"] == nil || r["sentenceIndex"] != float64(0) {
		t.Fatalf("result = %v", r)
	}
	if _, leaked := r["userId"]; leaked {
		t.Fatal("response leaks userId")
	}

	rec = call(t, mux, http.MethodPost, dictationPath, "an", `{"sentenceIndex":0,"typed":"i don't like green apples","correctWords":5,"totalWords":5}`)
	if b := decode(t, rec); rec.Code != http.StatusOK || b.Summary.CheckedCount != 1 || b.Summary.CorrectWords != 5 {
		t.Fatalf("re-post: %d %s", rec.Code, rec.Body)
	}

	for i := 1; i < 3; i++ {
		rec = call(t, mux, http.MethodPost, dictationPath, "an", fmt.Sprintf(`{"sentenceIndex":%d,"typed":"x","correctWords":0,"totalWords":5}`, i))
	}
	if b := decode(t, rec); !b.Summary.Completed || b.Summary.Rate != 5.0/15 {
		t.Fatalf("completed: %s", rec.Body)
	}

	// Another user's summary is empty.
	rec = call(t, mux, http.MethodGet, dictationPath+"/summary", "binh", "")
	if b := decode(t, rec); rec.Code != http.StatusOK || b.Summary.CheckedCount != 0 || len(b.Summary.Results) != 0 {
		t.Fatalf("other user: %d %s", rec.Code, rec.Body)
	}
	rec = call(t, mux, http.MethodGet, dictationPath+"/summary", "an", "")
	if b := decode(t, rec); rec.Code != http.StatusOK || b.Summary.CheckedCount != 3 {
		t.Fatalf("own summary: %d %s", rec.Code, rec.Body)
	}
}

func TestRecordEndpointErrors(t *testing.T) {
	t.Parallel()
	mux := newAPI(t)
	cases := map[string]struct {
		body string
		want string
	}{
		"index out of range": {`{"sentenceIndex":3,"typed":"a","correctWords":1,"totalWords":1}`, `"sentenceIndex"`},
		"empty typed":        {`{"sentenceIndex":0,"typed":"","correctWords":0,"totalWords":1}`, `"typed"`},
		"total zero":         {`{"sentenceIndex":0,"typed":"a","correctWords":0,"totalWords":0}`, `"totalWords"`},
		"correct > total":    {`{"sentenceIndex":0,"typed":"a","correctWords":2,"totalWords":1}`, `"correctWords"`},
		"userId field":       {`{"sentenceIndex":0,"typed":"a","correctWords":1,"totalWords":1,"userId":"u2"}`, `"invalid_body"`},
		"not json":           {`nope`, `"invalid_body"`},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			rec := call(t, mux, http.MethodPost, dictationPath, "an", tc.body)
			if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), tc.want) {
				t.Fatalf("%d %s", rec.Code, rec.Body)
			}
		})
	}
}

func TestEndpointsNotFoundAndUnauthenticated(t *testing.T) {
	t.Parallel()
	mux := newAPI(t)
	body := `{"sentenceIndex":0,"typed":"a","correctWords":1,"totalWords":1}`
	if rec := call(t, mux, http.MethodPost, "/api/lessons/nope/dictation", "an", body); rec.Code != http.StatusNotFound {
		t.Fatalf("post 404: %d", rec.Code)
	}
	if rec := call(t, mux, http.MethodGet, "/api/lessons/nope/dictation/summary", "an", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("get 404: %d", rec.Code)
	}
	if rec := call(t, mux, http.MethodPost, dictationPath, "", body); rec.Code != http.StatusUnauthorized {
		t.Fatalf("post 401: %d", rec.Code)
	}
	if rec := call(t, mux, http.MethodGet, dictationPath+"/summary", "", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("get 401: %d", rec.Code)
	}
}

func TestResetDictationEndpoint(t *testing.T) {
	t.Parallel()
	mux := newAPI(t)
	body := `{"sentenceIndex":0,"typed":"a","correctWords":1,"totalWords":1}`
	for _, token := range []string{"an", "binh"} {
		if rec := call(t, mux, http.MethodPost, dictationPath, token, body); rec.Code != http.StatusOK {
			t.Fatalf("post %s: %d", token, rec.Code)
		}
	}

	rec := call(t, mux, http.MethodDelete, dictationPath, "an", "")
	if rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
		t.Fatalf("delete: %d %s", rec.Code, rec.Body)
	}
	if b := decode(t, call(t, mux, http.MethodGet, dictationPath+"/summary", "an", "")); b.Summary.CheckedCount != 0 || b.Summary.Completed {
		t.Fatalf("own summary after delete = %+v", b.Summary)
	}
	// Another user's results are untouched.
	if b := decode(t, call(t, mux, http.MethodGet, dictationPath+"/summary", "binh", "")); b.Summary.CheckedCount != 1 {
		t.Fatalf("other user's summary = %+v", b.Summary)
	}
	// Deleting again is fine; the step can be redone.
	if rec := call(t, mux, http.MethodDelete, dictationPath, "an", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete again: %d", rec.Code)
	}
	if rec := call(t, mux, http.MethodPost, dictationPath, "an", body); rec.Code != http.StatusOK || decode(t, rec).Summary.CheckedCount != 1 {
		t.Fatalf("post after delete: %d %s", rec.Code, rec.Body)
	}

	if rec := call(t, mux, http.MethodDelete, "/api/lessons/nope/dictation", "an", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("delete 404: %d", rec.Code)
	}
	if rec := call(t, mux, http.MethodDelete, dictationPath, "", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("delete 401: %d", rec.Code)
	}
}

func TestResetDictationIsGuarded(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService()
	mux := http.NewServeMux()
	deny := func(http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusForbidden) })
	}
	NewHandler(svc, slog.New(slog.DiscardHandler)).Register(mux, httpx.RequireAuth(resolver), deny)
	if rec := call(t, mux, http.MethodDelete, dictationPath, "an", ""); rec.Code != http.StatusForbidden {
		t.Fatalf("delete: %d", rec.Code)
	}
}
