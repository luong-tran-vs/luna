package grammar

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

func resolver(_ context.Context, token string) (httpx.Principal, error) {
	switch token {
	case "admin":
		return httpx.Principal{UserID: "1", Role: "admin"}, nil
	case "learner":
		return httpx.Principal{UserID: "2", Role: "learner"}, nil
	}
	return httpx.Principal{}, errors.New("no session")
}

type api struct {
	env *env
	mux *http.ServeMux
}

func newAPI(t *testing.T) *api {
	t.Helper()
	e := newEnv(t)
	mux := http.NewServeMux()
	NewHandler(e.svc, slog.New(slog.DiscardHandler)).Register(mux, httpx.RequireAuth(resolver))
	return &api{env: e, mux: mux}
}

func (a *api) do(t *testing.T, method, path, token, body string) *httptest.ResponseRecorder {
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
	return rec
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("body %q: %v", rec.Body, err)
	}
	return out
}

const adminPath = "/api/admin/grammar-lessons/" + point

func (a *api) generate(t *testing.T) {
	t.Helper()
	if rec := a.do(t, http.MethodPost, adminPath+"/generate", "admin", `{"force":false}`); rec.Code != http.StatusCreated {
		t.Fatalf("generate: %d %s", rec.Code, rec.Body)
	}
}

func (a *api) publish(t *testing.T) {
	t.Helper()
	if rec := a.do(t, http.MethodPost, adminPath+"/publish", "admin", ""); rec.Code != http.StatusOK {
		t.Fatalf("publish: %d %s", rec.Code, rec.Body)
	}
}

func TestEndpointsRequireTheRightRole(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	admin := []struct{ method, path, body string }{
		{http.MethodGet, "/api/admin/grammar-lessons", ""},
		{http.MethodGet, adminPath, ""},
		{http.MethodPost, adminPath + "/generate", `{}`},
		{http.MethodPut, adminPath, `{"content":{}}`},
		{http.MethodPost, adminPath + "/publish", ""},
		{http.MethodPost, adminPath + "/unpublish", ""},
		{http.MethodPost, adminPath + "/check", ""},
		{http.MethodGet, "/api/admin/grammar-reports", ""},
		{http.MethodPost, "/api/admin/grammar-reports/resolve", `{}`},
	}
	for _, ep := range admin {
		if rec := a.do(t, ep.method, ep.path, "", ep.body); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s anonymous = %d", ep.method, ep.path, rec.Code)
		}
		if rec := a.do(t, ep.method, ep.path, "learner", ep.body); rec.Code != http.StatusForbidden {
			t.Errorf("%s %s learner = %d", ep.method, ep.path, rec.Code)
		}
	}
	learner := []struct{ method, path, body string }{
		{http.MethodGet, "/api/grammar", ""},
		{http.MethodGet, "/api/grammar/" + point, ""},
		{http.MethodPost, "/api/grammar/" + point + "/attempts", `{}`},
		{http.MethodPost, "/api/grammar/" + point + "/reports", `{}`},
	}
	for _, ep := range learner {
		if rec := a.do(t, ep.method, ep.path, "", ep.body); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s anonymous = %d", ep.method, ep.path, rec.Code)
		}
	}
	if a.env.ai.calls != 0 {
		t.Error("a refused request must not reach the AI")
	}
}

