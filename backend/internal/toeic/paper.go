package toeic

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"sort"
)

// partCountDefault is how many questions (Part 1, 2, 5) or groups (the others) a part round has.
var partCountDefault = [8]int{0, 6, 10, 3, 3, 10, 2, 3}

// countsQuestions reports whether a part round counts questions (one per group) rather than groups.
func countsQuestions(p Part) bool { return p == 1 || p == 2 || p == 5 }

// paper is the questions an attempt is built with.
type paper struct {
	title       string
	groups      []Group
	questionIDs []string
	numbers     map[string]int
}

// parts returns the parts of the paper, in order.
func (p paper) parts() []Part {
	var out []Part
	for _, g := range p.groups {
		if !slices.Contains(out, g.Part) {
			out = append(out, g.Part)
		}
	}
	slices.Sort(out)
	return out
}

// buildPaper picks the groups of a new attempt.
func (s *Service) buildPaper(ctx context.Context, userID string, src Source, mode Mode, parts []Part) (paper, error) {
	switch src.Kind {
	case SourceTest:
		return s.testPaper(ctx, src.TestKey, mode, parts)
	case SourcePart:
		return s.partPaper(ctx, userID, src.Part, src.Count)
	case SourceRetry:
		return s.retryPaper(ctx, userID, src.FromAttempt)
	}
	return paper{}, invalid("source.kind", "Cách luyện này sẽ có ở đợt sau")
}

// testPaper is a fixed test, or the parts of it the learner ticked.
func (s *Service) testPaper(ctx context.Context, key string, mode Mode, parts []Part) (paper, error) {
	t, err := s.d.Tests.Get(ctx, key)
	if err != nil {
		return paper{}, err
	}
	if t.Status != StatusPublished {
		return paper{}, ErrNotFound
	}
	groups, err := s.groupsOf(ctx, t.Groups)
	if err != nil {
		return paper{}, err
	}
	numbers := testNumbers(t, groups)
	inTest := map[Part]bool{}
	for _, k := range t.Groups {
		if g, ok := groups[k]; ok {
			inTest[g.Part] = true
		}
	}
	p := paper{numbers: map[string]int{}}
	if mode == ModeTest {
		if len(inTest) < 7 {
			return paper{}, invalid("mode", "Đề chưa có đủ 7 Part: làm full test sẽ có ở đợt sau")
		}
		return paper{}, invalid("mode", "Làm full test sẽ có ở đợt sau")
	}
	for _, part := range parts {
		if !inTest[part] {
			return paper{}, invalid("parts", fmt.Sprintf("Đề chưa có Part %d", part))
		}
	}
	p.title = t.Title + " · Luyện tập"
	for _, k := range t.Groups {
		g, ok := groups[k]
		if !ok || !slices.Contains(parts, g.Part) {
			continue
		}
		p.groups = append(p.groups, g)
		for _, q := range g.Questions {
			id := QuestionID(g.Key, q.No)
			p.questionIDs = append(p.questionIDs, id)
			p.numbers[id] = numbers[id]
		}
	}
	return p, nil
}

// partPaper draws published groups of one part, the ones with fewest questions answered first.
func (s *Service) partPaper(ctx context.Context, userID string, part Part, count int) (paper, error) {
	if !part.Valid() {
		return paper{}, invalid("source.part", "Part phải từ 1 đến 7")
	}
	if count <= 0 {
		count = partCountDefault[part]
	}
	if count > 30 {
		return paper{}, invalid("source.count", "Tối đa 30")
	}
	pool, err := s.d.Groups.Published(ctx, part)
	if err != nil {
		return paper{}, err
	}
	if len(pool) == 0 {
		return paper{}, ErrNotEnough
	}
	answered, err := s.d.Attempts.AnsweredQuestions(ctx, userID)
	if err != nil {
		return paper{}, err
	}
	picked := pickFresh(pool, answered, count, s.newRand())
	p := paper{title: fmt.Sprintf("Part %d · luyện tập", part), groups: picked}
	p.number()
	return p, nil
}

// pickFresh shuffles the pool, then keeps the n groups with the fewest answered questions.
func pickFresh(pool []Group, answered map[string]bool, n int, rnd *rand.Rand) []Group {
	pool = slices.Clone(pool)
	rnd.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	seen := func(g Group) int {
		c := 0
		for _, q := range g.Questions {
			if answered[QuestionID(g.Key, q.No)] {
				c++
			}
		}
		return c
	}
	sort.SliceStable(pool, func(i, j int) bool { return seen(pool[i]) < seen(pool[j]) })
	return pool[:min(n, len(pool))]
}

// retryPaper is the wrong and skipped questions of a submitted attempt, with their whole groups
// shown and only those questions counted (FR-032).
func (s *Service) retryPaper(ctx context.Context, userID, fromID string) (paper, error) {
	if fromID == "" {
		return paper{}, invalid("source.fromAttempt", "Cần bài làm")
	}
	from, err := s.d.Attempts.Get(ctx, userID, fromID)
	if err != nil {
		return paper{}, err
	}
	if from.Status != AttemptSubmitted || from.Result == nil {
		return paper{}, &ErrAttemptInProgress{ID: from.ID}
	}
	groups, err := s.groupsOf(ctx, from.GroupKeys)
	if err != nil {
		return paper{}, err
	}
	p := paper{title: from.Title + " · làm lại câu sai", numbers: map[string]int{}}
	added := map[string]bool{}
	for _, id := range from.QuestionIDs {
		if from.Result.ByQuestion[id] == OutcomeRight {
			continue
		}
		key, no, ok := SplitQuestionID(id)
		g, found := groups[key]
		if !ok || !found {
			continue
		}
		if _, found := g.Question(no); !found {
			continue
		}
		if !added[key] {
			added[key] = true
			p.groups = append(p.groups, g)
		}
		p.questionIDs = append(p.questionIDs, id)
		p.numbers[id] = from.Numbers[id]
	}
	if len(p.questionIDs) == 0 {
		return paper{}, ErrNotEnough
	}
	return p, nil
}

