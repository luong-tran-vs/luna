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

	r := do(t, mux, http.MethodPost, "/api/goals", "an", `{"topicId":"family","level":"A1"}`)
	b := body(t, r)
	active := b["active"].(map[string]any)
	if r.code != http.StatusOK || len(b) != 1 || active["effectiveFrom"] != "2026-09-30" ||
		active["topicName"] != "Gia đình" || active["totalLessons"] != 3.0 || active["status"] != "active" {
		t.Fatalf("set goal: %d %s", r.code, r.text)
	}

	for payload, code := range map[string]int{
		`{"topicId":""}`:                    http.StatusBadRequest,
		`{"topicId":"family"}`:              http.StatusBadRequest,
		`{"topicId":"family","level":"Z1"}`: http.StatusBadRequest,
		`{"topicId":"x","extra":1}`:         http.StatusBadRequest,
		`{"topicId":"nope","level":"A1"}`:   http.StatusNotFound,
	} {
		if r := do(t, mux, http.MethodPost, "/api/goals", "an", payload); r.code != code {
			t.Fatalf("%s: %d %s", payload, r.code, r.text)
		}
	}
	for _, p := range []string{"/api/goals", "/api/lessons/f1/study", "/api/lessons/mine"} {
		if r := do(t, mux, http.MethodGet, p, "", ""); r.code != http.StatusUnauthorized {
			t.Fatalf("anonymous %s: %d", p, r.code)
		}
	}
	// The daily page is gone (2026-10-02).
	if r := do(t, mux, http.MethodGet, "/api/today", "an", ""); r.code != http.StatusNotFound {
		t.Fatalf("today: %d", r.code)
	}
}

func TestStudyEndpoints(t *testing.T) {
	t.Parallel()
	mux, e := newStudyAPI(t)
	do(t, mux, http.MethodPost, "/api/goals", "an", `{"topicId":"family","level":"A1"}`)

	r := do(t, mux, http.MethodGet, "/api/lessons/f1/study", "an", "")
	b := body(t, r)
	steps := b["steps"].(map[string]any)
	if r.code != http.StatusOK || b["status"] != "studying" || b["currentStep"] != "read" || steps["read"] != "current" ||
		steps["listen"] != "locked" || len(steps) != 3 || b["next"] != nil || b["goal"].(map[string]any)["topicId"] != "family" ||
		b["streak"] != 0.0 {
		t.Fatalf("study: %d %s", r.code, r.text)
	}
	// An upcoming lesson is locked, as its content.
	if r := do(t, mux, http.MethodGet, "/api/lessons/f2/study", "an", ""); r.code != http.StatusForbidden {
		t.Fatalf("upcoming: %d %s", r.code, r.text)
	}

	cases := []struct {
		path, token string
		code        int
		want        string
	}{
		{"/api/lessons/f1/steps/listen/complete", "an", http.StatusConflict, `"listen_incomplete"`},
		{"/api/lessons/f1/steps/speak/complete", "an", http.StatusBadRequest, `"step"`},
		{"/api/lessons/f2/steps/read/complete", "an", http.StatusConflict, `"not_current_lesson"`},
		{"/api/lessons/f1/steps/read/complete", "binh", http.StatusConflict, `"not_current_lesson"`},
		{"/api/lessons/f1/steps/read/complete", "", http.StatusUnauthorized, ""},
	}
	for _, c := range cases {
		if r := do(t, mux, http.MethodPost, c.path, c.token, ""); r.code != c.code || !strings.Contains(r.text, c.want) {
			t.Fatalf("%s as %s: %d %s", c.path, c.token, r.code, r.text)
		}
	}

	if r := do(t, mux, http.MethodPut, "/api/lessons/f1/position", "an", `{"step":"read","sentenceIndex":2}`); r.code != http.StatusNoContent {
		t.Fatalf("position: %d %s", r.code, r.text)
	}
	for payload, code := range map[string]int{
		`{"step":"read","sentenceIndex":-1}`:      http.StatusBadRequest,
		`{"step":"read","sentenceIndex":1,"x":1}`: http.StatusBadRequest,
	} {
		if r := do(t, mux, http.MethodPut, "/api/lessons/f1/position", "an", payload); r.code != code {
			t.Fatalf("%s: %d %s", payload, r.code, r.text)
		}
	}
	if r := do(t, mux, http.MethodGet, "/api/lessons/f1/study", "an", ""); !strings.Contains(r.text, `"sentenceIndex":2`) {
		t.Fatalf("saved position: %s", r.text)
	}

	do(t, mux, http.MethodPost, "/api/lessons/f1/steps/read/complete", "an", "")
	// A done step takes no position.
	if r := do(t, mux, http.MethodPut, "/api/lessons/f1/position", "an", `{"step":"read","sentenceIndex":1}`); r.code != http.StatusConflict ||
		!strings.Contains(r.text, `"not_current_step"`) {
		t.Fatalf("position of a done step: %d %s", r.code, r.text)
	}
	if r := do(t, mux, http.MethodPost, "/api/lessons/f1/steps/listen/complete", "an", ""); r.code != http.StatusConflict || !strings.Contains(r.text, `"listen_incomplete"`) {
		t.Fatalf("listen early: %d %s", r.code, r.text)
	}
	e.finishDictation(t)
	r = do(t, mux, http.MethodPost, "/api/lessons/f1/steps/listen/complete", "an", "")
	if !strings.Contains(r.text, `"currentStep":"write"`) || !strings.Contains(r.text, `"status":"studying"`) {
		t.Fatalf("listen: %s", r.text)
	}

	// F8: the lesson is done once the writing is submitted, and the next one opens at once.
	if r := do(t, mux, http.MethodPost, "/api/lessons/f1/steps/write/complete", "an", ""); r.code != http.StatusConflict || !strings.Contains(r.text, `"write_incomplete"`) {
		t.Fatalf("write early: %d %s", r.code, r.text)
	}
	e.submitWriting(t)
	r = do(t, mux, http.MethodPost, "/api/lessons/f1/steps/write/complete", "an", "")
	if !strings.Contains(r.text, `"status":"completed"`) || !strings.Contains(r.text, `"streak":1`) ||
		!strings.Contains(r.text, `"currentStep":"done"`) || !strings.Contains(r.text, `"next":{"id":"f2","title":"Family 2"}`) {
		t.Fatalf("write: %s", r.text)
	}
	if r := do(t, mux, http.MethodGet, "/api/lessons/f2/study", "an", ""); r.code != http.StatusOK || !strings.Contains(r.text, `"status":"studying"`) {
		t.Fatalf("f2 after f1: %d %s", r.code, r.text)
	}
}