func TestAdminFlow(t *testing.T) {
	t.Parallel()
	a := newAPI(t)

	rec := a.do(t, http.MethodGet, adminPath, "admin", "")
	if rec.Code != http.StatusNotFound || decode(t, rec)["error"] != "not_found" {
		t.Fatalf("missing lesson: %d %s", rec.Code, rec.Body)
	}
	if rec := a.do(t, http.MethodGet, "/api/admin/grammar-lessons/nope", "admin", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown point: %d", rec.Code)
	}

	rec = a.do(t, http.MethodPost, adminPath+"/generate", "admin", `{"force":false}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("generate: %d %s", rec.Code, rec.Body)
	}
	l := decode(t, rec)["lesson"].(map[string]any)
	if l["pointId"] != point || l["status"] != "draft" || l["edited"] != false || l["publishedAt"] != nil || l["updatedAt"] == "" {
		t.Errorf("lesson = %v", l)
	}
	content := l["content"].(map[string]any)
	practice := content["practice"].([]any)
	first := practice[0].(map[string]any)
	if len(practice) != 8 || first["id"] != "p1" || first["kind"] != "choice" || first["answerIndex"] != float64(1) {
		t.Errorf("practice[0] = %v", first)
	}
	if fill := practice[1].(map[string]any); fill["kind"] != "fill" || fill["options"] != nil || fill["answers"] == nil {
		t.Errorf("fill exercise = %v", fill)
	}

	// 200 when it replaces, 409 once edited.
	if rec := a.do(t, http.MethodPost, adminPath+"/generate", "admin", `{}`); rec.Code != http.StatusOK {
		t.Fatalf("regenerate: %d %s", rec.Code, rec.Body)
	}
	body, _ := json.Marshal(map[string]any{"content": content})
	rec = a.do(t, http.MethodPut, adminPath, "admin", string(body))
	if rec.Code != http.StatusOK || decode(t, rec)["lesson"].(map[string]any)["edited"] != true {
		t.Fatalf("save: %d %s", rec.Code, rec.Body)
	}
	rec = a.do(t, http.MethodPost, adminPath+"/generate", "admin", `{"force":false}`)
	if rec.Code != http.StatusConflict || decode(t, rec)["error"] != "grammar_lesson_exists" {
		t.Fatalf("overwrite: %d %s", rec.Code, rec.Body)
	}
	if rec := a.do(t, http.MethodPost, adminPath+"/generate", "admin", `{"force":true}`); rec.Code != http.StatusOK {
		t.Fatalf("force: %d %s", rec.Code, rec.Body)
	}

	rec = a.do(t, http.MethodPost, adminPath+"/publish", "admin", "")
	published := decode(t, rec)["lesson"].(map[string]any)
	if rec.Code != http.StatusOK || published["status"] != "published" || published["publishedAt"] == nil {
		t.Fatalf("publish: %d %s", rec.Code, rec.Body)
	}
	rec = a.do(t, http.MethodGet, "/api/admin/grammar-lessons", "admin", "")
	items := decode(t, rec)["lessons"].([]any)
	if len(items) != 1 || items[0].(map[string]any)["pointId"] != point || items[0].(map[string]any)["status"] != "published" {
		t.Errorf("list = %s", rec.Body)
	}
	rec = a.do(t, http.MethodPost, adminPath+"/unpublish", "admin", "")
	if rec.Code != http.StatusOK || decode(t, rec)["lesson"].(map[string]any)["status"] != "draft" {
		t.Fatalf("unpublish: %d %s", rec.Code, rec.Body)
	}
}

func TestSaveReportsFieldErrors(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	a.generate(t)
	rec := a.do(t, http.MethodGet, adminPath, "admin", "")
	content := decode(t, rec)["lesson"].(map[string]any)["content"].(map[string]any)
	practice := content["practice"].([]any)
	practice[0].(map[string]any)["options"] = []string{"only", "two"}
	delete(practice[1].(map[string]any), "answers")
	content["objective"] = ""
	body, _ := json.Marshal(map[string]any{"content": content})

	rec = a.do(t, http.MethodPut, adminPath, "admin", string(body))
	got := decode(t, rec)
	fields, _ := got["fields"].(map[string]any)
	if rec.Code != http.StatusBadRequest || got["error"] != "validation_failed" {
		t.Fatalf("save: %d %s", rec.Code, rec.Body)
	}
	for _, key := range []string{"content.objective", "content.practice[0].options", "content.practice[1].answers"} {
		if fields[key] == nil {
			t.Errorf("missing field error %q in %v", key, fields)
		}
	}

	// A choice without answerIndex is reported, not read as the first option.
	delete(practice[3].(map[string]any), "answerIndex")
	body, _ = json.Marshal(map[string]any{"content": map[string]any{"practice": []any{practice[3]}}})
	rec = a.do(t, http.MethodPut, adminPath, "admin", string(body))
	if f, _ := decode(t, rec)["fields"].(map[string]any); f["content.practice[0].answerIndex"] == nil {
		t.Errorf("fields = %v", f)
	}

	for _, bad := range []string{`{"content":{"surprise":1}}`, `not json`, `{}`} {
		if rec := a.do(t, http.MethodPut, adminPath, "admin", bad); rec.Code != http.StatusBadRequest {
			t.Errorf("PUT %q = %d", bad, rec.Code)
		}
	}
	if rec := a.do(t, http.MethodPut, "/api/admin/grammar-lessons/a1-articles", "admin", string(body)); rec.Code != http.StatusNotFound {
		t.Errorf("PUT without lesson = %d", rec.Code)
	}
	if rec := a.do(t, http.MethodPost, adminPath+"/generate", "admin", `{"force":"yes"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("bad generate body = %d", rec.Code)
	}
}

func TestGenerateMapsAIErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		err    error
		status int
		code   string
	}{
		{ai.ErrNotConfigured, http.StatusServiceUnavailable, "ai_not_configured"},
		{ai.ErrInvalidKey, http.StatusServiceUnavailable, "ai_not_configured"},
		{ai.ErrQuota, http.StatusTooManyRequests, "ai_quota"},
		{errors.New("gemini: status 500"), http.StatusBadGateway, "ai_failed"},
	}
	for _, tt := range tests {
		a := newAPI(t)
		a.env.ai.err = tt.err
		rec := a.do(t, http.MethodPost, adminPath+"/generate", "admin", `{}`)
		if rec.Code != tt.status || decode(t, rec)["error"] != tt.code {
			t.Errorf("%v -> %d %s", tt.err, rec.Code, rec.Body)
		}
	}
	a := newAPI(t)
	a.env.ai.content = ai.GrammarLessonContent{}
	rec := a.do(t, http.MethodPost, adminPath+"/generate", "admin", `{}`)
	if rec.Code != http.StatusBadGateway || decode(t, rec)["error"] != "ai_failed" {
		t.Errorf("unusable content -> %d %s", rec.Code, rec.Body)
	}
	if rec := a.do(t, http.MethodPost, "/api/admin/grammar-lessons/nope/generate", "admin", `{}`); rec.Code != http.StatusNotFound {
		t.Errorf("unknown point = %d", rec.Code)
	}
}

