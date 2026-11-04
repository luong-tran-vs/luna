package lesson

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/ai"
)

func importInput() Input {
	return Input{
		Title: "Park", Content: sampleContent, TopicID: "topic-b1", Level: "B1", Source: ImportSource, License: ImportLicense,
		AppendToRoadmap: true, Draft: true,
	}
}

func importContent() ImportContent {
	return ImportContent{
		Annotate: ai.LessonExtras{
			Annotations:   slices.Concat(sampleAnnotations, []ai.Annotation{{Text: "nowhere", Lemma: "nowhere", MeaningVi: "không đâu"}}),
			WritingPrompt: "Kể về một ngày đi chơi.",
		},
		Practice: samplePractice(),
		Glosses:  []ai.Gloss{{Text: "park", Lemma: "park", MeaningVi: "công viên", SentenceIndex: 0}},
	}
}

func part(r ImportReport, key string) ImportPart {
	for _, p := range r.Parts {
		if p.Key == key {
			return p
		}
	}
	return ImportPart{}
}

func TestImport(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	glosses := newFakeGlosses()
	e.svc.Glosses = glosses

	// A dry run checks everything and saves nothing.
	l, rep, err := e.svc.Import(t.Context(), importInput(), importContent(), true)
	if err != nil {
		t.Fatal(err)
	}
	if l.ID != "" || len(l.Annotations) != 3 || l.Practice == nil || rep.Sentences != 3 {
		t.Fatalf("dry run lesson %+v", l)
	}
	// The annotation whose text is not in the lesson is dropped, as from the AI.
	if p := part(rep, "annotate.annotations"); p.Given != 4 || p.Kept != 3 || p.RuleVi == "" ||
		!slices.Equal(p.Reasons, []string{`annotations[3] "nowhere": không có trong câu nào của bài`}) {
		t.Errorf("annotations part %+v", p)
	}
	// A part kept whole needs no rule nor reason.
	if p := part(rep, "practice.examples"); p.Given != 2 || p.Kept != 2 || p.RuleVi != "" || p.Reasons != nil {
		t.Errorf("examples part %+v", p)
	}
	if items, _ := e.lessons.List(t.Context(), Filter{}); len(items) != 0 {
		t.Fatalf("dry run saved %+v", items)
	}

	// Saved: done like an AI lesson, in its roadmap, and no AI job queued.
	l, _, err = e.svc.Import(t.Context(), importInput(), importContent(), false)
	if err != nil {
		t.Fatal(err)
	}
	if l.ID == "" || !l.Draft || l.AnnotationStatus != StatusDone || l.PracticeStatus != StatusDone ||
		len(l.Annotations) != 3 || l.Extras.WritingPrompt == "" || l.Practice == nil || len(l.Practice.Translations) != 1 {
		t.Fatalf("saved lesson %+v", l)
	}
	if jobs := e.jobs.all(); len(jobs) != 0 {
		t.Errorf("jobs %+v", jobs)
	}
	if r := e.topics.roadmap("topic-b1"); len(r) != 1 || r[0] != l.ID {
		t.Errorf("roadmap %v", r)
	}
	if rev, g, ok, _ := glosses.Get(t.Context(), l.ID); !ok || rev != l.Revision || len(g) != 1 {
		t.Errorf("glosses %d %+v %v", rev, g, ok)
	}
}

func TestImportInvalid(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c := importContent()
	c.Annotate.Annotations = []ai.Annotation{{Text: "nowhere", Lemma: "nowhere", MeaningVi: "x"}}
	c.Practice = ai.Practice{ObjectiveVi: "x"}
	_, rep, err := e.svc.Import(t.Context(), importInput(), c, false)
	var verr *ValidationError
	if !errors.As(err, &verr) || verr.Fields["annotate.annotations"] == "" || verr.Fields["practice"] == "" {
		t.Fatalf("err = %v", err)
	}
	if len(rep.Parts) == 0 {
		t.Error("no report with the errors")
	}
	if items, _ := e.lessons.List(t.Context(), Filter{}); len(items) != 0 {
		t.Fatalf("saved %+v", items)
	}

	in := importInput()
	in.Content = ""
	if _, _, err := e.svc.Import(t.Context(), in, importContent(), true); !errors.As(err, &verr) || verr.Fields["content"] == "" {
		t.Errorf("no content: %v", err)
	}
}

