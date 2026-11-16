package games

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

// testNow is 2026-11-13 15:00 in Ho Chi Minh City.
var testNow = time.Date(2026, 11, 13, 8, 0, 0, 0, time.UTC)

type fakeBests struct {
	mu   sync.Mutex
	recs map[[2]string]Best
}

func (f *fakeBests) List(_ context.Context, userID string) ([]Best, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Best
	for k, b := range f.recs {
		if k[0] == userID {
			out = append(out, b)
		}
	}
	return out, nil
}

func (f *fakeBests) Raise(_ context.Context, userID string, game Game, score int, at time.Time) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	k := [2]string{userID, string(game)}
	if b, ok := f.recs[k]; ok && b.Score >= score {
		return false, nil
	}
	f.recs[k] = Best{UserID: userID, Game: game, Score: score, At: at}
	return true, nil
}

type fakeDaily struct {
	mu   sync.Mutex
	recs map[[2]string]DailyResult
}

func (f *fakeDaily) Get(_ context.Context, userID, day string) (DailyResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.recs[[2]string{userID, day}]
	if !ok {
		return DailyResult{}, ErrNotFound
	}
	return r, nil
}

func (f *fakeDaily) Save(_ context.Context, r DailyResult) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	k := [2]string{r.UserID, r.Day}
	if _, ok := f.recs[k]; ok {
		return ErrExists
	}
	f.recs[k] = r
	return nil
}

func (f *fakeDaily) List(_ context.Context, userID string) ([]DailyResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []DailyResult
	for k, r := range f.recs {
		if k[0] == userID {
			out = append(out, r)
		}
	}
	return out, nil
}

type fakeSpots struct {
	mu    sync.Mutex
	items []SpotItem
	next  int
}

func (f *fakeSpots) ListByPoint(_ context.Context, pointID string) ([]SpotItem, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []SpotItem
	for _, it := range f.items {
		if it.PointID == pointID {
			out = append(out, it)
		}
	}
	return out, nil
}

func (f *fakeSpots) Approved(_ context.Context, pointIDs []string) ([]SpotItem, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []SpotItem
	for _, it := range f.items {
		if it.Status == SpotApproved && (pointIDs == nil || slices.Contains(pointIDs, it.PointID)) {
			out = append(out, it)
		}
	}
	return out, nil
}

func (f *fakeSpots) Counts(context.Context) (map[string]SpotCount, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[string]SpotCount{}
	for _, it := range f.items {
		c := out[it.PointID]
		if it.Status == SpotApproved {
			c.Approved++
		} else {
			c.Draft++
		}
		out[it.PointID] = c
	}
	return out, nil
}

func (f *fakeSpots) Create(_ context.Context, items []SpotItem) ([]SpotItem, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]SpotItem, len(items))
	for i, it := range items {
		f.next++
		it.ID = "s" + strconv.Itoa(f.next)
		f.items = append(f.items, it)
		out[i] = it
	}
	return out, nil
}

func (f *fakeSpots) Get(_ context.Context, id string) (SpotItem, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, it := range f.items {
		if it.ID == id {
			return it, nil
		}
	}
	return SpotItem{}, ErrNotFound
}

func (f *fakeSpots) Update(_ context.Context, it SpotItem) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := range f.items {
		if f.items[i].ID == it.ID {
			f.items[i] = it
			return nil
		}
	}
	return ErrNotFound
}

func (f *fakeSpots) Delete(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := range f.items {
		if f.items[i].ID == id {
			f.items = slices.Delete(f.items, i, i+1)
			return nil
		}
	}
	return ErrNotFound
}

type fakeCards struct{ cards map[string][]Card }

func (f fakeCards) Cards(_ context.Context, userID string) ([]Card, error) {
	return f.cards[userID], nil
}

type fakePictures struct{ urls map[string]string }

func (f fakePictures) ImageURLs(_ context.Context, lemmas []string) (map[string]string, error) {
	out := map[string]string{}
	for _, l := range lemmas {
		if u, ok := f.urls[l]; ok {
			out[l] = u
		}
	}
	return out, nil
}

type fakeMeanings struct{ m map[string]string }

func (f fakeMeanings) MeaningsOf(_ context.Context, lemmas []string) (map[string]string, error) {
	out := map[string]string{}
	for _, l := range lemmas {
		if v, ok := f.m[l]; ok {
			out[l] = v
		}
	}
	return out, nil
}

type fakeLessons struct{ words map[string][]LessonWord }

func (f fakeLessons) LessonWords(_ context.Context, id string) ([]LessonWord, error) {
	return f.words[id], nil
}

type fakeStudy struct {
	studied map[string][]string
	current map[string]string
	open    map[string]bool // lesson ids the learners may open
	level   string
}

func (f fakeStudy) StudiedLessons(_ context.Context, userID string) ([]string, error) {
	return f.studied[userID], nil
}

func (f fakeStudy) CurrentLesson(_ context.Context, userID string) (string, error) {
	return f.current[userID], nil
}

func (f fakeStudy) CanOpen(_ context.Context, _ string, isAdmin bool, lessonID string) (bool, error) {
	return isAdmin || f.open[lessonID], nil
}

