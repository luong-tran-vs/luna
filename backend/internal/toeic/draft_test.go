package toeic

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/ai"
)

type fakeWriter struct {
	text string
	err  error
	got  ai.ToeicGroupsRequest
}

func (f *fakeWriter) ToeicGroups(_ context.Context, req ai.ToeicGroupsRequest) (string, error) {
	f.got = req
	return f.text, f.err
}

func draftJSON(t *testing.T, gs ...Group) string {
	t.Helper()
	b, err := json.Marshal(map[string]any{"groups": gs})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestDraftGroupsKeepsValidOnesAsDrafts(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := t.Context()
	e.seed(p5("p5-old", 0))
	bad := p5("", 1)
	bad.Questions[0].Options = bad.Questions[0].Options[:3] // Part 5 needs 4 options
	w := &fakeWriter{text: draftJSON(t, p5Says(2, "The new catalog will be mailed ___ all customers next week."), bad, p5Says(3, "Mr. Haddad has worked at the firm ___ more than ten years."))}
	e.svc.d.Writer = w

	rep, err := e.svc.DraftGroups(ctx, DraftInput{Part: 5, Level: 750, Topic: "prepositions"})
	if err != nil {
		t.Fatalf("draft = %v", err)
	}
	if len(rep.Created) != 2 || len(rep.Dropped) != 1 || rep.Dropped[0].Fields["questions[0].options"] == "" {
		t.Fatalf("report = %+v", rep)
	}
	for _, k := range rep.Created {
		g := e.groups.recs[k]
		if !strings.HasPrefix(k, "p5-ai-") || g.Status != StatusDraft || g.Source != SourceAI || g.Level != 750 || g.Part != 5 {
			t.Fatalf("stored %s = %+v", k, g)
		}
	}
	if w.got.Part != 5 || w.got.Count != 10 || w.got.Questions != 0 || w.got.Topic != "prepositions" || len(w.got.Existing) != 1 {
		t.Fatalf("request = %+v", w.got)
	}
	for _, tag := range w.got.Tags {
		if !TagAllowed(tag.Code, 5) || tag.Label == "" {
			t.Fatalf("tag %+v", tag)
		}
	}
}

func TestDraftGroupsChecksInputAndAIErrors(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := t.Context()
	var verr *ValidationError
	if _, err := e.svc.DraftGroups(ctx, DraftInput{Part: 8, Level: 700}); !errors.As(err, &verr) ||
		verr.Fields["part"] == "" || verr.Fields["level"] == "" {
		t.Fatalf("input = %v", err)
	}
	if _, err := e.svc.DraftGroups(ctx, DraftInput{Part: 5, Level: 600}); !errors.Is(err, ai.ErrNotConfigured) {
		t.Fatalf("no writer = %v", err)
	}
	e.svc.d.Writer = &fakeWriter{err: ai.ErrQuota}
	mux := e.mux()
	rec := call(t, mux, http.MethodPost, "/api/admin/toeic/groups/draft", "admin", `{"part":5,"level":600}`)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("quota = %d %s", rec.Code, rec.Body)
	}
	e.svc.d.Writer = &fakeWriter{text: "not json"}
	rec = call(t, mux, http.MethodPost, "/api/admin/toeic/groups/draft", "admin", `{"part":5,"level":600}`)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("bad json = %d %s", rec.Code, rec.Body)
	}
	rec = call(t, mux, http.MethodPost, "/api/admin/toeic/groups/draft", "member", `{"part":5,"level":600}`)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("member = %d", rec.Code)
	}
}

func TestDraftGroupsAsksForAboutTenQuestions(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		part      Part
		count     int
		questions int
	}{{1, 10, 0}, {2, 10, 0}, {3, 4, 0}, {4, 4, 0}, {5, 10, 0}, {6, 3, 0}, {7, 0, DraftQuestions}} {
		e := newEnv(t)
		w := &fakeWriter{text: `{"groups":[]}`}
		e.svc.d.Writer = w
		if _, err := e.svc.DraftGroups(t.Context(), DraftInput{Part: c.part, Level: 600}); err != nil {
			t.Fatalf("part %d: %v", c.part, err)
		}
		if w.got.Count != c.count || w.got.Questions != c.questions {
			t.Fatalf("part %d asked %d groups, %d questions", c.part, w.got.Count, w.got.Questions)
		}
	}
}

