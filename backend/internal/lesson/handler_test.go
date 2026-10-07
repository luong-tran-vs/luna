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
	"github.com/luongtran/luna/backend/internal/job"
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
	h := NewHandler(e.svc, slog.New(slog.DiscardHandler))
	mux := http.NewServeMux()
	h.Register(mux, httpx.RequireAuth(resolver))
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

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("body %q: %v", rec.Body.String(), err)
	}
	return out
}

const validBody = `{"title":"Park","content":"We went to the park. He gave up smoking.","topicId":"topic-b1","level":"B1","source":"Tự viết","license":"CC BY"}`

func (a *api) createLesson(t *testing.T) map[string]any {
	t.Helper()
	rec := a.do(t, http.MethodPost, "/api/admin/lessons", "admin", validBody)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	return decodeBody(t, rec)["lesson"].(map[string]any)
}

// --- permissions for every admin endpoint ---

func TestAdminEndpointsRequireAdmin(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	endpoints := []struct{ method, path, body string }{
		{http.MethodGet, "/api/admin/lessons", ""},
		{http.MethodPost, "/api/admin/lessons", validBody},
		{http.MethodGet, "/api/admin/lessons/x", ""},
		{http.MethodPut, "/api/admin/lessons/x", validBody},
		{http.MethodDelete, "/api/admin/lessons/x", ""},
		{http.MethodPut, "/api/admin/lessons/x/annotations", `{"annotations":[]}`},
		{http.MethodPost, "/api/admin/lessons/x/retry?job=annotate", ""},
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

// --- US1 ---

func TestCreateAndGetEndpoints(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	l := a.createLesson(t)

	if _, has := l["audioStatus"]; has || l["annotationStatus"] != "running" || l["revision"] != float64(1) || l["inRoadmap"] != false {
		t.Fatalf("lesson = %v", l)
	}
	if l["topicId"] != "topic-b1" || l["topicName"] != "Work" || l["level"] != "B1" {
		t.Fatalf("topic = %v %v %v", l["topicId"], l["topicName"], l["level"])
	}
	if _, old := l["topic"]; old {
		t.Fatal("lesson still has topic")
	}
	sentences := l["sentences"].([]any)
	if _, has := sentences[0].(map[string]any)["audioUrl"]; len(sentences) != 2 || has {
		t.Fatalf("sentences = %v", sentences)
	}

	rec := a.do(t, http.MethodGet, "/api/admin/lessons/"+l["id"].(string), "admin", "")
	if rec.Code != http.StatusOK || decodeBody(t, rec)["lesson"].(map[string]any)["title"] != "Park" {
		t.Fatalf("get: %d %s", rec.Code, rec.Body)
	}
	if rec := a.do(t, http.MethodGet, "/api/admin/lessons/missing", "admin", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("get missing: %d", rec.Code)
	}
}

func TestCreateValidation(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	rec := a.do(t, http.MethodPost, "/api/admin/lessons", "admin", `{"title":"","content":"","topicId":"","source":"","license":""}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d", rec.Code)
	}
	fields := decodeBody(t, rec)["fields"].(map[string]any)
	for _, f := range []string{"title", "content", "topicId", "source", "license"} {
		if fields[f] == nil {
			t.Errorf("missing field error %s: %v", f, fields)
		}
	}
}

func TestListEndpoint(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	a.createLesson(t)

	rec := a.do(t, http.MethodGet, "/api/admin/lessons?level=B1&topicId=topic-b1", "admin", "")
	lessons := decodeBody(t, rec)["lessons"].([]any)
	if rec.Code != http.StatusOK || len(lessons) != 1 || lessons[0].(map[string]any)["topicName"] != "Work" {
		t.Fatalf("list: %d %s", rec.Code, rec.Body)
	}
	rec = a.do(t, http.MethodGet, "/api/admin/lessons?topicId=topic-a1", "admin", "")
	if lessons := decodeBody(t, rec)["lessons"].([]any); len(lessons) != 0 {
		t.Fatalf("filter topic-a1 returned %v", lessons)
	}
	rec = a.do(t, http.MethodGet, "/api/admin/lessons?level=C2", "admin", "")
	if lessons := decodeBody(t, rec)["lessons"].([]any); len(lessons) != 0 {
		t.Fatalf("filter C2 returned %v", lessons)
	}
	if rec := a.do(t, http.MethodGet, "/api/admin/lessons?level=Z9", "admin", ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad level: %d", rec.Code)
	}
}

// --- US2 ---

func TestRetryEndpoint(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	l := a.createLesson(t)
	id := l["id"].(string)

	if rec := a.do(t, http.MethodPost, "/api/admin/lessons/"+id+"/retry?job=annotate", "admin", ""); rec.Code != http.StatusConflict {
		t.Fatalf("retry running: %d", rec.Code)
	}
	a.env.svc.JobFailed(context.Background(), job.Job{Type: job.TypeAnnotate, LessonID: id, Revision: 1}, ai.ErrQuota)
	rec := a.do(t, http.MethodPost, "/api/admin/lessons/"+id+"/retry?job=annotate", "admin", "")
	if rec.Code != http.StatusAccepted || decodeBody(t, rec)["lesson"].(map[string]any)["annotationStatus"] != "running" {
		t.Fatalf("retry failed: %d %s", rec.Code, rec.Body)
	}
	for _, bad := range []string{"x", "tts"} { // audio is no longer generated (Kokoro was removed)
		if rec := a.do(t, http.MethodPost, "/api/admin/lessons/"+id+"/retry?job="+bad, "admin", ""); rec.Code != http.StatusBadRequest {
			t.Fatalf("job %s: %d", bad, rec.Code)
		}
	}
	if rec := a.do(t, http.MethodPost, "/api/admin/lessons/missing/retry?job=annotate", "admin", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("missing: %d", rec.Code)
	}
}

// --- F14: topics replace the global roadmap ---

func TestOldBodyAndRoadmapRoutesAreGone(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	old := `{"title":"Park","content":"We went.","level":"B1","topic":"Daily","source":"s","license":"l"}`
	if rec := a.do(t, http.MethodPost, "/api/admin/lessons", "admin", old); rec.Code != http.StatusBadRequest {
		t.Fatalf("old body: %d", rec.Code)
	}
	unknown := `{"title":"Park","content":"We went.","topicId":"nope","level":"B1","source":"s","license":"l"}`
	rec := a.do(t, http.MethodPost, "/api/admin/lessons", "admin", unknown)
	if rec.Code != http.StatusBadRequest || decodeBody(t, rec)["fields"].(map[string]any)["topicId"] != "Chủ đề không tồn tại" {
		t.Fatalf("unknown topic: %d %s", rec.Code, rec.Body)
	}
	for _, m := range []string{http.MethodGet, http.MethodPut} {
		if rec := a.do(t, m, "/api/admin/roadmap", "admin", `{"lessonIds":[]}`); rec.Code != http.StatusNotFound && rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("%s roadmap: %d", m, rec.Code)
		}
	}
}

// --- US4 ---

func TestAnnotationsEndpoint(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	id := a.createLesson(t)["id"].(string)
	path := "/api/admin/lessons/" + id + "/annotations"

	if rec := a.do(t, http.MethodPut, path, "admin", `{"annotations":[]}`); rec.Code != http.StatusConflict {
		t.Fatalf("while running: %d", rec.Code)
	}
	a.env.svc.JobFailed(context.Background(), job.Job{Type: job.TypeAnnotate, LessonID: id, Revision: 1}, ai.ErrQuota)

	rec := a.do(t, http.MethodPut, path, "admin", `{"annotations":[{"text":"banana","lemma":"b","meaningVi":"c"}]}`)
	if rec.Code != http.StatusBadRequest || decodeBody(t, rec)["fields"].(map[string]any)["annotations.0.text"] == nil {
		t.Fatalf("invalid: %d %s", rec.Code, rec.Body)
	}
	rec = a.do(t, http.MethodPut, path, "admin", `{"annotations":[{"text":"gave up","lemma":"give up","meaningVi":"từ bỏ"}]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("save: %d %s", rec.Code, rec.Body)
	}
	anns := decodeBody(t, rec)["lesson"].(map[string]any)["annotations"].([]any)
	if first := anns[0].(map[string]any); first["editedByAdmin"] != true || first["sentenceIndex"] != float64(1) {
		t.Fatalf("annotations = %v", anns)
	}
	if rec := a.do(t, http.MethodPut, "/api/admin/lessons/missing/annotations", "admin", `{"annotations":[]}`); rec.Code != http.StatusNotFound {
		t.Fatalf("missing: %d", rec.Code)
	}
}

// --- US5 ---

func TestUpdateAndDeleteEndpoints(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	id := a.createLesson(t)["id"].(string)
	path := "/api/admin/lessons/" + id

	rec := a.do(t, http.MethodPut, path, "admin", strings.Replace(validBody, `"Park"`, `"Park 2"`, 1))
	if rec.Code != http.StatusOK || decodeBody(t, rec)["lesson"].(map[string]any)["revision"] != float64(1) {
		t.Fatalf("update info: %d %s", rec.Code, rec.Body)
	}
	rec = a.do(t, http.MethodPut, path, "admin", strings.Replace(validBody, "We went", "They went", 1))
	if rec.Code != http.StatusOK || decodeBody(t, rec)["lesson"].(map[string]any)["revision"] != float64(2) {
		t.Fatalf("update content: %d %s", rec.Code, rec.Body)
	}
	if rec := a.do(t, http.MethodPut, path, "admin", `{"title":""}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid update: %d", rec.Code)
	}
	if rec := a.do(t, http.MethodPut, "/api/admin/lessons/missing", "admin", validBody); rec.Code != http.StatusNotFound {
		t.Fatalf("update missing: %d", rec.Code)
	}

	a.env.topics.setRoadmap("topic-b1", id)
	rec = a.do(t, http.MethodDelete, path, "admin", "")
	if rec.Code != http.StatusConflict || decodeBody(t, rec)["error"] != "lesson_in_roadmap" {
		t.Fatalf("delete in roadmap: %d %s", rec.Code, rec.Body)
	}
	a.env.topics.setRoadmap("topic-b1")
	if rec := a.do(t, http.MethodDelete, path, "admin", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: %d", rec.Code)
	}
	if rec := a.do(t, http.MethodDelete, path, "admin", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("delete again: %d", rec.Code)
	}
}
