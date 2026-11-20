package toeic

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestDeleteGroupsKeepsTheOnesInUse(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := t.Context()
	e.seed(p1("p1-free"), p5("p5-free", 0), p5("p5-in-test", 1), p5("p5-done", 2))
	_ = e.images.Save(ctx, Image{GroupKey: "p1-free", Slot: SlotPhoto, MIME: "image/png", Data: pngOf(t, 4, 4)})
	e.tests.recs["mini-01"] = Test{Key: "mini-01", Kind: KindMini50, Status: StatusDraft, Groups: []string{"p5-in-test"}}
	e.attempts.recs = append(e.attempts.recs, Attempt{ID: "a1", UserID: "u1", Status: AttemptSubmitted, GroupKeys: []string{"p5-done"}})

	rep, err := e.svc.DeleteGroups(ctx, []string{"p1-free", " p5-free ", "p5-free", "p5-in-test", "p5-done", "nope"})
	if err != nil {
		t.Fatalf("delete = %v", err)
	}
	if strings.Join(rep.Deleted, ",") != "p1-free,p5-free" || strings.Join(rep.Missing, ",") != "nope" || len(rep.Blocked) != 2 {
		t.Fatalf("report = %+v", rep)
	}
	if !strings.Contains(rep.Blocked[0].Reason, "mini-01") || !strings.Contains(rep.Blocked[1].Reason, "1 bài làm") {
		t.Fatalf("blocked = %+v", rep.Blocked)
	}
	for _, k := range []string{"p1-free", "p5-free"} {
		if _, ok := e.groups.recs[k]; ok {
			t.Fatalf("%s still stored", k)
		}
	}
	for _, k := range []string{"p5-in-test", "p5-done"} {
		if _, ok := e.groups.recs[k]; !ok {
			t.Fatalf("%s was deleted", k)
		}
	}
	if _, err := e.images.Get(ctx, "p1-free", SlotPhoto); !errors.Is(err, ErrNotFound) {
		t.Fatal("the photo of a deleted group is still stored")
	}
}

func TestDeleteGroupsEndpoint(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.seed(p5("p5-x", 0))
	if rec := call(t, e.mux(), http.MethodPost, "/api/admin/toeic/groups/delete", "admin", `{"keys":[]}`); rec.Code != http.StatusBadRequest && rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("no keys = %d %s", rec.Code, rec.Body)
	}
	if rec := call(t, e.mux(), http.MethodPost, "/api/admin/toeic/groups/delete", "learner", `{"keys":["p5-x"]}`); rec.Code != http.StatusForbidden && rec.Code != http.StatusUnauthorized {
		t.Fatalf("not an admin = %d", rec.Code)
	}
	rec := call(t, e.mux(), http.MethodPost, "/api/admin/toeic/groups/delete", "admin", `{"keys":["p5-x"]}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"deleted":["p5-x"]`) {
		t.Fatalf("delete = %d %s", rec.Code, rec.Body)
	}
}
