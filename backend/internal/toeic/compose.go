package toeic

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sort"
	"strings"
)

// ComposeInput asks for a test made by the app from the published groups ("Ghép đề tự động").
type ComposeInput struct {
	Kind TestKind `json:"kind"`
	// Title and Key are optional: the next free "Luna Test NN" / "Mini Test NN" is used when empty.
	Title string `json:"title"`
	Key   string `json:"key"`
	// Partial creates the test even when some parts cannot be filled (those parts stay empty).
	Partial bool `json:"partial"`
	// DryRun only returns the plan.
	DryRun bool `json:"dryRun"`
}

// ComposePart is what the plan takes for one part.
type ComposePart struct {
	Part Part `json:"part"`
	Need int  `json:"need"`
	// Have is how many questions were picked (0 when the part cannot be filled).
	Have int `json:"have"`
	// Groups and Reused are how many groups were picked and how many of them are already in a test.
	Groups int `json:"groups"`
	Reused int `json:"reused"`
	// Available is how many published questions the part has in all.
	Available int  `json:"available"`
	Filled    bool `json:"filled"`
}

// ComposePlan is the test the app would make.
type ComposePlan struct {
	Key    string        `json:"key"`
	Title  string        `json:"title"`
	Kind   TestKind      `json:"kind"`
	Order  int           `json:"order"`
	Parts  []ComposePart `json:"parts"`
	Groups []string      `json:"groups"`
	// Complete is true when every part is filled.
	Complete bool `json:"complete"`
}

// ComposeResult is the plan and, unless it was a dry run, the test made from it.
type ComposeResult struct {
	Plan ComposePlan
	Test *TestAdmin
}

// ComposeTest picks published groups so that each part has the questions a test of that kind needs,
// preferring groups that are in no test yet (then the least used), and stores the result as a draft
// test. Without Partial, a part that cannot be filled is a ValidationError naming it.
func (s *Service) ComposeTest(ctx context.Context, in ComposeInput) (ComposeResult, error) {
	if !in.Kind.Valid() {
		return ComposeResult{}, invalid("kind", "Loại đề phải là full, mini50 hoặc mini100")
	}
	published, err := s.d.Groups.Published(ctx, 0)
	if err != nil {
		return ComposeResult{}, err
	}
	tests, err := s.d.Tests.List(ctx, false)
	if err != nil {
		return ComposeResult{}, err
	}
	uses := map[string]int{}
	for _, t := range tests {
		for _, k := range t.Groups {
			uses[k]++
		}
	}
	plan := ComposePlan{Kind: in.Kind, Groups: []string{}, Complete: true}
	plan.Key, plan.Title, plan.Order = nextTestName(tests, in)
	rnd := s.newRand()
	missing := map[string]string{}
	for p := Part(1); p <= 7; p++ {
		var cands []Group
		for _, g := range published {
			if g.Part == p {
				cands = append(cands, g)
			}
		}
		picked, cp := pickPart(p, in.Kind, cands, uses, rnd)
		plan.Parts = append(plan.Parts, cp)
		if !cp.Filled {
			plan.Complete = false
			missing[fmt.Sprintf("parts.%d", p)] = fmt.Sprintf("Part %d: có %d câu đã đăng, cần %d", p, cp.Available, cp.Need)
			continue
		}
		plan.Groups = append(plan.Groups, picked...)
	}
	if len(plan.Groups) == 0 {
		return ComposeResult{Plan: plan}, invalid("groups", "Chưa đủ cụm đã đăng cho Part nào")
	}
	if in.DryRun {
		return ComposeResult{Plan: plan}, nil
	}
	if !plan.Complete && !in.Partial {
		return ComposeResult{Plan: plan}, &ValidationError{Fields: missing}
	}
	ta, err := s.CreateTest(ctx, TestInput{Key: plan.Key, Title: plan.Title, Kind: plan.Kind, Order: plan.Order, Groups: plan.Groups})
	if err != nil {
		return ComposeResult{Plan: plan}, err
	}
	return ComposeResult{Plan: plan, Test: &ta}, nil
}

