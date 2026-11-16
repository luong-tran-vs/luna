package games

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/ai"
)

func TestListEndpoint(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	mux := e.mux()
	if rec := call(t, mux, http.MethodGet, "/api/games", "", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous: %d", rec.Code)
	}
	_, _ = e.bests.Raise(t.Context(), "u2", GamePicture, 2100, testNow)
	rec := call(t, mux, http.MethodGet, "/api/games", "guest", "")
	var body struct {
		Games []struct {
			ID     string  `json:"id"`
			Best   int     `json:"best"`
			BestAt *string `json:"bestAt"`
			Locked bool    `json:"locked"`
		} `json:"games"`
		Daily struct {
			Day    string `json:"day"`
			Done   bool   `json:"done"`
			Streak int    `json:"streak"`
		} `json:"daily"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); rec.Code != http.StatusOK || err != nil {
		t.Fatalf("list: %d %s", rec.Code, rec.Body)
	}
	if len(body.Games) != len(released) || body.Games[0].ID != "picture" || body.Games[0].Best != 2100 || body.Games[0].BestAt == nil {
		t.Fatalf("games = %+v", body.Games)
	}
	for _, g := range body.Games {
		if g.Locked != (g.ID == "grammar") {
			t.Fatalf("locked = %+v", g)
		}
	}
	if body.Daily.Day != "2026-11-13" || body.Daily.Done {
		t.Fatalf("daily = %+v", body.Daily)
	}
}

func TestRoundEndpoints(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	mux := e.mux()
	rec := call(t, mux, http.MethodGet, "/api/games/picture/round", "member", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"imageUrl":"/api/words/`) {
		t.Fatalf("picture: %d %s", rec.Code, rec.Body)
	}
	if rec := call(t, mux, http.MethodGet, "/api/games/picture/round", "new", ""); rec.Code != http.StatusConflict ||
		!strings.Contains(rec.Body.String(), "not_enough_words") {
		t.Fatalf("picture, new learner: %d %s", rec.Code, rec.Body)
	}
	if rec := call(t, mux, http.MethodGet, "/api/games/picture/round", "", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("picture, anonymous: %d", rec.Code)
	}
	if rec := call(t, mux, http.MethodGet, "/api/games/letters/round", "member", ""); rec.Code != http.StatusOK ||
		!strings.Contains(rec.Body.String(), `"words":[`) {
		t.Fatalf("letters: %d %s", rec.Code, rec.Body)
	}

	e.lessons.words["L5"] = []LessonWord{{Lemma: "apple"}, {Lemma: "pear"}, {Lemma: "lemon"}, {Lemma: "mango"}, {Lemma: "grape"}, {Lemma: "peach"}}
	if rec := call(t, mux, http.MethodGet, "/api/games/memory/round?lessonId=L5", "member", ""); rec.Code != http.StatusForbidden {
		t.Fatalf("memory, locked lesson: %d %s", rec.Code, rec.Body)
	}
	e.study.open["L5"] = true
	rec = call(t, mux, http.MethodGet, "/api/games/memory/round?lessonId=L5", "member", "")
	var mem struct {
		Pairs []struct{ Lemma string } `json:"pairs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &mem); rec.Code != http.StatusOK || err != nil || len(mem.Pairs) != 6 {
		t.Fatalf("memory: %d %s", rec.Code, rec.Body)
	}
	if rec := call(t, mux, http.MethodGet, "/api/games/memory/round", "new", ""); rec.Code != http.StatusConflict {
		t.Fatalf("memory, new learner: %d", rec.Code)
	}
}

func TestGrammarRoundEndpoint(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.grammar.sources = []PointSource{presentSimple(), pastSimple()}
	e.spots.items = []SpotItem{
		{ID: "x1", PointID: "a1-present-simple", Sentence: "My dog eat bones.", Wrong: "eat", Right: "eats", Distractors: []string{"eating", "ate"}, RuleVi: "r", Status: SpotApproved},
		{ID: "x2", PointID: "a1-present-simple", Sentence: "Tom like milk.", Wrong: "like", Right: "likes", Distractors: []string{"liking", "liked"}, RuleVi: "r", Status: SpotApproved},
	}
	mux := e.mux()
	if rec := call(t, mux, http.MethodGet, "/api/games/grammar/round", "guest", ""); rec.Code != http.StatusForbidden ||
		!strings.Contains(rec.Body.String(), "members_only") {
		t.Fatalf("guest: %d %s", rec.Code, rec.Body)
	}
	rec := call(t, mux, http.MethodGet, "/api/games/grammar/round", "member", "")
	var body struct {
		Questions []struct {
			Tokens  []string        `json:"tokens"`
			Wrong   *map[string]int `json:"wrong"`
			Options []string        `json:"options"`
			TopicVi string          `json:"topicVi"`
		} `json:"questions"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); rec.Code != http.StatusOK || err != nil || len(body.Questions) != 10 {
		t.Fatalf("member: %d %s", rec.Code, rec.Body)
	}
	for _, q := range body.Questions {
		if q.Options == nil || q.TopicVi == "" {
			t.Fatalf("question = %+v", q)
		}
	}
}

