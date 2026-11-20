package toeic

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
)

// miniQuestions is how many questions each part has in a Mini Test (research R9).
var miniQuestions = map[TestKind][8]int{
	KindMini50:  {0, 2, 6, 9, 9, 8, 4, 12},
	KindMini100: {0, 3, 12, 21, 15, 15, 8, 26},
}

// groupSize is one group's worth of questions in each part: how far a Mini Test may be off.
var groupSize = [8]int{0, 1, 1, 3, 3, 1, 4, 5}

// needOf returns the questions part p needs in a test of kind k.
func needOf(k TestKind, p Part) int {
	if m, ok := miniQuestions[k]; ok {
		return m[p]
	}
	return PartQuestions[p]
}

// partWhole reports whether have questions fill part p of a test of kind k.
func partWhole(k TestKind, p Part, have int) bool {
	need := needOf(k, p)
	if k.Mini() {
		return have >= need-groupSize[p] && have <= need+groupSize[p]
	}
	return have == need
}

// partStart is the number of the first question of each part in a full test.
func partStart(p Part) int {
	n := 1
	for q := Part(1); q < p; q++ {
		n += PartQuestions[q]
	}
	return n
}

// testNumbers numbers the questions of a test: a full test uses the exam's numbers (Part 5 starts
// at 101 even while Part 1–4 are missing), a Mini Test counts from 1.
func testNumbers(t Test, groups map[string]Group) map[string]int {
	out := map[string]int{}
	next := map[Part]int{}
	seq := 1
	for _, key := range t.Groups {
		g, ok := groups[key]
		if !ok {
			continue
		}
		for _, q := range g.Questions {
			id := QuestionID(g.Key, q.No)
			if t.Kind == KindFull {
				if next[g.Part] == 0 {
					next[g.Part] = partStart(g.Part)
				}
				out[id] = next[g.Part]
				next[g.Part]++
			} else {
				out[id] = seq
				seq++
			}
		}
	}
	return out
}

// check says what a test misses; groups are the test's groups that exist, photos the slots with a
// picture by group key.
func check(t Test, groups map[string]Group, photos map[string][]string) TestCheck {
	c := TestCheck{MissingGroups: []string{}, DraftGroups: []string{}, MissingPhotos: []PhotoGap{}}
	have := [8]int{}
	numbers := testNumbers(t, groups)
	var listening []Group
	for _, key := range t.Groups {
		g, ok := groups[key]
		if !ok {
			c.MissingGroups = append(c.MissingGroups, key)
			continue
		}
		have[g.Part] += len(g.Questions)
		if g.Status != StatusPublished {
			c.DraftGroups = append(c.DraftGroups, key)
		}
		if g.Graphic != nil {
			c.Graphics++
		}
		c.Docs += len(g.Docs)
		if g.Part == 1 && !slices.Contains(photos[key], SlotPhoto) {
			c.MissingPhotos = append(c.MissingPhotos, PhotoGap{GroupKey: key, Number: numbers[QuestionID(key, 1)]})
		}
		if g.Part <= 4 {
			listening = append(listening, g)
		}
	}
	c.ListeningMinutes = estimateListening(listening)
	c.Complete = true
	partsOK := true
	for p := Part(1); p <= 7; p++ {
		c.Parts = append(c.Parts, PartCheck{Part: p, Have: have[p], Need: needOf(t.Kind, p)})
		whole := partWhole(t.Kind, p, have[p])
		if !whole {
			c.Complete = false
		}
		// While the listening is not written yet (batch 1), a test of Part 5–7 alone may be published.
		if (have[p] > 0 || p >= 5) && !whole {
			partsOK = false
		}
	}
	c.Ready = partsOK && len(c.MissingGroups) == 0 && len(c.DraftGroups) == 0 && len(c.MissingPhotos) == 0
	return c
}

// TestInput is a test as the admin writes it (a file of content/toeic/tests/).
type TestInput struct {
	Key    string   `json:"key"`
	Title  string   `json:"title"`
	Kind   TestKind `json:"kind"`
	Order  int      `json:"order"`
	Groups []string `json:"groups"`
}

// TestAdmin is a test with what it misses.
type TestAdmin struct {
	Test  Test
	Check TestCheck
	// Groups are the test's groups that exist, in the test's order.
	Groups []Group
}

// CreateTest stores a new draft test; ErrExists when the key is taken.
func (s *Service) CreateTest(ctx context.Context, in TestInput) (TestAdmin, error) {
	t, err := s.testOf(ctx, in)
	if err != nil {
		return TestAdmin{}, err
	}
	if _, err := s.d.Tests.Get(ctx, t.Key); err == nil {
		return TestAdmin{}, ErrExists
	} else if !errors.Is(err, ErrNotFound) {
		return TestAdmin{}, err
	}
	now := s.now()
	t.Status, t.CreatedAt, t.UpdatedAt = StatusDraft, now, now
	if err := s.d.Tests.Save(ctx, t); err != nil {
		return TestAdmin{}, err
	}
	return s.TestAdmin(ctx, t.Key)
}

