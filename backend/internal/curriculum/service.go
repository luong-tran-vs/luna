package curriculum

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
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
	// ErrNoDraftPlan: the level has no draft course to edit or approve.
	ErrNoDraftPlan = errors.New("curriculum: no draft course")
	// ErrNoAI: no AI provider can draft courses.
	ErrNoAI = errors.New("curriculum: AI not configured")
	// ErrAIFailed wraps an error of the AI while drafting a course.
	ErrAIFailed = errors.New("curriculum: AI request failed")
)

// PlanError is a course the admin or the AI wrote that cannot be saved or approved; Message is in
// Vietnamese, for the admin.
type PlanError struct {
	Message string
}

func (e *PlanError) Error() string { return e.Message }

// Source and licence of generated lessons, as the admin's generate screen saves them.
const (
	aiSource  = "AI sinh"
	aiLicense = "Nội dung do AI tạo"
)

// Words of a generated lesson and target words per lesson, by level (as the admin's defaults,
// frontend core/models/generate.ts).
var (
	lessonWords = map[string]int{"A1": 120, "A2": 160, "B1": 220, "B2": 300, "C1": 400, "C2": 400}
	targetWords = map[string]int{"A1": 8, "A2": 8, "B1": 10, "B2": 10, "C1": 12, "C2": 12}
)

// Levels are the levels a course can be written for, lowest first.
var Levels = []string{"A1", "A2", "B1", "B2", "C1", "C2"}

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

// Service runs the course: the admin's page and what learners see of it.
type Service struct {
	Topics  Topics
	Lessons Lessons
	Grammar GrammarLessons
	Repo    Repository
	// Writer drafts courses; nil when the AI cannot.
	Writer ai.CurriculumWriter
	Now    func() time.Time

	cache planCache
}

func (s *Service) now() time.Time {
	if s.Now == nil {
		return time.Now()
	}
	return s.Now()
}

// course reads the course of level. A level without one that has a seed (A1) gets it, approved.
func (s *Service) course(ctx context.Context, level string) (Course, error) {
	c, err := s.Repo.Get(ctx, level)
	if errors.Is(err, ErrNoPlan) {
		p, ok := seed(level)
		if !ok {
			return Course{}, ErrNoPlan
		}
		now := s.now()
		c = Course{Level: level, Approved: &p, ApprovedAt: now, UpdatedAt: now}
		if err := s.Repo.Save(ctx, c); err != nil {
			return Course{}, fmt.Errorf("curriculum: save seed %s: %w", level, err)
		}
		return c, nil
	}
	if err != nil {
		return Course{}, fmt.Errorf("curriculum: read %s: %w", level, err)
	}
	for _, p := range []*Plan{c.Approved, c.Draft} {
		if p != nil {
			p.Level = level
			p.Normalize()
		}
	}
	return c, nil
}

func (s *Service) save(ctx context.Context, c Course) error {
	c.UpdatedAt = s.now()
	if err := s.Repo.Save(ctx, c); err != nil {
		return fmt.Errorf("curriculum: save %s: %w", c.Level, err)
	}
	s.cache.drop(c.Level)
	return nil
}

// approved returns the approved course of level, kept for cacheTTL; nil when there is none.
func (s *Service) approved(ctx context.Context, level string) (*Plan, error) {
	now := s.now()
	if p, ok := s.cache.get(level, now); ok {
		return p, nil
	}
	c, err := s.course(ctx, level)
	if err != nil && !errors.Is(err, ErrNoPlan) {
		return nil, err
	}
	s.cache.put(level, c.Approved, now)
	return c.Approved, nil
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
	// Words counts the catalogue topic's words that a lesson of the level may be given (added
	// 2026-10-23): generated lessons take their target words from them.
	Words   int
	Grammar []GrammarView
	Slots   []SlotView
	// Others are roadmap lessons of the topic and level that fill no slot.
	Others []topic.LessonRef
}

// Status tells where a level's course is.
type Status string

const (
	StatusNone     Status = "none"
	StatusDraft    Status = "draft"
	StatusApproved Status = "approved"
)