func TestLearnerList(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	a.generate(t)
	a.publish(t)

	rec := a.do(t, http.MethodGet, "/api/grammar?level=A1", "learner", "")
	got := decode(t, rec)
	points := got["points"].([]any)
	if rec.Code != http.StatusOK || len(points) != len(Default().ByLevel("A1")) || got["next"] != point {
		t.Fatalf("list: %d next %v, %d points", rec.Code, got["next"], len(points))
	}
	first := points[0].(map[string]any)
	if first["id"] != point || first["available"] != true || first["status"] != "new" || first["bestMastery"] != float64(0) || first["titleVi"] == "" || first["hintVi"] == "" {
		t.Errorf("first = %v", first)
	}
	if second := points[1].(map[string]any); second["available"] != false {
		t.Errorf("second = %v", second)
	}

	rec = a.do(t, http.MethodGet, "/api/grammar", "learner", "")
	if all := decode(t, rec)["points"].([]any); len(all) != len(Default().All()) {
		t.Errorf("all levels = %d", len(all))
	}
	if rec := a.do(t, http.MethodGet, "/api/grammar?level=Z9", "learner", ""); rec.Code != http.StatusBadRequest {
		t.Errorf("bad level = %d", rec.Code)
	}
	rec = a.do(t, http.MethodGet, "/api/grammar?level=B2", "learner", "")
	if got := decode(t, rec); got["next"] != "" {
		t.Errorf("B2 next = %v", got["next"])
	}
}