func TestResultsEndpoint(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	mux := e.mux()
	rec := call(t, mux, http.MethodPost, "/api/games/picture/results", "member", `{"score":2100,"completed":true,"wrongWords":["bicycle"]}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"best":2100,"newBest":true,"rescheduled":1`) {
		t.Fatalf("results: %d %s", rec.Code, rec.Body)
	}
	for path, body := range map[string]string{
		"/api/games/picture/results": `{"score":30001,"completed":true}`,
		"/api/games/chess/results":   `{"score":1}`,
		"/api/games/verbs/results":   `{"score":1,"wrongVerbs":[{"base":"go","form":"future","typed":"x"}]}`,
	} {
		if rec := call(t, mux, http.MethodPost, path, "member", body); rec.Code != http.StatusBadRequest {
			t.Fatalf("%s %s: %d %s", path, body, rec.Code, rec.Body)
		}
	}
	if rec := call(t, mux, http.MethodPost, "/api/games/grammar/results", "guest", `{"score":100,"completed":true}`); rec.Code != http.StatusForbidden {
		t.Fatalf("guest grammar: %d", rec.Code)
	}
	if rec := call(t, mux, http.MethodPost, "/api/games/verbs/results", "member",
		`{"score":3,"completed":true,"wrongVerbs":[{"base":"run","form":"past","typed":"runned"}]}`); rec.Code != http.StatusOK || len(e.verbs.misses) != 1 {
		t.Fatalf("verbs: %d %s", rec.Code, rec.Body)
	}
}

func dailyBody(day string, score int) string {
	var b strings.Builder
	b.WriteString(`{"day":"` + day + `","score":` + itoa(score) + `,"answers":[`)
	for i := 0; i < 10; i++ {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(`{"kind":"picture","right":true}`)
	}
	b.WriteString(`],"wrongWords":[]}`)
	return b.String()
}

func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}

func TestDailyEndpoints(t *testing.T) {
	t.Parallel()
	e := dailyEnv(t)
	mux := e.mux()
	rec := call(t, mux, http.MethodGet, "/api/games/daily", "member", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"done":false`) || !strings.Contains(rec.Body.String(), `"result":null`) {
		t.Fatalf("daily: %d %s", rec.Code, rec.Body)
	}
	for _, body := range []string{
		dailyBody("2026-11-01", 900),
		dailyBody("2026-11-13", 3001),
		strings.Replace(dailyBody("2026-11-13", 900), `{"kind":"picture","right":true},`, "", 1),
		strings.Replace(dailyBody("2026-11-13", 900), `"kind":"picture"`, `"kind":"memory"`, 1),
	} {
		if rec := call(t, mux, http.MethodPost, "/api/games/daily/result", "member", body); rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: %d %s", body, rec.Code, rec.Body)
		}
	}
	rec = call(t, mux, http.MethodPost, "/api/games/daily/result", "member", dailyBody("2026-11-13", 900))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"streak":{"current":1,"longest":1}`) {
		t.Fatalf("save: %d %s", rec.Code, rec.Body)
	}
	rec = call(t, mux, http.MethodPost, "/api/games/daily/result", "member", dailyBody("2026-11-13", 3000))
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), `"error":"already_done"`) || !strings.Contains(rec.Body.String(), `"score":900`) {
		t.Fatalf("again: %d %s", rec.Code, rec.Body)
	}
	rec = call(t, mux, http.MethodGet, "/api/games/daily", "member", "")
	if !strings.Contains(rec.Body.String(), `"done":true`) || !strings.Contains(rec.Body.String(), `"score":900`) {
		t.Fatalf("daily done: %s", rec.Body)
	}
	if rec := call(t, mux, http.MethodPost, "/api/games/daily/practice", "member", `{"wrongWords":["apple"]}`); rec.Code != http.StatusOK ||
		!strings.Contains(rec.Body.String(), `"rescheduled":1`) {
		t.Fatalf("practice: %d %s", rec.Code, rec.Body)
	}
}

