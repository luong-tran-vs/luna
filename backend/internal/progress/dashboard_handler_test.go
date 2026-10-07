package progress

import (
	"net/http"
	"testing"
)

func TestDashboardEndpoint(t *testing.T) {
	t.Parallel()
	mux, _ := newStudyAPI(t)

	r := do(t, mux, http.MethodGet, "/api/dashboard", "an", "")
	want := `{"kind":"noGoal","goal":null,"goalCompleted":false,"skills":null,"lesson":null,` +
		`"steps":{"listen":"locked","read":"locked","write":"locked"},"currentStep":"","action":null,"streak":0,"tomorrowCards":0}` + "\n"
	if r.code != http.StatusOK || r.text != want {
		t.Fatalf("no goal: %d %s", r.code, r.text)
	}

	do(t, mux, http.MethodPost, "/api/goals", "an", `{"topicId":"family"}`)
	b := body(t, do(t, mux, http.MethodGet, "/api/dashboard", "an", ""))
	goal, _ := b["goal"].(map[string]any)
	skills, _ := b["skills"].(map[string]any)
	lesson, _ := b["lesson"].(map[string]any)
	action, _ := b["action"].(map[string]any)
	if b["kind"] != "studying" || goal["topicName"] != "Gia đình" || goal["totalLessons"] != 3.0 ||
		skills["read"] != 0.0 || skills["listen"] != 0.0 || skills["total"] != 3.0 ||
		lesson["id"] != "f1" || lesson["title"] != "Family 1" || lesson["topicName"] != "Gia đình" || lesson["level"] != "A1" ||
		action["kind"] != "start" || action["step"] != "read" || b["currentStep"] != "read" {
		t.Fatalf("studying: %v", b)
	}

	if r := do(t, mux, http.MethodGet, "/api/dashboard", "", ""); r.code != http.StatusUnauthorized {
		t.Fatalf("no session: %d", r.code)
	}
}

func TestStatsEndpoint(t *testing.T) {
	t.Parallel()
	mux, e := newStudyAPI(t)

	r := do(t, mux, http.MethodGet, "/api/stats", "an", "")
	want := `{"period":"all","cards":0,"dictation":{"sentences":0,"correctWords":0,"totalWords":0,"rate":null,"lessons":0},` +
		`"lessons":{"read":0,"listen":0,"write":0,"completed":0},"reading":{"answered":0,"correct":0,"rate":null},` +
		`"writing":{"submitted":0,"averageScore":null},"grammar":{"lessons":0}}` + "\n"
	if r.code != http.StatusOK || r.text != want {
		t.Fatalf("new learner: %d %s", r.code, r.text)
	}

	if _, err := e.dictation.Record(t.Context(), "u1", "f1", Input{SentenceIndex: 0, Typed: "x", CorrectWords: 4, TotalWords: 5}); err != nil {
		t.Fatal(err)
	}
	e.grammar.mastered = 2
	b := body(t, do(t, mux, http.MethodGet, "/api/stats", "an", ""))
	d, _ := b["dictation"].(map[string]any)
	g, _ := b["grammar"].(map[string]any)
	if b["period"] != "all" || d["sentences"] != 1.0 || d["rate"] != 0.8 || d["lessons"] != 1.0 || g["lessons"] != 2.0 {
		t.Fatalf("stats: %v", b)
	}

	for _, p := range []string{"week", "month", "all"} {
		b := body(t, do(t, mux, http.MethodGet, "/api/stats?period="+p, "an", ""))
		if b["period"] != p {
			t.Fatalf("period %s: %v", p, b)
		}
	}
	r = do(t, mux, http.MethodGet, "/api/stats?period=year", "an", "")
	b = body(t, r)
	fields, _ := b["fields"].(map[string]any)
	if r.code != http.StatusBadRequest || b["error"] != "validation_failed" || fields["period"] == nil {
		t.Fatalf("bad period: %d %s", r.code, r.text)
	}

	if r := do(t, mux, http.MethodGet, "/api/stats", "", ""); r.code != http.StatusUnauthorized {
		t.Fatalf("no session: %d", r.code)
	}
}
