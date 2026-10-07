package vocab

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

// fakeRepo is an in-memory Repository enforcing (userId, lemma) uniqueness.
type fakeRepo struct {
	mu     sync.Mutex
	cards  []Card
	nextID int
}

func (f *fakeRepo) Create(_ context.Context, c Card) (Card, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, x := range f.cards {
		if x.UserID == c.UserID && x.Lemma == c.Lemma {
			return Card{}, ErrExists
		}
	}
	c.ID = "c" + strconv.Itoa(f.nextID)
	f.nextID++
	f.cards = append(f.cards, c)
	return c, nil
}

// put stores c as is (for cards saved before F5 or with a chosen schedule).
func (f *fakeRepo) put(c Card) Card {
	f.mu.Lock()
	defer f.mu.Unlock()
	c.ID = "c" + strconv.Itoa(f.nextID)
	f.nextID++
	f.cards = append(f.cards, c)
	return c
}

func (f *fakeRepo) find(userID, id string) int {
	return slices.IndexFunc(f.cards, func(c Card) bool { return c.UserID == userID && c.ID == id })
}

func (f *fakeRepo) FindByLemma(_ context.Context, userID, lemma string) (Card, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, x := range f.cards {
		if x.UserID == userID && x.Lemma == lemma {
			return x, nil
		}
	}
	return Card{}, ErrNotFound
}

func (f *fakeRepo) Words(_ context.Context, userID string) ([]WordRef, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []WordRef
	for _, x := range f.cards {
		if x.UserID == userID {
			out = append(out, WordRef{Lemma: x.Lemma, Text: x.Text})
		}
	}
	return out, nil
}

func (f *fakeRepo) Get(_ context.Context, userID, id string) (Card, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if i := f.find(userID, id); i >= 0 {
		return f.cards[i], nil
	}
	return Card{}, ErrNotFound
}

func (f *fakeRepo) List(_ context.Context, userID string, q ListQuery, limit, skip int) ([]Card, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	needle := strings.ToLower(q.Q)
	var all []Card
	for _, c := range f.cards {
		if c.UserID != userID {
			continue
		}
		if q.LessonID == ManualLesson && c.LessonID != "" || q.LessonID != "" && q.LessonID != ManualLesson && c.LessonID != q.LessonID {
			continue
		}
		if needle != "" && !strings.Contains(strings.ToLower(c.Text), needle) && !strings.Contains(strings.ToLower(c.Lemma), needle) {
			continue
		}
		all = append(all, c)
	}
	slices.SortStableFunc(all, func(a, b Card) int { return b.CreatedAt.Compare(a.CreatedAt) })
	if skip >= len(all) {
		return nil, false, nil
	}
	end := min(skip+limit, len(all))
	return all[skip:end], end < len(all), nil
}

func (f *fakeRepo) LessonCounts(_ context.Context, userID string) (map[string]int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[string]int{}
	for _, c := range f.cards {
		if c.UserID == userID {
			out[c.LessonID]++
		}
	}
	return out, nil
}

func (f *fakeRepo) UpdateDetails(_ context.Context, userID, id string, d Details) (Card, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	i := f.find(userID, id)
	if i < 0 {
		return Card{}, ErrNotFound
	}
	if d.MeaningVi != nil {
		f.cards[i].MeaningVi = *d.MeaningVi
	}
	if d.IPA != nil {
		f.cards[i].IPA = *d.IPA
	}
	if d.ContextSentence != nil {
		f.cards[i].ContextSentence = *d.ContextSentence
	}
	return f.cards[i], nil
}

func (f *fakeRepo) UpdateSchedule(_ context.Context, userID, id string, expectedReps uint64, s Schedule) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	i := f.find(userID, id)
	if i < 0 || f.cards[i].Schedule.Reps != expectedReps {
		return false, nil
	}
	f.cards[i].Schedule = s
	return true, nil
}

// isDue mirrors the Mongo query: stored due ≤ now, or no schedule and saved before today.
func isDue(c Card, now, startOfToday time.Time) bool {
	if c.Schedule.Due.IsZero() {
		return c.CreatedAt.Before(startOfToday)
	}
	return !c.Schedule.Due.After(now)
}

func (f *fakeRepo) Due(_ context.Context, userID string, now, startOfToday time.Time, limit int) ([]Card, int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var due []Card
	for _, c := range f.cards {
		if c.UserID == userID && isDue(c, now, startOfToday) {
			due = append(due, c)
		}
	}
	// Cards without a schedule sort first (zero time), then by due.
	slices.SortStableFunc(due, func(a, b Card) int { return a.Schedule.Due.Compare(b.Schedule.Due) })
	return due[:min(limit, len(due))], len(due), nil
}

