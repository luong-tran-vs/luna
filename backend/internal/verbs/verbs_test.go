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

func TestApplyMovesUpOnlyWhenDue(t *testing.T) {
	t.Parallel()
	p := Apply(Progress{Base: "go"}, true, testNow)
	if p.Level != 1 || !p.DueAt.Equal(testNow.Add(Intervals[0])) || p.Right != 1 || !p.Known() {
		t.Fatalf("first right = %+v", p)
	}
	// Right again the same day: not due yet, the box and the wait stay.
	again := Apply(p, true, testNow.Add(time.Hour))
	if again.Level != 1 || !again.DueAt.Equal(p.DueAt) || again.Right != 2 {
		t.Fatalf("early right = %+v", again)
	}
	// Right once due: next box, longer wait.
	later := testNow.Add(Intervals[0])
	up := Apply(again, true, later)
	if up.Level != 2 || !up.DueAt.Equal(later.Add(Intervals[1])) {
		t.Fatalf("due right = %+v", up)
	}
	// Wrong: back to box 0, no longer known.
	down := Apply(up, false, later.Add(time.Hour))
	if down.Level != 0 || !down.DueAt.IsZero() || down.Wrong != 1 || down.Known() {
		t.Fatalf("wrong = %+v", down)
	}
	// The last box keeps its wait.
	top := Progress{Level: len(Intervals), DueAt: testNow}
	if got := Apply(top, true, testNow); got.Level != len(Intervals) || !got.DueAt.Equal(testNow.Add(Intervals[len(Intervals)-1])) {
		t.Fatalf("top box = %+v", got)
	}
}

func TestRecordKeepsTheLastResultOfEachVerb(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := NewService(repo, func() time.Time { return testNow })
	got, err := svc.Record(t.Context(), "u1", []Result{{"go", false}, {"see", true}, {"go", true}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Base != "go" || got[0].Level != 1 || got[0].Right != 1 || got[0].Wrong != 0 || got[1].Base != "see" {
		t.Fatalf("records = %+v", got)
	}
	got, _ = svc.Record(t.Context(), "u1", []Result{{"see", false}})
	if got[0].Level != 0 || got[0].Right != 1 || got[0].Wrong != 1 {
		t.Fatalf("see after wrong = %+v", got[0])
	}
	list, _ := svc.List(t.Context(), "u1")
	if len(list) != 2 {
		t.Fatalf("list = %+v", list)
	}
	if other, _ := svc.List(t.Context(), "u2"); len(other) != 0 {
		t.Fatalf("u2 = %+v", other)
	}
}

func TestRecordRejectsBadInput(t *testing.T) {
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
			t.Errorf("%s: %v", name, err)
		}
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

	if code, out := call("GET", "/api/verbs/progress", ""); code != 200 || len(out["verbs"].([]any)) != 0 {
		t.Fatalf("empty list = %d %v", code, out)
	}
	code, out := call("POST", "/api/verbs/results", `{"results":[{"base":"go","right":true},{"base":"see","right":false}]}`)
	if code != 200 {
		t.Fatalf("record = %d %v", code, out)
	}
	verbs := out["verbs"].([]any)
	goRec, seeRec := verbs[0].(map[string]any), verbs[1].(map[string]any)
	if goRec["base"] != "go" || goRec["level"] != 1.0 || goRec["dueAt"] == nil || seeRec["dueAt"] != nil || seeRec["wrong"] != 1.0 {
		t.Fatalf("verbs = %v", verbs)
	}
	if code, _ := call("POST", "/api/verbs/results", `{"results":[]}`); code != http.StatusBadRequest {
		t.Fatalf("empty round = %d", code)
	}

	repo.fail = errors.New("db down")
	if code, _ := call("GET", "/api/verbs/progress", ""); code != http.StatusInternalServerError {
		t.Fatalf("db down = %d", code)
	}
}
