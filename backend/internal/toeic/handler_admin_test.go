package toeic

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func importBody(t *testing.T, replace bool, gs ...Group) string {
	t.Helper()
	b, err := json.Marshal(map[string]any{"groups": gs, "replace": replace})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestAdminImportEndpoint(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	mux := e.mux()
	body := importBody(t, false, p5("p5-a", 1), p7("p7-a", 2))
	if rec := call(t, mux, http.MethodPost, "/api/admin/toeic/import?dryRun=true", "member", body); rec.Code != http.StatusForbidden {
		t.Fatalf("member: %d", rec.Code)
	}
	rec := call(t, mux, http.MethodPost, "/api/admin/toeic/import?dryRun=true", "admin", body)
	rep := decode[ImportReport](t, rec)
	if rec.Code != http.StatusOK || !rep.DryRun || len(rep.Created) != 2 || len(e.groups.recs) != 0 {
		t.Fatalf("dry run: %d %s", rec.Code, rec.Body)
	}
	bad := p5("p5-b", 0)
	bad.Questions[0].Tags = []string{"nope"}
	rec = call(t, mux, http.MethodPost, "/api/admin/toeic/import", "admin", importBody(t, false, bad))
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "groups[0].questions[0].tags") {
		t.Fatalf("invalid: %d %s", rec.Code, rec.Body)
	}
	rec = call(t, mux, http.MethodPost, "/api/admin/toeic/import", "admin", `{"groups": [{"key": "x", "nope": 1}]}`)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "unknown field") {
		t.Fatalf("unknown field: %d %s", rec.Code, rec.Body)
	}
	if rec := call(t, mux, http.MethodPost, "/api/admin/toeic/import", "admin", body); rec.Code != http.StatusOK || len(e.groups.recs) != 2 {
		t.Fatalf("import: %d %s", rec.Code, rec.Body)
	}

	rec = call(t, mux, http.MethodGet, "/api/admin/toeic/groups?part=7", "admin", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"key":"p7-a"`) || strings.Contains(rec.Body.String(), "p5-a") {
		t.Fatalf("list: %d %s", rec.Code, rec.Body)
	}
	rec = call(t, mux, http.MethodPatch, "/api/admin/toeic/groups/p7-a", "admin", `{"status": "published"}`)
	if rec.Code != http.StatusOK || e.groups.recs["p7-a"].Status != StatusPublished {
		t.Fatalf("publish: %d %s", rec.Code, rec.Body)
	}
	if rec := call(t, mux, http.MethodPatch, "/api/admin/toeic/groups/nope", "admin", `{"status": "published"}`); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown group: %d", rec.Code)
	}
	rec = call(t, mux, http.MethodGet, "/api/admin/toeic/groups/p7-a", "admin", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"explainVi"`) {
		t.Fatalf("preview: %d %s", rec.Code, rec.Body)
	}
}

func TestAdminTestEndpoints(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	mux := e.mux()
	e.seed(p5("p5-a", 0))
	rec := call(t, mux, http.MethodPost, "/api/admin/toeic/tests", "admin", `{"key": "luna-t1", "title": "Luna Test 01", "kind": "full", "order": 1, "groups": ["p5-a"]}`)
	if rec.Code != http.StatusCreated || !strings.Contains(rec.Body.String(), `"ready":false`) {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	rec = call(t, mux, http.MethodPatch, "/api/admin/toeic/tests/luna-t1", "admin", `{"status": "published"}`)
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "test_not_ready") || !strings.Contains(rec.Body.String(), `"need":30`) {
		t.Fatalf("publish not ready: %d %s", rec.Code, rec.Body)
	}
	e.readingTest("t2")
	e.tests.recs["t2"] = func() Test { x := e.tests.recs["t2"]; x.Status = StatusDraft; return x }()
	if rec := call(t, mux, http.MethodPatch, "/api/admin/toeic/tests/t2", "admin", `{"status": "published"}`); rec.Code != http.StatusOK {
		t.Fatalf("publish: %d %s", rec.Code, rec.Body)
	}
	if rec := call(t, mux, http.MethodPut, "/api/admin/toeic/tests/t2", "admin", `{"title": "x", "kind": "full", "groups": ["p5-a"]}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("edit published: %d %s", rec.Code, rec.Body)
	}
	if rec := call(t, mux, http.MethodGet, "/api/admin/toeic/tests", "member", ""); rec.Code != http.StatusForbidden {
		t.Fatalf("member list: %d", rec.Code)
	}
}
