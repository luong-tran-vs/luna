package toeic

import (
	"context"
	"fmt"
	"math/rand/v2"
	"slices"
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
	// Levels is how many of the picked questions are of each level.
	Levels map[int]int `json:"levels,omitempty"`
	// Note says when the part could not follow the real exam's layout (not enough graphic groups in
	// Part 3/4, double or triple passages in Part 7) and was made by question count alone.
	Note string `json:"note,omitempty"`
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

// levelShare is the share of a part's questions each level gets in a composed test, so that every
// test mixes easy and hard questions alike (as a real test does) instead of a random draw.
var levelShare = map[int]float64{450: 0.2, 600: 0.3, 750: 0.3, 900: 0.2}

// slot is a share of a part that comes from one kind of group: Need questions of the groups that
// fit (Need < 0 is what the part's other slots leave).
type slot struct {
	fits func(Group) bool
	need int
}

// layouts is how many questions of a part come from graphic groups (Part 3/4) or from double and
// triple passages (Part 7) in each kind of test: the real exam's layout for a full test (Part 3:
// 3 graphic conversations, Part 4: 2 graphic talks, Part 7: 2 double and 3 triple passages) and a
// smaller copy of it for a Mini Test.
var layouts = map[TestKind]map[Part][]int{
	KindFull:    {3: {9}, 4: {6}, 7: {10, 15}},
	KindMini100: {3: {3}, 4: {3}, 7: {5, 5}},
	KindMini50:  {3: {3}, 4: {3}, 7: {5, 0}},
}

// structureOf is how a test of the kind builds part p and the note given when the bank cannot follow
// it; the other parts take any group.
func structureOf(kind TestKind, p Part) ([]slot, string) {
	l := layouts[kind][p]
	if len(l) == 0 {
		return nil, ""
	}
	withGraphic := func(g Group) bool { return g.Graphic != nil }
	plain := func(g Group) bool { return g.Graphic == nil }
	docs := func(n int) func(Group) bool { return func(g Group) bool { return len(g.Docs) == n } }
	if p == 7 {
		return []slot{{docs(2), l[0]}, {docs(3), l[1]}, {docs(1), -1}},
			fmt.Sprintf("Chưa đủ cụm 2 bài (cần %d câu) hoặc 3 bài (cần %d câu) nên chưa theo đúng cấu trúc đề thật: đã ghép theo số câu", l[0], l[1])
	}
	return []slot{{withGraphic, l[0]}, {plain, -1}},
		fmt.Sprintf("Chưa đủ cụm có biểu đồ (cần %d câu) nên chưa theo đúng cấu trúc đề thật: đã ghép theo số câu", l[0])
}

// pickPart chooses the groups of one part: questions spread over the levels as levelShare says,
// unused groups first (then the least used, random among equals), in the layout of the real exam
// for a full test (Part 3/4: graphic groups; Part 7: double and triple passages). When the bank cannot
// follow the layout the part is made by question count alone and Note says so. A Mini Test may be
// off by one group when the exact number cannot be made. The groups come easy first, and in Part 3/4
// the graphic groups and in Part 7 the multi-passage groups last.
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
	slots, note := structureOf(kind, p)
	picked, ok := pickTargets(targets, func(target int) ([]Group, bool) { return pickSlots(slots, cands, uses, target, rnd) })
	if !ok && len(slots) > 0 {
		picked, ok = pickTargets(targets, func(target int) ([]Group, bool) { return pickSlots(nil, cands, uses, target, rnd) })
		if ok {
			cp.Note = note
		}
	}
	if !ok {
		return nil, cp
	}
	sort.SliceStable(picked, func(i, j int) bool {
		a, b := picked[i], picked[j]
		if ra, rb := layoutRank(a), layoutRank(b); ra != rb {
			return ra < rb
		}
		if a.Level != b.Level {
			return a.Level < b.Level
		}
		return a.Key < b.Key
	})
	cp.Levels = map[int]int{}
	keys := make([]string, 0, len(picked))
	for _, g := range picked {
		keys = append(keys, g.Key)
		cp.Have += len(g.Questions)
		cp.Levels[g.Level] += len(g.Questions)
		if uses[g.Key] > 0 {
			cp.Reused++
		}
	}
	cp.Groups = len(keys)
	cp.Filled = true
	return keys, cp
}

// pickTargets returns what pick makes for the first target it can make.
func pickTargets(targets []int, pick func(int) ([]Group, bool)) ([]Group, bool) {
	for _, t := range targets {
		if t > 0 {
			if gs, ok := pick(t); ok {
				return gs, true
			}
		}
	}
	return nil, false
}

// layoutRank orders the groups of a part as the exam does: graphic groups after the others (Part
// 3/4), and passages of one, then two, then three documents (Part 7).
func layoutRank(g Group) int {
	switch g.Part {
	case 3, 4:
		if g.Graphic != nil {
			return 1
		}
	case 7:
		return len(g.Docs)
	}
	return 0
}

