package toeic

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPaperViewHidesAnswers(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.readingTest("t1")
	v, err := e.svc.Start(t.Context(), "u1", false, StartReq{Source: Source{Kind: SourceTest, TestKey: "t1"}, Parts: []Part{5, 6, 7}})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(v)
	for _, key := range []string{`"answer"`, `"evidence"`, `"explainVi"`, `"translateVi"`, `"vocab"`} {
		if strings.Contains(string(b), key) {
			t.Fatalf("paper has %s", key)
		}
	}
	if v.Paper[0].Part != 5 || v.Paper[0].Number != 101 || v.Paper[30].Part != 6 || v.Paper[30].Number != 131 ||
		v.Paper[34].Questions[0].Number != 147 || v.Paper[len(v.Paper)-1].Questions[2].Number != 200 {
		t.Fatalf("numbers: %d %d %d", v.Paper[0].Number, v.Paper[30].Number, v.Paper[34].Number)
	}
}

func TestPracticeOfSomeParts(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.readingTest("t1")
	v, err := e.svc.Start(t.Context(), "u1", false, StartReq{Source: Source{Kind: SourceTest, TestKey: "t1"}, Parts: []Part{6, 5}, TimeLimitMin: 20})
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range v.Paper {
		if g.Part != 5 && g.Part != 6 {
			t.Fatalf("part %d in a Part 5+6 round", g.Part)
		}
	}
	if len(v.Paper) != 34 || v.TimeLimitMs != 20*60000 || v.RemainingMs != v.TimeLimitMs || len(v.Parts) != 2 {
		t.Fatalf("view = %d groups, limit %d, parts %v", len(v.Paper), v.TimeLimitMs, v.Parts)
	}
	if _, err := e.svc.Start(t.Context(), "u4", false, StartReq{Source: Source{Kind: SourceTest, TestKey: "t1"}, Parts: []Part{3}}); fieldsOf(t, err)["parts"] == "" {
		t.Fatal("a part missing from the test")
	}
}

func TestPartPaperPrefersUnanswered(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := t.Context()
	for _, k := range []string{"p5-a", "p5-b", "p5-c", "p5-d"} {
		e.seed(p5(k, 0))
	}
	e.attempts.recs = append(e.attempts.recs, Attempt{ID: "old", UserID: "u1", Status: AttemptSubmitted,
		QuestionIDs: []string{"p5-a#1", "p5-b#1"}})
	v, err := e.svc.Start(ctx, "u1", false, StartReq{Source: Source{Kind: SourcePart, Part: 5, Count: 2}})
	if err != nil {
		t.Fatal(err)
	}
	if len(v.Paper) != 2 {
		t.Fatalf("paper = %d groups", len(v.Paper))
	}
	for _, g := range v.Paper {
		if g.Key == "p5-a" || g.Key == "p5-b" {
			t.Fatalf("answered group %s picked first", g.Key)
		}
	}
	if v.Paper[0].Questions[0].Number != 1 || v.Paper[1].Questions[0].Number != 2 {
		t.Fatal("a part round counts from 1")
	}
	if _, err := e.svc.Start(ctx, "u1", false, StartReq{Source: Source{Kind: SourcePart, Part: 3}}); err != ErrNotEnough {
		t.Fatalf("empty part = %v", err)
	}
}
