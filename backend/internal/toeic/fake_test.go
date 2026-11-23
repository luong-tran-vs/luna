package toeic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

var testNow = time.Date(2026, 11, 19, 3, 0, 0, 0, time.UTC)

// clone deep-copies through JSON so the fakes never share slices or maps with callers.
func clone[T any](v T) T {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	var out T
	if err := json.Unmarshal(b, &out); err != nil {
		panic(err)
	}
	return out
}

// --- groups ---

type fakeGroups struct {
	mu   sync.Mutex
	recs map[string]Group
}

func (f *fakeGroups) copyOf(g Group) Group {
	c := clone(g)
	c.ID, c.Status, c.Version, c.CreatedAt, c.UpdatedAt = g.ID, g.Status, g.Version, g.CreatedAt, g.UpdatedAt
	return c
}

func (f *fakeGroups) List(_ context.Context, flt GroupFilter) ([]Group, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Group
	for _, g := range f.recs {
		if (flt.Part == 0 || g.Part == flt.Part) && (flt.Status == "" || g.Status == flt.Status) {
			out = append(out, f.copyOf(g))
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Part != out[j].Part {
			return out[i].Part < out[j].Part
		}
		return out[i].Key < out[j].Key
	})
	return out, nil
}

func (f *fakeGroups) ByKeys(_ context.Context, keys []string) ([]Group, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Group
	for _, k := range keys {
		if g, ok := f.recs[k]; ok {
			out = append(out, f.copyOf(g))
		}
	}
	return out, nil
}

func (f *fakeGroups) Published(ctx context.Context, part Part) ([]Group, error) {
	return f.List(ctx, GroupFilter{Part: part, Status: StatusPublished})
}

func (f *fakeGroups) Upsert(_ context.Context, gs []Group) (created, replaced []string, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, g := range gs {
		if old, ok := f.recs[g.Key]; ok {
			n := f.copyOf(g)
			n.ID, n.Status, n.CreatedAt, n.Version = old.ID, old.Status, old.CreatedAt, old.Version+1
			f.recs[g.Key] = n
			replaced = append(replaced, g.Key)
			continue
		}
		n := f.copyOf(g)
		n.ID = "g-" + g.Key
		f.recs[g.Key] = n
		created = append(created, g.Key)
	}
	return created, replaced, nil
}

func (f *fakeGroups) SetStatus(_ context.Context, key string, s Status) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	g, ok := f.recs[key]
	if !ok {
		return ErrNotFound
	}
	g.Status = s
	f.recs[key] = g
	return nil
}

func (f *fakeGroups) Delete(_ context.Context, keys []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, k := range keys {
		delete(f.recs, k)
	}
	return nil
}

// --- images ---

type fakeImages struct {
	mu   sync.Mutex
	recs map[[2]string]Image
}

func (f *fakeImages) Get(_ context.Context, key, slot string) (Image, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	img, ok := f.recs[[2]string{key, slot}]
	if !ok {
		return Image{}, ErrNotFound
	}
	return img, nil
}

func (f *fakeImages) Save(_ context.Context, img Image) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.recs[[2]string{img.GroupKey, img.Slot}] = img
	return nil
}

func (f *fakeImages) Delete(_ context.Context, key, slot string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.recs, [2]string{key, slot})
	return nil
}

func (f *fakeImages) Has(_ context.Context, keys []string) (map[string][]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[string][]string{}
	for k := range f.recs {
		if slices.Contains(keys, k[0]) {
			out[k[0]] = append(out[k[0]], k[1])
		}
	}
	return out, nil
}

// --- tests ---

type fakeTests struct {
	mu   sync.Mutex
	recs map[string]Test
}

func (f *fakeTests) List(_ context.Context, onlyPublished bool) ([]Test, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Test
	for _, t := range f.recs {
		if !onlyPublished || t.Status == StatusPublished {
			t.Groups = slices.Clone(t.Groups)
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Order != out[j].Order {
			return out[i].Order < out[j].Order
		}
		return out[i].Key < out[j].Key
	})
	return out, nil
}

func (f *fakeTests) Get(_ context.Context, key string) (Test, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.recs[key]
	if !ok {
		return Test{}, ErrNotFound
	}
	t.Groups = slices.Clone(t.Groups)
	return t, nil
}

