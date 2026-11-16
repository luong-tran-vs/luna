package games

import (
	"errors"
	"testing"
)

func TestFinish(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := t.Context()
	var verr *ValidationError
	for _, r := range []Result{{Score: -1}, {Score: 30001}, {WrongWords: make([]string, 61)}, {WrongVerbs: make([]VerbMiss, 51)},
		{WrongVerbs: []VerbMiss{{Base: "go", Form: "present"}}}} {
		if _, err := e.svc.Finish(ctx, "u1", false, GamePicture, r); !errors.As(err, &verr) {
			t.Fatalf("%+v: %v", r, err)
		}
	}
	for _, g := range []Game{GameDaily, "chess"} {
		if _, err := e.svc.Finish(ctx, "u1", false, g, Result{}); !errors.As(err, &verr) {
			t.Fatalf("game %q: %v", g, err)
		}
	}
	if _, err := e.svc.Finish(ctx, "u2", true, GameGrammar, Result{Completed: true}); !errors.Is(err, ErrMembersOnly) {
		t.Fatalf("guest grammar: %v", err)
	}

	got, err := e.svc.Finish(ctx, "u1", false, GamePicture, Result{Score: 2100, Completed: true, WrongWords: []string{"bicycle"}})
	if err != nil || got.Best != 2100 || !got.NewBest || got.Rescheduled != 1 {
		t.Fatalf("first = %+v, %v", got, err)
	}
	got, _ = e.svc.Finish(ctx, "u1", false, GamePicture, Result{Score: 1500, Completed: true})
	if got.Best != 2100 || got.NewBest {
		t.Fatalf("lower = %+v", got)
	}
	// Left halfway: no best, the misses still come back.
	got, _ = e.svc.Finish(ctx, "u1", false, GamePicture, Result{Score: 9000, WrongWords: []string{"apple"}})
	if got.Best != 2100 || got.NewBest || len(e.words.calls) != 2 {
		t.Fatalf("left = %+v, calls %v", got, e.words.calls)
	}
}

func TestFinishVerbs(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	_, err := e.svc.Finish(t.Context(), "u1", false, GameVerbs, Result{Score: 12, Completed: true,
		WrongVerbs: []VerbMiss{{Base: " Run ", Form: "past", Typed: "runned"}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(e.verbs.misses) != 1 || e.verbs.misses[0] != (VerbMiss{Base: "run", Form: "past", Typed: "runned"}) {
		t.Fatalf("misses = %+v", e.verbs.misses)
	}
	if len(e.words.calls) != 0 {
		t.Fatalf("no words, yet %v", e.words.calls)
	}
}

func TestFinishZeroIsNoBest(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	got, err := e.svc.Finish(t.Context(), "u1", false, GameVerbs, Result{Score: 0, Completed: true})
	if err != nil || got.NewBest || got.Best != 0 {
		t.Fatalf("zero = %+v, %v", got, err)
	}
}
