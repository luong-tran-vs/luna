package grammar

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestImport(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c := validContent()

	// A dry run checks the lesson and saves nothing.
	l, replaced, err := e.svc.Import(t.Context(), point, c, false, true)
	if err != nil || replaced || l.Status != StatusDraft || !l.Edited || !l.Imported {
		t.Fatalf("dry run: %+v %v %v", l, replaced, err)
	}
	if _, err := e.svc.AdminGet(t.Context(), point); !errors.Is(err, ErrNotFound) {
		t.Fatalf("dry run saved: %v", err)
	}

	// Saved as a draft, marked imported, with no AI call.
	if _, _, err := e.svc.Import(t.Context(), point, c, false, false); err != nil {
		t.Fatal(err)
	}
	stored, err := e.svc.AdminGet(t.Context(), point)
	if err != nil || stored.Status != StatusDraft || !stored.Edited || !stored.Imported || len(stored.Content.Practice) != len(c.Practice) {
		t.Fatalf("stored: %+v %v", stored, err)
	}
	if e.ai.calls != 0 {
		t.Errorf("AI calls = %d", e.ai.calls)
	}

	// A point with a lesson is replaced only when asked; a published lesson goes back to draft.
	if _, err := e.svc.Publish(t.Context(), point, false); err != nil {
		t.Fatal(err)
	}
	if _, _, err := e.svc.Import(t.Context(), point, c, false, false); !errors.Is(err, ErrHasLesson) {
		t.Fatalf("without replace: %v", err)
	}
	l, replaced, err = e.svc.Import(t.Context(), point, c, true, false)
	if err != nil || !replaced || l.Status != StatusDraft || !l.CreatedAt.Equal(stored.CreatedAt) {
		t.Fatalf("replace: %+v %v %v", l, replaced, err)
	}

	if _, _, err := e.svc.Import(t.Context(), "nope", c, false, true); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown point: %v", err)
	}
	c.Objective = ""
	var verr *ValidationError
	if _, _, err := e.svc.Import(t.Context(), point, c, true, true); !errors.As(err, &verr) {
		t.Errorf("invalid: %v", err)
	}
}

func TestImportHandler(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	content := toContentJSON(validContent())
	body, _ := json.Marshal(map[string]any{"content": content})

	rec := a.do(t, http.MethodPost, adminPath+"/import?dryRun=true", "admin", string(body))
	if got := decode(t, rec); rec.Code != http.StatusOK || got["dryRun"] != true || got["replaced"] != false {
		t.Fatalf("dry run: %d %s", rec.Code, rec.Body)
	}
	if rec = a.do(t, http.MethodPost, adminPath+"/import", "admin", string(body)); rec.Code != http.StatusCreated ||
		decode(t, rec)["lesson"].(map[string]any)["imported"] != true {
		t.Fatalf("import: %d %s", rec.Code, rec.Body)
	}
	if rec = a.do(t, http.MethodPost, adminPath+"/import", "admin", string(body)); rec.Code != http.StatusConflict ||
		decode(t, rec)["error"] != "grammar_lesson_exists" {
		t.Fatalf("exists: %d %s", rec.Code, rec.Body)
	}
	body, _ = json.Marshal(map[string]any{"content": content, "replace": true})
	if rec = a.do(t, http.MethodPost, adminPath+"/import", "admin", string(body)); rec.Code != http.StatusCreated ||
		decode(t, rec)["replaced"] != true {
		t.Fatalf("replace: %d %s", rec.Code, rec.Body)
	}
	rec = a.do(t, http.MethodPost, adminPath+"/import", "admin", `{"content":{},"extra":1}`)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "extra") {
		t.Fatalf("unknown field: %d %s", rec.Code, rec.Body)
	}
	if rec = a.do(t, http.MethodPost, adminPath+"/import", "learner", string(body)); rec.Code != http.StatusForbidden {
		t.Errorf("learner: %d", rec.Code)
	}
}
