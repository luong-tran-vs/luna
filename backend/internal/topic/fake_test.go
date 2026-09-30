package topic

import (
	"context"
	"slices"
	"strconv"
	"sync"
	"time"
)

// fakeRepo is an in-memory Repository enforcing unique (level, name key).
type fakeRepo struct {
	mu     sync.Mutex
	topics []Topic
	nextID int
}

func (f *fakeRepo) index(id string) int {
	return slices.IndexFunc(f.topics, func(t Topic) bool { return t.ID == id })
}

func (f *fakeRepo) taken(level, name, except string) bool {
	return slices.ContainsFunc(f.topics, func(t Topic) bool {
		return t.ID != except && t.Level == level && NameKey(t.Name) == NameKey(name)
	})
}

func (f *fakeRepo) Create(_ context.Context, t Topic) (Topic, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.taken(t.Level, t.Name, "") {
		return Topic{}, ErrNameTaken
	}
	f.nextID++
	t.ID = "t" + strconv.Itoa(f.nextID)
	if t.LessonIDs == nil {
		t.LessonIDs = []string{}
	}
	f.topics = append(f.topics, t)
	return t, nil
}

func (f *fakeRepo) Get(_ context.Context, id string) (Topic, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if i := f.index(id); i >= 0 {
		t := f.topics[i]
		t.LessonIDs = slices.Clone(t.LessonIDs)
		return t, nil
	}
	return Topic{}, ErrNotFound
}

func (f *fakeRepo) List(_ context.Context, level string) ([]Topic, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Topic
	for _, t := range f.topics {
		if level == "" || t.Level == level {
			t.LessonIDs = slices.Clone(t.LessonIDs)
			out = append(out, t)
		}
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
	if f.taken(in.Level, in.Name, id) {
		return Topic{}, ErrNameTaken
	}
	f.topics[i].Name, f.topics[i].Level, f.topics[i].Description = in.Name, in.Level, in.Description
	return f.topics[i], nil
}

func (f *fakeRepo) Delete(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if i := f.index(id); i >= 0 {
		f.topics = slices.Delete(f.topics, i, i+1)
	}
	return nil
}

func (f *fakeRepo) SetLessons(_ context.Context, id string, ids []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	i := f.index(id)
	if i < 0 {
		return ErrNotFound
	}
	f.topics[i].LessonIDs = slices.Clone(ids)
	return nil
}

func (f *fakeRepo) RemoveLesson(_ context.Context, id, lessonID string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	i := f.index(id)
	if i < 0 || !slices.Contains(f.topics[i].LessonIDs, lessonID) {
		return false, nil
	}
	f.topics[i].LessonIDs = slices.DeleteFunc(f.topics[i].LessonIDs, func(l string) bool { return l == lessonID })
	return true, nil
}

func (f *fakeRepo) AppendLesson(_ context.Context, id, lessonID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	i := f.index(id)
	if i < 0 {
		return ErrNotFound
	}
	if !slices.Contains(f.topics[i].LessonIDs, lessonID) {
		f.topics[i].LessonIDs = append(f.topics[i].LessonIDs, lessonID)
	}
	return nil
}

// fakeLessons knows each lesson's topic and title.
type fakeLessons struct {
	mu        sync.Mutex
	lessons   map[string]LessonRef
	levelSets []string // "topicID=level" of SetLevelByTopic calls
}

func newFakeLessons() *fakeLessons { return &fakeLessons{lessons: map[string]LessonRef{}} }

func (f *fakeLessons) add(id, title, topicID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lessons[id] = LessonRef{ID: id, Title: title, TopicID: topicID, AudioStatus: "done", AnnotationStatus: "done"}
}

func (f *fakeLessons) CountByTopic(context.Context) (map[string]int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[string]int{}
	for _, l := range f.lessons {
		out[l.TopicID]++
	}
	return out, nil
}

func (f *fakeLessons) TopicOf(_ context.Context, ids []string) (map[string]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[string]string{}
	for _, id := range ids {
		if l, ok := f.lessons[id]; ok {
			out[id] = l.TopicID
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

func (f *fakeLessons) SetLevelByTopic(_ context.Context, topicID, level string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.levelSets = append(f.levelSets, topicID+"="+level)
	for id, l := range f.lessons {
		if l.TopicID == topicID {
			l.Level = level
			f.lessons[id] = l
		}
	}
	return nil
}

type testEnv struct {
	svc     *Service
	repo    *fakeRepo
	lessons *fakeLessons
}

var testNow = time.Date(2026, 9, 30, 8, 0, 0, 0, time.UTC)

func newEnv() *testEnv {
	e := &testEnv{repo: &fakeRepo{}, lessons: newFakeLessons()}
	e.svc = NewService(e.repo, e.lessons, func() time.Time { return testNow })
	return e
}
