package vocab

import (
	"context"
	"errors"
	"testing"
	"time"
)

// fakeTranslations are the sentence translations of each lesson, keyed by SentenceKey.
type fakeTranslations map[string]map[string]string

func (f fakeTranslations) Translations(_ context.Context, lessonID string) (map[string]string, error) {
	if lessonID == "broken" {
		return nil, errors.New("down")
	}
	return f[lessonID], nil
}

func TestSentenceKey(t *testing.T) {
	t.Parallel()
	if got := SentenceKey("  We  went\tto the PARK. "); got != "we went to the park." {
		t.Fatalf("SentenceKey = %q", got)
	}
}

func TestSaveFillsTheLessonTranslationOfTheExample(t *testing.T) {
	t.Parallel()
	svc, repo := newTestService()
	in := validInput()
	in.ContextSentence = "We  went to the PARK."
	c, err := svc.Save(t.Context(), "u1", in)
	if err != nil {
		t.Fatal(err)
	}
	if c.ContextSentenceVi != "Chúng tôi đã đi công viên." || repo.cards[0].ContextSentenceVi != c.ContextSentenceVi {
		t.Fatalf("card = %+v", c)
	}
}

func TestSaveKeepsAGivenTranslationAndLeavesUnknownOnesEmpty(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService()
	in := validInput()
	in.ContextSentenceVi = "  Tự dịch.  "
	c, err := svc.Save(t.Context(), "u1", in)
	if err != nil || c.ContextSentenceVi != "Tự dịch." {
		t.Fatalf("card = %+v, err = %v", c, err)
	}

	in = validInput()
	in.Text, in.Lemma, in.ContextSentence = "park", "park", "A sentence the lesson does not have."
	c, err = svc.Save(t.Context(), "u1", in)
	if err != nil || c.ContextSentenceVi != "" {
		t.Fatalf("card = %+v, err = %v", c, err)
	}
}

func TestSaveValidatesTheTranslationLength(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService()
	in := validInput()
	in.ContextSentenceVi = string(make([]byte, 1001))
	var verr *ValidationError
	if _, err := svc.Save(t.Context(), "u1", in); !errors.As(err, &verr) || verr.Fields["contextSentenceVi"] == "" {
		t.Fatalf("err = %v", err)
	}
}

// Cards saved before translations existed get them when read, and keep them.
func TestListAndDueFillMissingTranslations(t *testing.T) {
	t.Parallel()
	e := newEnv()
	old := Card{
		UserID: "u1", Text: "went", Lemma: "go", MeaningVi: "đi", ContextSentence: "We went to the park.",
		LessonID: "lesson1", Source: SourceAI, CreatedAt: e.clock.Now().Add(-48 * time.Hour),
		Schedule: Schedule{Due: e.clock.Now().Add(-time.Hour), State: StateReview, Reps: 1},
	}
	broken := old
	broken.Text, broken.Lemma, broken.LessonID = "run", "run", "broken"
	for _, c := range []Card{old, broken} {
		if _, err := e.repo.Create(t.Context(), c); err != nil {
			t.Fatal(err)
		}
	}

	page, err := e.svc.List(t.Context(), "u1", ListQuery{Page: 1})
	if err != nil {
		t.Fatal(err)
	}
	vi := map[string]string{}
	for _, c := range page.Cards {
		vi[c.Lemma] = c.ContextSentenceVi
	}
	if vi["go"] != "Chúng tôi đã đi công viên." || vi["run"] != "" {
		t.Fatalf("translations = %v", vi)
	}
	stored, _ := e.repo.FindByLemma(t.Context(), "u1", "go")
	if stored.ContextSentenceVi == "" {
		t.Fatal("the translation found was not stored")
	}

	due, err := e.svc.Due(t.Context(), "u1", 10)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range due.Cards {
		if c.Lemma == "go" && c.ContextSentenceVi == "" {
			t.Fatalf("due card = %+v", c)
		}
	}
}

func TestUpdateSetsTheTranslation(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService()
	c, err := svc.Save(t.Context(), "u1", validInput())
	if err != nil {
		t.Fatal(err)
	}
	vi := "  Bản dịch mới. "
	got, err := svc.Update(t.Context(), "u1", c.ID, Details{ContextSentenceVi: &vi})
	if err != nil || got.ContextSentenceVi != "Bản dịch mới." {
		t.Fatalf("card = %+v, err = %v", got, err)
	}
}
