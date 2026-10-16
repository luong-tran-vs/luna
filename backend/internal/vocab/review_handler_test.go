package vocab

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

func newEnvAPI(t *testing.T) (*http.ServeMux, *testEnv) {
	t.Helper()
	e := newEnv()
	mux := http.NewServeMux()
	NewHandler(e.svc, slog.New(slog.DiscardHandler)).Register(mux, httpx.RequireAuth(resolver))
	return mux, e
}

type dueBody struct {
	Cards []struct {
		ID        string         `json:"id"`
		Reps      int            `json:"reps"`
		State     string         `json:"state"`
		Intervals map[string]int `json:"intervals"`
	} `json:"cards"`
	Total   int     `json:"total"`
	NextDue *string `json:"nextDue"`
}

func TestDueEndpoint(t *testing.T) {
	t.Parallel()
	mux, e := newEnvAPI(t)
	c := e.dueCard("go", Schedule{State: StateNew})

	rec := call(t, mux, http.MethodGet, "/api/vocab/review/due", "an", "")
	var b dueBody
	if err := json.Unmarshal(rec.Body.Bytes(), &b); rec.Code != http.StatusOK || err != nil {
		t.Fatalf("due: %d %s", rec.Code, rec.Body)
	}
	if b.Total != 1 || len(b.Cards) != 1 || b.Cards[0].ID != c.ID || b.Cards[0].State != "new" || b.NextDue != nil {
		t.Fatalf("body = %s", rec.Body)
	}
	iv := b.Cards[0].Intervals
	if iv["again"] != 60 || iv["hard"] != 300 || iv["good"] != 600 || iv["easy"] < 86400 {
		t.Fatalf("intervals = %v", iv)
	}

	for _, q := range []string{"?limit=0", "?limit=201", "?limit=x"} {
		if rec := call(t, mux, http.MethodGet, "/api/vocab/review/due"+q, "an", ""); rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: %d", q, rec.Code)
		}
	}
	if rec := call(t, mux, http.MethodGet, "/api/vocab/review/due", "binh", ""); !strings.Contains(rec.Body.String(), `"total":0`) {
		t.Fatalf("other user: %s", rec.Body)
	}
	if rec := call(t, mux, http.MethodGet, "/api/vocab/review/due", "", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous: %d", rec.Code)
	}
}

func TestReviewEndpoint(t *testing.T) {
	t.Parallel()
	mux, e := newEnvAPI(t)
	c := e.dueCard("go", Schedule{State: StateNew})
	path := "/api/vocab/cards/" + c.ID + "/review"
	body := `{"rating":3,"mode":"flip","reps":0}`

	rec := call(t, mux, http.MethodPost, path, "an", body)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"reps":1`) ||
		!strings.Contains(rec.Body.String(), `"state":"learning"`) || !strings.Contains(rec.Body.String(), `"intervals":{`) {
		t.Fatalf("review: %d %s", rec.Code, rec.Body)
	}

	rec = call(t, mux, http.MethodPost, path, "an", body)
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), `"review_conflict"`) ||
		!strings.Contains(rec.Body.String(), `"card":{`) {
		t.Fatalf("re-send: %d %s", rec.Code, rec.Body)
	}

	errs := map[string]string{
		`{"rating":7,"mode":"flip","reps":1}`:                 `"rating"`,
		`{"rating":3,"mode":"type","reps":1}`:                 `"mode"`,
		`{"rating":3,"mode":"flip","reps":1,"context":"x"}`:   `"context"`,
		`{"rating":3,"mode":"flip","reps":1,"userId":"u2"}`:   `"invalid_body"`,
		`{"rating":3,"mode":"flip","reps":1,"due":"2030-01"}`: `"invalid_body"`,
	}
	for b, want := range errs {
		if rec := call(t, mux, http.MethodPost, path, "an", b); rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), want) {
			t.Fatalf("%s: %d %s", b, rec.Code, rec.Body)
		}
	}
	if rec := call(t, mux, http.MethodPost, path, "binh", body); rec.Code != http.StatusNotFound {
		t.Fatalf("other user: %d", rec.Code)
	}
	if rec := call(t, mux, http.MethodPost, path, "", body); rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous: %d", rec.Code)
	}
}

func TestForecastEndpoint(t *testing.T) {
	t.Parallel()
	mux, e := newEnvAPI(t)
	e.dueCard("go", Schedule{State: StateNew})

	rec := call(t, mux, http.MethodGet, "/api/vocab/review/forecast", "an", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"new":1,"learning":0,"known":0,"days":[{"day":"2026-09-29","count":1}`) {
		t.Fatalf("forecast: %d %s", rec.Code, rec.Body)
	}
	if rec := call(t, mux, http.MethodGet, "/api/vocab/review/forecast", "", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous: %d", rec.Code)
	}
}

func TestLessonStatsEndpoint(t *testing.T) {
	t.Parallel()
	mux, e := newEnvAPI(t)
	e.dueCard("go", Schedule{State: StateLearning, Reps: 1})

	rec := call(t, mux, http.MethodGet, "/api/vocab/review/lessons", "an", "")
	want := `{"lessons":[{"lessonId":"lesson1","cards":1,"due":1,"remembered":0,"lastReview":null,"nextDue":null,"result":null}]}`
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != want {
		t.Fatalf("lesson stats: %d %s", rec.Code, rec.Body)
	}
	if rec := call(t, mux, http.MethodGet, "/api/vocab/review/lessons", "", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous: %d", rec.Code)
	}
}

func TestSaveLessonResultEndpoint(t *testing.T) {
	t.Parallel()
	mux, _ := newEnvAPI(t)
	path := "/api/vocab/review/lessons/lesson1"

	rec := call(t, mux, http.MethodPut, path, "an", `{"correct":3,"total":5,"wrong":["v:go","t:0"]}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"correct":3,"total":5,"wrong":["v:go","t:0"]`) {
		t.Fatalf("save: %d %s", rec.Code, rec.Body)
	}
	rec = call(t, mux, http.MethodGet, "/api/vocab/review/lessons", "an", "")
	if !strings.Contains(rec.Body.String(), `"lessonId":"lesson1","cards":0`) || !strings.Contains(rec.Body.String(), `"result":{"correct":3,"total":5`) {
		t.Fatalf("stats: %s", rec.Body)
	}
	if rec := call(t, mux, http.MethodPut, path, "an", `{"correct":6,"total":5}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("too many right: %d %s", rec.Code, rec.Body)
	}
	if rec := call(t, mux, http.MethodPut, "/api/vocab/review/lessons/nope", "an", `{"correct":1,"total":1}`); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown lesson: %d", rec.Code)
	}
	if rec := call(t, mux, http.MethodPut, path, "", `{"correct":1,"total":1}`); rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous: %d", rec.Code)
	}
}
