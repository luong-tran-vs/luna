package vocab

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestPracticeMisses(t *testing.T) {
	t.Parallel()
	e := newEnv()
	now := e.clock.Now()

	// "go" is saved and not due for a week: it is brought forward, FSRS state kept.
	saved := e.repo.put(Card{
		UserID: "u1", Text: "went", Lemma: "go", MeaningVi: "đi", LessonID: "lesson1", Source: SourceAI,
		CreatedAt: now.AddDate(0, 0, -10),
		Schedule:  Schedule{Due: now.AddDate(0, 0, 7), Stability: 12.5, Reps: 3, State: StateReview},
	})

	res, err := e.svc.PracticeMisses(t.Context(), "u1", "lesson1", []string{"went", " Give  Up ", "unknown", "give up"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Added != 1 || res.Rescheduled != 1 {
		t.Fatalf("result = %+v", res)
	}

	c, _ := e.repo.Get(t.Context(), "u1", saved.ID)
	if !c.Schedule.Due.Equal(now) || c.Schedule.Stability != 12.5 || c.Schedule.Reps != 3 || c.Schedule.State != StateReview {
		t.Fatalf("rescheduled card = %+v", c.Schedule)
	}
	n, err := e.repo.FindByLemma(t.Context(), "u1", "give up")
	if err != nil {
		t.Fatal(err)
	}
	if n.Text != "gave up" || n.MeaningVi != "từ bỏ" || n.ContextSentence != "He gave up smoking." || n.Source != SourceAI ||
		!n.Schedule.Due.Equal(now) || n.Schedule.State != StateNew {
		t.Fatalf("new card = %+v", n)
	}
	if list, _ := e.svc.Due(t.Context(), "u1", 10); list.Total != 2 {
		t.Fatalf("due = %d, want both cards due now", list.Total)
	}

	// Again: nothing changes, since the cards are already due.
	again, err := e.svc.PracticeMisses(t.Context(), "u1", "lesson1", []string{"go", "give up"})
	if err != nil || again.Added != 0 || again.Rescheduled != 0 {
		t.Fatalf("again = %+v, %v", again, err)
	}
	// A card due earlier than now keeps its due time.
	e.clock.t = now.Add(time.Hour)
	if r, _ := e.svc.PracticeMisses(t.Context(), "u1", "lesson1", []string{"go"}); r.Rescheduled != 0 {
		t.Fatalf("overdue card moved: %+v", r)
	}
}

func TestPracticeMissesErrors(t *testing.T) {
	t.Parallel()
	e := newEnv()
	var verr *ValidationError
	if _, err := e.svc.PracticeMisses(t.Context(), "u1", "lesson1", nil); !errors.As(err, &verr) || verr.Fields["words"] == "" {
		t.Fatalf("empty: %v", err)
	}
	if _, err := e.svc.PracticeMisses(t.Context(), "u1", "lesson1", make([]string, 31)); !errors.As(err, &verr) {
		t.Fatalf("too many: %v", err)
	}
	if _, err := e.svc.PracticeMisses(t.Context(), "u1", "nope", []string{"go"}); !errors.Is(err, ErrLessonNotFound) {
		t.Fatalf("unknown lesson: %v", err)
	}
}

func TestPracticeMissesEndpoint(t *testing.T) {
	t.Parallel()
	mux, _ := newEnvAPI(t)
	body := `{"lessonId":"lesson1","words":["go","give up"]}`
	rec := call(t, mux, http.MethodPost, "/api/vocab/practice-misses", "an", body)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `{"added":2,"rescheduled":0}`) {
		t.Fatalf("misses: %d %s", rec.Code, rec.Body)
	}
	for b, code := range map[string]int{
		`{"lessonId":"nope","words":["go"]}`:          http.StatusNotFound,
		`{"lessonId":"lesson1","words":[]}`:           http.StatusBadRequest,
		`{"lessonId":"lesson1","words":["go"],"x":1}`: http.StatusBadRequest,
	} {
		if rec := call(t, mux, http.MethodPost, "/api/vocab/practice-misses", "an", b); rec.Code != code {
			t.Fatalf("%s: %d %s", b, rec.Code, rec.Body)
		}
	}
	if rec := call(t, mux, http.MethodPost, "/api/vocab/practice-misses", "", body); rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous: %d", rec.Code)
	}
}