func (f *fakeRepo) NextDue(_ context.Context, userID string, now, startOfToday time.Time) (time.Time, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var best time.Time
	for _, c := range f.cards {
		if c.UserID != userID {
			continue
		}
		t := c.Schedule.Due
		if t.IsZero() {
			if c.CreatedAt.Before(startOfToday) {
				continue
			}
			t = startOfToday.AddDate(0, 0, 1)
		}
		if t.After(now) && (best.IsZero() || t.Before(best)) {
			best = t
		}
	}
	return best, !best.IsZero(), nil
}

func (f *fakeRepo) Delete(_ context.Context, userID, id string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	i := f.find(userID, id)
	if i < 0 {
		return false, nil
	}
	f.cards = slices.Delete(f.cards, i, i+1)
	return true, nil
}

type fakeLogs struct {
	mu   sync.Mutex
	logs []ReviewLog
}

func (f *fakeLogs) Add(_ context.Context, l ReviewLog) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.logs = append(f.logs, l)
	return nil
}

func (f *fakeLogs) DeleteByCard(_ context.Context, userID, cardID string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	before := len(f.logs)
	f.logs = slices.DeleteFunc(f.logs, func(l ReviewLog) bool { return l.UserID == userID && l.CardID == cardID })
	return before - len(f.logs), nil
}

func (f *fakeLogs) CountSince(_ context.Context, userID string, c Context, since time.Time) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, l := range f.logs {
		if l.UserID == userID && l.Context == c && !l.ReviewedAt.Before(since) {
			n++
		}
	}
	return n, nil
}

func (f *fakeLogs) forCard(cardID string) []ReviewLog {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []ReviewLog
	for _, l := range f.logs {
		if l.CardID == cardID {
			out = append(out, l)
		}
	}
	return out
}

// fakeTimezones: u2 lives in UTC, everyone else in Viet Nam.
type fakeTimezones struct{}

func (fakeTimezones) Location(_ context.Context, userID string) (*time.Location, error) {
	if userID == "u2" {
		return time.UTC, nil
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

type fakeVocabulary map[string][]VocabItem

func (f fakeVocabulary) Vocabulary(_ context.Context, lessonID string) ([]VocabItem, error) {
	items, ok := f[lessonID]
	if !ok {
		return nil, ErrLessonNotFound
	}
	return items, nil
}

// fakePronunciations knows the IPA of a few base forms.
type fakePronunciations map[string]string

func (f fakePronunciations) IPA(_ context.Context, lemma string) (string, error) {
	return f[lemma], nil
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

// fakeClock is a settable clock.
type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) set(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = t
}

type testEnv struct {
	svc   *Service
	repo  *fakeRepo
	logs  *fakeLogs
	clock *fakeClock
}

// newEnv starts at 2026-09-29 08:00 UTC (15:00 in Viet Nam). Lessons: lesson1 (annotated),
// lesson2 (no annotations yet).
func newEnv() *testEnv {
	e := &testEnv{repo: &fakeRepo{}, logs: &fakeLogs{}, clock: &fakeClock{t: time.Date(2026, 9, 29, 8, 0, 0, 0, time.UTC)}}
	e.svc = NewService(Deps{
		Repo: e.repo,
		Logs: e.logs,
		LessonExists: func(_ context.Context, id string) (bool, error) {
			return id == "lesson1" || id == "lesson2", nil
		},
		Timezones: fakeTimezones{},
		Vocabulary: fakeVocabulary{
			"lesson1": {
				{Lemma: "go", Text: "went", MeaningVi: "đi", IPA: "/ɡəʊ/", SentenceIndex: 0, Sentence: "We went to the park."},
				{Lemma: "give up", Text: "gave up", MeaningVi: "từ bỏ", SentenceIndex: 1, Sentence: "He gave up smoking."},
			},
			"lesson2": {},
		},
		Titles:         fakeTitles{"lesson1": "A day at the park"},
		Pronunciations: fakePronunciations{"house": "/haʊs/", "home": "/həʊm/"},
		Now:            e.clock.Now,
	})
	return e
}

func newTestService() (*Service, *fakeRepo) {
	e := newEnv()
	return e.svc, e.repo
}

func (f *fakeRepo) CountDue(_ context.Context, userID string, before, createdBefore time.Time) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, c := range f.cards {
		if c.UserID != userID {
			continue
		}
		if c.Schedule.Due.IsZero() && c.CreatedAt.Before(createdBefore) || !c.Schedule.Due.IsZero() && c.Schedule.Due.Before(before) {
			n++
		}
	}
	return n, nil
}

func (f *fakeRepo) Count(_ context.Context, userID string, since *time.Time) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, c := range f.cards {
		if c.UserID == userID && (since == nil || !c.CreatedAt.Before(*since)) {
			n++
		}
	}
	return n, nil
}
