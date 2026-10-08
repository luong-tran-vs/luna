package topic

import (
	"context"
	"maps"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
)

// fakeRepo is an in-memory Repository enforcing unique name keys.
type fakeRepo struct {
	mu     sync.Mutex
	topics []Topic
	nextID int
}

func (f *fakeRepo) index(id string) int {
	return slices.IndexFunc(f.topics, func(t Topic) bool { return t.ID == id })
}

func (f *fakeRepo) taken(name, except string) bool {
	return slices.ContainsFunc(f.topics, func(t Topic) bool {
		return t.ID != except && NameKey(t.Name) == NameKey(name)
	})
}

// clone copies the roadmaps and words so callers never share them with the store.
func clone(t Topic) Topic {
	t.Roadmaps = maps.Clone(t.Roadmaps)
	for l, ids := range t.Roadmaps {
		t.Roadmaps[l] = slices.Clone(ids)
	}
	if t.Roadmaps == nil {
		t.Roadmaps = map[string][]string{}
	}
	t.Words = slices.Clone(t.Words)
	return t
}

func (f *fakeRepo) Create(_ context.Context, t Topic) (Topic, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.taken(t.Name, "") {
		return Topic{}, ErrNameTaken
	}
	f.nextID++
	t.ID = "t" + strconv.Itoa(f.nextID)
	t = clone(t)
	f.topics = append(f.topics, t)
	return clone(t), nil
}

func (f *fakeRepo) Get(_ context.Context, id string) (Topic, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if i := f.index(id); i >= 0 {
		return clone(f.topics[i]), nil
	}
	return Topic{}, ErrNotFound
}

func (f *fakeRepo) List(context.Context) ([]Topic, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Topic
	for _, t := range f.topics {
		out = append(out, clone(t))
	}
	return out, nil
}

func (f *fakeRepo) Update(_ context.Context, id string, in Input) (Topic, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	i := f.index(id)
	if i < 0 {
		return Topic{}, ErrNotFound
	}
	if f.taken(in.Name, id) {
		return Topic{}, ErrNameTaken
	}
	f.topics[i].Name, f.topics[i].Description = in.Name, in.Description
	return clone(f.topics[i]), nil
}

func (f *fakeRepo) Delete(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if i := f.index(id); i >= 0 {
		f.topics = slices.Delete(f.topics, i, i+1)
	}
	return nil
}

func (f *fakeRepo) SetLessons(_ context.Context, id, level string, ids []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	i := f.index(id)
	if i < 0 {
		return ErrNotFound
	}
	f.topics[i].Roadmaps[level] = slices.Clone(ids)
	return nil
}

func (f *fakeRepo) RemoveLesson(_ context.Context, id, level, lessonID string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	i := f.index(id)
	if i < 0 || !slices.Contains(f.topics[i].Roadmaps[level], lessonID) {
		return false, nil
	}
	f.topics[i].Roadmaps[level] = slices.DeleteFunc(f.topics[i].Roadmaps[level], func(l string) bool { return l == lessonID })
	return true, nil
}

func (f *fakeRepo) AppendLesson(_ context.Context, id, level, lessonID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	i := f.index(id)
	if i < 0 {
		return ErrNotFound
	}
	if !slices.Contains(f.topics[i].Roadmaps[level], lessonID) {
		f.topics[i].Roadmaps[level] = append(f.topics[i].Roadmaps[level], lessonID)
	}
	return nil
}

func (f *fakeRepo) SetWords(_ context.Context, id string, words []Word) (Topic, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	i := f.index(id)
	if i < 0 {
		return Topic{}, ErrNotFound
	}
	f.topics[i].Words, f.topics[i].WordsSeeded = slices.Clone(words), true
	return clone(f.topics[i]), nil
}

// fakeLessons knows each lesson's topic, level and title.
type fakeLessons struct {
	mu        sync.Mutex
	lessons   map[string]LessonRef
	texts     map[string]string // lesson id → content
	textCalls int
}

func newFakeLessons() *fakeLessons { return &fakeLessons{lessons: map[string]LessonRef{}} }

func (f *fakeLessons) add(id, title, topicID, level string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lessons[id] = LessonRef{ID: id, Title: title, TopicID: topicID, Level: level, AnnotationStatus: "done"}
}

func (f *fakeLessons) CountByTopic(context.Context) (map[string]map[string]int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[string]map[string]int{}
	for _, l := range f.lessons {
		if out[l.TopicID] == nil {
			out[l.TopicID] = map[string]int{}
		}
		out[l.TopicID][l.Level]++
	}
	return out, nil
}

func (f *fakeLessons) PlaceOf(_ context.Context, ids []string) (map[string]Place, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[string]Place{}
	for _, id := range ids {
		if l, ok := f.lessons[id]; ok {
			out[id] = Place{TopicID: l.TopicID, Level: l.Level}
		}
	}
	return out, nil
}

func (f *fakeLessons) Refs(_ context.Context, ids []string) ([]LessonRef, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []LessonRef
	for _, id := range ids {
		if l, ok := f.lessons[id]; ok {
			out = append(out, l)
		}
	}
	return out, nil
}

// setDraft hides a lesson from learners.
func (f *fakeLessons) setDraft(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	l := f.lessons[id]
	l.Draft = true
	f.lessons[id] = l
}

// setText gives a lesson a content for coverage.
func (f *fakeLessons) setText(id, content string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.texts == nil {
		f.texts = map[string]string{}
	}
	f.texts[id] = content
}

func (f *fakeLessons) Texts(_ context.Context, topicIDs []string) (map[string][]LessonText, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.textCalls++
	out := map[string][]LessonText{}
	for id, l := range f.lessons {
		if slices.Contains(topicIDs, l.TopicID) {
			out[l.TopicID] = append(out[l.TopicID], LessonText{Content: f.texts[id]})
		}
	}
	return out, nil
}

type testEnv struct {
	svc     *Service
	repo    *fakeRepo
	lessons *fakeLessons
	ai      *fakeSuggester
}

// fakeSuggester returns words or err and records the requests.
type fakeSuggester struct {
	mu    sync.Mutex
	words []string
	err   error
	reqs  []ai.SuggestWordsRequest
}

func (f *fakeSuggester) SuggestWords(_ context.Context, req ai.SuggestWordsRequest) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reqs = append(f.reqs, req)
	return f.words, f.err
}

var testNow = time.Date(2026, 9, 30, 8, 0, 0, 0, time.UTC)

func newEnv() *testEnv {
	e := &testEnv{repo: &fakeRepo{}, lessons: newFakeLessons(), ai: &fakeSuggester{err: ai.ErrNotConfigured}}
	e.svc = NewService(e.repo, e.lessons, e.ai, func() time.Time { return testNow })
	return e
}

// words builds topic words without a level.
func words(texts ...string) []Word {
	out := make([]Word, len(texts))
	for i, t := range texts {
		out[i] = Word{Text: t}
	}
	return out
}
