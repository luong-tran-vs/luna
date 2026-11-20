package toeic

import (
	"errors"
	"testing"
)

func TestReplaceRescores(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := t.Context()
	e.seed(p5("p5-a", 1), p5("p5-b", 2))
	submit := func(user string, keys []string, answers map[string]int) string {
		var ids []string
		for _, k := range keys {
			ids = append(ids, QuestionID(k, 1))
		}
		a, _ := e.attempts.Create(ctx, Attempt{UserID: user, Mode: ModePractice, Source: Source{Kind: SourcePart, Part: 5},
			GroupKeys: keys, QuestionIDs: ids, Answers: answers, Status: AttemptInProgress, Numbers: map[string]int{}})
		if _, err := e.svc.Submit(ctx, user, a.ID, nil); err != nil {
			t.Fatal(err)
		}
		return a.ID
	}
	a1 := submit("u1", []string{"p5-a", "p5-b"}, map[string]int{"p5-a#1": 0, "p5-b#1": 2})
	other := submit("u1", []string{"p5-b"}, map[string]int{"p5-b#1": 2})
	busy, _ := e.attempts.Create(ctx, Attempt{UserID: "u1", GroupKeys: []string{"p5-a"}, QuestionIDs: []string{"p5-a#1"}, Status: AttemptInProgress})

	if _, err := e.mistakes.Get(ctx, "u1", "p5-a#1"); err != nil {
		t.Fatal("p5-a#1 was wrong")
	}
	if rep, err := e.svc.Import(ctx, []Group{p5("p5-a", 0), p5("p5-b", 3)}, true, true); err != nil || rep.Rescored != 2 {
		t.Fatalf("dry run = %+v, %v", rep, err)
	}
	// p5-a's key is now A (the learner's choice), p5-b's is D (they chose C).
	rep, err := e.svc.Import(ctx, []Group{p5("p5-a", 0), p5("p5-b", 3)}, true, false)
	if err != nil || rep.Rescored != 2 {
		t.Fatalf("import = %+v, %v", rep, err)
	}
	r1, _ := e.svc.Result(ctx, "u1", a1)
	if r1.Correct != 1 || r1.Wrong != 1 {
		t.Fatalf("rescored = %+v", r1)
	}
	if r2, _ := e.svc.Result(ctx, "u1", other); r2.Correct != 0 {
		t.Fatalf("other = %+v", r2)
	}
	if _, err := e.mistakes.Get(ctx, "u1", "p5-a#1"); !errors.Is(err, ErrNotFound) {
		t.Fatal("now right: out of the notebook")
	}
	if m, err := e.mistakes.Get(ctx, "u1", "p5-b#1"); err != nil || m.Streak != 0 {
		t.Fatalf("now wrong: %+v, %v", m, err)
	}
	if a, _ := e.attempts.Get(ctx, "u1", busy.ID); a.Result != nil {
		t.Fatal("an attempt in progress was graded")
	}
}
