package toeic

import (
	"errors"
	"testing"
)

func ptr[T any](v T) *T { return &v }

func startPractice(t *testing.T, e *env, user string, parts ...Part) AttemptView {
	t.Helper()
	v, err := e.svc.Start(t.Context(), user, false, StartReq{Source: Source{Kind: SourceTest, TestKey: "t1"}, Parts: parts, TimeLimitMin: 20})
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestOneAttemptInProgressPerTest(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.readingTest("t1")
	v := startPractice(t, e, "u1", 5)
	_, err := e.svc.Start(t.Context(), "u1", false, StartReq{Source: Source{Kind: SourceTest, TestKey: "t1"}, Parts: []Part{6}})
	var busy *ErrAttemptInProgress
	if !errors.As(err, &busy) || busy.ID != v.ID {
		t.Fatalf("second start = %v", err)
	}
	startPractice(t, e, "u4", 5) // another learner is not blocked
}

func TestSaveProgress(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := t.Context()
	e.readingTest("t1")
	v := startPractice(t, e, "u1", 5)
	q1, q2 := v.Paper[0].Questions[0].ID, v.Paper[1].Questions[0].ID
	if err := e.svc.SaveProgress(ctx, "u1", v.ID, Progress{Answers: map[string]int{q1: 2, q2: 1}, RemainingMs: ptr(int64(1000000)),
		Flags: []string{q2, "zzz"}, Position: ptr(q2)}); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.SaveProgress(ctx, "u1", v.ID, Progress{Answers: map[string]int{q2: -1}, RemainingMs: ptr(int64(1100000))}); err != nil {
		t.Fatal(err)
	}
	a, _ := e.attempts.Get(ctx, "u1", v.ID)
	if a.Answers[q1] != 2 || len(a.Answers) != 1 || a.RemainingMs != 1000000 || len(a.Flags) != 1 || a.Position != q2 {
		t.Fatalf("saved = %+v", a)
	}
	err := e.svc.SaveProgress(ctx, "u1", v.ID, Progress{Answers: map[string]int{"p7-nope#1": 0}})
	if fieldsOf(t, err)["answers.p7-nope#1"] == "" {
		t.Fatal("a question of another test was taken")
	}
	if err := e.svc.SaveProgress(ctx, "u4", v.ID, Progress{}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("another learner's attempt = %v", err)
	}
	if _, err := e.svc.Submit(ctx, "u1", v.ID, nil); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.SaveProgress(ctx, "u1", v.ID, Progress{}); !errors.Is(err, ErrAttemptSubmitted) {
		t.Fatalf("save after submit = %v", err)
	}
}

func TestSubmit(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := t.Context()
	e.readingTest("t1")
	v := startPractice(t, e, "u1", 5)
	// readingTest answers Part 5 question i with i%4.
	last := Progress{Answers: map[string]int{v.Paper[0].Questions[0].ID: 0, v.Paper[1].Questions[0].ID: 0}, ElapsedMs: ptr(int64(60000))}
	r, err := e.svc.Submit(ctx, "u1", v.ID, &last)
	if err != nil {
		t.Fatal(err)
	}
	if r.Correct != 1 || r.Wrong != 1 || r.Skipped != 28 || r.Total != nil || r.DurationMs != 60000 || r.Counted {
		t.Fatalf("result = %+v", r)
	}
	if e.days.count() != 1 {
		t.Fatal("study day not counted")
	}
	again, err := e.svc.Submit(ctx, "u1", v.ID, &Progress{Answers: map[string]int{v.Paper[1].Questions[0].ID: 1}})
	if err != nil || again.Correct != 1 || e.days.count() != 1 {
		t.Fatalf("second submit = %+v, %v", again, err)
	}
	if len(r.ByPart) != 1 || r.ByPart[0].Tags[0].Tag != "word-form" || r.ByPart[0].Tags[0].Right != 1 || r.Answers[1].Correct != "B" || r.Answers[1].Chosen != "A" {
		t.Fatalf("tables = %+v %+v", r.ByPart, r.Answers[:2])
	}
}

func TestOpenSubmitsWhenTimeIsUp(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := t.Context()
	e.readingTest("t1")
	v := startPractice(t, e, "u1", 6)
	if err := e.svc.SaveProgress(ctx, "u1", v.ID, Progress{RemainingMs: ptr(int64(0))}); err != nil {
		t.Fatal(err)
	}
	got, err := e.svc.Open(ctx, "u1", v.ID)
	if err != nil || got.Status != AttemptSubmitted {
		t.Fatalf("open = %+v, %v", got.Status, err)
	}
}

func TestResultAndDetailOnlyAfterSubmit(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := t.Context()
	e.readingTest("t1")
	v := startPractice(t, e, "u1", 7)
	qid := v.Paper[0].Questions[0].ID
	if _, err := e.svc.Result(ctx, "u1", v.ID); !errors.As(err, new(*ErrAttemptInProgress)) {
		t.Fatalf("result before submit = %v", err)
	}
	if _, err := e.svc.QuestionDetail(ctx, "u1", v.ID, qid); !errors.As(err, new(*ErrAttemptInProgress)) {
		t.Fatalf("detail before submit = %v", err)
	}
	if _, err := e.svc.Submit(ctx, "u1", v.ID, &Progress{Answers: map[string]int{qid: 0}}); err != nil {
		t.Fatal(err)
	}
	d, err := e.svc.QuestionDetail(ctx, "u1", v.ID, qid)
	if err != nil || d.Question.Answer != 1 || d.Question.Chosen != 0 || d.Question.State != OutcomeWrong || d.Question.ExplainVi == "" ||
		len(d.Question.Evidence) != 1 || d.Next == "" || d.Prev != "" || d.Question.Number != 147 {
		t.Fatalf("detail = %+v, %v", d, err)
	}
	if _, err := e.svc.QuestionDetail(ctx, "u1", v.ID, "p5-zz#1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("question of another attempt = %v", err)
	}
}

func TestLibrary(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := t.Context()
	e.readingTest("t1")
	lib, err := e.svc.Library(ctx, "u1", false)
	if err != nil || len(lib.Tests) != 1 || lib.Tests[0].State.Status != "new" || lib.Tests[0].Questions != 100 || lib.Tests[0].Parts != 3 {
		t.Fatalf("library = %+v, %v", lib, err)
	}
	v := startPractice(t, e, "u1", 5)
	_ = e.svc.SaveProgress(ctx, "u1", v.ID, Progress{Position: ptr(v.Paper[3].Questions[0].ID)})
	lib, _ = e.svc.Library(ctx, "u1", false)
	if st := lib.Tests[0].State; st.Status != "in_progress" || st.Position != 104 || st.AttemptID != v.ID {
		t.Fatalf("in progress = %+v", st)
	}
	_, _ = e.svc.Submit(ctx, "u1", v.ID, nil)
	lib, _ = e.svc.Library(ctx, "u1", false)
	if st := lib.Tests[0].State; st.Status != "submitted" || st.Correct == nil || *st.Correct != 0 || len(lib.Recent) != 1 || lib.Latest != nil {
		t.Fatalf("submitted = %+v", lib)
	}
	if *lib.Mistakes != 30 {
		t.Fatalf("mistakes = %d", *lib.Mistakes)
	}
	guest, _ := e.svc.Library(ctx, "u2", true)
	if !guest.Tests[0].Locked || guest.Mistakes != nil {
		t.Fatalf("guest = %+v", guest.Tests[0])
	}
	d, err := e.svc.TestDetail(ctx, "u1", false, "t1")
	if err != nil || len(d.Parts) != 3 || d.Parts[0].Questions != 30 || len(d.Attempts) != 1 || d.Attempts[0].Retake {
		t.Fatalf("detail = %+v, %v", d, err)
	}
	v2 := startPractice(t, e, "u1", 6)
	_, _ = e.svc.Submit(ctx, "u1", v2.ID, nil)
	d, _ = e.svc.TestDetail(ctx, "u1", false, "t1")
	if len(d.Attempts) != 2 || !d.Attempts[0].Retake || d.Attempts[1].Retake {
		t.Fatalf("retakes = %+v", d.Attempts)
	}
	if r, _ := e.svc.Result(ctx, "u1", v2.ID); !r.Retake {
		t.Fatal("second sitting is a retake")
	}
}
