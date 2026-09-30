package vocab

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestListEndpoint(t *testing.T) {
	t.Parallel()
	mux, e := newEnvAPI(t)
	e.saved("u1", "went", "lesson1", time.Date(2026, 9, 29, 1, 0, 0, 0, time.UTC))
	e.saved("u1", "hi", "", time.Date(2026, 9, 28, 1, 0, 0, 0, time.UTC))

	rec := call(t, mux, http.MethodGet, "/api/vocab/cards?page=1", "an", "")
	var b struct {
		Cards     []map[string]any `json:"cards"`
		HasMore   bool             `json:"hasMore"`
		Today     string           `json:"today"`
		Yesterday string           `json:"yesterday"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &b); rec.Code != http.StatusOK || err != nil {
		t.Fatalf("list: %d %s", rec.Code, rec.Body)
	}
	if len(b.Cards) != 2 || b.Cards[0]["day"] != "2026-09-29" || b.Cards[1]["lessonId"] != nil || b.Today != "2026-09-29" ||
		b.Yesterday != "2026-09-28" || b.HasMore {
		t.Fatalf("body = %s", rec.Body)
	}

	rec = call(t, mux, http.MethodGet, "/api/vocab/cards?lessonId=manual&q=H", "an", "")
	if !strings.Contains(rec.Body.String(), `"text":"hi"`) || strings.Contains(rec.Body.String(), `"went"`) {
		t.Fatalf("filtered: %s", rec.Body)
	}
	for _, q := range []string{"?page=0", "?page=x"} {
		if rec := call(t, mux, http.MethodGet, "/api/vocab/cards"+q, "an", ""); rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: %d", q, rec.Code)
		}
	}
	if rec := call(t, mux, http.MethodGet, "/api/vocab/cards", "binh", ""); !strings.Contains(rec.Body.String(), `"cards":[]`) {
		t.Fatalf("other user: %s", rec.Body)
	}
	if rec := call(t, mux, http.MethodGet, "/api/vocab/cards", "", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous: %d", rec.Code)
	}
}

func TestLessonsEndpoint(t *testing.T) {
	t.Parallel()
	mux, e := newEnvAPI(t)
	e.saved("u1", "went", "lesson1", e.clock.Now())
	e.saved("u1", "hi", "", e.clock.Now())
	rec := call(t, mux, http.MethodGet, "/api/vocab/lessons", "an", "")
	want := "{\"lessons\":[{\"id\":\"lesson1\",\"title\":\"A day at the park\",\"count\":1}],\"manualCount\":1}\n"
	if rec.Code != http.StatusOK || rec.Body.String() != want {
		t.Fatalf("lessons: %d %s", rec.Code, rec.Body)
	}
	if rec := call(t, mux, http.MethodGet, "/api/vocab/lessons", "", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous: %d", rec.Code)
	}
}

func TestManualCardEndpoint(t *testing.T) {
	t.Parallel()
	mux := newAPI(t)
	body := `{"text":"serendipity","lemma":"serendipity","ipa":"","meaningVi":"sự tình cờ may mắn","contextSentence":"","source":"manual"}`
	rec := call(t, mux, http.MethodPost, "/api/vocab/cards", "an", body)
	if rec.Code != http.StatusCreated || !strings.Contains(rec.Body.String(), `"lessonId":null`) ||
		!strings.Contains(rec.Body.String(), `"due":"2026-09-29T17:00:00Z"`) {
		t.Fatalf("manual: %d %s", rec.Code, rec.Body)
	}
	if rec := call(t, mux, http.MethodPost, "/api/vocab/cards", "an", body); rec.Code != http.StatusConflict {
		t.Fatalf("duplicate: %d", rec.Code)
	}
}

func TestUpdateEndpoint(t *testing.T) {
	t.Parallel()
	mux, e := newEnvAPI(t)
	c := e.dueCard("went", Schedule{Due: time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC), Reps: 2, State: StateReview})
	path := "/api/vocab/cards/" + c.ID

	rec := call(t, mux, http.MethodPatch, path, "an", `{"meaningVi":"đã đi","ipa":"/went/"}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"meaningVi":"đã đi"`) ||
		!strings.Contains(rec.Body.String(), `"reps":2`) || !strings.Contains(rec.Body.String(), `"due":"2026-10-05T00:00:00Z"`) {
		t.Fatalf("patch: %d %s", rec.Code, rec.Body)
	}
	for body, want := range map[string]string{
		`{"text":"go"}`:    `"invalid_body"`,
		`{"reps":0}`:       `"invalid_body"`,
		`{"meaningVi":""}`: `"meaningVi"`,
	} {
		if rec := call(t, mux, http.MethodPatch, path, "an", body); rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), want) {
			t.Fatalf("%s: %d %s", body, rec.Code, rec.Body)
		}
	}
	if rec := call(t, mux, http.MethodPatch, path, "binh", `{"meaningVi":"x"}`); rec.Code != http.StatusNotFound {
		t.Fatalf("other user: %d", rec.Code)
	}
	if rec := call(t, mux, http.MethodPatch, path, "", `{"meaningVi":"x"}`); rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous: %d", rec.Code)
	}
}

func TestDeleteEndpoint(t *testing.T) {
	t.Parallel()
	mux, e := newEnvAPI(t)
	c := e.dueCard("went", Schedule{State: StateNew})
	path := "/api/vocab/cards/" + c.ID
	if rec := call(t, mux, http.MethodDelete, path, "binh", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("other user: %d", rec.Code)
	}
	if rec := call(t, mux, http.MethodDelete, path, "an", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: %d %s", rec.Code, rec.Body)
	}
	if rec := call(t, mux, http.MethodDelete, path, "an", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("again: %d", rec.Code)
	}
	if rec := call(t, mux, http.MethodDelete, path, "", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous: %d", rec.Code)
	}
}