// Board is the course of a level with its progress: the draft while there is one, otherwise the
// approved course.
type Board struct {
	Level  string
	Status Status
	// HasApproved is true when the level has an approved course (also while a draft edits it).
	HasApproved bool
	Topics      []TopicView
	// Warnings are what the shown course lacks (Plan.Warnings).
	Warnings []string
	// Points are the grammar points of the level, for editing a draft.
	Points []GrammarView
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

// catalogue finds the catalogue's topics by id and by name.
type catalogue struct {
	byID   map[string]topic.Topic
	byName map[string]topic.Topic
}

func (s *Service) catalogue(ctx context.Context) (catalogue, error) {
	all, err := s.Topics.All(ctx)
	if err != nil {
		return catalogue{}, fmt.Errorf("curriculum: list topics: %w", err)
	}
	c := catalogue{byID: make(map[string]topic.Topic, len(all)), byName: make(map[string]topic.Topic, len(all))}
	for _, t := range all {
		c.byID[t.ID] = t
		c.byName[NameKey(t.Name)] = t
	}
	return c, nil
}

// find returns the catalogue's topic of a course topic: by id, else by name.
func (c catalogue) find(t Topic) (topic.Topic, bool) {
	if cat, ok := c.byID[t.TopicID]; ok && t.TopicID != "" {
		return cat, true
	}
	cat, ok := c.byName[NameKey(t.Name)]
	return cat, ok
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

// Board returns the course of level with each slot's lesson; a level without a course has an
// empty board with StatusNone.
func (s *Service) Board(ctx context.Context, level string) (Board, error) {
	c, err := s.course(ctx, level)
	if errors.Is(err, ErrNoPlan) {
		c, err = Course{Level: level}, nil
	}
	if err != nil {
		return Board{}, err
	}
	return s.board(ctx, level, c, true)
}

// board is the board of course c of level; withChecks runs the checks of every draft lesson (one
// read each), which the summary does without.
func (s *Service) board(ctx context.Context, level string, c Course, withChecks bool) (Board, error) {
	published, err := s.published(ctx)
	if err != nil {
		return Board{}, err
	}
	b := Board{Level: level, Status: StatusNone}
	for _, pt := range grammar.Default().ByLevel(level) {
		b.Points = append(b.Points, GrammarView{ID: pt.ID, TitleVi: pt.TitleVi, Published: published[pt.ID]})
	}
	p := c.Approved
	b.Status, b.HasApproved = StatusApproved, c.Approved != nil
	if c.Draft != nil {
		p, b.Status = c.Draft, StatusDraft
	}
	if p == nil {
		b.Status = StatusNone
		return b, nil
	}
	b.Warnings = p.Warnings(grammar.Default())
	cat, err := s.catalogue(ctx)
	if err != nil {
		return Board{}, err
	}
	b.Applied = b.Status == StatusApproved
	for _, t := range p.Topics {
		tv := TopicView{Topic: t, Grammar: grammarViews(t.Grammar, published)}
		found, ok := cat.find(t)
		var bySlot map[string]topic.LessonRef
		if ok {
			tv.TopicID, tv.Position = found.ID, found.Positions[level]
			for _, w := range found.Words {
				if w.FitsLevel(level) {
					tv.Words++
				}
			}
			if min := targetWords[level]; tv.Words < min {
				b.Warnings = append(b.Warnings, fmt.Sprintf("Chủ đề “%s” mới có %d từ vựng dùng được ở %s, nên có ít nhất %d để bài sinh ra có từ mục tiêu.", found.Name, tv.Words, level, min))
			}
			rm, err := s.Topics.Roadmap(ctx, found.ID, level)
			if err != nil {
				return Board{}, fmt.Errorf("curriculum: roadmap: %w", err)
			}
			bySlot, tv.Others = fill(t, rm.Lessons)
		}
		if !ok || tv.Position != t.Number {
			b.Applied = false
		}
		for _, sl := range t.Slots {
			sv := SlotView{Slot: sl, Grammar: grammarViews(sl.Grammar, published)}
			if l, ok := bySlot[sl.Key]; ok {
				sv.Lesson = &l
				if l.Draft {
					b.Drafts++
					if withChecks {
						if full, err := s.Lessons.Get(ctx, l.ID); err == nil {
							c := Inspect(full)
							sv.Checks = &c
						}
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

// Apply creates the approved course's missing topics, gives the topics their place in the level's
// order (other topics come after them) and orders each roadmap as the slots: main lessons, extra
// ones, then the lessons that fill no slot.
func (s *Service) Apply(ctx context.Context, level string) (Board, error) {
	c, err := s.course(ctx, level)
	if err != nil {
		return Board{}, err
	}
	if c.Approved == nil {
		return Board{}, ErrNoPlan
	}
	changed, err := s.createTopics(ctx, c.Approved)
	if err != nil {
		return Board{}, err
	}
	ids := make([]string, 0, len(c.Approved.Topics))
	for _, t := range c.Approved.Topics {
		ids = append(ids, t.TopicID)
		if err := s.arrange(ctx, t, t.TopicID, level); err != nil {
			return Board{}, err
		}
	}
	if _, err := s.Topics.SetOrder(ctx, level, ids); err != nil {
		return Board{}, fmt.Errorf("curriculum: set order: %w", err)
	}
	if changed {
		if err := s.save(ctx, c); err != nil {
			return Board{}, err
		}
	}
	return s.Board(ctx, level)
}

// createTopics creates the plan's topics the catalogue lacks and sets every topic's TopicID;
// changed tells whether an id was set.
func (s *Service) createTopics(ctx context.Context, p *Plan) (changed bool, err error) {
	cat, err := s.catalogue(ctx)
	if err != nil {
		return false, err
	}
	for i := range p.Topics {
		t := &p.Topics[i]
		found, ok := cat.find(*t)
		if !ok {
			sum, err := s.Topics.Create(ctx, topic.Input{Name: t.Name, Description: "Sau chủ đề này bạn có thể " + t.CanDoVi})
			if err != nil {
				return changed, fmt.Errorf("curriculum: create topic %q: %w", t.Name, err)
			}
			found = sum.Topic
		}
		if t.TopicID != found.ID {
			t.TopicID, changed = found.ID, true
		}
	}
	return changed, nil
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

// GenerateSlot writes the lesson of an empty slot of an approved course with one AI request and
// saves it as a draft at its place in the roadmap: the slot's title, its first grammar point, its
// idea (and its other points), and target words planned from the topic's words.
func (s *Service) GenerateSlot(ctx context.Context, key string) (SlotView, error) {
	level := LevelOfKey(key)
	p, err := s.approved(ctx, level)
	if err != nil {
		return SlotView{}, err
	}
	if p == nil {
		return SlotView{}, ErrSlotNotFound
	}
	t, sl, ok := p.FindSlot(key)
	if !ok {
		return SlotView{}, ErrSlotNotFound
	}
	cat, err := s.catalogue(ctx)
	if err != nil {
		return SlotView{}, err
	}
	found, ok := cat.find(t)
	if !ok {
		return SlotView{}, ErrNotApplied
	}
	rm, err := s.Topics.Roadmap(ctx, found.ID, level)
	if err != nil {
		return SlotView{}, fmt.Errorf("curriculum: roadmap: %w", err)
	}
	if bySlot, _ := fill(t, rm.Lessons); bySlot[sl.Key].ID != "" {
		return SlotView{}, ErrSlotFilled
	}
	in := lesson.GenerateInput{
		Level: lesson.Level(level), Count: 1, Words: lessonWords[level], Kind: "reading",
		Idea: slotIdea(sl), GrammarPointID: sl.Grammar[0],
	}
	if per := targetWords[level]; per > 0 {
		plan, err := s.Topics.WordPlan(ctx, found.ID, level, 1, per)
		if err == nil && len(plan.Groups) == 1 && len(plan.Groups[0]) > 0 {
			in.TargetWords = plan.Groups
		}
	}
	res, err := s.Lessons.Generate(ctx, found.ID, in)
	if err != nil {
		return SlotView{}, err
	}
	if len(res.Drafts) == 0 {
		return SlotView{}, ErrNoDraft
	}
	d := res.Drafts[0]
	l, err := s.Lessons.Create(ctx, lesson.Input{
		Title: sl.Title, Content: d.Content, TopicID: found.ID, Level: lesson.Level(level),
		Source: aiSource, License: aiLicense, GrammarPointID: sl.Grammar[0], TargetWords: d.TargetWords,
		AppendToRoadmap: true, Draft: true,
	})
	if err != nil {
		return SlotView{}, err
	}
	if err := s.arrange(ctx, t, found.ID, level); err != nil {
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

// LearnerTopic returns the approved course information of a catalogue topic at level, found by
// id, else by name.
func (s *Service) LearnerTopic(ctx context.Context, level, topicID, topicName string) (TopicInfo, bool) {
	p, err := s.approved(ctx, level)
	if err != nil || p == nil {
		return TopicInfo{}, false
	}
	for _, t := range p.Topics {
		byID := t.TopicID != "" && t.TopicID == topicID
		byName := t.TopicID == "" && NameKey(t.Name) == NameKey(topicName)
		if !byID && !byName {
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

// PointTopics maps each grammar point the approved course of level teaches to its first topic,
// and lists the points in course order; ok is false when the level has no approved course.
func (s *Service) PointTopics(ctx context.Context, level string) (map[string]PointTopic, []string, bool) {
	p, err := s.approved(ctx, level)
	if err != nil || p == nil {
		return nil, nil, false
	}
	topics, order := p.PointTopics()
	return topics, order, true
}

// Removed tells what Remove did: the lesson left its roadmap and was deleted.
type Removed struct {
	Deleted bool
}

// Remove takes a lesson out of its topic's roadmap and deletes it, as the lesson list does, so
// that its slot is empty again (added 2026-10-22, "Gỡ" on the course page; published lessons are
// deleted too since 2026-10-23, at the admin's request).
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
	if err := s.Lessons.Delete(ctx, lessonID); err != nil {
		return Removed{}, fmt.Errorf("curriculum: delete lesson: %w", err)
	}
	return Removed{Deleted: true}, nil
}

// CourseTopic is a catalogue topic in the approved course of a level, with its place.
type CourseTopic struct {
	TopicID string
	Name    string
	Number  int
}

// CourseTopics lists, per level with an approved course, its topics that the catalogue has (added
// 2026-10-23: the roadmap page says which topics follow a course).
func (s *Service) CourseTopics(ctx context.Context) (map[string][]CourseTopic, error) {
	cat, err := s.catalogue(ctx)
	if err != nil {
		return nil, err
	}
	out := map[string][]CourseTopic{}
	for _, level := range Levels {
		p, err := s.approved(ctx, level)
		if err != nil {
			return nil, err
		}
		if p == nil {
			continue
		}
		list := []CourseTopic{}
		for _, t := range p.Topics {
			if found, ok := cat.find(t); ok {
				list = append(list, CourseTopic{TopicID: found.ID, Name: found.Name, Number: t.Number})
			}
		}
		out[level] = list
	}
	return out, nil
}

// LevelSummary is where the course of a level is, for the admin home (added 2026-10-23).
type LevelSummary struct {
	Level  string
	Status Status
	// HasDraft is true while a draft edits or writes the course.
	HasDraft bool
	// Applied, Empty and Drafts describe the approved course: the order is applied, slots
	// without a lesson, slots holding a draft lesson.
	Applied       bool
	Empty, Drafts int
}

// Summary lists the levels that have a course or a draft.
func (s *Service) Summary(ctx context.Context) ([]LevelSummary, error) {
	var out []LevelSummary
	for _, level := range Levels {
		c, err := s.course(ctx, level)
		if errors.Is(err, ErrNoPlan) {
			continue
		}
		if err != nil {
			return nil, err
		}
		sum := LevelSummary{Level: level, Status: StatusDraft, HasDraft: c.Draft != nil}
		if c.Approved != nil {
			// The board of the approved course, even while a draft is open.
			c.Draft = nil
			b, err := s.board(ctx, level, c, false)
			if err != nil {
				return nil, err
			}
			sum.Status, sum.Applied, sum.Drafts = StatusApproved, b.Applied, b.Drafts
			sum.Empty = b.Main + b.Extra - b.Published - b.Drafts
		}
		out = append(out, sum)
	}
	return out, nil
}
