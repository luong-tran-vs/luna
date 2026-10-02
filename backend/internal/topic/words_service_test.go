package topic

import (
	"errors"
	"net/http"
	"slices"
	"strings"
	"testing"
)

// familyTopic creates "A1 · Gia đình" with three words and two lessons.
func (e *testEnv) familyTopic(t *testing.T) string {
	t.Helper()
	id := e.create(t, "Gia đình", "A1").ID
	if _, err := e.svc.SetWords(t.Context(), id, []string{"Family", "Grandmother", "Cousin"}); err != nil {
		t.Fatal(err)
	}
	e.lessons.add("l1", "Một", id)
	e.lessons.setText("l1", "My family loves my grandmothers.")
	e.lessons.add("l2", "Hai", id)
	e.lessons.setText("l2", "Every family is different.")
	other := e.create(t, "Công việc", "B1").ID
	e.lessons.add("l3", "Ba", other)
	e.lessons.setText("l3", "My cousin works here.")
	return id
}

func TestListWordCoverage(t *testing.T) {
	t.Parallel()
	e := newEnv()
	id := e.familyTopic(t)
	e.lessons.textCalls = 0

	items, err := e.svc.List(t.Context(), "")
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(items, func(s Summary) bool { return s.ID == id })
	if items[i].WordCount != 3 || items[i].UsedWordCount != 2 {
		t.Fatalf("coverage = %d/%d", items[i].UsedWordCount, items[i].WordCount)
	}
	if e.lessons.textCalls != 1 {
		t.Fatalf("Texts calls = %d, want 1", e.lessons.textCalls)
	}
}

func TestWordsAndSetWords(t *testing.T) {
	t.Parallel()
	e := newEnv()
	id := e.familyTopic(t)

	got, err := e.svc.Words(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	want := []WordUse{{"Family", true, 2}, {"Grandmother", true, 1}, {"Cousin", false, 0}}
	if !slices.Equal(got, want) {
		t.Fatalf("words = %+v", got)
	}

	got, err = e.svc.SetWords(t.Context(), id, []string{" Family ", "Uncle"})
	if err != nil || len(got) != 2 || got[1] != (WordUse{Text: "Uncle"}) {
		t.Fatalf("set = %+v, %v", got, err)
	}
	stored, _ := e.repo.Get(t.Context(), id)
	if !stored.WordsSeeded || !slices.Equal(stored.Words, []string{"Family", "Uncle"}) {
		t.Fatalf("stored = %+v", stored)
	}

	if _, err := e.svc.SetWords(t.Context(), id, []string{"Uncle", "uncle"}); fieldErr(t, err, "words.1") != "Từ bị trùng" {
		t.Fatal("duplicate accepted")
	}
	if stored, _ = e.repo.Get(t.Context(), id); len(stored.Words) != 2 {
		t.Fatal("invalid list was saved")
	}
	if _, err := e.svc.Words(t.Context(), "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing: %v", err)
	}
	if _, err := e.svc.SetWords(t.Context(), "missing", nil); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing: %v", err)
	}
}

func TestWordEndpoints(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	id := a.env.familyTopic(t)
	path := "/api/admin/topics/" + id + "/words"

	rec := a.do(t, http.MethodGet, path, "admin", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(),
		`{"words":[{"text":"Family","used":true,"lessonCount":2},{"text":"Grandmother","used":true,"lessonCount":1},{"text":"Cousin","used":false,"lessonCount":0}]}`) {
		t.Fatalf("get: %d %s", rec.Code, rec.Body)
	}

	rec = a.do(t, http.MethodPut, path, "admin", `{"words":["Family","  take   a shower ","cousin"]}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `{"text":"take a shower","used":false,"lessonCount":0}`) {
		t.Fatalf("put: %d %s", rec.Code, rec.Body)
	}

	rec = a.do(t, http.MethodPut, path, "admin", `{"words":["Family","family","bố"]}`)
	body := decode(t, rec)
	fields, _ := body["fields"].(map[string]any)
	if rec.Code != http.StatusBadRequest || fields["words.1"] != "Từ bị trùng" || fields["words.2"] == nil {
		t.Fatalf("invalid: %d %s", rec.Code, rec.Body)
	}
	if rec := a.do(t, http.MethodPut, path, "admin", `{"words":`); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad json: %d", rec.Code)
	}
	for _, m := range []string{http.MethodGet, http.MethodPut} {
		if rec := a.do(t, m, "/api/admin/topics/missing/words", "admin", `{"words":[]}`); rec.Code != http.StatusNotFound {
			t.Errorf("%s missing: %d", m, rec.Code)
		}
		if rec := a.do(t, m, path, "learner", `{"words":[]}`); rec.Code != http.StatusForbidden {
			t.Errorf("%s learner: %d", m, rec.Code)
		}
		if rec := a.do(t, m, path, "", `{"words":[]}`); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s anonymous: %d", m, rec.Code)
		}
	}

	rec = a.do(t, http.MethodGet, "/api/admin/topics", "admin", "")
	if !strings.Contains(rec.Body.String(), `"wordCount":3,"usedWordCount":1`) {
		t.Fatalf("list: %s", rec.Body)
	}
}

func TestWordPlanEndpoint(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	id := a.env.familyTopic(t)
	base := "/api/admin/topics/" + id + "/word-plan"

	rec := a.do(t, http.MethodGet, base+"?count=2&perLesson=1", "admin", "")
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != `{"groups":[["Cousin"],["Grandmother"]]}` {
		t.Fatalf("plan: %d %s", rec.Code, rec.Body)
	}
	rec = a.do(t, http.MethodGet, base+"?count=0&perLesson=16", "admin", "")
	fields, _ := decode(t, rec)["fields"].(map[string]any)
	if rec.Code != http.StatusBadRequest || fields["count"] == nil || fields["perLesson"] == nil {
		t.Fatalf("invalid: %d %s", rec.Code, rec.Body)
	}
	if rec := a.do(t, http.MethodGet, base+"?count=x", "admin", ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("not a number: %d", rec.Code)
	}
	if rec := a.do(t, http.MethodGet, "/api/admin/topics/missing/word-plan?count=1&perLesson=1", "admin", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("missing: %d", rec.Code)
	}
	if rec := a.do(t, http.MethodGet, base+"?count=1&perLesson=1", "learner", ""); rec.Code != http.StatusForbidden {
		t.Fatalf("learner: %d", rec.Code)
	}
}