func TestMyLessonsAndGuardEndpoints(t *testing.T) {
	t.Parallel()
	mux, e := newStudyAPI(t)
	do(t, mux, http.MethodPost, "/api/goals", "an", `{"topicId":"family","level":"A1"}`)
	e.studyLesson(t)

	r := do(t, mux, http.MethodGet, "/api/lessons/mine", "an", "")
	b := body(t, r)
	if b["current"].(map[string]any)["id"] != "f2" || len(b["completed"].([]any)) != 1 || len(b["upcoming"].([]any)) != 1 {
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
	if r := do(t, mux, http.MethodGet, "/api/lessons/f3/study", "binh", ""); r.code != http.StatusOK || !strings.Contains(r.text, `"status":"other"`) {
		t.Fatalf("admin study: %d %s", r.code, r.text)
	}
}

func TestSkipWriteEndpoint(t *testing.T) {
	t.Parallel()
	mux, e := newStudyAPI(t)
	do(t, mux, http.MethodPost, "/api/goals", "an", `{"topicId":"family","level":"A1"}`)

	if r := do(t, mux, http.MethodPost, "/api/lessons/f1/steps/write/skip", "", ""); r.code != http.StatusUnauthorized {
		t.Fatalf("anonymous: %d", r.code)
	}
	do(t, mux, http.MethodPost, "/api/lessons/f1/steps/read/complete", "an", "")
	e.finishDictation(t)

	// Write may be skipped before Listen; the lesson completes with Listen.
	if r := do(t, mux, http.MethodPost, "/api/lessons/f1/steps/write/skip", "an", ""); r.code != http.StatusOK ||
		!strings.Contains(r.text, `"status":"studying"`) {
		t.Fatalf("skip before listen: %d %s", r.code, r.text)
	}
	r := do(t, mux, http.MethodPost, "/api/lessons/f1/steps/listen/complete", "an", "")
	if r.code != http.StatusOK || !strings.Contains(r.text, `"status":"completed"`) || !strings.Contains(r.text, `"streak":1`) {
		t.Fatalf("skip: %d %s", r.code, r.text)
	}
}

// TestRedoCompletedLesson checks that a learner may send and delete dictation results of a
// lesson already completed (redoing it), but not of a locked lesson, and that the steps stay done.
func TestRedoCompletedLesson(t *testing.T) {
	t.Parallel()
	mux, e := newStudyAPI(t)
	auth := httpx.RequireAuth(resolver)
	NewHandler(e.dictation, slog.New(slog.DiscardHandler)).
		Register(mux, auth, NewStudyHandler(e.svc, slog.New(slog.DiscardHandler)).Guard)
	do(t, mux, http.MethodPost, "/api/goals", "an", `{"topicId":"family","level":"A1"}`)
	e.studyLesson(t) // f1 completed, f2 current, f3 locked

	record := `{"sentenceIndex":0,"typed":"again","correctWords":1,"totalWords":2}`
	if r := do(t, mux, http.MethodPost, "/api/lessons/f1/dictation", "an", record); r.code != http.StatusOK || !strings.Contains(r.text, `"typed":"again"`) {
		t.Fatalf("record completed lesson: %d %s", r.code, r.text)
	}
	if r := do(t, mux, http.MethodDelete, "/api/lessons/f1/dictation", "an", ""); r.code != http.StatusNoContent {
		t.Fatalf("delete completed lesson: %d %s", r.code, r.text)
	}
	if r := do(t, mux, http.MethodGet, "/api/lessons/f1/dictation/summary", "an", ""); !strings.Contains(r.text, `"checkedCount":0`) {
		t.Fatalf("summary after delete: %s", r.text)
	}
	// Deleting the results does not touch the lesson progress.
	if r := do(t, mux, http.MethodGet, "/api/lessons/f1/study", "an", ""); r.code != http.StatusOK || !strings.Contains(r.text, `"status":"completed"`) {
		t.Fatalf("f1 study after delete: %d %s", r.code, r.text)
	}

	for _, req := range []struct{ method, body string }{{http.MethodPost, record}, {http.MethodDelete, ""}} {
		r := do(t, mux, req.method, "/api/lessons/f3/dictation", "an", req.body)
		if r.code != http.StatusForbidden || !strings.Contains(r.text, `"lesson_locked"`) {
			t.Fatalf("%s locked lesson: %d %s", req.method, r.code, r.text)
		}
	}
}
