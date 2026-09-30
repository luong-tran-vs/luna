package vocab

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestSaveBulk(t *testing.T) {
	t.Parallel()
	e := newEnv()
	if _, err := e.svc.Save(t.Context(), "u1", validInput()); err != nil { // "go" is already saved
		t.Fatal(err)
	}

	res, err := e.svc.SaveBulk(t.Context(), "u1", "lesson1", []string{"go", " Give  Up ", "unknown", "give up"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Added != 1 || len(res.Cards) != 1 {
		t.Fatalf("result = %+v", res)
	}
	c := res.Cards[0]
	if c.Text != "gave up" || c.Lemma != "give up" || c.MeaningVi != "từ bỏ" || c.LessonID != "lesson1" || c.Source != SourceAI ||
		c.ContextSentence != "He gave up smoking." || c.UserID != "u1" ||
		!c.Schedule.Due.Equal(time.Date(2026, 9, 30, 0, 0, 0, 0, hcm)) || c.Schedule.State != StateNew {
		t.Fatalf("card = %+v", c)
	}

	// Saving everything again adds nothing.
	again, err := e.svc.SaveBulk(t.Context(), "u1", "lesson1", []string{"go", "give up"})
	if err != nil || again.Added != 0 || len(again.Cards) != 0 {
		t.Fatalf("again = %+v, %v", again, err)
	}
	if words, _ := e.svc.Words(t.Context(), "u1"); len(words) != 2 {
		t.Fatalf("words = %+v", words)
	}

	// Another learner gets their own cards.
	other, _ := e.svc.SaveBulk(t.Context(), "u2", "lesson1", []string{"go", "give up"})
	if other.Added != 2 {
		t.Fatalf("other = %+v", other)
	}
}

func TestSaveBulkErrors(t *testing.T) {
	t.Parallel()
	e := newEnv()
	var verr *ValidationError
	if _, err := e.svc.SaveBulk(t.Context(), "u1", "lesson1", nil); !errors.As(err, &verr) || verr.Fields["lemmas"] == "" {
		t.Fatalf("empty: %v", err)
	}
	many := make([]string, 201)
	for i := range many {
		many[i] = "w"
	}
	if _, err := e.svc.SaveBulk(t.Context(), "u1", "lesson1", many); !errors.As(err, &verr) {
		t.Fatalf("too many: %v", err)
	}
	if _, err := e.svc.SaveBulk(t.Context(), "u1", "nope", []string{"go"}); !errors.Is(err, ErrLessonNotFound) {
		t.Fatalf("unknown lesson: %v", err)
	}
	// A lesson without annotations yet adds nothing.
	if res, err := e.svc.SaveBulk(t.Context(), "u1", "lesson2", []string{"go"}); err != nil || res.Added != 0 {
		t.Fatalf("no annotations: %+v, %v", res, err)
	}
}

func TestBulkEndpoint(t *testing.T) {
	t.Parallel()
	mux, _ := newEnvAPI(t)
	body := `{"lessonId":"lesson1","lemmas":["go","give up"]}`
	rec := call(t, mux, http.MethodPost, "/api/vocab/cards/bulk", "an", body)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"added":2`) || !strings.Contains(rec.Body.String(), `"lemma":"give up"`) {
		t.Fatalf("bulk: %d %s", rec.Code, rec.Body)
	}
	if rec := call(t, mux, http.MethodPost, "/api/vocab/cards/bulk", "an", body); !strings.Contains(rec.Body.String(), `{"added":0,"cards":[]}`) {
		t.Fatalf("again: %s", rec.Body)
	}
	for b, code := range map[string]int{
		`{"lessonId":"nope","lemmas":["go"]}`:          http.StatusNotFound,
		`{"lessonId":"lesson1","lemmas":[]}`:           http.StatusBadRequest,
		`{"lessonId":"lesson1","lemmas":["go"],"x":1}`: http.StatusBadRequest,
	} {
		if rec := call(t, mux, http.MethodPost, "/api/vocab/cards/bulk", "an", b); rec.Code != code {
			t.Fatalf("%s: %d %s", b, rec.Code, rec.Body)
		}
	}
	if rec := call(t, mux, http.MethodPost, "/api/vocab/cards/bulk", "", body); rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous: %d", rec.Code)
	}
}
