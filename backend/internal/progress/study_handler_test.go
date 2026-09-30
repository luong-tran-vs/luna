package progress

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

func newStudyAPI(t *testing.T) (*http.ServeMux, *studyEnv) {
	t.Helper()
	e := newStudyEnv()
	mux := http.NewServeMux()
	h := NewStudyHandler(e.svc, slog.New(slog.DiscardHandler))
	requireAuth := httpx.RequireAuth(resolver)
	h.Register(mux, requireAuth)
	// A guarded lesson route, as lesson.ReadingHandler registers it in main.
	mux.Handle("GET /api/lessons/{id}", requireAuth(h.Guard(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))))
	return mux, e
}

// recorder is a response with its status code and body text.
type recorder struct {
	code int
	text string
}

func body(t *testing.T, r recorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal([]byte(r.text), &out); err != nil {
		t.Fatalf("body %q: %v", r.text, err)
	}
	return out
}

func do(t *testing.T, mux *http.ServeMux, method, path, token, payload string) recorder {
	t.Helper()
	rec := call(t, mux, method, path, token, payload)
	return recorder{code: rec.Code, text: rec.Body.String()}
}

func TestGoalEndpoints(t *testing.T) {
	t.Parallel()
	mux, _ := newStudyAPI(t)

	if r := do(t, mux, http.MethodGet, "/api/goals", "an", ""); r.code != http.StatusOK || r.text != "{\"active\":null,\"others\":[]}\n" {
		t.Fatalf("no goals: %d %s", r.code, r.text)
	}
	if r := do(t, mux, http.MethodGet, "/api/today", "an", ""); !strings.Contains(r.text, `"kind":"noGoal"`) {
		t.Fatalf("today: %s", r.text)
	}

	r := do(t, mux, http.MethodPost, "/api/goals", "an", `{"topicId":"family"}`)
	b := body(t, r)
	active := b["active"].(map[string]any)
	if r.code != http.StatusOK || b["startsTomorrow"] != false || b["effectiveFrom"] != "2026-09-30" ||
		active["topicName"] != "Gia đình" || active["totalLessons"] != 3.0 || active["status"] != "active" {
		t.Fatalf("set goal: %d %s", r.code, r.text)
	}

	for payload, code := range map[string]int{
		`{"topicId":""}`:            http.StatusBadRequest,
		`{"topicId":"x","extra":1}`: http.StatusBadRequest,
		`{"topicId":"nope"}`:        http.StatusNotFound,
	} {
		if r := do(t, mux, http.MethodPost, "/api/goals", "an", payload); r.code != code {
			t.Fatalf("%s: %d %s", payload, r.code, r.text)
		}
	}
	for _, p := range []string{"/api/goals", "/api/today", "/api/lessons/mine"} {
		if r := do(t, mux, http.MethodGet, p, "", ""); r.code != http.StatusUnauthorized {
			t.Fatalf("anonymous %s: %d", p, r.code)
		}
	}
}