func TestImportExplains(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c := importContent()
	c.Annotate.Questions = []ai.Question{
		{Kind: "gist", Prompt: "What is it about?", Options: []string{"a", "b", "c", "d"}, AnswerIndex: 0, ExplanationVi: "x"},
		{Kind: "detail", Prompt: "Where?", Options: []string{"a", "b"}, AnswerIndex: 0, ExplanationVi: "x"},
		{Kind: "gist", Prompt: "What is it about?", Options: []string{"a", "b", "c", "d"}, AnswerIndex: 1, ExplanationVi: "x"},
	}
	c.Practice.GrammarStep = &ai.GrammarStep{TitleVi: "x", Exercises: []ai.GrammarExercise{
		{Kind: "choice", PromptVi: "Chọn", Text: "We ___ here.", Options: []string{"am", "is", "are"}, AnswerIndex: 2, ExplanationVi: "x"},
	}}
	_, rep, err := e.svc.Import(t.Context(), importInput(), c, true)
	if err != nil {
		t.Fatal(err)
	}
	if p := part(rep, "annotate.questions"); !slices.Equal(p.Reasons, []string{
		"questions[1]: options: Cần đúng 4 lựa chọn", "questions[2]: trùng câu hỏi",
	}) {
		t.Errorf("questions %+v", p)
	}
	if p := part(rep, "practice.grammarStep.exercises"); len(p.Reasons) != 1 || !strings.Contains(p.Reasons[0], "bài 1 (choice): options") {
		t.Errorf("exercises %+v", p)
	}
}

func TestReimport(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	glosses := newFakeGlosses()
	e.svc.Glosses = glosses
	l, _, err := e.svc.Import(t.Context(), importInput(), importContent(), false)
	if err != nil {
		t.Fatal(err)
	}

	// The lesson is rewritten in place: same ID and roadmap place, a new revision and quiz.
	in := importInput()
	in.Content = l.Content + " It was great."
	got, _, err := e.svc.Reimport(t.Context(), l.ID, in, importContent(), false)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != l.ID || got.Content != in.Content || got.Revision != l.Revision+1 || got.QuizVersion <= l.QuizVersion ||
		got.AnnotationStatus != StatusDone || got.PracticeStatus != StatusDone || len(got.Sentences) != len(l.Sentences)+1 ||
		got.Practice == nil || len(got.Annotations) != 3 || got.Draft != l.Draft {
		t.Fatalf("reimported %+v", got)
	}
	if jobs := e.jobs.all(); len(jobs) != 0 {
		t.Errorf("jobs %+v", jobs)
	}
	if r := e.topics.roadmap("topic-b1"); len(r) != 1 || r[0] != l.ID {
		t.Errorf("roadmap %v", r)
	}
	if rev, _, ok, _ := glosses.Get(t.Context(), l.ID); !ok || rev != got.Revision {
		t.Errorf("glosses at revision %d, want %d", rev, got.Revision)
	}

	// A dry run changes nothing.
	if _, _, err := e.svc.Reimport(t.Context(), l.ID, importInput(), importContent(), true); err != nil {
		t.Fatal(err)
	}
	if cur, _ := e.lessons.Get(t.Context(), l.ID); cur.Revision != got.Revision {
		t.Errorf("dry run saved revision %d", cur.Revision)
	}
	if _, _, err := e.svc.Reimport(t.Context(), "nope", importInput(), importContent(), false); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing lesson: %v", err)
	}
}
