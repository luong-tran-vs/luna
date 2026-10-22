package lesson

import (
	"slices"
	"testing"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/job"
)

func TestProcessPracticeGrammarChoices(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.annotated(t)
	if _, err := e.lessons.update(l.ID, func(x *Lesson) bool { x.TargetWords = []string{"go", "sunny"}; return true }); err != nil {
		t.Fatal(err)
	}
	e.ai.practice = samplePractice()
	if err := e.svc.ProcessPractice(t.Context(), jobFor(l, job.TypePractice)); err != nil {
		t.Fatal(err)
	}
	_, req := e.ai.practices()
	if req.GrammarFocus != nil || len(req.GrammarChoices) == 0 {
		t.Fatalf("focus %+v choices %+v", req.GrammarFocus, req.GrammarChoices)
	}
	for _, c := range req.GrammarChoices {
		if c.ID[:3] != "b1-" || c.TitleEn == "" {
			t.Fatalf("choice %+v not of B1", c)
		}
	}
	if !slices.Equal(req.TargetWords, []string{"go", "sunny"}) {
		t.Fatalf("target words = %q", req.TargetWords)
	}
}

func TestProcessPracticeGrammarFocus(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.create(t, func(in *Input) { in.GrammarPointID = "b1-present-perfect" })
	e.ai.result = sampleAnnotations
	if err := e.svc.ProcessAnnotate(t.Context(), jobFor(l, job.TypeAnnotate)); err != nil {
		t.Fatal(err)
	}
	e.ai.practice = samplePractice()
	if err := e.svc.ProcessPractice(t.Context(), jobFor(l, job.TypePractice)); err != nil {
		t.Fatal(err)
	}
	_, req := e.ai.practices()
	if req.GrammarFocus == nil || req.GrammarChoices != nil {
		t.Fatalf("focus %+v choices %+v", req.GrammarFocus, req.GrammarChoices)
	}
}

func TestProcessPracticeSavesSteps(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.annotated(t)
	p := samplePractice()
	p.ListeningQuestions = []ai.ListenQuestion{
		{Kind: "tfn", Prompt: "It was sunny.", AnswerIndex: 0, ExplanationVi: "Lượt 4.", TurnIndex: 3},
	}
	p.SpeakQuestions = []ai.SpeakQuestion{{Question: "Do you like parks?"}}
	p.GrammarPointID = "b1-present-perfect"
	e.ai.practice = p
	if err := e.svc.ProcessPractice(t.Context(), jobFor(l, job.TypePractice)); err != nil {
		t.Fatal(err)
	}
	got, _ := e.lessons.Get(t.Context(), l.ID)
	s := got.Practice.Steps
	if s == nil || len(s.ListeningQuestions) != 1 || s.ListeningQuestions[0].TurnIndex != 3 ||
		len(s.SpeakQuestions) != 1 || s.GrammarPointID != "b1-present-perfect" {
		t.Fatalf("steps = %+v", s)
	}
	// The picked point is not written to the lesson itself.
	if got.GrammarPointID != "" {
		t.Fatalf("lesson point = %q", got.GrammarPointID)
	}
}

func TestQueueMissingSteps(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	old := e.withPractice(t)
	// A practice made before the six steps has no Steps.
	if _, err := e.lessons.update(old.ID, func(l *Lesson) bool { l.Practice.Steps = nil; return true }); err != nil {
		t.Fatal(err)
	}
	e.withPractice(t) // has Steps already
	failed := e.withPractice(t)
	running := e.withPractice(t)
	for id, st := range map[string]Status{failed.ID: StatusFailed, running.ID: StatusRunning} {
		if _, err := e.lessons.update(id, func(l *Lesson) bool { l.Practice.Steps, l.PracticeStatus = nil, st; return true }); err != nil {
			t.Fatal(err)
		}
	}
	bare := e.withPractice(t) // no annotations
	if _, err := e.lessons.update(bare.ID, func(l *Lesson) bool { l.Practice.Steps, l.Annotations = nil, nil; return true }); err != nil {
		t.Fatal(err)
	}
	e.create(t) // no practice
	before := len(jobsOf(e, job.TypePractice))

	n, err := e.svc.QueueMissingSteps(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	jobs := jobsOf(e, job.TypePractice)
	if n != 1 || len(jobs) != before+1 || jobs[len(jobs)-1].LessonID != old.ID {
		t.Fatalf("queued %d, jobs %+v", n, jobs)
	}
	got, _ := e.lessons.Get(t.Context(), old.ID)
	if got.PracticeStatus != StatusRunning || got.Practice == nil {
		t.Fatalf("status %q practice %+v", got.PracticeStatus, got.Practice)
	}

	// Queued once only: the status is no longer done.
	if n, err := e.svc.QueueMissingSteps(t.Context()); err != nil || n != 0 {
		t.Fatalf("second run: %d, %v", n, err)
	}
}