func (f *fakeTests) Save(_ context.Context, t Test) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if old, ok := f.recs[t.Key]; ok {
		t.ID, t.CreatedAt = old.ID, old.CreatedAt
	} else {
		t.ID = "t-" + t.Key
	}
	t.Groups = slices.Clone(t.Groups)
	f.recs[t.Key] = t
	return nil
}

func (f *fakeTests) SetStatus(_ context.Context, key string, s Status) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.recs[key]
	if !ok {
		return ErrNotFound
	}
	t.Status = s
	f.recs[key] = t
	return nil
}

func (f *fakeTests) Delete(_ context.Context, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.recs[key]; !ok {
		return ErrNotFound
	}
	delete(f.recs, key)
	return nil
}

// --- attempts ---

type fakeAttempts struct {
	mu   sync.Mutex
	seq  int
	recs []Attempt
}

func copyAttempt(a Attempt) Attempt {
	s := clone(a.State())
	out := a
	out.SetState(s)
	out.GroupKeys, out.QuestionIDs = slices.Clone(a.GroupKeys), slices.Clone(a.QuestionIDs)
	if a.Result != nil {
		r := clone(*a.Result)
		out.Result = &r
	}
	return out
}

func (f *fakeAttempts) Create(_ context.Context, a Attempt) (Attempt, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seq++
	a.ID = fmt.Sprintf("a%d", f.seq)
	f.recs = append(f.recs, copyAttempt(a))
	return copyAttempt(a), nil
}

func (f *fakeAttempts) find(id string) int {
	return slices.IndexFunc(f.recs, func(a Attempt) bool { return a.ID == id })
}

func (f *fakeAttempts) Get(_ context.Context, userID, id string) (Attempt, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	i := f.find(id)
	if i < 0 || f.recs[i].UserID != userID {
		return Attempt{}, ErrNotFound
	}
	return copyAttempt(f.recs[i]), nil
}

func (f *fakeAttempts) Save(_ context.Context, a Attempt) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	i := f.find(a.ID)
	if i < 0 {
		return ErrNotFound
	}
	f.recs[i] = copyAttempt(a)
	return nil
}

func (f *fakeAttempts) Submit(ctx context.Context, a Attempt) error { return f.Save(ctx, a) }

func (f *fakeAttempts) List(_ context.Context, userID string, flt AttemptFilter) ([]Attempt, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Attempt
	for i := len(f.recs) - 1; i >= 0; i-- {
		a := f.recs[i]
		if a.UserID != userID || (flt.TestKey != "" && a.Source.TestKey != flt.TestKey) ||
			(flt.Status != "" && a.Status != flt.Status) || (flt.Counted && !a.Counted) {
			continue
		}
		out = append(out, copyAttempt(a))
		if flt.Limit > 0 && len(out) == flt.Limit {
			break
		}
	}
	return out, nil
}

func (f *fakeAttempts) InProgress(ctx context.Context, userID, testKey string) (Attempt, error) {
	list, _ := f.List(ctx, userID, AttemptFilter{TestKey: testKey, Status: AttemptInProgress, Limit: 1})
	if len(list) == 0 {
		return Attempt{}, ErrNotFound
	}
	return list[0], nil
}

func (f *fakeAttempts) AnsweredQuestions(_ context.Context, userID string) (map[string]bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[string]bool{}
	for _, a := range f.recs {
		if a.UserID == userID && a.Status == AttemptSubmitted {
			for _, id := range a.QuestionIDs {
				out[id] = true
			}
		}
	}
	return out, nil
}

func (f *fakeAttempts) WithGroup(_ context.Context, key string) ([]Attempt, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Attempt
	for _, a := range f.recs {
		if a.Status == AttemptSubmitted && slices.Contains(a.GroupKeys, key) {
			out = append(out, copyAttempt(a))
		}
	}
	return out, nil
}

func (f *fakeAttempts) CountWithTest(_ context.Context, key string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, a := range f.recs {
		if a.Source.TestKey == key {
			n++
		}
	}
	return n, nil
}

func (f *fakeAttempts) UpdateResult(_ context.Context, id string, r Result) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	i := f.find(id)
	if i < 0 {
		return ErrNotFound
	}
	rc := clone(r)
	f.recs[i].Result = &rc
	return nil
}

// --- mistakes, goals ---

