package lesson

import (
	"errors"
	"testing"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/job"
)

// --- F15: extras written with the annotations ---

var parkQuestion = ai.Question{
	Prompt: "Where did we go?", Options: []string{"To the park", "Home", "To school", "To work"}, AnswerIndex: 0,
	ExplanationVi: "Câu 1: We went to the park.",
}

func TestProcessAnnotateSavesExtras(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.create(t)
	e.ai.result = []ai.Annotation{{Text: "gave up", Lemma: "give up", MeaningVi: "bỏ", SentenceIndex: 1}}
	e.ai.extras = ai.LessonExtras{
		Questions:     []ai.Question{parkQuestion, {Prompt: "Broken", Options: []string{"a", "b"}}},
		GrammarNote:   &ai.GrammarNote{Title: "Quá khứ đơn", BodyVi: "Việc đã xong.", Examples: []string{"We went to the park.", "Not in the text."}},
		WritingPrompt: "Write about your weekend.",
	}

	if err := e.svc.ProcessAnnotate(t.Context(), jobFor(l, job.TypeAnnotate)); err != nil {
		t.Fatal(err)
	}
	got, _ := e.lessons.Get(t.Context(), l.ID)
	if e.ai.calls != 1 {
		t.Fatalf("AI calls = %d, want 1", e.ai.calls)
	}
	if len(got.Annotations) != 1 || len(got.Extras.Questions) != 1 || got.Extras.Questions[0].Prompt != "Where did we go?" {
		t.Fatalf("lesson = %+v", got)
	}
	if got.Extras.GrammarNote == nil || len(got.Extras.GrammarNote.Examples) != 1 || got.Extras.WritingPrompt == "" {
		t.Fatalf("extras = %+v", got.Extras)
	}
	if got.QuizVersion != l.QuizVersion+1 || got.ExtrasEditedByAdmin {
		t.Fatalf("quizVersion %d edited %v", got.QuizVersion, got.ExtrasEditedByAdmin)
	}
}

func TestProcessAnnotateBrokenExtrasKeepAnnotations(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.create(t)
	e.ai.result = []ai.Annotation{{Text: "gave up", Lemma: "give up", MeaningVi: "bỏ", SentenceIndex: 1}}
	e.ai.extras = ai.LessonExtras{
		Questions:   []ai.Question{{Prompt: "?", Options: []string{"a", "a", "b", "c"}, AnswerIndex: 9}},
		GrammarNote: &ai.GrammarNote{Title: "x", BodyVi: "y", Examples: []string{"nowhere"}},
	}
	if err := e.svc.ProcessAnnotate(t.Context(), jobFor(l, job.TypeAnnotate)); err != nil {
		t.Fatalf("broken extras failed the job: %v", err)
	}
	got, _ := e.lessons.Get(t.Context(), l.ID)
	if got.AnnotationStatus != StatusDone || len(got.Annotations) != 1 || len(got.Extras.Questions) != 0 ||
		got.Extras.GrammarNote != nil {
		t.Fatalf("lesson = %+v", got)
	}
}

func TestRetryAnnotateWhenDone(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.create(t)
	e.ai.result = []ai.Annotation{{Text: "gave up", Lemma: "give up", MeaningVi: "bỏ", SentenceIndex: 1}}

	// Running: cannot retry yet.
	if _, err := e.svc.Retry(t.Context(), l.ID, job.TypeAnnotate); !errors.Is(err, ErrAnnotationRunning) {
		t.Fatalf("retry while running: %v", err)
	}
	if err := e.svc.ProcessAnnotate(t.Context(), jobFor(l, job.TypeAnnotate)); err != nil {
		t.Fatal(err)
	}
	before := len(e.jobs.all())
	got, err := e.svc.Retry(t.Context(), l.ID, job.TypeAnnotate)
	if err != nil {
		t.Fatalf("retry when done: %v", err)
	}
	if got.AnnotationStatus != StatusRunning || len(e.jobs.all()) != before+1 {
		t.Fatalf("status %s, jobs %d → %d", got.AnnotationStatus, before, len(e.jobs.all()))
	}
}

func TestUpdateClearsExtras(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.create(t)
	e.ai.result = []ai.Annotation{{Text: "gave up", Lemma: "give up", MeaningVi: "bỏ", SentenceIndex: 1}}
	e.ai.extras = ai.LessonExtras{Questions: []ai.Question{parkQuestion}}
	_ = e.svc.ProcessAnnotate(t.Context(), jobFor(l, job.TypeAnnotate))
	done, _ := e.lessons.Get(t.Context(), l.ID)

	next, err := e.svc.Update(t.Context(), l.ID, Input{Title: "Park", Content: "New text here.", TopicID: "topic-b1", Source: "s", License: "l"})
	if err != nil {
		t.Fatal(err)
	}
	if len(next.Extras.Questions) != 0 || next.QuizVersion != done.QuizVersion+1 {
		t.Fatalf("after content change: questions %d, quizVersion %d (was %d)", len(next.Extras.Questions), next.QuizVersion, done.QuizVersion)
	}
}
