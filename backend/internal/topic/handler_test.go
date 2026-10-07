package topic

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
	env *testEnv
	mux *http.ServeMux
}

func newAPI(t *testing.T) *api {
	t.Helper()
	e := newEnv()
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

func (a *api) createTopic(t *testing.T, name string) string {
	t.Helper()
	rec := a.do(t, http.MethodPost, "/api/admin/topics", "admin", `{"name":"`+name+`","description":""}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	return decode(t, rec)["topic"].(map[string]any)["id"].(string)
}

func TestAdminTopicEndpointsRequireAdmin(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	endpoints := []struct{ method, path, body string }{
		{http.MethodGet, "/api/admin/topics", ""},
		{http.MethodPost, "/api/admin/topics", `{"name":"x","description":""}`},
		{http.MethodPut, "/api/admin/topics/x", `{"name":"x","description":""}`},
		{http.MethodDelete, "/api/admin/topics/x", ""},
		{http.MethodGet, "/api/admin/topics/x/roadmap?level=A1", ""},
		{http.MethodPut, "/api/admin/topics/x/roadmap?level=A1", `{"lessonIds":[]}`},
	}
	for _, ep := range endpoints {
		if rec := a.do(t, ep.method, ep.path, "learner", ep.body); rec.Code != http.StatusForbidden {
			t.Errorf("learner %s %s: %d", ep.method, ep.path, rec.Code)
		}
		if rec := a.do(t, ep.method, ep.path, "", ep.body); rec.Code != http.StatusUnauthorized {
			t.Errorf("anonymous %s %s: %d", ep.method, ep.path, rec.Code)
		}
	}
}

func TestTopicCRUDEndpoints(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	rec := a.do(t, http.MethodPost, "/api/admin/topics", "admin", `{"name":"Gia đình","description":"Người thân"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	topic := decode(t, rec)["topic"].(map[string]any)
	for key, want := range map[string]any{"name": "Gia đình", "description": "Người thân", "lessonCount": 0.0} {
		if topic[key] != want {
			t.Fatalf("%s = %v in %v", key, topic[key], topic)
		}
	}
	if levels, ok := topic["levels"].([]any); !ok || len(levels) != 0 {
		t.Fatalf("levels = %v", topic["levels"])
	}
	id := topic["id"].(string)

	rec = a.do(t, http.MethodPost, "/api/admin/topics", "admin", `{"name":"gia đình","description":""}`)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "Chủ đề này đã có") {
		t.Fatalf("duplicate: %d %s", rec.Code, rec.Body)
	}
	// A level is no longer part of a topic.
	for _, body := range []string{`{"name":"","description":""}`, `{"name":"x","level":"A1","description":""}`} {
		if rec := a.do(t, http.MethodPost, "/api/admin/topics", "admin", body); rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: %d", body, rec.Code)
		}
	}

	rec = a.do(t, http.MethodPut, "/api/admin/topics/"+id, "admin", `{"name":"Gia đình 2","description":""}`)
	if rec.Code != http.StatusOK || decode(t, rec)["topic"].(map[string]any)["name"] != "Gia đình 2" {
		t.Fatalf("update: %d %s", rec.Code, rec.Body)
	}
	if rec := a.do(t, http.MethodPut, "/api/admin/topics/nope", "admin", `{"name":"x","description":""}`); rec.Code != http.StatusNotFound {
		t.Fatalf("update unknown: %d", rec.Code)
	}

	a.env.lessons.add("l0", "Lesson", id, "B1")
	_ = a.env.repo.SetLessons(t.Context(), id, "B1", []string{"l0"})
	rec = a.do(t, http.MethodGet, "/api/admin/topics", "admin", "")
	want := `"levels":[{"level":"B1","lessonCount":1,"roadmapCount":1,"remaining":1,"warning":true}]`
	if rec.Code != http.StatusOK || len(decode(t, rec)["topics"].([]any)) != 1 || !strings.Contains(rec.Body.String(), want) {
		t.Fatalf("list: %d %s", rec.Code, rec.Body)
	}
	delete(a.env.lessons.lessons, "l0")

	a.env.lessons.add("l1", "Lesson", id, "A1")
	rec = a.do(t, http.MethodDelete, "/api/admin/topics/"+id, "admin", "")
	if body := decode(t, rec); rec.Code != http.StatusConflict || body["error"] != "topic_in_use" || body["count"] != 1.0 {
		t.Fatalf("delete in use: %d %s", rec.Code, rec.Body)
	}
	delete(a.env.lessons.lessons, "l1")
	if rec := a.do(t, http.MethodDelete, "/api/admin/topics/"+id, "admin", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: %d", rec.Code)
	}
	if rec := a.do(t, http.MethodDelete, "/api/admin/topics/"+id, "admin", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("delete again: %d", rec.Code)
	}
}

func TestPublicTopicsEndpoint(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	id := a.createTopic(t, "Gia đình")
	other := a.createTopic(t, "Công việc")
	_ = a.env.repo.SetLessons(t.Context(), id, "A1", []string{"l1"})
	_ = a.env.repo.SetLessons(t.Context(), other, "B1", []string{"l2"})

	rec := a.do(t, http.MethodGet, "/api/topics?level=A1", "learner", "")
	want := `{"topics":[{"id":"` + id + `","name":"Gia đình","level":"A1","description":"","lessonCount":1}]}` + "\n"
	if rec.Code != http.StatusOK || rec.Body.String() != want {
		t.Fatalf("public: %d %s", rec.Code, rec.Body)
	}
	for _, q := range []string{"?level=Q", ""} {
		if rec := a.do(t, http.MethodGet, "/api/topics"+q, "learner", ""); rec.Code != http.StatusBadRequest {
			t.Fatalf("level %q: %d", q, rec.Code)
		}
	}
	if rec := a.do(t, http.MethodGet, "/api/topics", "", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous: %d", rec.Code)
	}
}

func TestRoadmapEndpoints(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	id := a.createTopic(t, "Gia đình")
	other := a.createTopic(t, "Công việc")
	a.env.lessons.add("l1", "One", id, "A1")
	a.env.lessons.add("l2", "Two", id, "A1")
	a.env.lessons.add("l3", "Three", id, "A2")
	a.env.lessons.add("w1", "Work", other, "A1")
	path := "/api/admin/topics/" + id + "/roadmap?level=A1"

	rec := a.do(t, http.MethodPut, path, "admin", `{"lessonIds":["l2","l1"]}`)
	body := decode(t, rec)
	if rec.Code != http.StatusOK || body["remaining"] != 2.0 || body["warning"] != true || body["level"] != "A1" {
		t.Fatalf("put: %d %s", rec.Code, rec.Body)
	}
	lessons := body["lessons"].([]any)
	first := lessons[0].(map[string]any)
	if len(lessons) != 2 || first["id"] != "l2" || first["title"] != "Two" || first["topicName"] != "Gia đình" || first["inRoadmap"] != true {
		t.Fatalf("lessons = %v", lessons)
	}
	if lv := body["topic"].(map[string]any)["levels"].([]any)[0].(map[string]any); lv["level"] != "A1" || lv["roadmapCount"] != 2.0 {
		t.Fatalf("topic = %v", body["topic"])
	}

	if rec := a.do(t, http.MethodGet, path, "admin", ""); rec.Code != http.StatusOK {
		t.Fatalf("get: %d", rec.Code)
	}
	for _, b := range []string{`{"lessonIds":["l1","l1"]}`, `{"lessonIds":["zz"]}`, `{"lessonIds":["w1"]}`, `{"lessonIds":["l3"]}`} {
		if rec := a.do(t, http.MethodPut, path, "admin", b); rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: %d", b, rec.Code)
		}
	}
	if rec := a.do(t, http.MethodGet, "/api/admin/topics/nope/roadmap?level=A1", "admin", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown: %d", rec.Code)
	}
	if rec := a.do(t, http.MethodGet, "/api/admin/topics/"+id+"/roadmap", "admin", ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("no level: %d", rec.Code)
	}
}
