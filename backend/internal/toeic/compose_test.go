package toeic

import (
	"errors"
	"fmt"
	"math/rand/v2"
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

// levelled returns g with the level and key given.
func levelled(g Group, key string, level int) Group {
	g.Key, g.Level = key, level
	return g
}

func TestComposeFullFollowsTheExamLayoutAndMixesLevels(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := t.Context()
	for i, lv := range Levels {
		for j := range 5 {
			e.seed(levelled(p3(""), fmt.Sprintf("p3-%d-%d", lv, j), lv))
		}
		g := levelled(p3(""), fmt.Sprintf("p3-graphic-%d", lv), lv)
		g.Graphic = &Graphic{Kind: "table", Columns: []string{"Day", "Room"}, Rows: [][]string{{"Mon", "A"}, {"Tue", "B"}}}
		e.seed(g)
		// Part 7: singles of 3 and 2 questions, doubles and triples of 5.
		for j := range 4 {
			e.seed(levelled(p7("", 3), fmt.Sprintf("p7-s3-%d-%d", lv, j), lv))
		}
		for j := range 2 {
			e.seed(levelled(p7("", 2), fmt.Sprintf("p7-s2-%d-%d", lv, j), lv))
		}
		for n := 2; n <= 3; n++ {
			g := levelled(p7("", 5), fmt.Sprintf("p7-d%d-%d", n, lv), lv)
			for len(g.Docs) < n {
				g.Docs = append(g.Docs, g.Docs[0])
			}
			if i%2 == 0 || n == 3 {
				e.seed(g)
			}
		}
	}

	res, err := e.svc.ComposeTest(ctx, ComposeInput{Kind: KindFull, DryRun: true})
	if err != nil {
		t.Fatalf("compose = %v", err)
	}
	groups, _ := e.svc.groupsOf(ctx, res.Plan.Groups)
	var p3keys, p7keys []string
	for _, k := range res.Plan.Groups {
		switch groups[k].Part {
		case 3:
			p3keys = append(p3keys, k)
		case 7:
			p7keys = append(p7keys, k)
		}
	}

	// Part 3: 13 groups, the 3 graphic ones last; the others easy first; every level is there.
	cp := planPart(res.Plan, 3)
	if !cp.Filled || cp.Note != "" || len(p3keys) != 13 || len(cp.Levels) != 4 {
		t.Fatalf("part 3 = %+v %v", cp, p3keys)
	}
	for i, k := range p3keys {
		if graphic := groups[k].Graphic != nil; graphic != (i >= 10) {
			t.Fatalf("part 3 order = %v", p3keys)
		}
		if i > 0 && i != 10 && groups[k].Level < groups[p3keys[i-1]].Level {
			t.Fatalf("part 3 not easy first: %v", p3keys)
		}
	}

	// Part 7: 29 questions of singles, then 2 doubles, then 3 triples.
	cp = planPart(res.Plan, 7)
	if !cp.Filled || cp.Note != "" || cp.Have != 54 {
		t.Fatalf("part 7 = %+v", cp)
	}
	docs := map[int]int{}
	single := 0
	for i, k := range p7keys {
		g := groups[k]
		docs[len(g.Docs)]++
		if len(g.Docs) == 1 {
			single += len(g.Questions)
		}
		if i > 0 && len(g.Docs) < len(groups[p7keys[i-1]].Docs) {
			t.Fatalf("part 7 order = %v", p7keys)
		}
	}
	if docs[2] != 2 || docs[3] != 3 || single != 29 {
		t.Fatalf("part 7 layout = %v, singles %d", docs, single)
	}
	for _, lv := range Levels {
		if cp.Levels[lv] == 0 {
			t.Fatalf("part 7 levels = %v", cp.Levels)
		}
	}
}

func TestComposeNotesALayoutTheBankCannotFollow(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.readingTest("luna-test-01") // Part 7 has single passages only
	res, err := e.svc.ComposeTest(t.Context(), ComposeInput{Kind: KindFull, DryRun: true})
	if err != nil {
		t.Fatalf("compose = %v", err)
	}
	if cp := planPart(res.Plan, 7); !cp.Filled || !strings.Contains(cp.Note, "3 bài") {
		t.Fatalf("part 7 = %+v", cp)
	}
	if cp := planPart(res.Plan, 5); cp.Note != "" {
		t.Fatalf("part 5 = %+v", cp)
	}
}

func TestLevelTargets(t *testing.T) {
	t.Parallel()
	for target, want := range map[int][]int{30: {6, 9, 9, 6}, 54: {11, 16, 16, 11}, 10: {2, 3, 3, 2}, 1: {0, 1, 0, 0}} {
		got := levelTargets(target)
		sum := 0
		for _, n := range got {
			sum += n
		}
		if sum != target || (target != 1 && !slices.Equal(got, want)) {
			t.Fatalf("levelTargets(%d) = %v, want %v", target, got, want)
		}
	}
}

func TestDeleteTest(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := t.Context()
	e.readingTest("luna-test-01")
	mux := e.mux()

	// Published: hide first.
	rec := call(t, mux, http.MethodDelete, "/api/admin/toeic/tests/luna-test-01", "admin", "")
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "ẩn đề") {
		t.Fatalf("published = %d %s", rec.Code, rec.Body.String())
	}
	if _, err := e.svc.SetTestStatus(ctx, "luna-test-01", StatusHidden); err != nil {
		t.Fatal(err)
	}
	// Taken by a learner: keep it.
	e.attempts.recs = append(e.attempts.recs, Attempt{ID: "x1", UserID: "u1", Status: AttemptInProgress,
		Source: Source{Kind: SourceTest, TestKey: "luna-test-01"}})
	var verr *ValidationError
	if err := e.svc.DeleteTest(ctx, "luna-test-01"); !errors.As(err, &verr) || verr.Fields["attempts"] == "" {
		t.Fatalf("with attempts = %v", err)
	}
	e.attempts.recs = nil

	if rec := call(t, mux, http.MethodDelete, "/api/admin/toeic/tests/luna-test-01", "member", ""); rec.Code != http.StatusForbidden {
		t.Fatalf("member = %d", rec.Code)
	}
	if rec := call(t, mux, http.MethodDelete, "/api/admin/toeic/tests/luna-test-01", "admin", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete = %d %s", rec.Code, rec.Body.String())
	}
	if _, err := e.tests.Get(ctx, "luna-test-01"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("still there: %v", err)
	}
	// The groups stay in the bank.
	if g, err := e.svc.GetGroup(ctx, "luna-test-01-p5-00"); err != nil || g.Status != StatusPublished {
		t.Fatalf("group = %+v %v", g, err)
	}
	if rec := call(t, mux, http.MethodDelete, "/api/admin/toeic/tests/luna-test-01", "admin", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("again = %d", rec.Code)
	}
}

func TestPickByLevelSpreadsBigGroupsOverEveryLevel(t *testing.T) {
	t.Parallel()
	var cands []Group
	for _, lv := range Levels {
		for j := range 2 {
			cands = append(cands, levelled(p7("", 5), fmt.Sprintf("d-%d-%d", lv, j), lv))
		}
	}
	seen := map[int]int{}
	for seed := range uint64(200) {
		gs, ok := pickByLevel(cands, map[string]int{}, levelTargets(10), rand.New(rand.NewPCG(seed, 1)))
		if !ok || len(gs) != 2 {
			t.Fatalf("seed %d = %v %v", seed, gs, ok)
		}
		for _, g := range gs {
			seen[g.Level]++
		}
	}
	for _, lv := range Levels {
		if seen[lv] < 40 {
			t.Fatalf("levels picked = %v", seen)
		}
	}
}
