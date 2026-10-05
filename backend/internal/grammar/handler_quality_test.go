package grammar

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/ai"
)

func TestAdminJSONCarriesChecksAndReports(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	a.env.ai.solve = func(req ai.SolveRequest) ([]ai.Solution, error) {
		out := solveLikeFixtures(req)
		out[0].Ambiguous, out[0].NoteVi = true, "Hai đáp án đúng."
		return out, nil
	}
	rec := a.do(t, http.MethodPost, adminPath+"/generate", "admin", `{"force":false}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("generate: %d %s", rec.Code, rec.Body)
	}
	l := decode(t, rec)["lesson"].(map[string]any)
	checks := l["checks"].([]any)
	if len(checks) != 1 || l["checkedAt"] == nil || len(l["reports"].([]any)) != 0 {
		t.Fatalf("lesson = %v", l)
	}
	c := checks[0].(map[string]any)
	if c["exerciseId"] != "p1" || c["kind"] != "ambiguous" || c["noteVi"] != "Hai đáp án đúng." {
		t.Errorf("check = %v", c)
	}

	// List: flags and checked.
	list := decode(t, a.do(t, http.MethodGet, "/api/admin/grammar-lessons", "admin", ""))["lessons"].([]any)
	row := list[0].(map[string]any)
	if row["flags"] != float64(1) || row["checked"] != true {
		t.Errorf("row = %v", row)
	}

	// PUT clears them (checks is still an array, checkedAt null).
	body := `{"content":` + mustJSON(t, l["content"]) + `}`
	rec = a.do(t, http.MethodPut, adminPath, "admin", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("save: %d %s", rec.Code, rec.Body)
	}
	saved := decode(t, rec)["lesson"].(map[string]any)
	if cs, ok := saved["checks"].([]any); !ok || len(cs) != 0 || saved["checkedAt"] != nil {
		t.Errorf("after save = %v", saved)
	}
}

func TestCheckEndpoint(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	if rec := a.do(t, http.MethodPost, adminPath+"/check", "admin", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("no lesson: %d", rec.Code)
	}
	a.generate(t)

	a.env.ai.solve = func(req ai.SolveRequest) ([]ai.Solution, error) {
		out := solveLikeFixtures(req)
		out[1].Answer = "are"
		return out, nil
	}
	rec := a.do(t, http.MethodPost, adminPath+"/check", "admin", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("check: %d %s", rec.Code, rec.Body)
	}
	checks := decode(t, rec)["lesson"].(map[string]any)["checks"].([]any)
	if len(checks) != 1 || checks[0].(map[string]any)["kind"] != "mismatch" {
		t.Errorf("checks = %v", checks)
	}

	for _, tc := range []struct {
		err    error
		status int
		code   string
	}{
		{ai.ErrNotConfigured, http.StatusServiceUnavailable, "ai_not_configured"},
		{ai.ErrQuota, http.StatusTooManyRequests, "ai_quota"},
		{errBoom, http.StatusBadGateway, "ai_failed"},
	} {
		a.env.ai.solve = func(ai.SolveRequest) ([]ai.Solution, error) { return nil, tc.err }
		rec := a.do(t, http.MethodPost, adminPath+"/check", "admin", "")
		if rec.Code != tc.status || decode(t, rec)["error"] != tc.code {
			t.Errorf("%v: %d %s", tc.err, rec.Code, rec.Body)
		}
	}
}

func TestPublishFlagGate(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	a.env.ai.solve = func(req ai.SolveRequest) ([]ai.Solution, error) {
		out := solveLikeFixtures(req)
		out[0].Ambiguous, out[2].Ambiguous = true, true
		return out, nil
	}
	a.generate(t)

	rec := a.do(t, http.MethodPost, adminPath+"/publish", "admin", "")
	got := decode(t, rec)
	if rec.Code != http.StatusConflict || got["error"] != "grammar_flags_unresolved" || got["count"] != float64(2) || got["message"] == "" {
		t.Fatalf("publish: %d %s", rec.Code, rec.Body)
	}
	if rec := a.do(t, http.MethodPost, adminPath+"/publish", "admin", `{"acknowledgeFlags":false}`); rec.Code != http.StatusConflict {
		t.Errorf("acknowledgeFlags false: %d", rec.Code)
	}
	if rec := a.do(t, http.MethodPost, adminPath+"/publish", "admin", `{"acknowledgeFlags":true}`); rec.Code != http.StatusOK {
		t.Fatalf("acknowledged: %d %s", rec.Code, rec.Body)
	}
}

func TestLearnerGetReturnsExpandedAnswers(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	c := aiContent()
	c.Practice[1] = mod(goodFill("x"), func(x *ai.GrammarExercise) { x.Text = "She ___ here."; x.Answers = []string{"is not"} })
	a.env.ai.content = c
	a.env.ai.solve = func(req ai.SolveRequest) ([]ai.Solution, error) {
		out := solveLikeFixtures(req)
		out[1].Answer = "is not"
		return out, nil
	}
	a.generate(t)
	a.publish(t)

	rec := a.do(t, http.MethodGet, "/api/grammar/"+point, "learner", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("get: %d %s", rec.Code, rec.Body)
	}
	fill := decode(t, rec)["content"].(map[string]any)["practice"].([]any)[1].(map[string]any)
	answers := fill["answers"].([]any)
	if len(answers) < 2 || answers[0] != "is not" || answers[1] != "isn't" {
		t.Errorf("answers = %v", answers)
	}
}

func TestReportEndpoints(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	path := "/api/grammar/" + point + "/reports"

	a.generate(t)
	if rec := a.do(t, http.MethodPost, path, "learner", `{"exerciseId":"p3","reason":"typo"}`); rec.Code != http.StatusNotFound {
		t.Fatalf("draft: %d %s", rec.Code, rec.Body)
	}
	a.publish(t)

	bad := []string{
		`{"exerciseId":"zz","reason":"typo"}`,
		`{"exerciseId":"p3","reason":"nope"}`,
		`{"exerciseId":"p3","reason":"typo","note":"` + longNote() + `"}`,
	}
	for _, body := range bad {
		rec := a.do(t, http.MethodPost, path, "learner", body)
		if rec.Code != http.StatusBadRequest || decode(t, rec)["fields"] == nil {
			t.Errorf("%s: %d %s", body, rec.Code, rec.Body)
		}
	}
	if rec := a.do(t, http.MethodPost, path, "learner", `{"exerciseId":"p3","reason":"typo","extra":1}`); rec.Code != http.StatusBadRequest {
		t.Errorf("unknown field: %d", rec.Code)
	}

	for _, body := range []string{
		`{"exerciseId":"p3","reason":"wrong_answer","note":"đáp án sai"}`,
		`{"exerciseId":"m2","reason":"typo"}`,
	} {
		rec := a.do(t, http.MethodPost, path, "learner", body)
		if rec.Code != http.StatusOK || decode(t, rec)["ok"] != true {
			t.Fatalf("report: %d %s", rec.Code, rec.Body)
		}
	}
	// Reporting the same exercise again is allowed (it reopens), as is an admin reporting.
	if rec := a.do(t, http.MethodPost, path, "learner", `{"exerciseId":"p3","reason":"ambiguous"}`); rec.Code != http.StatusOK {
		t.Fatalf("again: %d", rec.Code)
	}

	rec := a.do(t, http.MethodGet, "/api/admin/grammar-reports", "admin", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list: %d %s", rec.Code, rec.Body)
	}
	reports := decode(t, rec)["reports"].([]any)
	if len(reports) != 2 {
		t.Fatalf("reports = %v", reports)
	}
	g := reports[0].(map[string]any)
	if g["pointId"] != point || g["exerciseId"] != "p3" || g["count"] != float64(1) || g["latestAt"] == "" ||
		g["reasons"].(map[string]any)["ambiguous"] != float64(1) || len(g["notes"].([]any)) != 0 {
		t.Errorf("newest group = %v", g)
	}

	// The lesson detail lists the open reports of the point.
	detail := decode(t, a.do(t, http.MethodGet, adminPath, "admin", ""))["lesson"].(map[string]any)["reports"].([]any)
	if len(detail) != 2 || detail[0].(map[string]any)["exerciseId"] != "p3" {
		t.Errorf("detail reports = %v", detail)
	}

	rec = a.do(t, http.MethodPost, "/api/admin/grammar-reports/resolve", "admin", `{"pointId":"`+point+`","exerciseId":"p3"}`)
	if rec.Code != http.StatusOK || decode(t, rec)["resolved"] != float64(1) {
		t.Fatalf("resolve: %d %s", rec.Code, rec.Body)
	}
	if rec := a.do(t, http.MethodPost, "/api/admin/grammar-reports/resolve", "admin", `{"pointId":"nope","exerciseId":"p3"}`); rec.Code != http.StatusNotFound {
		t.Errorf("unknown point: %d", rec.Code)
	}
	if rec := a.do(t, http.MethodPost, "/api/admin/grammar-reports/resolve", "admin", `{"pointId":"`+point+`","exerciseId":""}`); rec.Code != http.StatusBadRequest {
		t.Errorf("empty exercise: %d", rec.Code)
	}
	list := decode(t, a.do(t, http.MethodGet, "/api/admin/grammar-reports", "admin", ""))["reports"].([]any)
	if len(list) != 1 || list[0].(map[string]any)["exerciseId"] != "m2" {
		t.Errorf("after resolve = %v", list)
	}
}

var errBoom = errors.New("boom")

func longNote() string { return strings.Repeat("a", 301) }

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
