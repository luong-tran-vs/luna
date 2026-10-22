package curriculum

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/grammar"
	"github.com/luongtran/luna/backend/internal/lesson"
	"github.com/luongtran/luna/backend/internal/topic"
)

type fakeTopics struct {
	topics   []topic.Topic
	lessons  map[string]topic.LessonRef
	order    []string
	setRoads int
	plans    [][]string
}

func (f *fakeTopics) All(context.Context) ([]topic.Topic, error) { return slices.Clone(f.topics), nil }

func (f *fakeTopics) find(id string) *topic.Topic {
	for i := range f.topics {
		if f.topics[i].ID == id {
			return &f.topics[i]
		}
	}
	return nil
}

func (f *fakeTopics) Create(_ context.Context, in topic.Input) (topic.Summary, error) {
	t := topic.Topic{ID: fmt.Sprintf("t%d", len(f.topics)+1), Name: in.Name, Description: in.Description, Roadmaps: map[string][]string{}, Positions: map[string]int{}}
	f.topics = append(f.topics, t)
	return topic.Summary{Topic: t}, nil
}

func (f *fakeTopics) SetOrder(_ context.Context, level string, ids []string) ([]topic.Summary, error) {
	f.order = ids
	for i := range f.topics {
		delete(f.topics[i].Positions, level)
		if n := slices.Index(ids, f.topics[i].ID); n >= 0 {
			if f.topics[i].Positions == nil {
				f.topics[i].Positions = map[string]int{}
			}
			f.topics[i].Positions[level] = n + 1
		}
	}
	return nil, nil
}

func (f *fakeTopics) Roadmap(_ context.Context, id, level string) (topic.Roadmap, error) {
	t := f.find(id)
	if t == nil {
		return topic.Roadmap{}, errors.New("no topic")
	}
	rm := topic.Roadmap{Level: level}
	for _, lid := range t.Roadmaps[level] {
		rm.Lessons = append(rm.Lessons, f.lessons[lid])
	}
	return rm, nil
}

func (f *fakeTopics) SetRoadmap(_ context.Context, id, level string, ids []string) (topic.Roadmap, error) {
	f.setRoads++
	f.find(id).Roadmaps[level] = ids
	return topic.Roadmap{}, nil
}

func (f *fakeTopics) WordPlan(_ context.Context, _, _ string, count, per int) (topic.TargetPlan, error) {
	f.plans = append(f.plans, []string{fmt.Sprint(count), fmt.Sprint(per)})
	return topic.TargetPlan{Groups: [][]string{{"hello", "name"}}}, nil
}

func (f *fakeTopics) add(topicID, title string, draft bool) string {
	id := fmt.Sprintf("l%d", len(f.lessons)+1)
	f.lessons[id] = topic.LessonRef{ID: id, Title: title, Level: "A1", TopicID: topicID, Draft: draft}
	t := f.find(topicID)
	t.Roadmaps["A1"] = append(t.Roadmaps["A1"], id)
	return id
}

type fakeLessons struct {
	topics    *fakeTopics
	generated []lesson.GenerateInput
	created   []lesson.Input
	drafts    []lesson.Draft
	deleted   []string
}

func (f *fakeLessons) Generate(_ context.Context, _ string, in lesson.GenerateInput) (lesson.GenerateResult, error) {
	f.generated = append(f.generated, in)
	return lesson.GenerateResult{Drafts: f.drafts, Requested: 1}, nil
}

func (f *fakeLessons) Create(_ context.Context, in lesson.Input) (lesson.Lesson, error) {
	f.created = append(f.created, in)
	id := f.topics.add(in.TopicID, in.Title, in.Draft)
	return lesson.Lesson{ID: id, Title: in.Title, Content: in.Content, Level: in.Level, TopicID: in.TopicID, Draft: in.Draft,
		GrammarPointID: in.GrammarPointID, TargetWords: in.TargetWords}, nil
}

