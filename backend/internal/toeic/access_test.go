package toeic

import (
	"errors"
	"testing"
)

func TestGuestAccess(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := t.Context()
	e.tests.recs["mini-02"] = Test{Key: "mini-02", Kind: KindMini50, Order: 5, Status: StatusPublished}
	e.tests.recs["mini-01"] = Test{Key: "mini-01", Kind: KindMini100, Order: 2, Status: StatusPublished}
	e.tests.recs["mini-00"] = Test{Key: "mini-00", Kind: KindMini50, Order: 1, Status: StatusDraft}
	e.tests.recs["full-01"] = Test{Key: "full-01", Kind: KindFull, Order: 0, Status: StatusPublished}

	allowed := []Source{{Kind: SourceTest, TestKey: "mini-01"}, {Kind: SourcePart, Part: 2}, {Kind: SourcePart, Part: 5}, {Kind: SourcePart, Part: 1}}
	for _, src := range allowed {
		if err := e.svc.checkStart(ctx, true, src); err != nil {
			t.Errorf("guest %+v: %v", src, err)
		}
	}
	refused := []Source{
		{Kind: SourceTest, TestKey: "mini-02"}, {Kind: SourceTest, TestKey: "full-01"}, {Kind: SourceTest, TestKey: "mini-00"},
		{Kind: SourcePart, Part: 3}, {Kind: SourcePart, Part: 7}, {Kind: SourceRandom, Size: 50}, {Kind: SourceTag, Tag: "detail"},
		{Kind: SourceMistakes}, {Kind: SourceRetry, FromAttempt: "a1"},
	}
	for _, src := range refused {
		if err := e.svc.checkStart(ctx, true, src); !errors.Is(err, ErrMembersOnly) {
			t.Errorf("guest %+v: %v", src, err)
		}
		if err := e.svc.checkStart(ctx, false, src); err != nil {
			t.Errorf("member %+v: %v", src, err)
		}
	}
	if !locked(e.tests.recs["full-01"], true, "mini-01") || locked(e.tests.recs["mini-01"], true, "mini-01") ||
		locked(e.tests.recs["full-01"], false, "mini-01") {
		t.Fatal("locked")
	}
}
