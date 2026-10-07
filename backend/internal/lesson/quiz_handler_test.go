package lesson

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

func newQuizAPI(t *testing.T) (*http.ServeMux, Lesson) {
	t.Helper()
	r, _, _, l := newQuizEnv(t)
	mux := http.NewServeMux()
	NewReadingHandler(r, slog.New(slog.DiscardHandler)).Register(mux, httpx.RequireAuth(resolver), allowAll)
	return mux, l
}

func postAnswer(t *testing.T, mux http.Handler, lessonID, token, body string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/lessons/"+lessonID+"/answers", strings.NewReader(body))
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

func TestReadingQuizJSON(t *testing.T) {
	t.Parallel()
	mux, l := newQuizAPI(t)
	if code, _ := postAnswer(t, mux, l.ID, "learner", `{"version":2,"questionIndex":0,"choice":1}`); code != http.StatusOK {
		t.Fatalf("answer: %d", code)
	}

	rec := get(t, mux, "/api/lessons/"+l.ID, "learner")
	body := rec.Body.String()
	var out struct {
		Lesson map[string]any `json:"lesson"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	quiz := out.Lesson["quiz"].(map[string]any)
	questions := quiz["questions"].([]any)
	if quiz["version"] != float64(2) || len(questions) != 3 {
		t.Fatalf("quiz = %v", quiz)
	}
	for _, q := range questions {
		if _, leak := q.(map[string]any)["answerIndex"]; leak {
			t.Fatalf("question leaks its answer: %v", q)
		}
	}
	// Only the answered question carries its explanation.
	if strings.Count(body, "explanationVi") != 1 || strings.Contains(body, "Câu 2.") || strings.Contains(body, "secret prompt") {
		t.Fatalf("body leaks answers or the writing prompt: %s", body)
	}
	answers := quiz["answers"].([]any)
	if len(answers) != 1 || answers[0].(map[string]any)["correct"] != false || answers[0].(map[string]any)["answerIndex"] != float64(0) {
		t.Fatalf("answers = %v", answers)
	}
	if note := out.Lesson["grammarNote"].(map[string]any); note["title"] != "Quá khứ đơn" {
		t.Fatalf("grammarNote = %v", note)
	}

	// A lesson without questions has quiz: null.
	plainMux, plain := newReadingAPI(t)
	rec = get(t, plainMux, "/api/lessons/"+plain, "learner")
	if !strings.Contains(rec.Body.String(), `"quiz":null`) || !strings.Contains(rec.Body.String(), `"grammarNote":null`) {
		t.Fatalf("plain lesson: %s", rec.Body)
	}
}

func TestAnswerEndpoint(t *testing.T) {
	t.Parallel()
	mux, l := newQuizAPI(t)

	code, body := postAnswer(t, mux, l.ID, "learner", `{"version":2,"questionIndex":1,"choice":1}`)
	answer, _ := body["answer"].(map[string]any)
	if code != http.StatusOK || answer["correct"] != true || answer["answerIndex"] != float64(1) || answer["explanationVi"] != "Câu 2." ||
		body["answered"] != float64(1) || body["total"] != float64(3) || body["correct"] != float64(1) {
		t.Fatalf("answer: %d %v", code, body)
	}

	// Answering again replaces the stored answer and returns the same shape.
	code, body = postAnswer(t, mux, l.ID, "learner", `{"version":2,"questionIndex":1,"choice":0}`)
	answer, _ = body["answer"].(map[string]any)
	if code != http.StatusOK || answer["choice"] != float64(0) || answer["correct"] != false || answer["answerIndex"] != float64(1) ||
		body["answered"] != float64(1) || body["total"] != float64(3) || body["correct"] != float64(0) {
		t.Fatalf("answer again: %d %v", code, body)
	}

	tests := []struct {
		name, token, body string
		status            int
		code              string
	}{
		{"old version", "learner", `{"version":1,"questionIndex":0,"choice":0}`, http.StatusConflict, "quiz_changed"},
		{"bad index", "learner", `{"version":2,"questionIndex":9,"choice":0}`, http.StatusBadRequest, "validation_failed"},
		{"broken json", "learner", `{"version":`, http.StatusBadRequest, "invalid_body"},
		{"no session", "", `{"version":2,"questionIndex":0,"choice":0}`, http.StatusUnauthorized, "unauthenticated"},
	}
	for _, tt := range tests {
		code, body := postAnswer(t, mux, l.ID, tt.token, tt.body)
		if code != tt.status || body["error"] != tt.code {
			t.Errorf("%s: %d %v", tt.name, code, body)
		}
	}

	if code, body := postAnswer(t, mux, "missing", "learner", `{"version":2,"questionIndex":0,"choice":0}`); code != http.StatusNotFound {
		t.Errorf("missing lesson: %d %v", code, body)
	}
	plainMux, plain := newReadingAPI(t)
	if code, body := postAnswer(t, plainMux, plain, "learner", `{"version":0,"questionIndex":0,"choice":0}`); code != http.StatusConflict ||
		body["error"] != "no_quiz" {
		t.Errorf("no quiz: %d %v", code, body)
	}
}

func deleteAnswers(t *testing.T, mux http.Handler, lessonID, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodDelete, "/api/lessons/"+lessonID+"/answers", nil)
	if token != "" {
		req.AddCookie(&http.Cookie{Name: httpx.SessionCookieName, Value: token})
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestResetAnswersEndpoint(t *testing.T) {
	t.Parallel()
	mux, l := newQuizAPI(t)
	for _, token := range []string{"learner", "admin"} {
		if code, body := postAnswer(t, mux, l.ID, token, `{"version":2,"questionIndex":0,"choice":0}`); code != http.StatusOK {
			t.Fatalf("answer %s: %d %v", token, code, body)
		}
	}

	if rec := deleteAnswers(t, mux, l.ID, "learner"); rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
		t.Fatalf("delete: %d %s", rec.Code, rec.Body)
	}
	answersOf := func(token string) []any {
		var out struct {
			Lesson struct {
				Quiz struct {
					Answers []any `json:"answers"`
				} `json:"quiz"`
			} `json:"lesson"`
		}
		_ = json.Unmarshal(get(t, mux, "/api/lessons/"+l.ID, token).Body.Bytes(), &out)
		return out.Lesson.Quiz.Answers
	}
	if a := answersOf("learner"); len(a) != 0 {
		t.Fatalf("learner answers after delete = %v", a)
	}
	// Another user's answers are untouched.
	if a := answersOf("admin"); len(a) != 1 {
		t.Fatalf("admin answers = %v", a)
	}
	// The quiz can be taken again.
	if code, body := postAnswer(t, mux, l.ID, "learner", `{"version":2,"questionIndex":0,"choice":1}`); code != http.StatusOK ||
		body["answered"] != float64(1) || body["correct"] != float64(0) {
		t.Fatalf("answer after delete: %d %v", code, body)
	}

	if rec := deleteAnswers(t, mux, "missing", "learner"); rec.Code != http.StatusNotFound {
		t.Errorf("missing lesson: %d", rec.Code)
	}
	if rec := deleteAnswers(t, mux, l.ID, ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("no session: %d", rec.Code)
	}
	plainMux, plain := newReadingAPI(t)
	if rec := deleteAnswers(t, plainMux, plain, "learner"); rec.Code != http.StatusNoContent {
		t.Errorf("no quiz: %d %s", rec.Code, rec.Body)
	}

	r, _, _, l2 := newQuizEnv(t)
	locked := http.NewServeMux()
	deny := func(http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			httpx.WriteError(w, http.StatusForbidden, "lesson_locked", "Bài này sẽ mở khi tới lượt")
		})
	}
	NewReadingHandler(r, slog.New(slog.DiscardHandler)).Register(locked, httpx.RequireAuth(resolver), deny)
	if rec := deleteAnswers(t, locked, l2.ID, "learner"); rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "lesson_locked") {
		t.Errorf("locked: %d %s", rec.Code, rec.Body)
	}
}
