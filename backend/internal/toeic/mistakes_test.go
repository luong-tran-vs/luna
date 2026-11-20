package toeic

import (
	"errors"
	"testing"
	"time"
)

func TestApplyMistake(t *testing.T) {
	t.Parallel()
	m := Mistake{Streak: 1}
	if keep := applyMistake(&m, true, false); !keep || m.Streak != 1 {
		t.Fatalf("right in a submitted attempt = %+v", m)
	}
	if keep := applyMistake(&m, false, false); !keep || m.Streak != 0 {
		t.Fatalf("wrong in a submitted attempt = %+v", m)
	}
	if keep := applyMistake(&m, true, true); !keep || m.Streak != 1 {
		t.Fatalf("first right in the notebook = %+v", m)
	}
	if keep := applyMistake(&m, false, true); !keep || m.Streak != 0 {
		t.Fatalf("wrong in the notebook = %+v", m)
	}
	applyMistake(&m, true, true)
	if keep := applyMistake(&m, true, true); keep {
		t.Fatal("two in a row should leave the notebook")
	}
}

func TestNotebookFlow(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := t.Context()
	e.readingTest("t1")
	v := startPractice(t, e, "u1", 7)
	q1, q2, q3 := v.Paper[0].Questions[0].ID, v.Paper[0].Questions[1].ID, v.Paper[0].Questions[2].ID
	// Part 7 answers are 1: q1 right, q2 wrong, q3 and the rest skipped.
	if _, err := e.svc.Submit(ctx, "u1", v.ID, &Progress{Answers: map[string]int{q1: 1, q2: 0}}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.mistakes.Get(ctx, "u1", q1); !errors.Is(err, ErrNotFound) {
		t.Fatal("a right answer went in the notebook")
	}
	if m, err := e.mistakes.Get(ctx, "u1", q2); err != nil || m.Streak != 0 || m.Part != 7 || m.Tags[0] != "detail" {
		t.Fatalf("wrong = %+v, %v", m, err)
	}
	if _, err := e.mistakes.Get(ctx, "u1", q3); err != nil {
		t.Fatal("a skipped question is in the notebook too")
	}

	view, err := e.svc.Mistakes(ctx, "u1", 7)
	if err != nil || len(view.Items) != 53 || view.Counts[7] != 53 || view.Items[0].Prompt == "" {
		t.Fatalf("notebook = %d items, %v", len(view.Items), err)
	}

	e.now = e.now.Add(time.Hour)
	if r, err := e.svc.CheckMistake(ctx, "u1", q2, 1); err != nil || !r.Right || r.Streak != 1 || r.Removed {
		t.Fatalf("first right = %+v, %v", r, err)
	}
	if r, _ := e.svc.CheckMistake(ctx, "u1", q2, 0); r.Right || r.Streak != 0 || r.Answer != 1 {
		t.Fatalf("wrong again = %+v", r)
	}
	_, _ = e.svc.CheckMistake(ctx, "u1", q2, 1)
	if r, _ := e.svc.CheckMistake(ctx, "u1", q2, 1); !r.Removed {
		t.Fatalf("two in a row = %+v", r)
	}
	if _, err := e.svc.CheckMistake(ctx, "u1", q2, 1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("after leaving = %v", err)
	}

	// The question wrong most recently (q2 was, at e.now) comes last; it left anyway.
	round, err := e.svc.MistakeRound(ctx, "u1", 7, 4)
	if err != nil || len(round) != 4 || len(round[0].Group.Questions) != 1 || len(round[0].Group.Docs) != 1 {
		t.Fatalf("round = %+v, %v", round, err)
	}
}

func TestRetryPaper(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := t.Context()
	e.readingTest("t1")
	v := startPractice(t, e, "u1", 7)
	answers := map[string]int{}
	for _, g := range v.Paper {
		for _, q := range g.Questions {
			answers[q.ID] = 1
		}
	}
	wrong := v.Paper[2].Questions[1].ID
	answers[wrong] = 3
	if _, err := e.svc.Submit(ctx, "u1", v.ID, &Progress{Answers: answers}); err != nil {
		t.Fatal(err)
	}
	r, err := e.svc.Start(ctx, "u1", false, StartReq{Source: Source{Kind: SourceRetry, FromAttempt: v.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Paper) != 1 || len(r.Paper[0].Questions) != 1 || r.Paper[0].Questions[0].ID != wrong || r.Paper[0].Questions[0].Number != 154 {
		t.Fatalf("retry = %+v", r.Paper)
	}
	res, err := e.svc.Submit(ctx, "u1", r.ID, &Progress{Answers: map[string]int{wrong: 1}})
	if err != nil || res.Questions != 1 || res.Correct != 1 {
		t.Fatalf("retry result = %+v, %v", res, err)
	}
	if _, err := e.svc.Start(ctx, "u4", false, StartReq{Source: Source{Kind: SourceRetry, FromAttempt: v.ID}}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("another learner's attempt = %v", err)
	}
}