func TestTodayEndpoints(t *testing.T) {
	t.Parallel()
	mux, e := newStudyAPI(t)
	e.reviews.setDue("u1", 45)
	do(t, mux, http.MethodPost, "/api/goals", "an", `{"topicId":"family"}`)

	r := do(t, mux, http.MethodGet, "/api/today", "an", "")
	b := body(t, r)
	steps := b["steps"].(map[string]any)
	if b["kind"] != "studying" || b["currentStep"] != "review" || b["reviewCount"] != 30.0 || steps["read"] != "locked" ||
		b["lesson"].(map[string]any)["title"] != "Family 1" || b["streak"] != 0.0 {
		t.Fatalf("today: %s", r.text)
	}

	cases := []struct {
		path, token string
		code        int
		want        string
	}{
		{"/api/today/steps/read/complete", "an", http.StatusConflict, `"step_locked"`},
		{"/api/today/steps/write/complete", "an", http.StatusBadRequest, `"step"`},
		{"/api/today/steps/review/complete", "binh", http.StatusConflict, `"no_lesson"`},
		{"/api/today/steps/review/complete", "", http.StatusUnauthorized, ""},
	}
	for _, c := range cases {
		if r := do(t, mux, http.MethodPost, c.path, c.token, ""); r.code != c.code || !strings.Contains(r.text, c.want) {
			t.Fatalf("%s as %s: %d %s", c.path, c.token, r.code, r.text)
		}
	}

	if r := do(t, mux, http.MethodPost, "/api/today/steps/review/complete", "an", ""); !strings.Contains(r.text, `"currentStep":"read"`) {
		t.Fatalf("review: %s", r.text)
	}
	if r := do(t, mux, http.MethodPut, "/api/today/position", "an", `{"step":"read","sentenceIndex":2}`); r.code != http.StatusNoContent {
		t.Fatalf("position: %d %s", r.code, r.text)
	}
	for payload, code := range map[string]int{
		`{"step":"read","sentenceIndex":-1}`:      http.StatusBadRequest,
		`{"step":"listen","sentenceIndex":1}`:     http.StatusConflict,
		`{"step":"read","sentenceIndex":1,"x":1}`: http.StatusBadRequest,
	} {
		if r := do(t, mux, http.MethodPut, "/api/today/position", "an", payload); r.code != code {
			t.Fatalf("%s: %d %s", payload, r.code, r.text)
		}
	}
	if r := do(t, mux, http.MethodGet, "/api/today", "an", ""); !strings.Contains(r.text, `"sentenceIndex":2`) {
		t.Fatalf("saved position: %s", r.text)
	}

	do(t, mux, http.MethodPost, "/api/today/steps/read/complete", "an", "")
	if r := do(t, mux, http.MethodPost, "/api/today/steps/listen/complete", "an", ""); r.code != http.StatusConflict || !strings.Contains(r.text, `"listen_incomplete"`) {
		t.Fatalf("listen early: %d %s", r.code, r.text)
	}
	e.finishDictation(t)
	r = do(t, mux, http.MethodPost, "/api/today/steps/listen/complete", "an", "")
	if !strings.Contains(r.text, `"kind":"doneToday"`) || !strings.Contains(r.text, `"streak":1`) || !strings.Contains(r.text, `"currentStep":"done"`) {
		t.Fatalf("listen: %s", r.text)
	}
}

func TestMyLessonsAndGuardEndpoints(t *testing.T) {
	t.Parallel()
	mux, e := newStudyAPI(t)
	do(t, mux, http.MethodPost, "/api/goals", "an", `{"topicId":"family"}`)
	e.studyLesson(t)
	e.nextDay(1)

	r := do(t, mux, http.MethodGet, "/api/lessons/mine", "an", "")
	b := body(t, r)
	if b["today"].(map[string]any)["id"] != "f2" || len(b["completed"].([]any)) != 1 || len(b["upcoming"].([]any)) != 1 {
		t.Fatalf("mine: %s", r.text)
	}
	first := b["completed"].([]any)[0].(map[string]any)
	if first["title"] != "Family 1" || first["topicName"] != "Gia đình" || first["completedAt"] == nil {
		t.Fatalf("completed: %v", first)
	}
	if up := b["upcoming"].([]any)[0].(map[string]any); len(up) != 2 || up["id"] != "f3" {
		t.Fatalf("upcoming: %v", up)
	}

	for id, code := range map[string]int{"f1": http.StatusOK, "f2": http.StatusOK, "f3": http.StatusForbidden} {
		r := do(t, mux, http.MethodGet, "/api/lessons/"+id, "an", "")
		if r.code != code || (code == http.StatusForbidden && !strings.Contains(r.text, `"lesson_locked"`)) {
			t.Fatalf("learner %s: %d %s", id, r.code, r.text)
		}
	}
	if r := do(t, mux, http.MethodGet, "/api/lessons/f3", "binh", ""); r.code != http.StatusOK {
		t.Fatalf("admin: %d", r.code)
	}
}
