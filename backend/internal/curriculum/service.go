package curriculum

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/luongtran/luna/backend/internal/grammar"
	"github.com/luongtran/luna/backend/internal/lesson"
	"github.com/luongtran/luna/backend/internal/topic"
)

// Errors of the admin actions.
var (
	ErrNoPlan       = errors.New("curriculum: no course for this level")
	ErrSlotNotFound = errors.New("curriculum: slot not found")
	// ErrNotApplied: the slot's topic is not in the catalogue yet (apply the course first).
	ErrNotApplied = errors.New("curriculum: topic not created yet")
	ErrSlotFilled = errors.New("curriculum: slot already has a lesson")
	// ErrNoDraft: the AI returned no usable lesson.
	ErrNoDraft = errors.New("curriculum: no lesson generated")
)

// Source and licence of generated lessons, as the admin's generate screen saves them.
const (
	aiSource  = "AI sinh"
	aiLicense = "Nội dung do AI tạo"
)

// Words of a generated lesson and target words per lesson, by level (as the admin's defaults).
var (
	lessonWords = map[string]int{"A1": 120, "A2": 160}
	targetWords = map[string]int{"A1": 8, "A2": 8}
)

// Topics is what the course needs of the topic catalogue.
type Topics interface {
	All(ctx context.Context) ([]topic.Topic, error)
	Create(ctx context.Context, in topic.Input) (topic.Summary, error)
	SetOrder(ctx context.Context, level string, topicIDs []string) ([]topic.Summary, error)
	Roadmap(ctx context.Context, id, level string) (topic.Roadmap, error)
	SetRoadmap(ctx context.Context, id, level string, lessonIDs []string) (topic.Roadmap, error)
	WordPlan(ctx context.Context, id, level string, count, perLesson int) (topic.TargetPlan, error)
}

// Lessons is what the course needs of the lessons.
type Lessons interface {
	Generate(ctx context.Context, topicID string, in lesson.GenerateInput) (lesson.GenerateResult, error)
	Create(ctx context.Context, in lesson.Input) (lesson.Lesson, error)
	Get(ctx context.Context, id string) (lesson.Lesson, error)
	Delete(ctx context.Context, id string) error
}

// GrammarLessons tells which grammar points have a published lesson.
type GrammarLessons interface {
	AdminList(ctx context.Context) ([]grammar.LessonSummary, error)
}

// Service runs the admin side of the course.
type Service struct {
	Topics  Topics
	Lessons Lessons
	Grammar GrammarLessons
}

// GrammarView is a grammar point of a topic, with whether its grammar lesson is published.
type GrammarView struct {
	ID        string
	TitleVi   string
	Published bool
}

// SlotView is a slot with its lesson, if the topic's roadmap has one with the slot's title.
type SlotView struct {
	Slot
	Grammar []GrammarView
	Lesson  *topic.LessonRef
	// Checks are run on a draft lesson only.
	Checks *Checks
}

// TopicView is a topic of the course as the admin sees it.
type TopicView struct {
	Topic
	// TopicID is "" when the catalogue has no topic of this name.
	TopicID  string
	Position int
	Grammar  []GrammarView
	Slots    []SlotView
	// Others are roadmap lessons of the topic and level that fill no slot.
	Others []topic.LessonRef
}

// Board is the course of a level with its progress.
type Board struct {
	Level  string
	Topics []TopicView
	// Applied is true when every topic exists and has its place in the order.
	Applied bool
	// Main and Extra count the slots, Published and Drafts the slots filled.
	Main, Extra, Published, Drafts int
}

func (s *Service) published(ctx context.Context) (map[string]bool, error) {
	list, err := s.Grammar.AdminList(ctx)
	if err != nil {
		return nil, fmt.Errorf("curriculum: grammar lessons: %w", err)
	}
	out := map[string]bool{}
	for _, l := range list {
		if l.Status == grammar.StatusPublished {
			out[l.PointID] = true
		}
	}
	return out, nil
}