func (f *fakeLessons) Get(_ context.Context, id string) (lesson.Lesson, error) {
	r, ok := f.topics.lessons[id]
	if !ok {
		return lesson.Lesson{}, lesson.ErrNotFound
	}
	return lesson.Lesson{ID: id, Title: r.Title, Level: "A1", TopicID: r.TopicID, Draft: r.Draft, Content: "He is not here. Is she?", GrammarPointID: "a1-to-be"}, nil
}

func (f *fakeLessons) Delete(_ context.Context, id string) error {
	f.deleted = append(f.deleted, id)
	delete(f.topics.lessons, id)
	return nil
}

type fakeGrammar struct{}

func (fakeGrammar) AdminList(context.Context) ([]grammar.LessonSummary, error) {
	return []grammar.LessonSummary{{PointID: "a1-like-love-ing", Status: grammar.StatusPublished}, {PointID: "a1-to-be", Status: "draft"}}, nil
}

func newService() (*Service, *fakeTopics, *fakeLessons) {
	ft := &fakeTopics{lessons: map[string]topic.LessonRef{}}
	// Âm nhạc exists with its 3 lessons out of order and one lesson of no slot; Công việc is not in the course.
	ft.topics = []topic.Topic{
		{ID: "music", Name: "Âm nhạc", Roadmaps: map[string][]string{}, Positions: map[string]int{"A1": 2}},
		{ID: "job", Name: "Công việc", Roadmaps: map[string][]string{}, Positions: map[string]int{"A1": 1}},
	}
	ft.add("music", "Learning Music at School", false)
	ft.add("music", "My Favorite Song", false)
	ft.add("music", "Old Lesson", false)
	ft.add("music", "listening to music at home", true)
	fl := &fakeLessons{topics: ft}
	return &Service{Topics: ft, Lessons: fl, Grammar: fakeGrammar{}}, ft, fl
}

func TestBoard(t *testing.T) {
	s, _, _ := newService()
	b, err := s.Board(context.Background(), "A1")
	if err != nil {
		t.Fatal(err)
	}
	if b.Applied || b.Main != 36 || b.Extra != 29 || b.Published != 2 || b.Drafts != 1 {
		t.Fatalf("board totals %+v", b)
	}
	music := b.Topics[8]
	if music.TopicID != "music" || music.Name != "Âm nhạc" || !music.Grammar[0].Published {
		t.Fatalf("music topic %+v", music)
	}
	if music.Slots[0].Lesson == nil || music.Slots[0].Lesson.Title != "My Favorite Song" {
		t.Errorf("slot 1 %+v", music.Slots[0].Lesson)
	}
	if l := music.Slots[1].Lesson; l == nil || !l.Draft || music.Slots[1].Checks == nil {
		t.Errorf("slot 2 should hold the draft, with checks: %+v", music.Slots[1])
	}
	if music.Slots[3].Lesson != nil || len(music.Others) != 1 || music.Others[0].Title != "Old Lesson" {
		t.Errorf("extra slot %+v, others %+v", music.Slots[3].Lesson, music.Others)
	}
	if b.Topics[0].TopicID != "" || b.Topics[0].Grammar[0].Published {
		t.Errorf("topic 1 is not in the catalogue and to be is not published: %+v", b.Topics[0])
	}
	if _, err := s.Board(context.Background(), "C2"); !errors.Is(err, ErrNoPlan) {
		t.Errorf("C2: %v", err)
	}
}

