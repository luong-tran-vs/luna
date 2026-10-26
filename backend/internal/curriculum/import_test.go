package curriculum

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
	"testing"

	"github.com/luongtran/luna/backend/internal/platform/httpx"

	"github.com/luongtran/luna/backend/internal/lesson"
)

func TestBrief(t *testing.T) {
	s, _, _ := newService()
	ctx := context.Background()
	b, err := s.Brief(ctx, "a1-09-4")
	if err != nil {
		t.Fatal(err)
	}
	if b.Title != "A Concert in the Park" || b.Level != "A1" || b.TopicID != "music" || b.Words != 120 || b.Filled ||
		!slices.Equal(b.TargetWords, []string{"hello", "name"}) || b.MinGrammarUses != MinGrammarUses {
		t.Errorf("brief %+v", b)
	}
	if len(b.Grammar) < 2 || b.Grammar[0].ID != "a1-like-love-ing" || b.Grammar[0].Pattern == "" {
		t.Errorf("grammar %+v", b.Grammar)
	}
	if b, err := s.Brief(ctx, "a1-09-1"); err != nil || !b.Filled {
		t.Errorf("filled slot: %+v, %v", b, err)
	}
	if _, err := s.Brief(ctx, "a1-01-1"); !errors.Is(err, ErrNotApplied) {
		t.Errorf("not applied: %v", err)
	}
}

func TestImportSlot(t *testing.T) {
	s, ft, fl := newService()
	ctx := context.Background()
	in := SlotImport{Content: "We like singing."}
	if _, err := s.ImportSlot(ctx, "a1-09-1", in, false); !errors.Is(err, ErrSlotFilled) {
		t.Fatalf("filled: %v", err)
	}

	// A dry run checks the lesson but saves nothing.
	before := slices.Clone(ft.find("music").Roadmaps["A1"])
	res, err := s.ImportSlot(ctx, "a1-09-4", in, true)
	if err != nil {
		t.Fatal(err)
	}
	if fl.dryRuns != 1 || res.Slot.Lesson != nil || len(res.Checks.Items) == 0 || res.Report.Sentences != 1 ||
		!slices.Equal(ft.find("music").Roadmaps["A1"], before) {
		t.Errorf("dry run %+v, roadmap %v", res, ft.find("music").Roadmaps["A1"])
	}
	got := fl.imported[0]
	if got.Title != "A Concert in the Park" || !got.Draft || got.AppendToRoadmap || got.Source != lesson.ImportSource ||
		got.GrammarPointID != "a1-like-love-ing" || !slices.Equal(got.TargetWords, []string{"hello", "name"}) {
		t.Errorf("dry run input %+v", got)
	}

	// Saved, the lesson takes its place like a generated one; the writer's target words are kept.
	in.TargetWords = []string{"song"}
	res, err = s.ImportSlot(ctx, "a1-09-4", in, false)
	if err != nil {
		t.Fatal(err)
	}
	got = fl.imported[1]
	if !got.AppendToRoadmap || !slices.Equal(got.TargetWords, []string{"song"}) || res.Slot.Lesson == nil {
		t.Errorf("import input %+v, result %+v", got, res)
	}
	if r := ft.find("music").Roadmaps["A1"]; !slices.Equal(r, []string{"l2", "l4", "l1", "l5", "l3"}) {
		t.Errorf("music roadmap %v", r)
	}
	if _, err := s.ImportSlot(ctx, "a1-09-4", in, true); !errors.Is(err, ErrSlotFilled) {
		t.Errorf("now filled: %v", err)
	}

	// The report comes back with an invalid lesson.
	s, _, fl = newService()
	fl.importErr = &lesson.ValidationError{Fields: map[string]string{"practice": "x"}}
	res, err = s.ImportSlot(ctx, "a1-09-4", in, true)
	var verr *lesson.ValidationError
	if !errors.As(err, &verr) || len(res.Report.Parts) != 1 {
		t.Errorf("invalid: %+v, %v", res, err)
	}
}

// serveBody is serve with a JSON body.
func serveBody(t *testing.T, s *Service, method, path, body string) (int, map[string]any) {
	t.Helper()
	mux := http.NewServeMux()
	auth := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(httpx.WithPrincipal(r.Context(), httpx.Principal{UserID: "u1", Role: "admin"})))
		})
	}
	NewHandler(s, slog.New(slog.NewTextHandler(io.Discard, nil))).Register(mux, auth)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

func TestImportHandler(t *testing.T) {
	s, ft, fl := newService()
	code, body := serve(t, s, "admin", "GET", "/api/admin/curriculum/slots/a1-09-4/brief")
	if code != http.StatusOK || body["title"] != "A Concert in the Park" || len(body["targetWords"].([]any)) != 2 {
		t.Errorf("brief: %d %v", code, body)
	}
	if code, _ := serve(t, s, "learner", "GET", "/api/admin/curriculum/slots/a1-09-4/brief"); code != http.StatusForbidden {
		t.Errorf("learner brief: %d", code)
	}

	lessonBody := `{"content":"We like singing.","annotate":{"annotations":[{"text":"like","lemma":"like","meaningVi":"thích","sentenceIndex":0}]},"practice":{}}`
	code, body = serveBody(t, s, "POST", "/api/admin/curriculum/slots/a1-09-4/import?dryRun=true", lessonBody)
	if code != http.StatusOK || body["dryRun"] != true || body["slot"].(map[string]any)["lesson"] != nil || body["report"] == nil {
		t.Errorf("dry run: %d %v", code, body)
	}
	code, body = serveBody(t, s, "POST", "/api/admin/curriculum/slots/a1-09-4/import", lessonBody)
	if code != http.StatusCreated || body["slot"].(map[string]any)["lesson"] == nil {
		t.Errorf("import: %d %v", code, body)
	}

	// An unknown field names itself, so the writer can fix it.
	code, body = serveBody(t, s, "POST", "/api/admin/curriculum/slots/a1-09-5/import", `{"content":"x","annotations":[]}`)
	if code != http.StatusBadRequest || !strings.Contains(body["fields"].(map[string]any)["body"].(string), "annotations") {
		t.Errorf("unknown field: %d %v", code, body)
	}
	fl.importErr = &lesson.ValidationError{Fields: map[string]string{"practice": "x"}}
	code, body = serveBody(t, s, "POST", "/api/admin/curriculum/slots/a1-09-5/import", lessonBody)
	if code != http.StatusBadRequest || body["fields"].(map[string]any)["practice"] != "x" || body["report"] == nil {
		t.Errorf("invalid: %d %v", code, body)
	}

	// {"delete": false} keeps the lesson as a draft.
	code, body = serveBody(t, s, "POST", "/api/admin/curriculum/lessons/l1/remove", `{"delete":false}`)
	if code != http.StatusOK || body["deleted"] != false || slices.Contains(fl.deleted, "l1") || !ft.lessons["l1"].Draft {
		t.Errorf("keep: %d %v", code, body)
	}
}
