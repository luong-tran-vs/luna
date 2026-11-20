package toeic

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestLearnerEndpoints(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	mux := e.mux()
	e.readingTest("t1")

	if rec := call(t, mux, http.MethodGet, "/api/toeic", "", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous: %d", rec.Code)
	}
	rec := call(t, mux, http.MethodGet, "/api/toeic", "member", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"key":"t1"`) {
		t.Fatalf("library: %d %s", rec.Code, rec.Body)
	}
	if rec := call(t, mux, http.MethodPost, "/api/toeic/attempts", "member", `{"source": {"kind": "test"}, "parts": [5]}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("no test key: %d", rec.Code)
	}
	if rec := call(t, mux, http.MethodPost, "/api/toeic/attempts", "member", `{"source": {"kind": "test", "testKey": "t1"}, "parts": [9]}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad part: %d", rec.Code)
	}
	if rec := call(t, mux, http.MethodPost, "/api/toeic/attempts", "guest", `{"source": {"kind": "test", "testKey": "t1"}, "parts": [5]}`); rec.Code != http.StatusForbidden ||
		!strings.Contains(rec.Body.String(), "members_only") {
		t.Fatalf("guest: %d %s", rec.Code, rec.Body)
	}
	rec = call(t, mux, http.MethodPost, "/api/toeic/attempts", "member", `{"source": {"kind": "test", "testKey": "t1"}, "parts": [7], "timeLimitMin": 30}`)
	if rec.Code != http.StatusCreated || strings.Contains(rec.Body.String(), "explainVi") {
		t.Fatalf("start: %d %s", rec.Code, rec.Body)
	}
	v := decode[AttemptView](t, rec)
	rec = call(t, mux, http.MethodPost, "/api/toeic/attempts", "member", `{"source": {"kind": "test", "testKey": "t1"}, "parts": [5]}`)
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), `"attemptId":"`+v.ID+`"`) {
		t.Fatalf("busy: %d %s", rec.Code, rec.Body)
	}
	if rec := call(t, mux, http.MethodGet, "/api/toeic/attempts/"+v.ID, "other", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("another learner: %d", rec.Code)
	}
	if rec := call(t, mux, http.MethodGet, "/api/toeic/attempts/"+v.ID+"/result", "member", ""); rec.Code != http.StatusConflict {
		t.Fatalf("result before submit: %d", rec.Code)
	}
	qid := v.Paper[0].Questions[0].ID
	if rec := call(t, mux, http.MethodPut, "/api/toeic/attempts/"+v.ID, "member", `{"answers": {"`+qid+`": 1}, "remainingMs": 1000}`); rec.Code != http.StatusNoContent {
		t.Fatalf("save: %d %s", rec.Code, rec.Body)
	}
	if rec := call(t, mux, http.MethodPost, "/api/toeic/attempts/"+v.ID+"/submit", "member", `{}`); rec.Code != http.StatusOK ||
		!strings.Contains(rec.Body.String(), `"correct":1`) {
		t.Fatalf("submit: %d %s", rec.Code, rec.Body)
	}
	rec = call(t, mux, http.MethodGet, "/api/toeic/attempts/"+v.ID+"/questions/"+url.PathEscape(qid), "member", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"explainVi"`) || !strings.Contains(rec.Body.String(), `"answer":1`) {
		t.Fatalf("detail: %d %s", rec.Code, rec.Body)
	}
	if rec := call(t, mux, http.MethodGet, "/api/toeic/tests/t1", "member", ""); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"part":7`) {
		t.Fatalf("test detail: %d %s", rec.Code, rec.Body)
	}
	if rec := call(t, mux, http.MethodGet, "/api/toeic/tests/nope", "member", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown test: %d", rec.Code)
	}
	if rec := call(t, mux, http.MethodPost, "/api/toeic/vocab", "member", `{"word": "garden", "meaningVi": "khu vườn"}`); rec.Code != http.StatusNoContent || len(e.cards.saved) != 1 {
		t.Fatalf("vocab: %d %s", rec.Code, rec.Body)
	}
	if rec := call(t, mux, http.MethodPost, "/api/toeic/vocab", "member", `{"word": ""}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("empty word: %d", rec.Code)
	}
}

func TestMistakeEndpoints(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	mux := e.mux()
	e.readingTest("t1")
	v := startPractice(t, e, "u1", 5)
	if _, err := e.svc.Submit(t.Context(), "u1", v.ID, nil); err != nil {
		t.Fatal(err)
	}
	rec := call(t, mux, http.MethodGet, "/api/toeic/mistakes?part=5", "member", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"5":30`) {
		t.Fatalf("mistakes: %d %s", rec.Code, rec.Body)
	}
	if rec := call(t, mux, http.MethodGet, "/api/toeic/mistakes", "guest", ""); rec.Code != http.StatusForbidden {
		t.Fatalf("guest: %d", rec.Code)
	}
	rec = call(t, mux, http.MethodGet, "/api/toeic/mistakes/round?count=3", "member", "")
	if rec.Code != http.StatusOK || strings.Count(rec.Body.String(), `"questionId"`) != 3 || strings.Contains(rec.Body.String(), "explainVi") {
		t.Fatalf("round: %d %s", rec.Code, rec.Body)
	}
	qid := url.PathEscape(v.Paper[0].Questions[0].ID)
	if rec := call(t, mux, http.MethodPost, "/api/toeic/mistakes/"+qid+"/check", "member", `{"choice": 9}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("choice out of range: %d", rec.Code)
	}
	rec = call(t, mux, http.MethodPost, "/api/toeic/mistakes/"+qid+"/check", "member", `{"choice": 0}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"right":true`) || !strings.Contains(rec.Body.String(), `"streak":1`) {
		t.Fatalf("check: %d %s", rec.Code, rec.Body)
	}
	if rec := call(t, mux, http.MethodPost, "/api/toeic/mistakes/finish", "member", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("finish: %d", rec.Code)
	}
}