// number counts every question of the groups from 1.
func (p *paper) number() {
	p.numbers = map[string]int{}
	p.questionIDs = nil
	n := 1
	for _, g := range p.groups {
		for _, q := range g.Questions {
			id := QuestionID(g.Key, q.No)
			p.questionIDs = append(p.questionIDs, id)
			p.numbers[id] = n
			n++
		}
	}
}

// --- what the learner sees ---

// PaperGroup is a group as the exam room shows it: no answer, evidence, explanation, translation
// or vocabulary (research R2).
type PaperGroup struct {
	Key       string          `json:"key"`
	Part      Part            `json:"part"`
	Number    int             `json:"number"`
	ImageURL  *string         `json:"imageUrl"`
	Docs      []PaperDoc      `json:"docs,omitempty"`
	Script    []PaperTurn     `json:"script,omitempty"`
	Graphic   *Graphic        `json:"graphic"`
	Questions []PaperQuestion `json:"questions"`
}

// PaperDoc is a text without its translation.
type PaperDoc struct {
	Kind     string  `json:"kind"`
	Title    string  `json:"title,omitempty"`
	Meta     []KV    `json:"meta"`
	Body     []Block `json:"body"`
	ImageURL *string `json:"imageUrl"`
}

// PaperTurn is a line of the script without its translation.
type PaperTurn struct {
	Speaker string `json:"speaker"`
	Accent  Accent `json:"accent"`
	Text    string `json:"text"`
}

// PaperQuestion is a question without its answer.
type PaperQuestion struct {
	ID      string   `json:"id"`
	No      int      `json:"no"`
	Number  int      `json:"number"`
	Prompt  string   `json:"prompt,omitempty"`
	Options []string `json:"options"`
}

// imageURL is where a picture of a group is served.
func imageURL(key, slot string) *string {
	u := "/api/toeic/images/" + key + "/" + slot
	return &u
}

// paperView turns the groups of an attempt into what the learner sees; only the questions of
// questionIDs are kept.
func (s *Service) paperView(ctx context.Context, groups []Group, questionIDs []string, numbers map[string]int) ([]PaperGroup, error) {
	keys := make([]string, len(groups))
	for i, g := range groups {
		keys[i] = g.Key
	}
	slots, err := s.d.Images.Has(ctx, keys)
	if err != nil {
		return nil, err
	}
	keep := make(map[string]bool, len(questionIDs))
	for _, id := range questionIDs {
		keep[id] = true
	}
	out := make([]PaperGroup, 0, len(groups))
	for _, g := range groups {
		pg := paperGroupOf(g, slots[g.Key])
		pg.Questions = pg.Questions[:0]
		for _, q := range g.Questions {
			id := QuestionID(g.Key, q.No)
			if !keep[id] {
				continue
			}
			pq := PaperQuestion{ID: id, No: q.No, Number: numbers[id], Prompt: q.Prompt, Options: q.Options}
			if pg.Number == 0 {
				pg.Number = pq.Number
			}
			pg.Questions = append(pg.Questions, pq)
		}
		if len(pg.Questions) > 0 {
			out = append(out, pg)
		}
	}
	return out, nil
}

// paperGroupOf is the learner's view of g; slots are the group's pictures.
func paperGroupOf(g Group, slots []string) PaperGroup {
	pg := PaperGroup{Key: g.Key, Part: g.Part, Graphic: g.Graphic, Questions: []PaperQuestion{}}
	if g.Part == 1 && slices.Contains(slots, SlotPhoto) {
		pg.ImageURL = imageURL(g.Key, SlotPhoto)
	}
	for i, d := range g.Docs {
		pd := PaperDoc{Kind: d.Kind, Title: d.Title, Meta: d.Meta, Body: d.Body}
		if pd.Meta == nil {
			pd.Meta = []KV{}
		}
		if slot := DocSlot(d, i); slices.Contains(slots, slot) {
			pd.ImageURL = imageURL(g.Key, slot)
		}
		pg.Docs = append(pg.Docs, pd)
	}
	for _, t := range g.Script {
		pg.Script = append(pg.Script, PaperTurn{Speaker: t.Speaker, Accent: t.Accent, Text: t.Text})
	}
	return pg
}

// attemptGroups loads the groups of an attempt in its order; a group removed since is left out.
func (s *Service) attemptGroups(ctx context.Context, a Attempt) ([]Group, map[string]Group, error) {
	byKey, err := s.groupsOf(ctx, a.GroupKeys)
	if err != nil {
		return nil, nil, err
	}
	out := make([]Group, 0, len(a.GroupKeys))
	for _, k := range a.GroupKeys {
		if g, ok := byKey[k]; ok {
			out = append(out, g)
		}
	}
	return out, byKey, nil
}

// errIsNotFound reports whether err is ErrNotFound.
func errIsNotFound(err error) bool { return errors.Is(err, ErrNotFound) }
