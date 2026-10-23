package curriculum

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/grammar"
)

// The draft of a level's course (added 2026-10-23): the AI writes its topics, then the lessons of
// each topic one request at a time; the admin edits it and approves it. Learners and lesson
// generation only ever read the approved course.

//go:embed suggestions.json
var suggestionsJSON []byte

// suggestions are the topics suggested per level with why they come there (the topic page's
// suggestions until 2026-10-23): what the AI starts from.
var suggestions = func() map[string][]ai.CurriculumIdea {
	var raw map[string][]struct {
		Name string `json:"name"`
		Why  string `json:"why"`
	}
	if err := json.Unmarshal(suggestionsJSON, &raw); err != nil {
		panic(fmt.Sprintf("curriculum: suggestions: %v", err))
	}
	out := make(map[string][]ai.CurriculumIdea, len(raw))
	for level, list := range raw {
		for _, s := range list {
			out[level] = append(out[level], ai.CurriculumIdea{Name: s.Name, Why: s.Why})
		}
	}
	return out
}()

// extraPerTopic is how many extra lessons the AI writes for a topic.
const extraPerTopic = MaxExtra

func points(ids []string) []ai.CurriculumPoint {
	out := make([]ai.CurriculumPoint, 0, len(ids))
	for _, id := range ids {
		if p, ok := grammar.Default().Get(id); ok {
			out = append(out, ai.CurriculumPoint{ID: p.ID, Title: p.TitleEn, Pattern: p.Pattern})
		}
	}
	return out
}

func levelPoints(level string) []ai.CurriculumPoint {
	var ids []string
	for _, p := range grammar.Default().ByLevel(level) {
		ids = append(ids, p.ID)
	}
	return points(ids)
}

// lowerCourses returns the approved courses of the levels below level.
func (s *Service) lowerCourses(ctx context.Context, level string) ([]*Plan, error) {
	var out []*Plan
	for _, l := range Levels {
		if l == level {
			break
		}
		p, err := s.approved(ctx, l)
		if err != nil {
			return nil, err
		}
		if p != nil {
			out = append(out, p)
		}
	}
	return out, nil
}

func (s *Service) writer() (ai.CurriculumWriter, error) {
	if s.Writer == nil {
		return nil, ErrNoAI
	}
	return s.Writer, nil
}

// DraftWithAI asks the AI for the topics of level's course, in one request, and keeps them as the
// level's draft, without lessons yet (WriteTopic writes them); it replaces a draft already there.
// A result that cannot be saved is sent back once with what is wrong.
func (s *Service) DraftWithAI(ctx context.Context, level string) (Board, error) {
	w, err := s.writer()
	if err != nil {
		return Board{}, err
	}
	if len(grammar.Default().ByLevel(level)) == 0 {
		return Board{}, ErrNoPlan
	}
	c, err := s.course(ctx, level)
	if err != nil && !errors.Is(err, ErrNoPlan) {
		return Board{}, err
	}
	c.Level = level
	cat, err := s.catalogue(ctx)
	if err != nil {
		return Board{}, err
	}
	req := ai.OutlineRequest{
		Level: level, Points: levelPoints(level), Suggestions: suggestions[level],
		MinTopics: MinTopics, MaxTopics: MaxTopics,
	}
	for _, t := range cat.byName {
		req.Topics = append(req.Topics, t.Name)
	}
	slices.Sort(req.Topics)
	lower, err := s.lowerCourses(ctx, level)
	if err != nil {
		return Board{}, err
	}
	for _, p := range lower {
		for _, t := range p.Topics {
			req.Earlier = append(req.Earlier, p.Level+": "+t.Name)
		}
	}
	var plan Plan
	for attempt := 0; attempt < 2; attempt++ {
		out, err := w.OutlineCurriculum(ctx, req)
		if err != nil {
			return Board{}, fmt.Errorf("%w: %w", ErrAIFailed, err)
		}
		plan = outlinePlan(level, out, cat)
		problems := outlineProblems(plan)
		if len(problems) == 0 {
			break
		}
		if attempt == 1 {
			return Board{}, &PlanError{Message: "AI soạn chưa đạt: " + strings.Join(problems, "; ")}
		}
		req.PreviousErrors = problems
	}
	c.Draft = &plan
	if err := s.save(ctx, c); err != nil {
		return Board{}, err
	}
	return s.Board(ctx, level)
}