// UpdateTest replaces a test that is not published.
func (s *Service) UpdateTest(ctx context.Context, key string, in TestInput) (TestAdmin, error) {
	old, err := s.d.Tests.Get(ctx, key)
	if err != nil {
		return TestAdmin{}, err
	}
	if old.Status == StatusPublished {
		return TestAdmin{}, invalid("status", "Đề đã đăng: ẩn đề trước khi sửa")
	}
	in.Key = key
	t, err := s.testOf(ctx, in)
	if err != nil {
		return TestAdmin{}, err
	}
	t.ID, t.Status, t.CreatedAt, t.UpdatedAt = old.ID, old.Status, old.CreatedAt, s.now()
	if err := s.d.Tests.Save(ctx, t); err != nil {
		return TestAdmin{}, err
	}
	return s.TestAdmin(ctx, key)
}

// testOf checks a test input and orders its groups by part.
func (s *Service) testOf(ctx context.Context, in TestInput) (Test, error) {
	t := Test{Key: strings.ToLower(strings.TrimSpace(in.Key)), Title: strings.TrimSpace(in.Title), Kind: in.Kind, Order: in.Order}
	f := map[string]string{}
	if !keyPattern.MatchString(t.Key) {
		f["key"] = "Mã đề gồm 3–48 ký tự: chữ thường, số, gạch ngang"
	}
	if t.Title == "" || len([]rune(t.Title)) > 80 {
		f["title"] = "Tên đề cần 1–80 ký tự"
	}
	if !t.Kind.Valid() {
		f["kind"] = "Loại đề phải là full, mini50 hoặc mini100"
	}
	if t.Order < 0 {
		f["order"] = "Thứ tự không được âm"
	}
	seen := map[string]bool{}
	for _, k := range in.Groups {
		k = strings.ToLower(strings.TrimSpace(k))
		if k == "" || seen[k] {
			f["groups"] = "Danh sách cụm có mã trống hoặc trùng"
			continue
		}
		seen[k] = true
		t.Groups = append(t.Groups, k)
	}
	if len(t.Groups) == 0 && f["groups"] == "" {
		f["groups"] = "Đề cần ít nhất một cụm"
	}
	if len(f) > 0 {
		return Test{}, &ValidationError{Fields: f}
	}
	gs, err := s.d.Groups.ByKeys(ctx, t.Groups)
	if err != nil {
		return Test{}, err
	}
	byKey := make(map[string]Group, len(gs))
	for _, g := range gs {
		byKey[g.Key] = g
	}
	var missing []string
	for _, k := range t.Groups {
		if _, ok := byKey[k]; !ok {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return Test{}, invalid("groups", "Chưa có cụm: "+strings.Join(missing, ", "))
	}
	sort.SliceStable(t.Groups, func(i, j int) bool { return byKey[t.Groups[i]].Part < byKey[t.Groups[j]].Part })
	return t, nil
}

// TestAdmin returns a test with its groups and what it misses.
func (s *Service) TestAdmin(ctx context.Context, key string) (TestAdmin, error) {
	t, err := s.d.Tests.Get(ctx, key)
	if err != nil {
		return TestAdmin{}, err
	}
	return s.testAdminOf(ctx, t)
}

func (s *Service) testAdminOf(ctx context.Context, t Test) (TestAdmin, error) {
	groups, err := s.groupsOf(ctx, t.Groups)
	if err != nil {
		return TestAdmin{}, err
	}
	photos, err := s.d.Images.Has(ctx, t.Groups)
	if err != nil {
		return TestAdmin{}, err
	}
	out := TestAdmin{Test: t, Check: check(t, groups, photos)}
	for _, k := range t.Groups {
		if g, ok := groups[k]; ok {
			out.Groups = append(out.Groups, g)
		}
	}
	return out, nil
}

// ListTestsAdmin returns every test with what it misses.
func (s *Service) ListTestsAdmin(ctx context.Context) ([]TestAdmin, error) {
	tests, err := s.d.Tests.List(ctx, false)
	if err != nil {
		return nil, err
	}
	out := make([]TestAdmin, 0, len(tests))
	for _, t := range tests {
		ta, err := s.testAdminOf(ctx, t)
		if err != nil {
			return nil, err
		}
		ta.Groups = nil
		out = append(out, ta)
	}
	return out, nil
}

// SetTestStatus publishes, hides or unpublishes a test; publishing needs a ready test.
func (s *Service) SetTestStatus(ctx context.Context, key string, st Status) (TestAdmin, error) {
	if !st.Valid() {
		return TestAdmin{}, invalid("status", "Trạng thái phải là draft, published hoặc hidden")
	}
	ta, err := s.TestAdmin(ctx, key)
	if err != nil {
		return TestAdmin{}, err
	}
	if st == StatusPublished && !ta.Check.Ready {
		return TestAdmin{}, &ErrTestNotReady{Check: ta.Check}
	}
	if err := s.d.Tests.SetStatus(ctx, key, st); err != nil {
		return TestAdmin{}, err
	}
	ta.Test.Status = st
	return ta, nil
}

// groupsOf loads groups by key.
func (s *Service) groupsOf(ctx context.Context, keys []string) (map[string]Group, error) {
	if len(keys) == 0 {
		return map[string]Group{}, nil
	}
	gs, err := s.d.Groups.ByKeys(ctx, keys)
	if err != nil {
		return nil, err
	}
	out := make(map[string]Group, len(gs))
	for _, g := range gs {
		out[g.Key] = g
	}
	return out, nil
}

// --- groups (admin) ---

// GroupRow is a group in the admin list.
type GroupRow struct {
	Key       string   `json:"key"`
	Part      Part     `json:"part"`
	Level     int      `json:"level"`
	Questions int      `json:"questions"`
	Tags      []string `json:"tags"`
	Status    Status   `json:"status"`
	Source    string   `json:"source"`
	Version   int      `json:"version"`
	// NeedsPhoto is true for Part 1; HasPhoto tells whether it is drawn.
	NeedsPhoto bool `json:"needsPhoto"`
	HasPhoto   bool `json:"hasPhoto"`
	// Pictures and PictureSlots are how many of the group's picture slots have a picture (Part 1, 6, 7).
	Pictures     int      `json:"pictures"`
	PictureSlots int      `json:"pictureSlots"`
	Tests        []string `json:"tests"`
	Title        string   `json:"title"`
}

// ListGroups returns the admin list of groups.
func (s *Service) ListGroups(ctx context.Context, f GroupFilter) ([]GroupRow, error) {
	if f.Part != 0 && !f.Part.Valid() {
		return nil, invalid("part", "Part phải từ 1 đến 7")
	}
	if f.Status != "" && !f.Status.Valid() {
		return nil, invalid("status", "Trạng thái không hợp lệ")
	}
	gs, err := s.d.Groups.List(ctx, f)
	if err != nil {
		return nil, err
	}
	tests, err := s.d.Tests.List(ctx, false)
	if err != nil {
		return nil, err
	}
	inTests := map[string][]string{}
	for _, t := range tests {
		for _, k := range t.Groups {
			inTests[k] = append(inTests[k], t.Key)
		}
	}
	var withSlots []string
	for _, g := range gs {
		if len(ImageSlots(g)) > 0 {
			withSlots = append(withSlots, g.Key)
		}
	}
	photos := map[string][]string{}
	if len(withSlots) > 0 {
		if photos, err = s.d.Images.Has(ctx, withSlots); err != nil {
			return nil, err
		}
	}
	out := make([]GroupRow, len(gs))
	for i, g := range gs {
		out[i] = GroupRow{
			Key: g.Key, Part: g.Part, Level: g.Level, Questions: len(g.Questions), Tags: groupTags(g), Status: g.Status,
			Source: g.Source, Version: g.Version, NeedsPhoto: g.Part == 1, HasPhoto: slices.Contains(photos[g.Key], SlotPhoto),
			Tests: inTests[g.Key], Title: groupTitle(g),
		}
		for _, slot := range ImageSlots(g) {
			out[i].PictureSlots++
			if slices.Contains(photos[g.Key], slot) {
				out[i].Pictures++
			}
		}
		if out[i].Tests == nil {
			out[i].Tests = []string{}
		}
	}
	return out, nil
}

// GetGroup returns a whole group (admin preview).
func (s *Service) GetGroup(ctx context.Context, key string) (Group, error) {
	gs, err := s.d.Groups.ByKeys(ctx, []string{key})
	if err != nil {
		return Group{}, err
	}
	if len(gs) == 0 {
		return Group{}, ErrNotFound
	}
	return gs[0], nil
}

// SetGroupStatus publishes, hides or unpublishes a group.
func (s *Service) SetGroupStatus(ctx context.Context, key string, st Status) error {
	if !st.Valid() {
		return invalid("status", "Trạng thái phải là draft, published hoặc hidden")
	}
	return s.d.Groups.SetStatus(ctx, key, st)
}

// SetGroupsStatus publishes (or hides) many groups at once; unknown keys are reported.
func (s *Service) SetGroupsStatus(ctx context.Context, keys []string, st Status) (int, error) {
	if !st.Valid() {
		return 0, invalid("status", "Trạng thái phải là draft, published hoặc hidden")
	}
	n := 0
	for _, k := range keys {
		err := s.d.Groups.SetStatus(ctx, k, st)
		if errors.Is(err, ErrNotFound) {
			continue
		}
		if err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

// groupTags returns the tags of a group's questions, each once, in order.
func groupTags(g Group) []string {
	out := []string{}
	for _, q := range g.Questions {
		for _, t := range q.Tags {
			if !slices.Contains(out, t) {
				out = append(out, t)
			}
		}
	}
	return out
}

// groupTitle is a short line telling a group apart: the first text's title, the question or the scene.
func groupTitle(g Group) string {
	switch {
	case len(g.Docs) > 0 && g.Docs[0].Title != "":
		return g.Docs[0].Title
	case len(g.Questions) > 0 && g.Questions[0].Prompt != "" && (g.Part == 2 || g.Part == 5):
		return g.Questions[0].Prompt
	case g.Scene != "":
		return g.Scene
	case len(g.Script) > 0:
		return g.Script[0].Text
	}
	return fmt.Sprintf("Part %d", g.Part)
}