type fakeMistakes struct {
	mu   sync.Mutex
	recs map[[2]string]Mistake
}

func (f *fakeMistakes) List(_ context.Context, userID string, part Part) ([]Mistake, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Mistake
	for k, m := range f.recs {
		if k[0] == userID && (part == 0 || m.Part == part) {
			out = append(out, m)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].LastWrongAt.Equal(out[j].LastWrongAt) {
			return out[i].LastWrongAt.Before(out[j].LastWrongAt)
		}
		return out[i].QuestionID < out[j].QuestionID
	})
	return out, nil
}

func (f *fakeMistakes) Get(_ context.Context, userID, qid string) (Mistake, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, ok := f.recs[[2]string{userID, qid}]
	if !ok {
		return Mistake{}, ErrNotFound
	}
	return m, nil
}

func (f *fakeMistakes) Upsert(_ context.Context, m Mistake) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.recs[[2]string{m.UserID, m.QuestionID}] = m
	return nil
}

func (f *fakeMistakes) Delete(_ context.Context, userID, qid string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.recs, [2]string{userID, qid})
	return nil
}

func (f *fakeMistakes) Count(ctx context.Context, userID string) (int, error) {
	list, _ := f.List(ctx, userID, 0)
	return len(list), nil
}

type fakeGoals struct {
	mu   sync.Mutex
	recs map[string]Goal
}

func (f *fakeGoals) Get(_ context.Context, userID string) (Goal, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	g, ok := f.recs[userID]
	if !ok {
		return Goal{}, ErrNotFound
	}
	return g, nil
}

func (f *fakeGoals) Save(_ context.Context, g Goal) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.recs[g.UserID] = g
	return nil
}

func (f *fakeGoals) Delete(_ context.Context, userID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.recs, userID)
	return nil
}

// --- ports ---

type fakeDays struct {
	mu   sync.Mutex
	days []string
}

func (f *fakeDays) MarkStudied(_ context.Context, userID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.days = append(f.days, userID)
	return nil
}

func (f *fakeDays) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.days)
}

type fakeCards struct {
	mu    sync.Mutex
	saved []CardInput
}

func (f *fakeCards) SaveWord(_ context.Context, _ string, w CardInput) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.saved = append(f.saved, w)
	return nil
}

type fakeZones struct{}

func (fakeZones) Location(context.Context, string) (*time.Location, error) {
	return time.FixedZone("ICT", 7*3600), nil
}

// --- environment ---

type env struct {
	groups   *fakeGroups
	images   *fakeImages
	tests    *fakeTests
	attempts *fakeAttempts
	mistakes *fakeMistakes
	goals    *fakeGoals
	days     *fakeDays
	cards    *fakeCards
	now      time.Time
	svc      *Service
}

func newEnv(t *testing.T) *env {
	t.Helper()
	e := &env{
		groups:   &fakeGroups{recs: map[string]Group{}},
		images:   &fakeImages{recs: map[[2]string]Image{}},
		tests:    &fakeTests{recs: map[string]Test{}},
		attempts: &fakeAttempts{},
		mistakes: &fakeMistakes{recs: map[[2]string]Mistake{}},
		goals:    &fakeGoals{recs: map[string]Goal{}},
		days:     &fakeDays{},
		cards:    &fakeCards{},
		now:      testNow,
	}
	e.svc = NewService(Deps{
		Groups: e.groups, Images: e.images, Tests: e.tests, Attempts: e.attempts, Mistakes: e.mistakes, Goals: e.goals,
		StudyDays: e.days, Cards: e.cards, Timezones: fakeZones{}, Now: func() time.Time { return e.now }, Seed: 7,
	})
	return e
}

// --- content builders ---

func q4(answer int, tags ...string) Question {
	return Question{No: 1, Options: []string{"alpha", "bravo", "charlie", "delta"}, Answer: answer, ExplainVi: "Vì vậy.", Tags: tags}
}

func p5(key string, answer int) Group {
	q := q4(answer, "word-form")
	q.Prompt = "The report was written ___ by the team."
	return Group{Key: key, Part: 5, Level: 600, Questions: []Question{q}}
}

