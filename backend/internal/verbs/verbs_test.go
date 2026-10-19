package verbs

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

var testNow = time.Date(2026, 10, 16, 8, 0, 0, 0, time.UTC)

type fakeRepo struct {
	mu   sync.Mutex
	recs map[[2]string]Progress
	fail error
}

func newFakeRepo() *fakeRepo { return &fakeRepo{recs: map[[2]string]Progress{}} }

func (f *fakeRepo) List(_ context.Context, userID string) ([]Progress, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fail != nil {
		return nil, f.fail
	}
	var out []Progress
	for k, p := range f.recs {
		if k[0] == userID {
			out = append(out, p)
		}
	}
	slices.SortFunc(out, func(a, b Progress) int { return strings.Compare(a.Base, b.Base) })
	return out, nil
}

func (f *fakeRepo) Save(_ context.Context, records []Progress) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, p := range records {
		f.recs[[2]string{p.UserID, p.Base}] = p
	}
	return nil
}

func TestLearnAndUnlearn(t *testing.T) {
	t.Parallel()
	p := Learn(Progress{Base: "go"}, testNow)
	if !p.Learned() || p.Level != 1 || !p.DueAt.Equal(testNow.Add(Intervals[0])) || !p.LearnedAt.Equal(testNow) {
		t.Fatalf("learned = %+v", p)
	}
	// Learning again leaves the record as it is.
	if again := Learn(p, testNow.Add(time.Hour)); again != p {
		t.Fatalf("learned twice = %+v", again)
	}
	p.Right = 3
	u := Unlearn(p, testNow.Add(time.Hour))
	if u.Learned() || u.Level != 0 || !u.DueAt.IsZero() || u.Right != 3 {
		t.Fatalf("unlearned = %+v", u)
	}
}

func TestApplyReviewsOnlyLearnedVerbs(t *testing.T) {
	t.Parallel()
	// A verb not learned only counts the answer.
	if p := Apply(Progress{Base: "go"}, true, testNow); p.Learned() || p.Level != 0 || !p.DueAt.IsZero() || p.Right != 1 {
		t.Fatalf("not learned = %+v", p)
	}
	p := Learn(Progress{Base: "go"}, testNow)
	// Right before it is due: the box and the wait stay.
	early := Apply(p, true, testNow.Add(time.Hour))
	if early.Level != 1 || !early.DueAt.Equal(p.DueAt) || early.Right != 1 {
		t.Fatalf("early right = %+v", early)
	}
	// Right once due: next box, longer wait.
	due := p.DueAt
	up := Apply(early, true, due)
	if up.Level != 2 || !up.DueAt.Equal(due.Add(Intervals[1])) {
		t.Fatalf("due right = %+v", up)
	}
	// Wrong: still learned, back in box 1, asked again a day later.
	down := Apply(up, false, due.Add(time.Hour))
	if !down.Learned() || down.Level != 1 || !down.DueAt.Equal(due.Add(time.Hour+Intervals[0])) || down.Wrong != 1 {
		t.Fatalf("wrong = %+v", down)
	}
	// The last box keeps its wait.
	top := Progress{LearnedAt: testNow, Level: len(Intervals), DueAt: testNow}
	if got := Apply(top, true, testNow); got.Level != len(Intervals) || !got.DueAt.Equal(testNow.Add(Intervals[len(Intervals)-1])) {
		t.Fatalf("top box = %+v", got)
	}
}