func TestLearnerGetAndAttempts(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	path := "/api/grammar/" + point

	a.generate(t)
	if rec := a.do(t, http.MethodGet, path, "learner", ""); rec.Code != http.StatusNotFound || decode(t, rec)["error"] != "not_found" {
		t.Fatalf("draft: %d %s", rec.Code, rec.Body)
	}
	if rec := a.do(t, http.MethodPost, path+"/attempts", "learner", `{"kind":"practice","correct":8,"total":8,"wrong":[]}`); rec.Code != http.StatusNotFound {
		t.Fatalf("attempt on a draft: %d", rec.Code)
	}
	a.publish(t)

	rec := a.do(t, http.MethodGet, path, "learner", "")
	got := decode(t, rec)
	pt := got["point"].(map[string]any)
	progress := got["progress"].(map[string]any)
	if rec.Code != http.StatusOK || pt["id"] != point || pt["pattern"] == "" || len(pt["examples"].([]any)) == 0 ||
		got["lessonCount"] != float64(3) || got["content"].(map[string]any)["objective"] == "" {
		t.Fatalf("get: %d %s", rec.Code, rec.Body)
	}
	if progress["status"] != "new" || progress["mastered"] != false || progress["weak"] == nil || len(progress["weak"].([]any)) != 0 {
		t.Errorf("progress = %v", progress)
	}

	rec = a.do(t, http.MethodPost, path+"/attempts", "learner", `{"kind":"practice","correct":6,"total":8,"wrong":["p2","p5"]}`)
	res := decode(t, rec)
	progress = res["progress"].(map[string]any)
	if rec.Code != http.StatusOK || res["passed"] != false || progress["status"] != "learning" || progress["practiceAttempts"] != float64(1) ||
		progress["lastPractice"] != float64(75) || len(progress["weak"].([]any)) != 2 {
		t.Fatalf("practice attempt: %d %s", rec.Code, rec.Body)
	}
	rec = a.do(t, http.MethodPost, path+"/attempts", "learner", `{"kind":"mastery","correct":5,"total":6,"wrong":["m4"]}`)
	res = decode(t, rec)
	progress = res["progress"].(map[string]any)
	if res["passed"] != true || progress["status"] != "mastered" || progress["mastered"] != true || progress["bestMastery"] != float64(83) {
		t.Fatalf("mastery attempt: %d %s", rec.Code, rec.Body)
	}
	if rec := a.do(t, http.MethodGet, "/api/grammar?level=A1", "learner", ""); decode(t, rec)["next"] != "" {
		t.Errorf("after mastering the only lesson, next = %v", decode(t, rec)["next"])
	}

	for _, bad := range []string{
		`{"kind":"practice","correct":6,"total":8,"wrong":["p2"]}`,
		`{"kind":"mastery","correct":8,"total":8,"wrong":[]}`,
		`{"kind":"exam","correct":0,"total":0,"wrong":[]}`,
		`{"kind":"practice","correct":1,"total":8,"extra":true}`,
		`oops`,
	} {
		if rec := a.do(t, http.MethodPost, path+"/attempts", "learner", bad); rec.Code != http.StatusBadRequest {
			t.Errorf("attempt %s = %d", bad, rec.Code)
		}
	}
	rec = a.do(t, http.MethodPost, path+"/attempts", "learner", `{"kind":"practice","correct":6,"total":8,"wrong":["p2"]}`)
	if f, _ := decode(t, rec)["fields"].(map[string]any); f["wrong"] == nil {
		t.Errorf("fields = %s", rec.Body)
	}
}