func TestApply(t *testing.T) {
	s, ft, _ := newService()
	b, err := s.Apply(context.Background(), "A1")
	if err != nil {
		t.Fatal(err)
	}
	if !b.Applied {
		t.Error("the board should be applied")
	}
	if len(ft.topics) != 13 || len(ft.order) != 12 || ft.order[8] != "music" {
		t.Fatalf("%d topics, order %v", len(ft.topics), ft.order)
	}
	if ft.find("job").Positions["A1"] != 0 {
		t.Error("a topic outside the course loses its place")
	}
	if got := ft.find("music").Roadmaps["A1"]; !slices.Equal(got, []string{"l2", "l4", "l1", "l3"}) {
		t.Errorf("music roadmap %v", got)
	}
	if ft.find("t3").Description == "" {
		t.Error("a created topic gets the can-do as description")
	}
	again := ft.setRoads
	if _, err := s.Apply(context.Background(), "A1"); err != nil || ft.setRoads != again {
		t.Errorf("applying twice reorders nothing (%d → %d), %v", again, ft.setRoads, err)
	}
}

func TestGenerateSlot(t *testing.T) {
	s, ft, fl := newService()
	ctx := context.Background()
	if _, err := s.GenerateSlot(ctx, "a1-01-1"); !errors.Is(err, ErrNotApplied) {
		t.Fatalf("not applied: %v", err)
	}
	if _, err := s.GenerateSlot(ctx, "a1-09-1"); !errors.Is(err, ErrSlotFilled) {
		t.Fatalf("filled: %v", err)
	}
	if _, err := s.GenerateSlot(ctx, "nope"); !errors.Is(err, ErrSlotNotFound) {
		t.Fatalf("unknown: %v", err)
	}
	if _, err := s.GenerateSlot(ctx, "a1-09-4"); !errors.Is(err, ErrNoDraft) {
		t.Fatalf("no draft: %v", err)
	}
	fl.drafts = []lesson.Draft{{Title: "Some AI Title", Content: "We like singing.", TargetWords: []string{"hello", "name"}}}
	sv, err := s.GenerateSlot(ctx, "a1-09-4")
	if err != nil {
		t.Fatal(err)
	}
	in := fl.generated[len(fl.generated)-1]
	if in.Count != 1 || in.Words != 120 || in.GrammarPointID != "a1-like-love-ing" || len(in.TargetWords) != 1 {
		t.Errorf("generate input %+v", in)
	}
	if !strings.Contains(in.Idea, "Also use Conjunctions and, but, or, because") {
		t.Errorf("idea %q should name the other point", in.Idea)
	}
	c := fl.created[len(fl.created)-1]
	if c.Title != "A Concert in the Park" || !c.Draft || !c.AppendToRoadmap || c.Source != aiSource || c.GrammarPointID != "a1-like-love-ing" {
		t.Errorf("created %+v", c)
	}
	if sv.Lesson == nil || sv.Checks == nil {
		t.Fatalf("slot view %+v", sv)
	}
	// The new lesson goes after the main ones, before the lesson of no slot.
	if got := ft.find("music").Roadmaps["A1"]; !slices.Equal(got, []string{"l2", "l4", "l1", "l5", "l3"}) {
		t.Errorf("music roadmap %v", got)
	}
}

func TestRemove(t *testing.T) {
	s, ft, fl := newService()
	ctx := context.Background()
	// A published lesson leaves the roadmap and stays.
	res, err := s.Remove(ctx, "l2")
	if err != nil || res.Deleted || !slices.Equal(ft.find("music").Roadmaps["A1"], []string{"l1", "l3", "l4"}) || len(fl.deleted) != 0 {
		t.Fatalf("published: %+v, %v, roadmap %v", res, err, ft.find("music").Roadmaps["A1"])
	}
	// A draft is deleted too.
	res, err = s.Remove(ctx, "l4")
	if err != nil || !res.Deleted || !slices.Equal(fl.deleted, []string{"l4"}) || slices.Contains(ft.find("music").Roadmaps["A1"], "l4") {
		t.Fatalf("draft: %+v, %v", res, err)
	}
	if _, err := s.Remove(ctx, "nope"); !errors.Is(err, lesson.ErrNotFound) {
		t.Errorf("unknown lesson: %v", err)
	}
}
