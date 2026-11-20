package toeic

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
)

func planPart(p ComposePlan, part Part) ComposePart {
	for _, cp := range p.Parts {
		if cp.Part == part {
			return cp
		}
	}
	return ComposePart{}
}

func TestComposeTestPrefersUnusedGroups(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := t.Context()
	used := e.readingTest("luna-test-01")
	// A second set of reading groups that no test has: 30 Part 5, 4 Part 6, Part 7 of 2, 3, 4 and 5
	// questions (enough to make exactly 54 in several ways).
	for i := range 32 {
		e.seed(p5(fmt.Sprintf("new-p5-%02d", i), i%4))
	}
	for i := range 4 {
		e.seed(p6(fmt.Sprintf("new-p6-%d", i)))
	}
	for i, n := range []int{5, 5, 5, 5, 5, 5, 4, 4, 4, 4, 3, 3, 2, 2} {
		e.seed(p7(fmt.Sprintf("new-p7-%02d", i), n))
	}

	res, err := e.svc.ComposeTest(ctx, ComposeInput{Kind: KindFull, Partial: true})
	if err != nil {
		t.Fatalf("compose = %v", err)
	}
	if res.Plan.Key != "luna-test-02" || res.Plan.Title != "Luna Test 02" || res.Plan.Order != 2 {
		t.Fatalf("name = %q %q %d", res.Plan.Key, res.Plan.Title, res.Plan.Order)
	}
	for _, p := range []Part{5, 6, 7} {
		cp := planPart(res.Plan, p)
		if !cp.Filled || cp.Have != PartQuestions[p] || cp.Reused != 0 {
			t.Fatalf("part %d = %+v", p, cp)
		}
	}
	if cp := planPart(res.Plan, 3); cp.Filled || cp.Have != 0 || res.Plan.Complete {
		t.Fatalf("part 3 = %+v, complete %v", cp, res.Plan.Complete)
	}
	for _, k := range res.Plan.Groups {
		if slices.Contains(used.Groups, k) {
			t.Fatalf("reused %s while unused groups were enough", k)
		}
	}
	if res.Test == nil || res.Test.Test.Status != StatusDraft || !res.Test.Check.Ready {
		t.Fatalf("test = %+v", res.Test)
	}
}

func TestComposeTestReusesWhenShortAndReportsMissingParts(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := t.Context()
	e.readingTest("luna-test-01")

	// Dry run: Part 5–7 come from the only test there is; the listening parts cannot be filled.
	res, err := e.svc.ComposeTest(ctx, ComposeInput{Kind: KindFull, DryRun: true})
	if err != nil || res.Test != nil {
		t.Fatalf("dry run = %+v, %v", res, err)
	}
	if cp := planPart(res.Plan, 7); !cp.Filled || cp.Reused != cp.Groups {
		t.Fatalf("part 7 = %+v", cp)
	}

	// Without partial, the missing parts are named and nothing is stored.
	_, err = e.svc.ComposeTest(ctx, ComposeInput{Kind: KindFull})
	var verr *ValidationError
	if !errors.As(err, &verr) || !strings.Contains(verr.Fields["parts.1"], "cần 6") {
		t.Fatalf("missing parts = %v", err)
	}
	if _, err := e.tests.Get(ctx, "luna-test-02"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("stored = %v", err)
	}
}

func TestComposeMiniTest(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := t.Context()
	for i := range 10 {
		e.seed(p1(fmt.Sprintf("p1-%02d", i)), p2(fmt.Sprintf("p2-%02d", i)), p3(fmt.Sprintf("p3-%02d", i)),
			p4(fmt.Sprintf("p4-%02d", i)), p5(fmt.Sprintf("p5-%02d", i), 0))
	}
	for i := range 3 {
		e.seed(p6(fmt.Sprintf("p6-%d", i)))
	}
	for i := range 6 {
		e.seed(p7(fmt.Sprintf("p7-%d", i), 3))
	}
	res, err := e.svc.ComposeTest(ctx, ComposeInput{Kind: KindMini50, Title: "Thử nhanh"})
	if err != nil {
		t.Fatalf("compose = %v", err)
	}
	if res.Plan.Key != "mini-01" || res.Plan.Title != "Thử nhanh" || !res.Plan.Complete {
		t.Fatalf("plan = %+v", res.Plan)
	}
	// Mini 50: Part 3 needs 9 (three groups of 3); Part 7 needs 12 (four groups of 3).
	if cp := planPart(res.Plan, 3); cp.Have != 9 || cp.Groups != 3 {
		t.Fatalf("part 3 = %+v", cp)
	}
	if cp := planPart(res.Plan, 7); cp.Have != 12 || cp.Groups != 4 {
		t.Fatalf("part 7 = %+v", cp)
	}
	if !res.Test.Check.Complete {
		t.Fatalf("check = %+v", res.Test.Check)
	}
}

func TestComposeHTTP(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.readingTest("luna-test-01")
	mux := e.mux()

	rec := call(t, mux, http.MethodPost, "/api/admin/toeic/tests/compose", "admin", `{"kind":"full"}`)
	if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(rec.Body.String(), `"plan"`) {
		t.Fatalf("short = %d %s", rec.Code, rec.Body.String())
	}
	rec = call(t, mux, http.MethodPost, "/api/admin/toeic/tests/compose", "admin", `{"kind":"full","partial":true}`)
	if rec.Code != http.StatusCreated || !strings.Contains(rec.Body.String(), `"key":"luna-test-02"`) {
		t.Fatalf("partial = %d %s", rec.Code, rec.Body.String())
	}
	rec = call(t, mux, http.MethodPost, "/api/admin/toeic/tests/compose", "member", `{"kind":"full"}`)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("member = %d", rec.Code)
	}
}
