package toeic

import (
	"errors"
	"fmt"
	"testing"
)

func partOf(c TestCheck, p Part) PartCheck {
	for _, pc := range c.Parts {
		if pc.Part == p {
			return pc
		}
	}
	return PartCheck{}
}

func TestCheckFullTest(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := t.Context()
	test := e.readingTest("t1")

	ta, err := e.svc.TestAdmin(ctx, "t1")
	if err != nil || !ta.Check.Ready || ta.Check.Complete || ta.Check.Docs != 22 {
		t.Fatalf("reading test = %+v, %v", ta.Check, err)
	}
	if pc := partOf(ta.Check, 7); pc.Have != 54 || pc.Need != 54 {
		t.Fatalf("part 7 = %+v", pc)
	}

	// Part 3 started but 4 questions short.
	var p3keys []string
	for i := range 11 {
		g := p3(fmt.Sprintf("p3-%02d", i))
		e.seed(g)
		p3keys = append(p3keys, g.Key)
	}
	test.Groups = append(p3keys, test.Groups...)
	e.tests.recs["t1"] = test
	ta, _ = e.svc.TestAdmin(ctx, "t1")
	if pc := partOf(ta.Check, 3); pc.Have != 33 || pc.Need != 39 || ta.Check.Ready {
		t.Fatalf("part 3 short = %+v, ready %v", pc, ta.Check.Ready)
	}
	if _, err := e.svc.SetTestStatus(ctx, "t1", StatusPublished); !errors.As(err, new(*ErrTestNotReady)) {
		t.Fatalf("publish not ready = %v", err)
	}

	// A draft group and a Part 1 question without its photo.
	test = e.readingTest("t2")
	_ = e.groups.SetStatus(ctx, test.Groups[0], StatusDraft)
	ta, _ = e.svc.TestAdmin(ctx, "t2")
	if len(ta.Check.DraftGroups) != 1 || ta.Check.Ready {
		t.Fatalf("draft = %+v", ta.Check)
	}
	_ = e.groups.SetStatus(ctx, test.Groups[0], StatusPublished)
	var p1keys []string
	for i := range 6 {
		g := p1(fmt.Sprintf("p1-%d", i))
		e.seed(g)
		p1keys = append(p1keys, g.Key)
	}
	_ = e.images.Save(ctx, Image{GroupKey: "p1-0", Slot: SlotPhoto})
	test.Groups = append(p1keys, test.Groups...)
	e.tests.recs["t2"] = test
	ta, _ = e.svc.TestAdmin(ctx, "t2")
	if len(ta.Check.MissingPhotos) != 5 || ta.Check.MissingPhotos[0].Number != 2 || ta.Check.Ready {
		t.Fatalf("photos = %+v", ta.Check.MissingPhotos)
	}
	if partOf(ta.Check, 1).Have != 6 {
		t.Fatal("part 1 count")
	}
}

func TestCheckMiniTest(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	test := Test{Key: "mini", Kind: KindMini50, Status: StatusDraft}
	add := func(g Group) {
		e.seed(g)
		test.Groups = append(test.Groups, g.Key)
	}
	for i := range 7 { // need 8, one group off is fine
		add(p5(fmt.Sprintf("m5-%d", i), 0))
	}
	add(p6("m6"))
	for i := range 4 { // 12
		add(p7(fmt.Sprintf("m7-%d", i), 3))
	}
	e.tests.recs["mini"] = test
	ta, _ := e.svc.TestAdmin(t.Context(), "mini")
	if !ta.Check.Ready || partOf(ta.Check, 5).Need != 8 {
		t.Fatalf("mini = %+v", ta.Check)
	}
	for i := 7; i < 12; i++ { // 12 is 4 more than 8
		add(p5(fmt.Sprintf("m5-%d", i), 0))
	}
	e.tests.recs["mini"] = test
	ta, _ = e.svc.TestAdmin(t.Context(), "mini")
	if ta.Check.Ready {
		t.Fatalf("too many part 5 = %+v", partOf(ta.Check, 5))
	}
}

func TestCreateTestOrdersByPartAndChecksGroups(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := t.Context()
	e.seed(p7("p7-a", 2), p5("p5-a", 0))
	if _, err := e.svc.CreateTest(ctx, TestInput{Key: "luna-t1", Title: "T", Kind: KindFull, Groups: []string{"p7-a", "nope"}}); fieldsOf(t, err)["groups"] == "" {
		t.Fatal("missing group not reported")
	}
	ta, err := e.svc.CreateTest(ctx, TestInput{Key: "LUNA-T1", Title: "Luna Test", Kind: KindFull, Groups: []string{"p7-a", "p5-a"}})
	if err != nil || ta.Test.Key != "luna-t1" || ta.Test.Groups[0] != "p5-a" || ta.Test.Status != StatusDraft {
		t.Fatalf("create = %+v, %v", ta.Test, err)
	}
	if _, err := e.svc.CreateTest(ctx, TestInput{Key: "luna-t1", Title: "Again", Kind: KindFull, Groups: []string{"p5-a"}}); !errors.Is(err, ErrExists) {
		t.Fatalf("again = %v", err)
	}
}
