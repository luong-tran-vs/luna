package topic

import (
	"errors"
	"net/http"
	"slices"
	"strings"
	"testing"
)

// familyTopic creates "Gia đình" with three words (Cousin of level A2) and two A1 lessons.
func (e *testEnv) familyTopic(t *testing.T) string {
	t.Helper()
	id := e.create(t, "Gia đình").ID
	if _, err := e.svc.SetWords(t.Context(), id, []Word{{Text: "Family"}, {Text: "Grandmother"}, {Text: "Cousin", Level: "A2"}}); err != nil {
		t.Fatal(err)
	}
	e.lessons.add("l1", "Một", id, "A1")
	e.lessons.setText("l1", "My family loves my grandmothers.")
	e.lessons.add("l2", "Hai", id, "A1")
	e.lessons.setText("l2", "Every family is different.")
	other := e.create(t, "Công việc").ID
	e.lessons.add("l3", "Ba", other, "B1")
	e.lessons.setText("l3", "My cousin works here.")
	return id
}

func TestListWordCoverage(t *testing.T) {
	t.Parallel()
	e := newEnv()
	id := e.familyTopic(t)
	e.lessons.textCalls = 0

	items, err := e.svc.List(t.Context())
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
	want := []WordUse{{"Family", "", true, 2}, {"Grandmother", "", true, 1}, {"Cousin", "A2", false, 0}}
	if !slices.Equal(got, want) {
		t.Fatalf("words = %+v", got)
	}

	got, err = e.svc.SetWords(t.Context(), id, []Word{{Text: " Family "}, {Text: "Uncle", Level: "b1"}})
	if err != nil || len(got) != 2 || got[1] != (WordUse{Text: "Uncle", Level: "B1"}) {
		t.Fatalf("set = %+v, %v", got, err)
	}
	stored, _ := e.repo.Get(t.Context(), id)
	if !stored.WordsSeeded || !slices.Equal(stored.Words, []Word{{Text: "Family"}, {Text: "Uncle", Level: "B1"}}) {
		t.Fatalf("stored = %+v", stored)
	}

	if _, err := e.svc.SetWords(t.Context(), id, words("Uncle", "uncle")); fieldErr(t, err, "words.1") != "Từ bị trùng" {
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
		`{"words":[{"text":"Family","level":"","used":true,"lessonCount":2},{"text":"Grandmother","level":"","used":true,"lessonCount":1},{"text":"Cousin","level":"A2","used":false,"lessonCount":0}]}`) {
		t.Fatalf("get: %d %s", rec.Code, rec.Body)
	}
	if rec := a.do(t, http.MethodGet, path+"?level=A2", "admin", ""); !strings.Contains(rec.Body.String(), `{"words":[{"text":"Cousin","level":"A2"`) {
		t.Fatalf("get A2: %d %s", rec.Code, rec.Body)
	}

	// Words are objects with a level, or plain strings (any level).
	rec = a.do(t, http.MethodPut, path, "admin", `{"words":["Family",{"text":"  take   a shower ","level":"B1"},"cousin"]}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `{"text":"take a shower","level":"B1","used":false,"lessonCount":0}`) {
		t.Fatalf("put: %d %s", rec.Code, rec.Body)
	}
	rec = a.do(t, http.MethodPut, path, "admin", `{"words":[{"text":"Family","level":"Z9"}]}`)
	if fields, _ := decode(t, rec)["fields"].(map[string]any); rec.Code != http.StatusBadRequest || fields["words.0.level"] == nil {
		t.Fatalf("bad level: %d %s", rec.Code, rec.Body)
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

	rec := a.do(t, http.MethodGet, base+"?level=A2&count=2&perLesson=1", "admin", "")
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != `{"groups":[["Cousin"],["Grandmother"]],"shortage":1}` {
		t.Fatalf("plan: %d %s", rec.Code, rec.Body)
	}
	// Cousin (A2) is not given to an A1 lesson.
	rec = a.do(t, http.MethodGet, base+"?level=A1&count=2&perLesson=1", "admin", "")
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != `{"groups":[["Grandmother"],["Family"]],"shortage":2}` {
		t.Fatalf("plan A1: %d %s", rec.Code, rec.Body)
	}
	rec = a.do(t, http.MethodGet, base+"?count=0&perLesson=16", "admin", "")
	fields, _ := decode(t, rec)["fields"].(map[string]any)
	if rec.Code != http.StatusBadRequest || fields["count"] == nil || fields["perLesson"] == nil || fields["level"] == nil {
		t.Fatalf("invalid: %d %s", rec.Code, rec.Body)
	}
	if rec := a.do(t, http.MethodGet, base+"?count=x", "admin", ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("not a number: %d", rec.Code)
	}
	if rec := a.do(t, http.MethodGet, "/api/admin/topics/missing/word-plan?level=A1&count=1&perLesson=1", "admin", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("missing: %d", rec.Code)
	}
	if rec := a.do(t, http.MethodGet, base+"?level=A1&count=1&perLesson=1", "learner", ""); rec.Code != http.StatusForbidden {
		t.Fatalf("learner: %d", rec.Code)
	}
}

func TestWordPlanShortage(t *testing.T) {
	t.Parallel()
	e := newEnv()
	id := e.familyTopic(t) // Cousin is the only unused word

	plan, err := e.svc.WordPlan(t.Context(), id, "A2", 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Shortage != 5 || len(plan.Groups) != 3 {
		t.Fatalf("plan = %+v", plan)
	}
	if plan, _ := e.svc.WordPlan(t.Context(), id, "A2", 1, 1); plan.Shortage != 0 {
		t.Fatalf("enough words: %+v", plan)
	}
	// At A1 the only unused word (Cousin, A2) does not count.
	if plan, _ := e.svc.WordPlan(t.Context(), id, "A1", 1, 1); plan.Shortage != 1 {
		t.Fatalf("A1: %+v", plan)
	}
	empty := e.create(t, "Màu sắc").ID
	if plan, err := e.svc.WordPlan(t.Context(), empty, "A1", 2, 3); err != nil || plan.Shortage != 6 || len(plan.Groups) != 2 {
		t.Fatalf("no words: %+v, %v", plan, err)
	}
}

func TestSuggestWords(t *testing.T) {
	t.Parallel()
	e := newEnv()
	id := e.familyTopic(t)
	e.ai.err = nil
	e.ai.words = []string{"aunt", "FAMILY", "uncle", "bad_word!", "  baby   sister ", "nephew"}

	added, uses, err := e.svc.SuggestWords(t.Context(), id, "B1", 3)
	if err != nil {
		t.Fatal(err)
	}
	// Existing words (any case) and invalid ones are skipped; at most count words are added.
	if !slices.Equal(added, []string{"aunt", "uncle", "baby sister"}) {
		t.Fatalf("added = %v", added)
	}
	if len(uses) != 6 || uses[3].Text != "aunt" || uses[3].Level != "B1" || uses[3].Used {
		t.Fatalf("uses = %+v", uses)
	}
	req := e.ai.reqs[0]
	if req.Level != "B1" || req.TopicName != "Gia đình" || req.Count != 3 || len(req.Existing) != 3 {
		t.Fatalf("request = %+v", req)
	}

	e.ai.words = []string{"aunt"}
	if _, _, err := e.svc.SuggestWords(t.Context(), id, "B1", 1); !errors.Is(err, ErrNoSuggestion) {
		t.Fatalf("nothing new: %v", err)
	}
	var verr *ValidationError
	if _, _, err := e.svc.SuggestWords(t.Context(), id, "B1", MaxSuggestWords+1); !errors.As(err, &verr) {
		t.Fatalf("too many: %v", err)
	}
	if _, _, err := e.svc.SuggestWords(t.Context(), id, "", 1); fieldErr(t, err, "level") == "" {
		t.Fatal("missing level accepted")
	}
	if _, _, err := e.svc.SuggestWords(t.Context(), "missing", "B1", 1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing: %v", err)
	}
}

func TestSuggestWordsEndpoint(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	id := a.env.familyTopic(t)
	url := "/api/admin/topics/" + id + "/words/suggest"

	if rec := a.do(t, http.MethodPost, url, "admin", `{"count":2,"level":"A1"}`); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("not configured: %d %s", rec.Code, rec.Body)
	}
	a.env.ai.err, a.env.ai.words = nil, []string{"aunt", "uncle"}
	rec := a.do(t, http.MethodPost, url, "admin", `{"count":2,"level":"A1"}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"added":["aunt","uncle"]`) ||
		!strings.Contains(rec.Body.String(), `{"text":"uncle","level":"A1","used":false,"lessonCount":0}`) {
		t.Fatalf("suggest: %d %s", rec.Code, rec.Body)
	}
	if rec := a.do(t, http.MethodPost, url, "learner", `{"count":2,"level":"A1"}`); rec.Code != http.StatusForbidden {
		t.Fatalf("learner: %d", rec.Code)
	}
}

func TestAddWord(t *testing.T) {
	t.Parallel()
	e := newEnv()
	id := e.familyTopic(t)

	if err := e.svc.AddWord(t.Context(), id, "  uncle "); err != nil {
		t.Fatal(err)
	}
	got, _ := e.svc.Words(t.Context(), id)
	if last := got[len(got)-1]; len(got) != 4 || last.Text != "uncle" || last.Level != "" {
		t.Fatalf("words = %+v", got)
	}
	if err := e.svc.AddWord(t.Context(), id, "FAMILY"); !errors.Is(err, ErrWordInList) {
		t.Fatalf("duplicate: %v", err)
	}
	var verr *ValidationError
	if err := e.svc.AddWord(t.Context(), id, "covid-19"); !errors.As(err, &verr) || verr.Fields["word"] == "" {
		t.Fatalf("bad word: %v", err)
	}
	if err := e.svc.AddWord(t.Context(), "nope", "aunt"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown topic: %v", err)
	}
}