func (f fakeStudy) Level(context.Context, string) (string, error) { return f.level, nil }

type fakeWords struct {
	mu    sync.Mutex
	calls [][]string
}

func (f *fakeWords) GameMisses(_ context.Context, _ string, lemmas []string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, lemmas)
	return len(lemmas), nil
}

type fakeVerbs struct {
	mu     sync.Mutex
	misses []VerbMiss
	due    []string
	other  []string
}

func (f *fakeVerbs) GameMisses(_ context.Context, _ string, misses []VerbMiss) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.misses = append(f.misses, misses...)
	return nil
}

func (f *fakeVerbs) LearnedBases(context.Context, string) ([]string, []string, error) {
	return f.due, f.other, nil
}

type fakeGrammar struct {
	sources []PointSource
	started map[string][]string
}

func (f fakeGrammar) PublishedSources(context.Context) ([]PointSource, error) { return f.sources, nil }

func (f fakeGrammar) StartedPoints(_ context.Context, userID string) ([]string, error) {
	return f.started[userID], nil
}

type fakeAI struct {
	items []ai.GrammarSpotItem
	err   error
	req   ai.GrammarSpotRequest
}

func (f *fakeAI) GrammarSpotItems(_ context.Context, req ai.GrammarSpotRequest) ([]ai.GrammarSpotItem, error) {
	f.req = req
	return f.items, f.err
}

type fakeDays struct{ n int }

func (f *fakeDays) MarkStudied(context.Context, string) error { f.n++; return nil }

type fakeZones struct{}

func (fakeZones) Location(context.Context, string) (*time.Location, error) {
	return time.LoadLocation("Asia/Ho_Chi_Minh")
}

type env struct {
	svc     *Service
	bests   *fakeBests
	daily   *fakeDaily
	spots   *fakeSpots
	words   *fakeWords
	verbs   *fakeVerbs
	ai      *fakeAI
	days    *fakeDays
	cards   fakeCards
	pics    fakePictures
	lessons fakeLessons
	study   *fakeStudy
	grammar *fakeGrammar
	now     time.Time
}

// fruit are words with pictures and different meanings.
var fruit = []struct{ lemma, vi string }{
	{"apple", "quả táo"}, {"banana", "quả chuối"}, {"orange", "quả cam"}, {"pear", "quả lê"},
	{"peach", "quả đào"}, {"grape", "quả nho"}, {"lemon", "quả chanh"}, {"mango", "quả xoài"},
}

func newEnv(t *testing.T) *env {
	t.Helper()
	e := &env{
		bests:   &fakeBests{recs: map[[2]string]Best{}},
		daily:   &fakeDaily{recs: map[[2]string]DailyResult{}},
		spots:   &fakeSpots{},
		words:   &fakeWords{},
		verbs:   &fakeVerbs{},
		ai:      &fakeAI{},
		days:    &fakeDays{},
		cards:   fakeCards{cards: map[string][]Card{}},
		pics:    fakePictures{urls: map[string]string{}},
		lessons: fakeLessons{words: map[string][]LessonWord{}},
		study:   &fakeStudy{studied: map[string][]string{}, current: map[string]string{}, open: map[string]bool{}, level: "A1"},
		grammar: &fakeGrammar{started: map[string][]string{}},
		now:     testNow,
	}
	for i, f := range fruit {
		due := testNow.AddDate(0, 0, 3)
		if i < 2 {
			due = testNow.Add(-time.Hour)
		}
		e.cards.cards["u1"] = append(e.cards.cards["u1"], Card{Lemma: f.lemma, Text: f.lemma, MeaningVi: f.vi, LessonID: "L1", Due: due})
		e.pics.urls[f.lemma] = "/api/words/" + f.lemma + "/image"
	}
	e.svc = NewService(Deps{
		Bests: e.bests, Daily: e.daily, Spots: e.spots, Cards: e.cards, Pictures: e.pics,
		Meanings: fakeMeanings{m: map[string]string{"house": "ngôi nhà"}}, Lessons: e.lessons, Study: e.study,
		Words: e.words, Verbs: e.verbs, Grammar: e.grammar, AI: e.ai, StudyDays: e.days, Timezones: fakeZones{},
		Now: func() time.Time { return e.now }, Seed: 7,
	})
	return e
}

// --- HTTP ---

func resolver(_ context.Context, token string) (httpx.Principal, error) {
	switch token {
	case "member":
		return httpx.Principal{UserID: "u1", Role: "member"}, nil
	case "guest":
		return httpx.Principal{UserID: "u2", Role: httpx.RoleGuest}, nil
	case "admin":
		return httpx.Principal{UserID: "u3", Role: httpx.RoleAdmin}, nil
	case "new":
		return httpx.Principal{UserID: "u9", Role: "member"}, nil
	}
	return httpx.Principal{}, errors.New("no session")
}

func (e *env) mux() *http.ServeMux {
	mux := http.NewServeMux()
	NewHandler(e.svc, slog.New(slog.DiscardHandler)).Register(mux, httpx.RequireAuth(resolver))
	return mux
}

func call(t *testing.T, mux http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.AddCookie(&http.Cookie{Name: httpx.SessionCookieName, Value: token})
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}
