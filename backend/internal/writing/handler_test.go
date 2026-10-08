package writing

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
	case "u1", "u2":
		return httpx.Principal{UserID: token, Role: "learner"}, nil
	}
	return httpx.Principal{}, errors.New("no session")
}

// allowAll is a guard that lets every request through.
func allowAll(next http.Handler) http.Handler { return next }

type api struct {
	env *env
	mux *http.ServeMux
}

func newAPI(t *testing.T) *api {
	t.Helper()
	e := newEnv()
	mux := http.NewServeMux()
	NewHandler(e.svc, slog.New(slog.DiscardHandler)).Register(mux, httpx.RequireAuth(resolver), allowAll)
	return &api{env: e, mux: mux}
}

func (a *api) do(t *testing.T, method, path, token, body string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.AddCookie(&http.Cookie{Name: httpx.SessionCookieName, Value: token})
	}
	rec := httptest.NewRecorder()
	a.mux.ServeHTTP(rec, req)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

func textBody(text string) string {
	b, _ := json.Marshal(map[string]string{"text": text})
	return string(b)
}

func TestLessonWritingEndpoints(t *testing.T) {
	t.Parallel()
	a := newAPI(t)

	code, body := a.do(t, http.MethodGet, "/api/lessons/l1/writing", "u1", "")
	if code != http.StatusOK || body["prompt"] != "Write about your family." || body["level"] != "A1" ||
		body["canWrite"] != true || body["writing"] != nil {
		t.Fatalf("get: %d %v", code, body)
	}

	code, body = a.do(t, http.MethodPut, "/api/lessons/l1/writing", "u1", textBody("My family"))
	w, _ := body["writing"].(map[string]any)
	if code != http.StatusOK || w["text"] != "My family" || w["status"] != "draft" || w["grade"] != nil || w["submittedAt"] != nil {
		t.Fatalf("draft: %d %v", code, body)
	}

	code, body = a.do(t, http.MethodPost, "/api/lessons/l1/writing/submit", "u1", textBody("one two three four"))
	if fields, _ := body["fields"].(map[string]any); code != http.StatusBadRequest || fields["text"] != "Bài viết cần ít nhất 5 từ" {
		t.Fatalf("short: %d %v", code, body)
	}

	code, body = a.do(t, http.MethodPost, "/api/lessons/l1/writing/submit", "u1", textBody("My family has four people."))
	w, _ = body["writing"].(map[string]any)
	g, _ := w["grade"].(map[string]any)
	if code != http.StatusOK || w["status"] != "submitted" || w["prompt"] != "Write about your family." || g["status"] != "pending" ||
		g["average"] != nil || len(g["criteria"].([]any)) != 0 {
		t.Fatalf("submit: %d %v", code, body)
	}

	if code, body := a.do(t, http.MethodPost, "/api/lessons/l1/writing/submit", "u1", textBody(words(10))); code != http.StatusConflict ||
		body["error"] != "already_submitted" {
		t.Fatalf("again: %d %v", code, body)
	}
	if code, body := a.do(t, http.MethodPut, "/api/lessons/l1/writing", "u1", textBody("x")); code != http.StatusConflict ||
		body["error"] != "already_submitted" {
		t.Fatalf("draft after submit: %d %v", code, body)
	}
}

func TestLessonWritingErrors(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	for _, c := range []struct {
		method, path, token, body string
		code                      int
		err                       string
	}{
		{http.MethodPut, "/api/lessons/l2/writing", "u1", textBody("x"), http.StatusConflict, "write_locked"},
		{http.MethodPost, "/api/lessons/l2/writing/submit", "u1", textBody(words(10)), http.StatusConflict, "write_locked"},
		{http.MethodGet, "/api/lessons/missing/writing", "u1", "", http.StatusNotFound, "not_found"},
		{http.MethodPut, "/api/lessons/l1/writing", "u1", `{"text":`, http.StatusBadRequest, "invalid_body"},
		{http.MethodGet, "/api/lessons/l1/writing", "", "", http.StatusUnauthorized, "unauthenticated"},
	} {
		if code, body := a.do(t, c.method, c.path, c.token, c.body); code != c.code || body["error"] != c.err {
			t.Errorf("%s %s: %d %v", c.method, c.path, code, body)
		}
	}
}

func TestLessonWritingRoutesAreGuarded(t *testing.T) {
	t.Parallel()
	e := newEnv()
	mux := http.NewServeMux()
	deny := func(http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusForbidden) })
	}
	NewHandler(e.svc, slog.New(slog.DiscardHandler)).Register(mux, httpx.RequireAuth(resolver), deny)
	a := &api{env: e, mux: mux}
	for _, p := range []string{"/api/lessons/l1/writing"} {
		if code, _ := a.do(t, http.MethodGet, p, "u1", ""); code != http.StatusForbidden {
			t.Errorf("%s: %d", p, code)
		}
	}
	if code, _ := a.do(t, http.MethodPost, "/api/lessons/l1/writing/submit", "u1", textBody(words(10))); code != http.StatusForbidden {
		t.Errorf("submit: %d", code)
	}
}

// --- US2 / US3: the learner's writings ---