func grammarViews(ids []string, published map[string]bool) []GrammarView {
	out := make([]GrammarView, 0, len(ids))
	for _, id := range ids {
		p, _ := grammar.Default().Get(id)
		out = append(out, GrammarView{ID: id, TitleVi: p.TitleVi, Published: published[id]})
	}
	return out
}

// topicsByName finds the catalogue's topics by name.
func (s *Service) topicsByName(ctx context.Context) (map[string]topic.Topic, error) {
	all, err := s.Topics.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("curriculum: list topics: %w", err)
	}
	out := make(map[string]topic.Topic, len(all))
	for _, t := range all {
		out[NameKey(t.Name)] = t
	}
	return out, nil
}

// fill matches the roadmap's lessons with the slots by title.
func fill(t Topic, lessons []topic.LessonRef) (bySlot map[string]topic.LessonRef, others []topic.LessonRef) {
	byTitle := map[string]string{}
	for _, sl := range t.Slots {
		byTitle[TitleKey(sl.Title)] = sl.Key
	}
	bySlot = map[string]topic.LessonRef{}
	for _, l := range lessons {
		key, ok := byTitle[TitleKey(l.Title)]
		if _, taken := bySlot[key]; ok && !taken {
			bySlot[key] = l
			continue
		}
		others = append(others, l)
	}
	return bySlot, others
}

// Board returns the course of level with each slot's lesson.
func (s *Service) Board(ctx context.Context, level string) (Board, error) {
	p, ok := ByLevel(level)
	if !ok {
		return Board{}, ErrNoPlan
	}
	published, err := s.published(ctx)
	if err != nil {
		return Board{}, err
	}
	topics, err := s.topicsByName(ctx)
	if err != nil {
		return Board{}, err
	}
	b := Board{Level: level, Applied: true}
	for _, t := range p.Topics {
		tv := TopicView{Topic: t, Grammar: grammarViews(t.Grammar, published)}
		cat, found := topics[NameKey(t.Name)]
		var bySlot map[string]topic.LessonRef
		if found {
			tv.TopicID, tv.Position = cat.ID, cat.Positions[level]
			rm, err := s.Topics.Roadmap(ctx, cat.ID, level)
			if err != nil {
				return Board{}, fmt.Errorf("curriculum: roadmap: %w", err)
			}
			bySlot, tv.Others = fill(t, rm.Lessons)
		}
		if !found || tv.Position != t.Number {
			b.Applied = false
		}
		for _, sl := range t.Slots {
			sv := SlotView{Slot: sl, Grammar: grammarViews(sl.Grammar, published)}
			if l, ok := bySlot[sl.Key]; ok {
				sv.Lesson = &l
				if l.Draft {
					b.Drafts++
					if full, err := s.Lessons.Get(ctx, l.ID); err == nil {
						c := Inspect(full)
						sv.Checks = &c
					}
				} else {
					b.Published++
				}
			}
			if sl.Kind == KindMain {
				b.Main++
			} else {
				b.Extra++
			}
			tv.Slots = append(tv.Slots, sv)
		}
		b.Topics = append(b.Topics, tv)
	}
	return b, nil
}

// Apply creates the course's missing topics, gives the topics their place in the level's order
// (other topics come after them) and orders each roadmap as the slots: main lessons, extra ones,
// then the lessons that fill no slot.
func (s *Service) Apply(ctx context.Context, level string) (Board, error) {
	p, ok := ByLevel(level)
	if !ok {
		return Board{}, ErrNoPlan
	}
	topics, err := s.topicsByName(ctx)
	if err != nil {
		return Board{}, err
	}
	ids := make([]string, 0, len(p.Topics))
	for _, t := range p.Topics {
		cat, found := topics[NameKey(t.Name)]
		if !found {
			sum, err := s.Topics.Create(ctx, topic.Input{Name: t.Name, Description: "Sau chủ đề này bạn có thể " + t.CanDoVi})
			if err != nil {
				return Board{}, fmt.Errorf("curriculum: create topic %q: %w", t.Name, err)
			}
			cat = sum.Topic
		}
		ids = append(ids, cat.ID)
		if err := s.arrange(ctx, t, cat.ID, level); err != nil {
			return Board{}, err
		}
	}
	if _, err := s.Topics.SetOrder(ctx, level, ids); err != nil {
		return Board{}, fmt.Errorf("curriculum: set order: %w", err)
	}
	return s.Board(ctx, level)
}

