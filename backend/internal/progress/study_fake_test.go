package progress

import (
	"context"
	"maps"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
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

type fakeDays struct {
	mu   sync.Mutex
	days map[string]StudyDay // user|day
}

func newFakeDays() *fakeDays { return &fakeDays{days: map[string]StudyDay{}} }

func (f *fakeDays) Get(_ context.Context, userID, dayKey string) (*StudyDay, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	d, ok := f.days[userID+"|"+dayKey]
	if !ok {
		return nil, nil
	}
	return &d, nil
}

func (f *fakeDays) Start(_ context.Context, userID, dayKey, lessonID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := userID + "|" + dayKey
	if _, ok := f.days[key]; !ok {
		f.days[key] = StudyDay{DayKey: dayKey, LessonID: lessonID}
	}
	return nil
}

func (f *fakeDays) Update(_ context.Context, userID, dayKey string, reviewed int, completed bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := userID + "|" + dayKey
	d := f.days[key]
	d.DayKey, d.ReviewedCount, d.Completed = dayKey, reviewed, completed
	f.days[key] = d
	return nil
}

func (f *fakeDays) CompletedKeys(_ context.Context, userID string) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []string
	for key, d := range f.days {
		if d.Completed && key[:len(userID)+1] == userID+"|" {
			out = append(out, d.DayKey)
		}
	}
	return out, nil
}

// put stores a finished day directly (to build a streak).
func (f *fakeDays) put(userID, dayKey, lessonID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.days[userID+"|"+dayKey] = StudyDay{DayKey: dayKey, LessonID: lessonID, Completed: true}
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
	mu       sync.Mutex
	due      map[string]int
	reviewed map[string][]time.Time
	cards    map[string][]fakeCard
}

func (f *fakeReviews) DueCount(_ context.Context, userID string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.due[userID], nil
}

func (f *fakeReviews) ReviewedSince(_ context.Context, userID string, since time.Time) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, t := range f.reviewed[userID] {
		if !t.Before(since) {
			n++
		}
	}
	return n, nil
}

// review records n daily reviews at t, each lowering the due count.
func (f *fakeReviews) review(userID string, n int, t time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for range n {
		f.reviewed[userID] = append(f.reviewed[userID], t)
	}
	f.due[userID] = max(0, f.due[userID]-n)
}

func (f *fakeReviews) setDue(userID string, n int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.due[userID] = n
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
	// limit is the daily card limit from the settings (F12), 30 by default.
	limit atomic.Int64
}

// newStudyEnv: topics "family" (A1, lessons f1..f3), "shopping" (A1, s1..s2), "work" (B1, w1),
// "empty" (A1, no lessons). The clock starts at 2026-09-30 10:00 in Viet Nam.
func newStudyEnv() *studyEnv {
	e := &studyEnv{
		goals:    &fakeGoals{},
		progress: newFakeProgress(),
		days:     newFakeDays(),
		roadmaps: &fakeRoadmaps{topics: map[string]TopicInfo{}},
		reviews:  &fakeReviews{due: map[string]int{}, reviewed: map[string][]time.Time{}, cards: map[string][]fakeCard{}},
		zones:    fakeZones{},
		clock:    &studyClock{t: time.Date(2026, 9, 30, 10, 0, 0, 0, hcm)},
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
	e.limit.Store(DefaultReviewLimit)
	titles := fakeTitles{"f1": "Family 1", "f2": "Family 2", "f3": "Family 3", "s1": "Shop 1", "s2": "Shop 2", "w1": "Work 1"}
	e.svc = NewStudyService(StudyDeps{
		Goals: e.goals, Progress: e.progress, Days: e.days, Dictation: e.dictation, Lessons: e.lessons,
		Roadmaps: e.roadmaps, Titles: titles, Reviews: e.reviews, Timezones: e.zones, Now: e.clock.now,
		ReviewLimit: func(context.Context, string) int { return int(e.limit.Load()) },
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

func (f *fakeDays) LatestKey(_ context.Context, userID string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	latest := ""
	for key, d := range f.days {
		if key[:len(userID)+1] == userID+"|" {
			latest = max(latest, d.DayKey)
		}
	}
	return latest, nil
}