// outlinePlan turns the AI's topics into a plan without lessons, linked to the catalogue's topics
// of the same name.
func outlinePlan(level string, out []ai.OutlineTopic, cat catalogue) Plan {
	p := Plan{Level: level}
	for _, o := range out {
		t := Topic{Name: strings.TrimSpace(o.Name), CanDoVi: strings.TrimSpace(o.CanDoVi), Grammar: o.Grammar}
		if found, ok := cat.byName[NameKey(t.Name)]; ok {
			t.Name, t.TopicID = found.Name, found.ID
		}
		p.Topics = append(p.Topics, t)
	}
	p.Normalize()
	return p
}

// outlineProblems lists, in English for the AI, what makes an outline unusable.
func outlineProblems(p Plan) []string {
	var out []string
	if n := len(p.Topics); n < MinTopics || n > MaxTopics {
		out = append(out, fmt.Sprintf("there are %d topics, write %d to %d", n, MinTopics, MaxTopics))
	}
	names := map[string]bool{}
	for _, t := range p.Topics {
		if t.Name == "" || t.CanDoVi == "" {
			out = append(out, fmt.Sprintf("topic %d has no name or no canDoVi", t.Number))
		}
		if names[NameKey(t.Name)] {
			out = append(out, fmt.Sprintf("topic %q is listed twice", t.Name))
		}
		names[NameKey(t.Name)] = true
		if len(t.Grammar) == 0 {
			out = append(out, fmt.Sprintf("topic %q has no grammar point", t.Name))
		}
		for _, id := range t.Grammar {
			if pt, ok := grammar.Default().Get(id); !ok || pt.Level != p.Level {
				out = append(out, fmt.Sprintf("%q is not a grammar point id of the list", id))
			}
		}
	}
	return out
}

// WriteTopic asks the AI for the lessons of the draft's topic number (from 1), in one request,
// replacing the ones it has.
//
// With extraOnly the topic keeps its lessons and the AI only adds the extra lessons it lacks, up
// to MaxExtra (added 2026-10-23: A1 was approved with 2–3 extra lessons, and rewriting a topic
// would leave its lessons out of their slots).
func (s *Service) WriteTopic(ctx context.Context, level string, number int, extraOnly bool) (Board, error) {
	w, err := s.writer()
	if err != nil {
		return Board{}, err
	}
	c, err := s.course(ctx, level)
	if errors.Is(err, ErrNoPlan) || (err == nil && c.Draft == nil) {
		return Board{}, ErrNoDraftPlan
	}
	if err != nil {
		return Board{}, err
	}
	if number < 1 || number > len(c.Draft.Topics) {
		return Board{}, ErrSlotNotFound
	}
	t := &c.Draft.Topics[number-1]
	req := ai.SlotsRequest{
		Level:  level,
		Topic:  ai.OutlineTopic{Name: t.Name, CanDoVi: t.CanDoVi, Grammar: t.Grammar},
		Points: points(t.Grammar), Main: MainPerTopic, Extra: extraPerTopic,
	}
	var keep []Slot
	if extraOnly {
		missing := MainPerTopic + MaxExtra - len(t.Slots)
		if len(t.Slots) < MainPerTopic || missing <= 0 {
			return Board{}, &PlanError{Message: fmt.Sprintf("Chủ đề “%s” cần có đủ %d bài chính và chưa đủ %d bài thêm.", t.Name, MainPerTopic, MaxExtra)}
		}
		keep, req.Main, req.Extra = t.Slots, 0, missing
	}
	lower, err := s.lowerCourses(ctx, level)
	if err != nil {
		return Board{}, err
	}
	for _, p := range append(lower, c.Draft) {
		for _, other := range p.Topics {
			if p == c.Draft && other.Number == number && !extraOnly {
				continue
			}
			for _, sl := range other.Slots {
				req.Avoid = append(req.Avoid, sl.Title)
			}
		}
	}
	out, err := w.WriteTopicSlots(ctx, req)
	if err != nil {
		return Board{}, fmt.Errorf("%w: %w", ErrAIFailed, err)
	}
	slots := topicSlots(level, *t, keep, out)
	if need := MainPerTopic + MinExtraHard; len(slots) < need || len(slots) == len(keep) {
		return Board{}, &PlanError{Message: fmt.Sprintf("AI chỉ soạn được %d bài dùng được cho chủ đề “%s”, vui lòng thử lại.", len(slots)-len(keep), t.Name)}
	}
	t.Slots = slots
	c.Draft.Normalize()
	if err := s.save(ctx, c); err != nil {
		return Board{}, err
	}
	return s.Board(ctx, level)
}