func p6(key string) Group {
	g := Group{Key: key, Part: 6, Level: 600, Docs: []Doc{{Kind: "memo", Title: "Office move", Body: []Block{
		{Kind: "p", Text: "We will [[1]] on Monday. [[2]] Please [[3]] your desk. Thanks for your [[4]]."},
	}}}}
	for i := 1; i <= 4; i++ {
		q := q4(0, "vocabulary")
		q.No = i
		g.Questions = append(g.Questions, q)
	}
	g.Questions[1].Tags = []string{"sentence-insert"}
	return g
}

func p7(key string, n int) Group {
	g := Group{Key: key, Part: 7, Level: 750, Docs: []Doc{{Kind: "notice", Title: "Garden notice", Body: []Block{
		{Kind: "p", Text: "The garden opens at 9 a.m. on Saturdays."},
	}}}}
	for i := 1; i <= n; i++ {
		q := q4(1, "detail")
		q.No, q.Prompt = i, fmt.Sprintf("Question %d?", i)
		q.Evidence = []Ref{{Doc: 0, Block: 0, Text: "opens at 9 a.m."}}
		q.TranslateVi = "Bản dịch."
		q.Vocab = []Vocab{{Word: "garden", MeaningVi: "khu vườn"}}
		g.Questions = append(g.Questions, q)
	}
	return g
}

func p1(key string) Group {
	q := q4(0, "photo-people")
	return Group{Key: key, Part: 1, Level: 450, Scene: "A man stacks boxes.", Questions: []Question{q}}
}

func p2(key string) Group {
	q := Question{No: 1, Prompt: "When is the shipment arriving?", Options: []string{"By Thursday.", "Yes, it was.", "To the office."},
		Answer: 0, ExplainVi: "Hỏi When.", Tags: []string{"q-wh"}}
	return Group{Key: key, Part: 2, Level: 600, Script: []Turn{{Speaker: "w", Accent: "gb", Text: q.Prompt}}, Questions: []Question{q}}
}

func p3(key string) Group {
	g := Group{Key: key, Part: 3, Level: 600, Script: []Turn{
		{Speaker: "m", Accent: "us", Text: "I'd like to pick up the brochures I ordered."},
		{Speaker: "w", Accent: "gb", Text: "Our printer broke down this morning."},
	}}
	for i := 1; i <= 3; i++ {
		q := q4(2, "detail")
		q.No, q.Prompt = i, "What is the problem?"
		g.Questions = append(g.Questions, q)
	}
	return g
}

func p4(key string) Group {
	g := p3(key)
	g.Part = 4
	g.Script = []Turn{{Speaker: "m", Accent: "au", Text: "Welcome to the tour of the factory."}}
	return g
}

// seed stores groups as published.
func (e *env) seed(gs ...Group) {
	for _, g := range gs {
		Normalize(&g)
		g.Status, g.Version, g.ID = StatusPublished, 1, "g-"+g.Key
		e.groups.recs[g.Key] = g
	}
}

// readingTest stores a published full test with Part 5–7: 30 Part 5 questions, 4 texts of Part 6
// and 54 questions of Part 7 (groups of 3).
func (e *env) readingTest(key string) Test {
	t := Test{Key: key, Title: "Luna Test 01", Kind: KindFull, Order: 1, Status: StatusPublished}
	for i := range 30 {
		g := p5(fmt.Sprintf("%s-p5-%02d", key, i), i%4)
		e.seed(g)
		t.Groups = append(t.Groups, g.Key)
	}
	for i := range 4 {
		g := p6(fmt.Sprintf("%s-p6-%d", key, i))
		e.seed(g)
		t.Groups = append(t.Groups, g.Key)
	}
	for i := range 18 {
		g := p7(fmt.Sprintf("%s-p7-%02d", key, i), 3)
		e.seed(g)
		t.Groups = append(t.Groups, g.Key)
	}
	e.tests.recs[key] = t
	return t
}

// --- HTTP ---

func resolver(_ context.Context, token string) (httpx.Principal, error) {
	switch token {
	case "member":
		return httpx.Principal{UserID: "u1", Role: "member"}, nil
	case "other":
		return httpx.Principal{UserID: "u4", Role: "member"}, nil
	case "guest":
		return httpx.Principal{UserID: "u2", Role: httpx.RoleGuest}, nil
	case "admin":
		return httpx.Principal{UserID: "u3", Role: httpx.RoleAdmin}, nil
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

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode %s: %v", rec.Body, err)
	}
	return v
}
