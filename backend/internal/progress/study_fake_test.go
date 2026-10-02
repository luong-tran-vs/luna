package progress

import (
	"context"
	"maps"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

type fakeGoals struct {
	mu     sync.Mutex
	goals  []Goal
	nextID int
}

func (f *fakeGoals) List(_ context.Context, userID string) ([]Goal, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Goal
	for _, g := range f.goals {
		if g.UserID == userID {
			out = append(out, g)
		}
	}
	return out, nil
}

func (f *fakeGoals) Activate(_ context.Context, userID, topicID, level, effectiveFrom string, now time.Time) (Goal, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	idx := -1
	for i := range f.goals {
		g := &f.goals[i]
		if g.UserID != userID {
			continue
		}
		if g.TopicID == topicID {
			idx = i
		} else if g.Status == GoalActive {
			g.Status = GoalPaused
		}
	}
	if idx < 0 {
		f.nextID++
		f.goals = append(f.goals, Goal{ID: "g" + strconv.Itoa(f.nextID), UserID: userID, TopicID: topicID, StartedAt: now})
		idx = len(f.goals) - 1
	}
	g := &f.goals[idx]
	g.Status, g.Level, g.EffectiveFrom = GoalActive, level, effectiveFrom
	return *g, nil
}

type fakeProgress struct {
	mu   sync.Mutex
	rows map[string]LessonProgress
}

func newFakeProgress() *fakeProgress { return &fakeProgress{rows: map[string]LessonProgress{}} }

func clone(p LessonProgress) LessonProgress {
	p.Done = maps.Clone(p.Done)
	return p
}

func (f *fakeProgress) Get(_ context.Context, userID, lessonID string) (LessonProgress, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.rows[userID+"|"+lessonID]
	return clone(p), ok, nil
}

func (f *fakeProgress) Completed(_ context.Context, userID string) ([]LessonProgress, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []LessonProgress
	for _, p := range f.rows {
		if p.UserID == userID && !p.CompletedAt.IsZero() {
			out = append(out, clone(p))
		}
	}
	slices.SortFunc(out, func(a, b LessonProgress) int { return b.CompletedAt.Compare(a.CompletedAt) })
	return out, nil
}

func (f *fakeProgress) Upsert(_ context.Context, p LessonProgress) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows[p.UserID+"|"+p.LessonID] = clone(p)
	return nil
}

func (f *fakeProgress) SetPosition(_ context.Context, userID, lessonID string, step Step, sentence int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := userID + "|" + lessonID
	p, ok := f.rows[key]
	if !ok {
		p = LessonProgress{UserID: userID, LessonID: lessonID, Done: map[Step]bool{}}
	}
	p.CurrentStep, p.SentenceIndex = step, sentence
	f.rows[key] = p
	return nil
}

// fakeDays holds the days with a completed lesson, per "user|day".
type fakeDays struct {
	mu   sync.Mutex
	days map[string]bool
}

func newFakeDays() *fakeDays { return &fakeDays{days: map[string]bool{}} }

func (f *fakeDays) MarkCompleted(_ context.Context, userID, dayKey string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.days[userID+"|"+dayKey] = true
	return nil
}

func (f *fakeDays) CompletedKeys(_ context.Context, userID string) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []string
	for key := range f.days {
		if day, ok := strings.CutPrefix(key, userID+"|"); ok {
			out = append(out, day)
		}
	}
	return out, nil
}

// put stores a finished day directly (to build a streak).
func (f *fakeDays) put(userID, dayKey string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.days[userID+"|"+dayKey] = true
}

type fakeRoadmaps struct {
	mu     sync.Mutex
	topics map[string]TopicInfo
}

func (f *fakeRoadmaps) Roadmap(_ context.Context, topicID string) (TopicInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.topics[topicID]
	if !ok {
		return TopicInfo{}, ErrTopicNotFound
	}
	t.LessonIDs = slices.Clone(t.LessonIDs)
	return t, nil
}

func (f *fakeRoadmaps) set(t TopicInfo) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.topics[t.ID] = t
}

type fakeTitles map[string]string

func (f fakeTitles) Titles(_ context.Context, ids []string) (map[string]string, error) {
	out := map[string]string{}
	for _, id := range ids {
		if t, ok := f[id]; ok {
			out[id] = t
		}
	}
	return out, nil
}

type fakeReviews struct {
	mu    sync.Mutex
	due   map[string]int
	cards map[string][]fakeCard
}

func (f *fakeReviews) DueCount(_ context.Context, userID string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.due[userID], nil
}

type fakeZones map[string]*time.Location

func (f fakeZones) Location(_ context.Context, userID string) (*time.Location, error) {
	if loc, ok := f[userID]; ok {
		return loc, nil
	}
	return hcm, nil
}

var hcm = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		panic(err)
	}
	return loc
}()

type studyClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *studyClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *studyClock) set(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = t
}