func TestServiceLearnsAndRecords(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := NewService(repo, func() time.Time { return testNow })
	got, err := svc.SetLearned(t.Context(), "u1", []string{"go", "see", "go"}, true)
	if err != nil || len(got) != 2 || !got[0].Learned() || got[0].Base != "go" || got[1].Base != "see" {
		t.Fatalf("learned = %+v, %v", got, err)
	}
	got, err = svc.Record(t.Context(), "u1", []Result{{"go", false}, {"see", true}, {"go", true}})
	if err != nil || len(got) != 2 || got[0].Right != 1 || got[0].Wrong != 0 || got[1].Right != 1 {
		t.Fatalf("records = %+v, %v", got, err)
	}
	got, _ = svc.SetLearned(t.Context(), "u1", []string{"see"}, false)
	if got[0].Learned() || got[0].Right != 1 {
		t.Fatalf("unlearned see = %+v", got[0])
	}
	list, _ := svc.List(t.Context(), "u1")
	if len(list) != 2 || !list[0].Learned() || list[1].Learned() {
		t.Fatalf("list = %+v", list)
	}
	if other, _ := svc.List(t.Context(), "u2"); len(other) != 0 {
		t.Fatalf("u2 = %+v", other)
	}
}

func TestServiceRejectsBadInput(t *testing.T) {
	t.Parallel()
	svc := NewService(newFakeRepo(), func() time.Time { return testNow })
	many := make([]Result, MaxResults+1)
	for i := range many {
		many[i] = Result{Base: "go"}
	}
	for name, results := range map[string][]Result{
		"empty":    nil,
		"too many": many,
		"bad base": {{Base: "Go!"}},
		"spaces":   {{Base: "go on"}},
	} {
		var verr *ValidationError
		if _, err := svc.Record(t.Context(), "u1", results); !errors.As(err, &verr) || verr.Fields["results"] == "" {
			t.Errorf("record %s: %v", name, err)
		}
	}
	var verr *ValidationError
	if _, err := svc.SetLearned(t.Context(), "u1", nil, true); !errors.As(err, &verr) || verr.Fields["bases"] == "" {
		t.Errorf("learn nothing: %v", err)
	}
	if _, err := svc.SetLearned(t.Context(), "u1", []string{"GO"}, true); !errors.As(err, &verr) {
		t.Errorf("learn bad base: %v", err)
	}
}

func TestHandler(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	mux := http.NewServeMux()
	auth := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(httpx.WithPrincipal(r.Context(), httpx.Principal{UserID: "u1"})))
		})
	}
	NewHandler(NewService(repo, func() time.Time { return testNow }), slog.New(slog.NewTextHandler(io.Discard, nil))).Register(mux, auth)
	call := func(method, path, body string) (int, map[string]any) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(rec, req)
		var out map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		return rec.Code, out
	}
	first := func(out map[string]any) map[string]any { return out["verbs"].([]any)[0].(map[string]any) }

	if code, out := call("GET", "/api/verbs/progress", ""); code != 200 || len(out["verbs"].([]any)) != 0 {
		t.Fatalf("empty list = %d %v", code, out)
	}
	code, out := call("POST", "/api/verbs/learned", `{"bases":["go"],"learned":true}`)
	if code != 200 || first(out)["base"] != "go" || first(out)["learnedAt"] == nil || first(out)["level"] != 1.0 || first(out)["dueAt"] == nil {
		t.Fatalf("learned = %d %v", code, out)
	}
	code, out = call("POST", "/api/verbs/results", `{"results":[{"base":"go","right":false}]}`)
	if code != 200 || first(out)["wrong"] != 1.0 || first(out)["learnedAt"] == nil {
		t.Fatalf("record = %d %v", code, out)
	}
	code, out = call("POST", "/api/verbs/learned", `{"bases":["go"],"learned":false}`)
	if code != 200 || first(out)["learnedAt"] != nil || first(out)["dueAt"] != nil {
		t.Fatalf("unlearned = %d %v", code, out)
	}
	if code, _ := call("POST", "/api/verbs/results", `{"results":[]}`); code != http.StatusBadRequest {
		t.Fatalf("empty round = %d", code)
	}
	if code, _ := call("POST", "/api/verbs/learned", `{"bases":[]}`); code != http.StatusBadRequest {
		t.Fatalf("empty learned = %d", code)
	}

	repo.fail = errors.New("db down")
	if code, _ := call("GET", "/api/verbs/progress", ""); code != http.StatusInternalServerError {
		t.Fatalf("db down = %d", code)
	}
}
