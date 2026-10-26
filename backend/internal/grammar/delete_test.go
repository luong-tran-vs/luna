package grammar

import (
	"errors"
	"net/http"
	"testing"
)

func TestDelete(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	if _, _, err := e.svc.Import(t.Context(), point, validContent(), false, false); err != nil {
		t.Fatal(err)
	}
	// A published lesson is unpublished first.
	if _, err := e.svc.Publish(t.Context(), point, false); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.Delete(t.Context(), point); !errors.Is(err, ErrPublished) {
		t.Fatalf("published: %v", err)
	}
	if _, err := e.svc.Unpublish(t.Context(), point); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.Delete(t.Context(), point); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.AdminGet(t.Context(), point); !errors.Is(err, ErrNotFound) {
		t.Errorf("still there: %v", err)
	}
	if err := e.svc.Delete(t.Context(), point); !errors.Is(err, ErrNotFound) {
		t.Errorf("twice: %v", err)
	}
}

func TestDeleteHandler(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	a.generate(t)
	if rec := a.do(t, http.MethodDelete, adminPath, "learner", ""); rec.Code != http.StatusForbidden {
		t.Errorf("learner: %d", rec.Code)
	}
	if rec := a.do(t, http.MethodPost, adminPath+"/publish", "admin", ""); rec.Code != http.StatusOK {
		t.Fatalf("publish: %d %s", rec.Code, rec.Body)
	}
	if rec := a.do(t, http.MethodDelete, adminPath, "admin", ""); rec.Code != http.StatusConflict ||
		decode(t, rec)["error"] != "grammar_lesson_published" {
		t.Fatalf("published: %d %s", rec.Code, rec.Body)
	}
	if rec := a.do(t, http.MethodPost, adminPath+"/unpublish", "admin", ""); rec.Code != http.StatusOK {
		t.Fatalf("unpublish: %d %s", rec.Code, rec.Body)
	}
	if rec := a.do(t, http.MethodDelete, adminPath, "admin", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: %d %s", rec.Code, rec.Body)
	}
	if rec := a.do(t, http.MethodDelete, adminPath, "admin", ""); rec.Code != http.StatusNotFound {
		t.Errorf("twice: %d", rec.Code)
	}
}
