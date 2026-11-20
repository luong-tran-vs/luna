package toeic

import (
	"testing"
)

func TestImportDryRunSavesNothing(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	rep, err := e.svc.Import(t.Context(), []Group{p5("p5-a", 1), p7("p7-a", 2)}, false, true)
	if err != nil || !rep.DryRun || len(rep.Created) != 2 || rep.ListeningMinutes != nil {
		t.Fatalf("dry run = %+v, %v", rep, err)
	}
	if len(e.groups.recs) != 0 {
		t.Fatal("dry run saved groups")
	}
	rep, err = e.svc.Import(t.Context(), []Group{p2("p2-a")}, false, true)
	if err != nil || rep.ListeningMinutes == nil || *rep.ListeningMinutes <= 0 {
		t.Fatalf("listening minutes = %+v, %v", rep, err)
	}
}

func TestImportRefusesAFileWithErrors(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	bad := p5("p5-b", 0)
	bad.Questions[0].Answer = 7
	gs := []Group{p5("p5-a", 1), bad}
	rep, err := e.svc.Import(t.Context(), gs, false, true)
	if err != nil || rep.Reports[1].Status != ReportError || rep.Reports[1].Fields["questions[0].answer"] == "" || rep.Reports[0].Status != ReportOK {
		t.Fatalf("dry run = %+v, %v", rep, err)
	}
	_, err = e.svc.Import(t.Context(), []Group{p5("p5-a", 1), bad}, false, false)
	if f := fieldsOf(t, err); f["groups[1].questions[0].answer"] == "" {
		t.Fatalf("fields = %v", f)
	}
	if len(e.groups.recs) != 0 {
		t.Fatal("a refused file saved groups")
	}
}

func TestImportDuplicatesAndReplace(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := t.Context()
	if _, err := e.svc.Import(ctx, []Group{p5("p5-a", 1)}, false, false); err != nil {
		t.Fatal(err)
	}
	if g := e.groups.recs["p5-a"]; g.Status != StatusDraft || g.Version != 1 {
		t.Fatalf("new group = %+v", g)
	}
	_ = e.groups.SetStatus(ctx, "p5-a", StatusPublished)

	rep, err := e.svc.Import(ctx, []Group{p5("p5-a", 2), p5("p5-b", 0)}, false, false)
	if err != nil || len(rep.Duplicates) != 1 || rep.Duplicates[0] != "p5-a" || len(rep.Created) != 1 {
		t.Fatalf("duplicate = %+v, %v", rep, err)
	}
	if e.groups.recs["p5-a"].Questions[0].Answer != 1 {
		t.Fatal("a duplicate was replaced without replace")
	}
	rep, err = e.svc.Import(ctx, []Group{p5("p5-a", 2)}, true, false)
	if err != nil || len(rep.Replaced) != 1 {
		t.Fatalf("replace = %+v, %v", rep, err)
	}
	g := e.groups.recs["p5-a"]
	if g.Status != StatusPublished || g.Version != 2 || g.Questions[0].Answer != 2 {
		t.Fatalf("replaced group = %+v", g)
	}
}

func TestImportKeepsQuestionCountOfPublishedTests(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.seed(p7("p7-a", 3))
	e.tests.recs["t1"] = Test{Key: "t1", Kind: KindFull, Status: StatusPublished, Groups: []string{"p7-a"}}
	rep, err := e.svc.Import(t.Context(), []Group{p7("p7-a", 2)}, true, true)
	if err != nil || rep.Reports[0].Status != ReportError || rep.Reports[0].Fields["questions"] == "" {
		t.Fatalf("report = %+v, %v", rep, err)
	}
	if rep, err := e.svc.Import(t.Context(), []Group{p7("p7-a", 3)}, true, true); err != nil || rep.Reports[0].Status != ReportReplace {
		t.Fatalf("same count = %+v, %v", rep, err)
	}
}