func TestSpotAdminEndpoints(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.grammar.sources = []PointSource{presentSimple()}
	e.ai.items = []ai.GrammarSpotItem{{Sentence: "My dog eat bones.", Wrong: "eat", Right: "eats", Distractors: []string{"eating", "ate"}, RuleVi: "r"}}
	mux := e.mux()
	for _, token := range []string{"member", "guest"} {
		if rec := call(t, mux, http.MethodGet, "/api/admin/games/spot-counts", token, ""); rec.Code != http.StatusForbidden {
			t.Fatalf("%s counts: %d", token, rec.Code)
		}
		if rec := call(t, mux, http.MethodPost, "/api/admin/games/spot-items/generate", token, `{"pointId":"a1-present-simple"}`); rec.Code != http.StatusForbidden {
			t.Fatalf("%s generate: %d", token, rec.Code)
		}
	}
	rec := call(t, mux, http.MethodGet, "/api/admin/games/spot-counts", "admin", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"pointId":"a1-present-simple"`) {
		t.Fatalf("counts: %d %s", rec.Code, rec.Body)
	}
	rec = call(t, mux, http.MethodPost, "/api/admin/games/spot-items/generate", "admin", `{"pointId":"a1-present-simple"}`)
	var gen struct {
		Created []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"created"`
		Dropped int `json:"dropped"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &gen); rec.Code != http.StatusOK || err != nil || len(gen.Created) != 1 || gen.Created[0].Status != "draft" {
		t.Fatalf("generate: %d %s", rec.Code, rec.Body)
	}
	id := gen.Created[0].ID
	if rec := call(t, mux, http.MethodGet, "/api/admin/games/spot-items?pointId=a1-present-simple", "admin", ""); rec.Code != http.StatusOK ||
		!strings.Contains(rec.Body.String(), id) {
		t.Fatalf("items: %d %s", rec.Code, rec.Body)
	}
	if rec := call(t, mux, http.MethodPut, "/api/admin/games/spot-items/"+id, "admin", `{"sentence":"x","wrong":"y","right":"z","distractors":[],"ruleVi":""}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad edit: %d %s", rec.Code, rec.Body)
	}
	if rec := call(t, mux, http.MethodPost, "/api/admin/games/spot-items/"+id+"/approve", "admin", ""); rec.Code != http.StatusOK ||
		!strings.Contains(rec.Body.String(), `"status":"approved"`) {
		t.Fatalf("approve: %d %s", rec.Code, rec.Body)
	}
	if rec := call(t, mux, http.MethodDelete, "/api/admin/games/spot-items/"+id, "admin", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: %d", rec.Code)
	}
	if rec := call(t, mux, http.MethodDelete, "/api/admin/games/spot-items/"+id, "admin", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("delete twice: %d", rec.Code)
	}
	e.ai.err = ai.ErrQuota
	if rec := call(t, mux, http.MethodPost, "/api/admin/games/spot-items/generate", "admin", `{"pointId":"a1-present-simple"}`); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("quota: %d %s", rec.Code, rec.Body)
	}
	if rec := call(t, mux, http.MethodPost, "/api/admin/games/spot-items/generate", "admin", `{"pointId":"zz"}`); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown point: %d %s", rec.Code, rec.Body)
	}
}