// pickPart chooses the groups of one part: the cheapest set (unused groups first, then the least
// used, random among equals) whose questions add up to what the kind needs; a Mini Test may be off
// by one group when the exact number cannot be made.
func pickPart(p Part, kind TestKind, cands []Group, uses map[string]int, rnd *rand.Rand) ([]string, ComposePart) {
	need := needOf(kind, p)
	cp := ComposePart{Part: p, Need: need}
	for _, g := range cands {
		cp.Available += len(g.Questions)
	}
	rnd.Shuffle(len(cands), func(i, j int) { cands[i], cands[j] = cands[j], cands[i] })
	sort.SliceStable(cands, func(i, j int) bool { return uses[cands[i].Key] < uses[cands[j].Key] })

	targets := []int{need}
	if kind.Mini() {
		for d := 1; d <= groupSize[p]; d++ {
			targets = append(targets, need-d, need+d)
		}
	}
	for _, target := range targets {
		if target <= 0 {
			continue
		}
		idx, ok := cheapestSubset(cands, uses, target)
		if !ok {
			continue
		}
		keys := make([]string, 0, len(idx))
		for _, i := range idx {
			g := cands[i]
			keys = append(keys, g.Key)
			cp.Have += len(g.Questions)
			if uses[g.Key] > 0 {
				cp.Reused++
			}
		}
		sort.Strings(keys)
		cp.Groups = len(keys)
		cp.Filled = true
		return keys, cp
	}
	return nil, cp
}

// cheapestSubset returns the groups (indexes into cands) whose questions add up to target with the
// lowest cost: a used group costs far more than an unused one, and a later candidate a little more
// than an earlier one (cands is in order of preference).
func cheapestSubset(cands []Group, uses map[string]int, target int) ([]int, bool) {
	const unreachable = int64(1) << 62
	n := len(cands)
	// cost[i][s] is the cheapest way to reach s with the first i candidates.
	cost := make([][]int64, n+1)
	for i := range cost {
		cost[i] = make([]int64, target+1)
		for s := range cost[i] {
			cost[i][s] = unreachable
		}
	}
	cost[0][0] = 0
	for i, g := range cands {
		size := len(g.Questions)
		c := int64(uses[g.Key])*1_000_000 + int64(i)
		for s := 0; s <= target; s++ {
			best := cost[i][s]
			if size > 0 && s >= size && cost[i][s-size] != unreachable && cost[i][s-size]+c < best {
				best = cost[i][s-size] + c
			}
			cost[i+1][s] = best
		}
	}
	if cost[n][target] == unreachable {
		return nil, false
	}
	var out []int
	for i, s := n, target; i > 0 && s > 0; i-- {
		if cost[i][s] != cost[i-1][s] {
			out = append(out, i-1)
			s -= len(cands[i-1].Questions)
		}
	}
	for l, r := 0, len(out)-1; l < r; l, r = l+1, r-1 {
		out[l], out[r] = out[r], out[l]
	}
	return out, true
}

// nextTestName returns the key, title and order of a new test: the input's when given, otherwise the
// next free "luna-test-NN" / "mini-NN" after the tests there are.
func nextTestName(tests []Test, in ComposeInput) (key, title string, order int) {
	taken := map[string]bool{}
	for _, t := range tests {
		taken[t.Key] = true
		if t.Order >= order {
			order = t.Order + 1
		}
	}
	if order == 0 {
		order = 1
	}
	prefix, name := "luna-test", "Luna Test"
	if in.Kind.Mini() {
		prefix, name = "mini", "Mini Test"
	}
	n := 1
	for taken[fmt.Sprintf("%s-%02d", prefix, n)] {
		n++
	}
	key = fmt.Sprintf("%s-%02d", prefix, n)
	title = fmt.Sprintf("%s %02d", name, n)
	if in.Kind.Mini() {
		title = fmt.Sprintf("%s %02d (%d câu)", name, n, miniTotal(in.Kind))
	}
	if k := strings.ToLower(strings.TrimSpace(in.Key)); k != "" {
		key = k
	}
	if t := strings.TrimSpace(in.Title); t != "" {
		title = t
	}
	return key, title, order
}

// miniTotal is how many questions a Mini Test of kind k has.
func miniTotal(k TestKind) int {
	n := 0
	for _, q := range miniQuestions[k] {
		n += q
	}
	return n
}