// arrange orders a topic's roadmap as its slots.
func (s *Service) arrange(ctx context.Context, t Topic, topicID, level string) error {
	rm, err := s.Topics.Roadmap(ctx, topicID, level)
	if err != nil {
		return fmt.Errorf("curriculum: roadmap: %w", err)
	}
	bySlot, others := fill(t, rm.Lessons)
	ids := make([]string, 0, len(rm.Lessons))
	for _, sl := range t.Slots {
		if l, ok := bySlot[sl.Key]; ok {
			ids = append(ids, l.ID)
		}
	}
	for _, l := range others {
		ids = append(ids, l.ID)
	}
	current := make([]string, len(rm.Lessons))
	for i, l := range rm.Lessons {
		current[i] = l.ID
	}
	if slices.Equal(ids, current) {
		return nil
	}
	if _, err := s.Topics.SetRoadmap(ctx, topicID, level, ids); err != nil {
		return fmt.Errorf("curriculum: order roadmap: %w", err)
	}
	return nil
}

// GenerateSlot writes the lesson of an empty slot with one AI request and saves it as a draft
// at its place in the roadmap: the slot's title, its first grammar point, its idea (and its other
// points), and target words planned from the topic's words.
func (s *Service) GenerateSlot(ctx context.Context, key string) (SlotView, error) {
	p, t, sl, ok := FindSlot(key)
	if !ok {
		return SlotView{}, ErrSlotNotFound
	}
	topics, err := s.topicsByName(ctx)
	if err != nil {
		return SlotView{}, err
	}
	cat, found := topics[NameKey(t.Name)]
	if !found {
		return SlotView{}, ErrNotApplied
	}
	rm, err := s.Topics.Roadmap(ctx, cat.ID, p.Level)
	if err != nil {
		return SlotView{}, fmt.Errorf("curriculum: roadmap: %w", err)
	}
	if bySlot, _ := fill(t, rm.Lessons); bySlot[sl.Key].ID != "" {
		return SlotView{}, ErrSlotFilled
	}
	in := lesson.GenerateInput{
		Level: lesson.Level(p.Level), Count: 1, Words: lessonWords[p.Level], Kind: "reading",
		Idea: slotIdea(sl), GrammarPointID: sl.Grammar[0],
	}
	if per := targetWords[p.Level]; per > 0 {
		plan, err := s.Topics.WordPlan(ctx, cat.ID, p.Level, 1, per)
		if err == nil && len(plan.Groups) == 1 && len(plan.Groups[0]) > 0 {
			in.TargetWords = plan.Groups
		}
	}
	res, err := s.Lessons.Generate(ctx, cat.ID, in)
	if err != nil {
		return SlotView{}, err
	}
	if len(res.Drafts) == 0 {
		return SlotView{}, ErrNoDraft
	}
	d := res.Drafts[0]
	l, err := s.Lessons.Create(ctx, lesson.Input{
		Title: sl.Title, Content: d.Content, TopicID: cat.ID, Level: lesson.Level(p.Level),
		Source: aiSource, License: aiLicense, GrammarPointID: sl.Grammar[0], TargetWords: d.TargetWords,
		AppendToRoadmap: true, Draft: true,
	})
	if err != nil {
		return SlotView{}, err
	}
	if err := s.arrange(ctx, t, cat.ID, p.Level); err != nil {
		return SlotView{}, err
	}
	published, err := s.published(ctx)
	if err != nil {
		return SlotView{}, err
	}
	c := Inspect(l)
	return SlotView{
		Slot: sl, Grammar: grammarViews(sl.Grammar, published), Checks: &c,
		Lesson: &topic.LessonRef{ID: l.ID, Title: l.Title, Level: string(l.Level), TopicID: l.TopicID, AnnotationStatus: string(l.AnnotationStatus), Draft: l.Draft, CreatedAt: l.CreatedAt},
	}, nil
}

