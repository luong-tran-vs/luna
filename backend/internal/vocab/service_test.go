package vocab

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
)

func validInput() Input {
	return Input{
		Text: "went", Lemma: "  Go ", IPA: "/ɡəʊ/", MeaningVi: "đã đi",
		ContextSentence: "We went to the park.", LessonID: "lesson1", Source: "ai",
	}
}

func TestSave(t *testing.T) {
	t.Parallel()
	svc, repo := newTestService()

	c, err := svc.Save(t.Context(), "u1", validInput())
	if err != nil {
		t.Fatal(err)
	}
	if c.Lemma != "go" || c.UserID != "u1" || c.Source != SourceAI || c.CreatedAt.IsZero() || c.ContextSentence != "We went to the park." {
		t.Fatalf("card = %+v", c)
	}
	if len(repo.cards) != 1 {
		t.Fatalf("stored %d cards", len(repo.cards))
	}
}

func TestSaveDuplicate(t *testing.T) {
	t.Parallel()
	svc, repo := newTestService()
	first, _ := svc.Save(t.Context(), "u1", validInput())

	in := validInput()
	in.Text, in.Lemma = "GO", "go  "
	_, err := svc.Save(t.Context(), "u1", in)
	var exists *ExistsError
	if !errors.As(err, &exists) || exists.Card.ID != first.ID {
		t.Fatalf("err = %v, want ExistsError with the first card", err)
	}
	if len(repo.cards) != 1 {
		t.Fatalf("duplicate stored: %d cards", len(repo.cards))
	}

	// Another learner has a separate notebook.
	if _, err := svc.Save(t.Context(), "u2", validInput()); err != nil {
		t.Fatalf("other user: %v", err)
	}
}

func TestSaveCollapsesSpacesInPhrases(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService()
	in := validInput()
	in.Text, in.Lemma = "gave  up", " Give   Up "
	c, err := svc.Save(t.Context(), "u1", in)
	if err != nil || c.Lemma != "give up" {
		t.Fatalf("card = %+v, %v", c, err)
	}
}

func TestSaveValidation(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService()

	tests := []struct {
		name   string
		mutate func(*Input)
		field  string
	}{
		{"empty text", func(in *Input) { in.Text = " " }, "text"},
		{"long lemma", func(in *Input) { in.Lemma = strings.Repeat("a", 101) }, "lemma"},
		{"empty meaning", func(in *Input) { in.MeaningVi = "" }, "meaningVi"},
		{"long meaning", func(in *Input) { in.MeaningVi = strings.Repeat("ă", 201) }, "meaningVi"},
		{"long sentence", func(in *Input) { in.ContextSentence = strings.Repeat("a", 1001) }, "contextSentence"},
		{"bad source", func(in *Input) { in.Source = "gpt" }, "source"},
		{"unknown lesson", func(in *Input) { in.LessonID = "nope" }, "lessonId"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			in := validInput()
			tt.mutate(&in)
			_, err := svc.Save(t.Context(), "u1", in)
			var verr *ValidationError
			if !errors.As(err, &verr) || verr.Fields[tt.field] == "" {
				t.Fatalf("err = %v, want field %s", err, tt.field)
			}
		})
	}
}

func TestWords(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService()
	_, _ = svc.Save(t.Context(), "u1", validInput())
	in := validInput()
	in.Text, in.Lemma = "park", "park"
	_, _ = svc.Save(t.Context(), "u2", in)

	words, err := svc.Words(t.Context(), "u1")
	if err != nil || !slices.Equal(words, []WordRef{{Lemma: "go", Text: "went"}}) {
		t.Fatalf("words = %+v, %v", words, err)
	}
}

func TestSaveSetsFirstDue(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		now  time.Time
		user string
		want time.Time
	}{
		// 23:59 and 00:01 in Viet Nam (UTC+7).
		{"23:59 local", time.Date(2026, 9, 29, 16, 59, 0, 0, time.UTC), "u1", time.Date(2026, 9, 30, 0, 0, 0, 0, hcm)},
		{"00:01 local", time.Date(2026, 9, 29, 17, 1, 0, 0, time.UTC), "u1", time.Date(2026, 10, 1, 0, 0, 0, 0, hcm)},
		{"utc learner", time.Date(2026, 9, 29, 17, 1, 0, 0, time.UTC), "u2", time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e := newEnv()
			e.clock.set(tc.now)
			c, err := e.svc.Save(t.Context(), tc.user, validInput())
			if err != nil {
				t.Fatal(err)
			}
			if !c.Schedule.Due.Equal(tc.want) || c.Schedule.State != StateNew || c.Schedule.Reps != 0 {
				t.Fatalf("schedule = %+v, want due %v", c.Schedule, tc.want)
			}
		})
	}
}

func TestSaveManualCard(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService()
	in := Input{Text: "Serendipity", Lemma: "serendipity", MeaningVi: "sự tình cờ may mắn", Source: "manual"}
	c, err := svc.Save(t.Context(), "u1", in)
	if err != nil || c.LessonID != "" || c.Source != SourceManual || c.Schedule.Due.IsZero() {
		t.Fatalf("manual card = %+v, %v", c, err)
	}

	// Only manual cards may have no lesson.
	in.Lemma, in.Source = "other", "ai"
	_, err = svc.Save(t.Context(), "u1", in)
	var verr *ValidationError
	if !errors.As(err, &verr) || verr.Fields["lessonId"] == "" {
		t.Fatalf("ai card without lesson: %v", err)
	}
}