// pickSlots fills each slot from the groups that fit it; without slots every group fits.
func pickSlots(slots []slot, cands []Group, uses map[string]int, target int, rnd *rand.Rand) ([]Group, bool) {
	if len(slots) == 0 {
		return pickByLevel(cands, uses, levelTargets(target), rnd)
	}
	rest := target
	for _, s := range slots {
		if s.need > 0 {
			rest -= s.need
		}
	}
	if rest < 0 {
		return nil, false
	}
	// The fixed slots (made of few big groups) come first; the rest slot then makes up the levels
	// they could not give their share, so that the part as a whole keeps the levelShare mix.
	partWant := levelTargets(target)
	got := map[int]int{}
	var out []Group
	for _, s := range slots {
		n := s.need
		want := levelTargets(max(n, 0))
		if n < 0 {
			n = rest
			weights := map[int]float64{}
			for i, lv := range Levels {
				weights[lv] = float64(max(0, partWant[i]-got[lv]))
			}
			want = splitByWeight(n, weights)
		}
		if n == 0 {
			continue
		}
		var fit []Group
		for _, g := range cands {
			if s.fits(g) {
				fit = append(fit, g)
			}
		}
		gs, ok := pickByLevel(fit, uses, want, rnd)
		if !ok {
			return nil, false
		}
		for _, g := range gs {
			got[g.Level] += len(g.Questions)
		}
		out = append(out, gs...)
	}
	return out, true
}

// pickByLevel returns groups whose questions add up to the sum of wants (by index of Levels), each
// level getting as near its want as the group sizes allow. A pass takes the levels one by one (a
// level that cannot make its want exactly passes the difference to the next one, and what is left
// at the end may come from any level); several passes in random orders are tried, so that a share
// made of few big groups (two double passages) does not always go to the same levels. Of the passes
// with the fewest reused groups, one is kept at random among those whose distance to the wants is
// within one group of the nearest (keeping only the nearest would always give the big groups to the
// middle levels). When no pass works it takes the cheapest groups of any level. cands is in order of
// preference.
func pickByLevel(cands []Group, uses map[string]int, wants []int, rnd *rand.Rand) ([]Group, bool) {
	want := map[int]int{}
	target := 0
	for i, n := range wants {
		want[Levels[i]] = n
		target += n
	}
	maxSize := 0
	for _, g := range cands {
		maxSize = max(maxSize, len(g.Questions))
	}
	type pass struct {
		gs          []Group
		reused, far int
	}
	const passes = 12
	var done []pass
	for range passes {
		order := slices.Clone(Levels)
		rnd.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })
		gs, ok := levelPass(cands, uses, want, order, target)
		if !ok {
			continue
		}
		p := pass{gs: gs}
		got := map[int]int{}
		for _, g := range gs {
			got[g.Level] += len(g.Questions)
			if uses[g.Key] > 0 {
				p.reused++
			}
		}
		for _, lv := range Levels {
			p.far += abs(got[lv] - want[lv])
		}
		done = append(done, p)
	}
	if len(done) > 0 {
		reused, dist := done[0].reused, done[0].far
		for _, p := range done {
			if p.reused < reused || (p.reused == reused && p.far < dist) {
				reused, dist = p.reused, p.far
			}
		}
		var near []pass
		for _, p := range done {
			if p.reused == reused && p.far <= dist+maxSize {
				near = append(near, p)
			}
		}
		return near[rnd.IntN(len(near))].gs, true
	}
	idx, ok := cheapestSubset(cands, uses, target)
	if !ok {
		return nil, false
	}
	out := make([]Group, 0, len(idx))
	for _, j := range idx {
		out = append(out, cands[j])
	}
	return out, true
}

// levelPass is one pass of pickByLevel with the levels in the order given.
func levelPass(cands []Group, uses map[string]int, want map[int]int, order []int, target int) ([]Group, bool) {
	taken := map[string]bool{}
	var out []Group
	left, carry, maxSize := target, 0, 0
	for _, g := range cands {
		maxSize = max(maxSize, len(g.Questions))
	}
	for i, lv := range order {
		w := min(want[lv]+carry, left)
		if i == len(order)-1 {
			w = left
		}
		if w <= 0 {
			carry = w
			continue
		}
		var of []Group
		for _, g := range cands {
			if g.Level == lv {
				of = append(of, g)
			}
		}
		got := 0
		for d := 0; d <= maxSize && got == 0; d++ {
			tries := []int{w}
			if d > 0 {
				tries = []int{w - d, w + d}
			}
			for _, t := range tries {
				if t <= 0 || t > left {
					continue
				}
				if idx, ok := cheapestSubset(of, uses, t); ok {
					for _, j := range idx {
						out = append(out, of[j])
						taken[of[j].Key] = true
					}
					got = t
					break
				}
			}
		}
		carry, left = w-got, left-got
	}
	if left > 0 {
		var rest []Group
		for _, g := range cands {
			if !taken[g.Key] {
				rest = append(rest, g)
			}
		}
		idx, ok := cheapestSubset(rest, uses, left)
		if !ok {
			return nil, false
		}
		for _, j := range idx {
			out = append(out, rest[j])
		}
	}
	return out, true
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// levelTargets splits target questions over Levels by levelShare.
func levelTargets(target int) []int { return splitByWeight(target, levelShare) }

// splitByWeight splits target questions over Levels in proportion to the weights of the levels
// (largest remainders get the rest); levelShare when no level weighs anything.
func splitByWeight(target int, weights map[int]float64) []int {
	out := make([]int, len(Levels))
	if target <= 0 {
		return out
	}
	total := 0.0
	for _, lv := range Levels {
		total += weights[lv]
	}
	if total <= 0 {
		weights, total = levelShare, 1
	}
	type rem struct {
		i int
		r float64
	}
	var rems []rem
	sum := 0
	for i, lv := range Levels {
		x := float64(target) * weights[lv] / total
		out[i] = int(x)
		sum += out[i]
		rems = append(rems, rem{i, x - float64(out[i])})
	}
	sort.SliceStable(rems, func(a, b int) bool { return rems[a].r > rems[b].r })
	for k := 0; sum < target; k++ {
		out[rems[k%len(rems)].i]++
		sum++
	}
	return out
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