// slotIdea is the idea given to the AI: the slot's story, and its other grammar points.
func slotIdea(sl Slot) string {
	idea := sl.Idea
	for _, id := range sl.Grammar[1:] {
		if p, ok := grammar.Default().Get(id); ok {
			idea += fmt.Sprintf(" Also use %s (%s) a few times.", p.TitleEn, p.Pattern)
		}
	}
	return idea
}

// Checks runs the checks on one lesson.
func (s *Service) Checks(ctx context.Context, lessonID string) (Checks, error) {
	l, err := s.Lessons.Get(ctx, lessonID)
	if err != nil {
		return Checks{}, err
	}
	return Inspect(l), nil
}

// TopicInfo is what learners see of a course topic.
type TopicInfo struct {
	Number    int
	CanDoVi   string
	GrammarVi []string
	// Extra holds the title keys (TitleKey) of the topic's extra lessons.
	Extra map[string]bool
}

// LearnerTopic returns the course information of a topic of level, by its catalogue name.
func LearnerTopic(level, topicName string) (TopicInfo, bool) {
	p, ok := ByLevel(level)
	if !ok {
		return TopicInfo{}, false
	}
	for _, t := range p.Topics {
		if NameKey(t.Name) != NameKey(topicName) {
			continue
		}
		info := TopicInfo{Number: t.Number, CanDoVi: t.CanDoVi, Extra: map[string]bool{}}
		for _, id := range t.Grammar {
			if pt, ok := grammar.Default().Get(id); ok {
				info.GrammarVi = append(info.GrammarVi, pt.TitleVi)
			}
		}
		for _, sl := range t.Slots {
			if sl.Kind == KindExtra {
				info.Extra[TitleKey(sl.Title)] = true
			}
		}
		return info, true
	}
	return TopicInfo{}, false
}

// Removed tells what Remove did: the lesson left its roadmap, and a draft was deleted too.
type Removed struct {
	Deleted bool
}

// Remove takes a lesson out of its topic's roadmap so that its slot is empty again (added
// 2026-10-22, "Gỡ" on the course page). A draft, never seen by learners, is deleted as well; a
// published lesson stays in the lesson list, as "Gỡ" on the roadmap page leaves it.
func (s *Service) Remove(ctx context.Context, lessonID string) (Removed, error) {
	l, err := s.Lessons.Get(ctx, lessonID)
	if err != nil {
		return Removed{}, err
	}
	rm, err := s.Topics.Roadmap(ctx, l.TopicID, string(l.Level))
	if err != nil {
		return Removed{}, fmt.Errorf("curriculum: roadmap: %w", err)
	}
	ids := make([]string, 0, len(rm.Lessons))
	for _, r := range rm.Lessons {
		if r.ID != lessonID {
			ids = append(ids, r.ID)
		}
	}
	if len(ids) < len(rm.Lessons) {
		if _, err := s.Topics.SetRoadmap(ctx, l.TopicID, string(l.Level), ids); err != nil {
			return Removed{}, fmt.Errorf("curriculum: remove from roadmap: %w", err)
		}
	}
	if !l.Draft {
		return Removed{}, nil
	}
	if err := s.Lessons.Delete(ctx, lessonID); err != nil {
		return Removed{}, fmt.Errorf("curriculum: delete draft: %w", err)
	}
	return Removed{Deleted: true}, nil
}