func TestDraftGroupsKeepsPart7GroupsOfTenQuestions(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.svc.d.Writer = &fakeWriter{text: draftJSON(t, p7Says(2, "Bring your own gloves and tools."), p7Says(3, "Volunteers receive a free lunch after the morning shift."), p7Says(5, "Parking is available behind the community center on Elm Street."))}
	rep, err := e.svc.DraftGroups(t.Context(), DraftInput{Part: 7, Level: 750})
	if err != nil || len(rep.Created) != 3 {
		t.Fatalf("part 7 = %+v, %v", rep, err)
	}
}

func TestDraftTimeoutSaysTheAIFailed(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.svc.d.Writer = &fakeWriter{err: context.DeadlineExceeded}
	rec := call(t, e.mux(), http.MethodPost, "/api/admin/toeic/groups/draft", "admin", `{"part":5,"level":600}`)
	if rec.Code != http.StatusBadGateway || !strings.Contains(rec.Body.String(), "AI soạn thất bại") {
		t.Fatalf("timeout = %d %s", rec.Code, rec.Body)
	}
}

func TestDraftKeepsTheSceneOnlyInPart1(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	g := p3("")
	g.Scene = "Two people in an office."
	e.svc.d.Writer = &fakeWriter{text: draftJSON(t, g)}
	rep, err := e.svc.DraftGroups(t.Context(), DraftInput{Part: 3, Level: 600})
	if err != nil || len(rep.Created) != 1 || e.groups.recs[rep.Created[0]].Scene != "" {
		t.Fatalf("report = %+v, %v", rep, err)
	}
}

// p5Says is p5 with its own sentence, so drafts are not taken for the same group.
func p5Says(answer int, sentence string) Group {
	g := p5("", answer)
	g.Questions[0].Prompt = sentence
	return g
}

// p7Says is p7 whose notice adds a sentence of its own.
func p7Says(n int, extra string) Group {
	g := p7("", n)
	g.Docs[0].Body[0].Text += " " + extra
	return g
}

func TestDraftDropsGroupsThePartAlreadyHas(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	old := p5Says(0, "All staff must submit their expense reports ___ the end of the month.")
	old.Key = "p5-old"
	e.seed(old)
	e.svc.d.Writer = &fakeWriter{text: draftJSON(t,
		// The same sentence with one word changed: a repeat of p5-old.
		p5Says(1, "All employees must submit their expense reports ___ the end of the month."),
		p5Says(2, "The shipment from our Lisbon supplier is expected to arrive ___ Thursday morning."),
		// Written twice in the same answer: only the first is kept.
		p5Says(3, "The shipment from our Lisbon supplier is expected to arrive ___ Thursday morning!"),
		p5Says(0, "Ms. Okafor asked the interns to review the contract ___ before signing it."),
	)}
	rep, err := e.svc.DraftGroups(t.Context(), DraftInput{Part: 5, Level: 600})
	if err != nil {
		t.Fatalf("draft = %v", err)
	}
	if len(rep.Created) != 2 || len(rep.Dropped) != 2 {
		t.Fatalf("report = %+v", rep)
	}
	if msg := rep.Dropped[0].Fields["duplicate"]; !strings.Contains(msg, "p5-old") || !strings.Contains(msg, "%") {
		t.Fatalf("first dropped = %+v", rep.Dropped[0])
	}
	if msg := rep.Dropped[1].Fields["duplicate"]; !strings.Contains(msg, rep.Created[0]) {
		t.Fatalf("second dropped = %+v", rep.Dropped[1])
	}
}

func TestLikenessTellsRepeatsFromNewGroups(t *testing.T) {
	t.Parallel()
	cases := []struct {
		a, b string
		dup  bool
	}{
		{"The report was written ___ by the team.", "The report was written ___ by the team.", true},
		{"Where is the meeting being held?", "Where is the meeting being held today?", true},
		{"Where is the meeting being held?", "When does the meeting start?", false},
		{"The new printer on the third floor is out of order.", "Please order new paper for the printer on the third floor.", false},
	}
	for _, c := range cases {
		got := likeness(wordSet(c.a), wordSet(c.b)) >= dupThreshold
		if got != c.dup {
			t.Errorf("%q vs %q: dup = %v, want %v (%.2f)", c.a, c.b, got, c.dup, likeness(wordSet(c.a), wordSet(c.b)))
		}
	}
}