func TestWritingsEndpoints(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	w, j := a.env.submitted(t)

	code, body := a.do(t, http.MethodGet, "/api/writings/unseen-count", "u1", "")
	if code != http.StatusOK || body["unseen"] != float64(0) || body["pending"] != float64(1) || body["latest"] != nil {
		t.Fatalf("pending count: %d %v", code, body)
	}
	if code, body := a.do(t, http.MethodPost, "/api/writings/"+w.ID+"/regrade", "u1", ""); code != http.StatusConflict ||
		body["error"] != "not_failed" {
		t.Fatalf("regrade pending: %d %v", code, body)
	}

	_ = a.env.svc.ProcessGrade(t.Context(), j)
	code, body = a.do(t, http.MethodGet, "/api/writings", "u1", "")
	list, _ := body["writings"].([]any)
	if code != http.StatusOK || len(list) != 1 {
		t.Fatalf("list: %d %v", code, body)
	}
	row := list[0].(map[string]any)
	if row["id"] != w.ID || row["lessonTitle"] != "My family" || row["gradeStatus"] != "done" || row["average"] != 3.8 || row["seen"] != false {
		t.Fatalf("row = %v", row)
	}
	code, body = a.do(t, http.MethodGet, "/api/writings/unseen-count", "u1", "")
	latest, _ := body["latest"].(map[string]any)
	if body["unseen"] != float64(1) || latest["id"] != w.ID || latest["status"] != "done" {
		t.Fatalf("unseen: %d %v", code, body)
	}

	code, body = a.do(t, http.MethodGet, "/api/writings/"+w.ID, "u1", "")
	wj, _ := body["writing"].(map[string]any)
	g, _ := wj["grade"].(map[string]any)
	if code != http.StatusOK || g["status"] != "done" || g["average"] != 3.8 || len(g["criteria"].([]any)) != 4 || g["gradedAt"] == nil {
		t.Fatalf("detail: %d %v", code, body)
	}
	if code, _ := a.do(t, http.MethodGet, "/api/writings/"+w.ID, "u2", ""); code != http.StatusNotFound {
		t.Fatalf("u2 detail: %d", code)
	}
	if code, _ := a.do(t, http.MethodPost, "/api/writings/"+w.ID+"/seen", "u2", ""); code != http.StatusNotFound {
		t.Fatalf("u2 seen: %d", code)
	}
	if code, _ := a.do(t, http.MethodPost, "/api/writings/"+w.ID+"/seen", "u1", ""); code != http.StatusNoContent {
		t.Fatalf("seen: %d", code)
	}
	if _, body := a.do(t, http.MethodGet, "/api/writings/unseen-count", "u1", ""); body["unseen"] != float64(0) {
		t.Fatalf("after seen: %v", body)
	}
	if code, _ := a.do(t, http.MethodGet, "/api/writings", "", ""); code != http.StatusUnauthorized {
		t.Fatalf("anonymous: %d", code)
	}
}

func TestRegradeEndpoint(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	w, j := a.env.submitted(t)
	a.env.svc.JobFailed(t.Context(), j, errors.New("boom"))

	_, body := a.do(t, http.MethodGet, "/api/writings/"+w.ID, "u1", "")
	g := body["writing"].(map[string]any)["grade"].(map[string]any)
	if g["status"] != "failed" || g["error"] != "Không chấm được bài" {
		t.Fatalf("failed grade = %v", g)
	}
	if code, _ := a.do(t, http.MethodPost, "/api/writings/"+w.ID+"/regrade", "u2", ""); code != http.StatusNotFound {
		t.Fatalf("u2 regrade: %d", code)
	}
	code, body := a.do(t, http.MethodPost, "/api/writings/"+w.ID+"/regrade", "u1", "")
	if code != http.StatusAccepted || body["writing"].(map[string]any)["grade"].(map[string]any)["status"] != "pending" {
		t.Fatalf("regrade: %d %v", code, body)
	}
}

func TestResubmitEndpoint(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	w, j := a.env.submitted(t)
	gradings := func(body map[string]any) any {
		wr, _ := body["writing"].(map[string]any)
		return wr["gradings"]
	}
	code, body := a.do(t, http.MethodGet, "/api/writings/"+w.ID, "u1", "")
	if code != http.StatusOK || fmt.Sprint(gradings(body)) != "map[max:2 used:1]" {
		t.Fatalf("detail: %d %v", code, body)
	}
	if code, body := a.do(t, http.MethodPost, "/api/writings/"+w.ID+"/resubmit", "u1", textBody("My family has four people now.")); code != http.StatusConflict || body["error"] != "grading" {
		t.Fatalf("while grading: %d %v", code, body)
	}
	_ = a.env.svc.ProcessGrade(t.Context(), j)
	code, body = a.do(t, http.MethodPost, "/api/writings/"+w.ID+"/resubmit", "u1", textBody("My family has four people now."))
	if code != http.StatusAccepted || fmt.Sprint(gradings(body)) != "map[max:2 used:2]" {
		t.Fatalf("resubmit: %d %v", code, body)
	}
	_ = a.env.svc.ProcessGrade(t.Context(), a.env.lastJob())
	code, body = a.do(t, http.MethodPost, "/api/writings/"+w.ID+"/resubmit", "u1", textBody("My family has four people again."))
	if code != http.StatusConflict || body["error"] != "no_gradings" || body["message"] != "Mỗi bài viết chỉ được chấm 2 lần" {
		t.Fatalf("third: %d %v", code, body)
	}
	if code, _ := a.do(t, http.MethodPost, "/api/writings/"+w.ID+"/resubmit", "u2", textBody("My family has four people now.")); code != http.StatusNotFound {
		t.Fatalf("u2: %d", code)
	}
}
