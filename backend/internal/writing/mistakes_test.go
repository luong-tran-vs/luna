package writing

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
)

func TestCheckGradeMistakes(t *testing.T) {
	t.Parallel()
	g := goodGrade()
	g.Mistakes = []ai.GradeMistake{
		{Original: " have ", Corrected: "has", Type: " verb-s ", NoteVi: "He/she + has."},
		{Original: "people", Corrected: "people", Type: "plural"}, // unchanged
		{Original: " ", Corrected: "", Type: "article"},          // empty
		{Original: "", Corrected: "a", Type: "article"},          // a word added
		{Original: "go", Corrected: "went", Type: "grammar"},     // unknown type
		{Original: "famly", Corrected: "family", Type: "spelling"},
	}
	out, err := CheckGrade(g)
	if err != nil {
		t.Fatal(err)
	}
	want := []Mistake{
		{Original: "have", Corrected: "has", Type: MistakeVerbS, NoteVi: "He/she + has."},
		{Original: "", Corrected: "a", Type: MistakeArticle},
		{Original: "go", Corrected: "went", Type: MistakeOther},
		{Original: "famly", Corrected: "family", Type: MistakeSpelling},
	}
	if len(out.Mistakes) != len(want) {
		t.Fatalf("mistakes = %+v", out.Mistakes)
	}
	for i := range want {
		if out.Mistakes[i] != want[i] {
			t.Errorf("mistake %d = %+v, want %+v", i, out.Mistakes[i], want[i])
		}
	}

	// At most 20.
	g.Mistakes = nil
	for i := range 25 {
		g.Mistakes = append(g.Mistakes, ai.GradeMistake{Original: fmt.Sprint("a", i), Corrected: "b", Type: "tense"})
	}
	if out, _ := CheckGrade(g); len(out.Mistakes) != maxMistakes || out.Mistakes[19].Original != "a19" {
		t.Fatalf("capped mistakes = %d", len(out.Mistakes))
	}

	// No mistakes is fine.
	if out, err := CheckGrade(goodGrade()); err != nil || out.Mistakes != nil {
		t.Fatalf("none: %+v, %v", out.Mistakes, err)
	}
}

// addGraded stores a submitted writing of user with a grade, submitted at the given hour.
func (e *env) addGraded(user string, hour int, status GradeStatus, ms ...Mistake) {
	e.repo.mu.Lock()
	defer e.repo.mu.Unlock()
	e.repo.nextID++
	id := fmt.Sprint("w", e.repo.nextID)
	e.repo.byID[id] = Writing{
		ID: id, UserID: user, LessonID: "l" + id, Status: StatusSubmitted,
		SubmittedAt: e.now.Add(time.Duration(hour) * time.Hour),
		Grade:       &Grade{Status: status, Mistakes: ms},
	}
}

func tense(orig string) Mistake  { return Mistake{Original: orig, Corrected: "x", Type: MistakeTense} }
func article(orig string) Mistake { return Mistake{Original: orig, Corrected: "x", Type: MistakeArticle} }

func TestServiceMistakes(t *testing.T) {
	t.Parallel()
	e := newEnv()
	e.addGraded("u1", 1, GradeDone, tense("t1"), article("a1"), Mistake{Original: "o", Corrected: "x", Type: MistakeOther})
	e.addGraded("u1", 2, GradeDone, tense("t2"), tense("t3"), Mistake{Original: "s", Corrected: "x", Type: MistakeSpelling})
	e.addGraded("u1", 3, GradeDone, tense("t4"), article("a2"))
	e.addGraded("u1", 4, GradeFailed, tense("failed"))
	e.addGraded("u1", 5, GradePending, tense("pending"))
	e.addGraded("u2", 6, GradeDone, tense("other user"))

	got, err := e.svc.Mistakes(t.Context(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("stats = %+v", got)
	}
	if got[0].Type != MistakeTense || got[0].Count != 4 || got[1].Type != MistakeArticle || got[1].Count != 2 {
		t.Fatalf("stats = %+v", got)
	}
	// The latest writings' examples first, at most 3.
	var ex []string
	for _, m := range got[0].Examples {
		ex = append(ex, m.Original)
	}
	if strings.Join(ex, ",") != "t4,t2,t3" {
		t.Fatalf("tense examples = %q", ex)
	}
	if got[1].Examples[0].Original != "a2" {
		t.Fatalf("article examples = %+v", got[1].Examples)
	}

	// No graded writing: an empty list.
	if got, err := e.svc.Mistakes(t.Context(), "u3"); err != nil || got == nil || len(got) != 0 {
		t.Fatalf("none = %+v, %v", got, err)
	}
}

func TestMistakesEndpoint(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	a.env.addGraded("u1", 1, GradeDone, tense("go"), tense("eat"), article("a dog"))

	code, body := a.do(t, http.MethodGet, "/api/writings/mistakes", "u1", "")
	stats, _ := body["mistakes"].([]any)
	if code != http.StatusOK || len(stats) != 2 {
		t.Fatalf("u1: %d %v", code, body)
	}
	first := stats[0].(map[string]any)
	examples, _ := first["examples"].([]any)
	if first["type"] != MistakeTense || first["count"] != float64(2) || len(examples) != 2 ||
		examples[0].(map[string]any)["original"] != "go" {
		t.Fatalf("first = %v", first)
	}

	code, body = a.do(t, http.MethodGet, "/api/writings/mistakes", "u2", "")
	if stats, ok := body["mistakes"].([]any); code != http.StatusOK || !ok || len(stats) != 0 {
		t.Fatalf("u2: %d %v", code, body)
	}
	if code, _ := a.do(t, http.MethodGet, "/api/writings/mistakes", "", ""); code != http.StatusUnauthorized {
		t.Fatalf("anonymous: %d", code)
	}
}