// topicSlots adds to keep the AI's lessons that can be used: a title (once in the topic) and an
// idea; points of another level are dropped and a lesson left without one takes the topic's first
// point.
func topicSlots(level string, t Topic, keep []Slot, out []ai.SlotDraft) []Slot {
	slots := slices.Clone(keep)
	titles := map[string]bool{}
	for _, sl := range keep {
		titles[TitleKey(sl.Title)] = true
	}
	for _, d := range out {
		title, idea := strings.TrimSpace(d.Title), strings.TrimSpace(d.Idea)
		if title == "" || idea == "" || titles[TitleKey(title)] {
			continue
		}
		titles[TitleKey(title)] = true
		var ids []string
		for _, id := range d.Grammar {
			if pt, ok := grammar.Default().Get(id); ok && pt.Level == level && !slices.Contains(ids, id) {
				ids = append(ids, id)
			}
		}
		if len(ids) == 0 {
			ids = []string{t.Grammar[0]}
		}
		slots = append(slots, Slot{Title: title, Idea: idea, Grammar: ids})
	}
	if max := MainPerTopic + MaxExtra; len(slots) > max {
		slots = slots[:max]
	}
	return slots
}

// SaveDraft replaces the draft of level with the admin's edit.
func (s *Service) SaveDraft(ctx context.Context, level string, p Plan) (Board, error) {
	c, err := s.course(ctx, level)
	if errors.Is(err, ErrNoPlan) || (err == nil && c.Draft == nil) {
		return Board{}, ErrNoDraftPlan
	}
	if err != nil {
		return Board{}, err
	}
	p.Level = level
	p.Normalize()
	if err := p.Validate(grammar.Default()); err != nil {
		return Board{}, &PlanError{Message: err.Error()}
	}
	c.Draft = &p
	if err := s.save(ctx, c); err != nil {
		return Board{}, err
	}
	return s.Board(ctx, level)
}

// EditApproved starts a draft from the approved course of level ("Sửa giáo trình").
func (s *Service) EditApproved(ctx context.Context, level string) (Board, error) {
	c, err := s.course(ctx, level)
	if err != nil {
		return Board{}, err
	}
	if c.Approved == nil {
		return Board{}, ErrNoPlan
	}
	if c.Draft == nil {
		data, err := json.Marshal(c.Approved)
		if err != nil {
			return Board{}, fmt.Errorf("curriculum: copy: %w", err)
		}
		var p Plan
		if err := json.Unmarshal(data, &p); err != nil {
			return Board{}, fmt.Errorf("curriculum: copy: %w", err)
		}
		p.Normalize()
		c.Draft = &p
		if err := s.save(ctx, c); err != nil {
			return Board{}, err
		}
	}
	return s.Board(ctx, level)
}

// DiscardDraft drops the draft of level; the approved course, if any, stays.
func (s *Service) DiscardDraft(ctx context.Context, level string) (Board, error) {
	c, err := s.course(ctx, level)
	if errors.Is(err, ErrNoPlan) || (err == nil && c.Draft == nil) {
		return Board{}, ErrNoDraftPlan
	}
	if err != nil {
		return Board{}, err
	}
	c.Draft = nil
	if err := s.save(ctx, c); err != nil {
		return Board{}, err
	}
	return s.Board(ctx, level)
}

// Approve makes the draft of level its course: it creates the topics the catalogue lacks, and the
// draft replaces the approved course. The topics' order and roadmaps are left to Apply.
func (s *Service) Approve(ctx context.Context, level string) (Board, error) {
	c, err := s.course(ctx, level)
	if errors.Is(err, ErrNoPlan) || (err == nil && c.Draft == nil) {
		return Board{}, ErrNoDraftPlan
	}
	if err != nil {
		return Board{}, err
	}
	p := c.Draft
	if err := p.Validate(grammar.Default()); err != nil {
		return Board{}, &PlanError{Message: err.Error()}
	}
	if err := p.Ready(); err != nil {
		return Board{}, &PlanError{Message: err.Error()}
	}
	if _, err := s.createTopics(ctx, p); err != nil {
		return Board{}, err
	}
	c.Approved, c.Draft, c.ApprovedAt = p, nil, s.now()
	if err := s.save(ctx, c); err != nil {
		return Board{}, err
	}
	return s.Board(ctx, level)
}