type studyEnv struct {
	svc       *StudyService
	goals     *fakeGoals
	progress  *fakeProgress
	days      *fakeDays
	roadmaps  *fakeRoadmaps
	reviews   *fakeReviews
	zones     fakeZones
	dictation *Service
	lessons   *fakeLessons
	clock     *studyClock
	quiz      *fakeQuiz
	writings  *fakeWritings
}

// newStudyEnv: topics "family" (A1, lessons f1..f3), "shopping" (A1, s1..s2), "work" (B1, w1),
// "empty" (A1, no lessons). The clock starts at 2026-09-30 10:00 in Viet Nam.
func newStudyEnv() *studyEnv {
	e := &studyEnv{
		goals:    &fakeGoals{},
		progress: newFakeProgress(),
		days:     newFakeDays(),
		roadmaps: &fakeRoadmaps{topics: map[string]TopicInfo{}},
		reviews:  &fakeReviews{due: map[string]int{}, cards: map[string][]fakeCard{}},
		zones:    fakeZones{},
		clock:    &studyClock{t: time.Date(2026, 9, 30, 10, 0, 0, 0, hcm)},
		quiz:     &fakeQuiz{status: map[string][2]int{}},
		writings: &fakeWritings{submitted: map[string]bool{}},
	}
	for _, t := range []TopicInfo{
		{ID: "family", Name: "Gia đình", Level: "A1", LessonIDs: []string{"f1", "f2", "f3"}},
		{ID: "shopping", Name: "Mua sắm", Level: "A1", LessonIDs: []string{"s1", "s2"}},
		{ID: "work", Name: "Công việc", Level: "B1", LessonIDs: []string{"w1"}},
		{ID: "empty", Name: "Trống", Level: "A1"},
	} {
		e.roadmaps.set(t)
	}
	e.lessons = &fakeLessons{lessons: map[string]lessonInfo{}}
	for _, id := range []string{"f1", "f2", "f3", "s1", "s2", "w1"} {
		e.lessons.set(id, 1, 3)
	}
	e.dictation = NewService(newFakeRepo(), e.lessons, e.clock.now)
	titles := fakeTitles{"f1": "Family 1", "f2": "Family 2", "f3": "Family 3", "s1": "Shop 1", "s2": "Shop 2", "w1": "Work 1"}
	e.svc = NewStudyService(StudyDeps{
		Goals: e.goals, Progress: e.progress, Days: e.days, Dictation: e.dictation, Lessons: e.lessons,
		Roadmaps: e.roadmaps, Titles: titles, Reviews: e.reviews, Timezones: e.zones, Now: e.clock.now,
		Quiz: e.quiz, Writings: e.writings,
	})
	return e
}

func (f *fakeProgress) StepCounts(_ context.Context, userID string, lessonIDs []string) (StepCounts, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var c StepCounts
	for _, p := range f.rows {
		if p.UserID != userID || lessonIDs != nil && !slices.Contains(lessonIDs, p.LessonID) {
			continue
		}
		if p.Done[StepRead] {
			c.Read++
		}
		if p.Done[StepListen] {
			c.Listen++
		}
		if p.Done[StepWrite] {
			c.Write++
		}
		if !p.CompletedAt.IsZero() {
			c.Completed++
		}
	}
	return c, nil
}

// fakeCard is a card for DueBefore: a due time, or none with the time it was saved.
type fakeCard struct {
	due, created time.Time
}

func (f *fakeReviews) DueBefore(_ context.Context, userID string, before, createdBefore time.Time) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, c := range f.cards[userID] {
		if c.due.IsZero() && c.created.Before(createdBefore) || !c.due.IsZero() && c.due.Before(before) {
			n++
		}
	}
	return n, nil
}

func (f *fakeReviews) CardCount(_ context.Context, userID string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.cards[userID]), nil
}

func (f *fakeReviews) addCard(userID string, c fakeCard) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cards[userID] = append(f.cards[userID], c)
}

// fakeQuiz holds [questions, answered] per "user/lesson" and answer totals per user (F15).
type fakeQuiz struct {
	mu     sync.Mutex
	status map[string][2]int
	totals [2]int
	err    error
}

func (f *fakeQuiz) set(userID, lessonID string, questions, answered int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.status[userID+"/"+lessonID] = [2]int{questions, answered}
}

func (f *fakeQuiz) Status(_ context.Context, userID, lessonID string) (questions, answered int, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s := f.status[userID+"/"+lessonID]
	return s[0], s[1], f.err
}

func (f *fakeQuiz) Totals(context.Context, string) (answered, correct int, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.totals[0], f.totals[1], f.err
}

// fakeWritings records submitted writings per "user/lesson" (F8).
type fakeWritings struct {
	mu        sync.Mutex
	submitted map[string]bool
	count     int
	average   *float64
	err       error
}

func (f *fakeWritings) submit(userID, lessonID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.submitted[userID+"/"+lessonID] = true
}

func (f *fakeWritings) Submitted(_ context.Context, userID, lessonID string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.submitted[userID+"/"+lessonID], f.err
}

func (f *fakeWritings) Stats(context.Context, string) (int, *float64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.count, f.average, f.err
}
