package progress

import (
	"net/http"
	"testing"
)

func TestDashboardEndpoint(t *testing.T) {
	t.Parallel()
	mux, e := newStudyAPI(t)

	r := do(t, mux, http.MethodGet, "/api/dashboard", "an", "")
	want := `{"kind":"noGoal","goal":null,"goalCompleted":false,"skills":null,"lesson":null,` +
		`"steps":{"listen":"locked","read":"locked","review":"locked","write":"locked"},"currentStep":"","action":null,"streak":0,"tomorrowCards":0}` + "\n"
	if r.code != http.StatusOK || r.text != want {
		t.Fatalf("no goal: %d %s", r.code, r.text)
	}

	e.reviews.setDue("u1", 2)
	do(t, mux, http.MethodPost, "/api/goals", "an", `{"topicId":"family"}`)
	b := body(t, do(t, mux, http.MethodGet, "/api/dashboard", "an", ""))
	goal, _ := b["goal"].(map[string]any)
	skills, _ := b["skills"].(map[string]any)
	lesson, _ := b["lesson"].(map[string]any)
	action, _ := b["action"].(map[string]any)
	if b["kind"] != "studying" || goal["topicName"] != "Gia đình" || goal["totalLessons"] != 3.0 ||
		skills["read"] != 0.0 || skills["listen"] != 0.0 || skills["total"] != 3.0 ||
		lesson["id"] != "f1" || lesson["title"] != "Family 1" || lesson["topicName"] != "Gia đình" || lesson["level"] != "A1" ||
		action["kind"] != "start" || action["step"] != "review" || b["currentStep"] != "review" {
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
	want := `{"cards":0,"dictation":{"sentences":0,"correctWords":0,"totalWords":0,"rate":null},` +
		`"lessons":{"read":0,"listen":0,"write":0,"completed":0},"reading":{"answered":0,"correct":0,"rate":null},` +
		`"writing":{"submitted":0,"averageScore":null}}` + "\n"
	if r.code != http.StatusOK || r.text != want {
		t.Fatalf("new learner: %d %s", r.code, r.text)
	}

	if _, err := e.dictation.Record(t.Context(), "u1", "f1", Input{SentenceIndex: 0, Typed: "x", CorrectWords: 4, TotalWords: 5}); err != nil {
		t.Fatal(err)
	}
	b := body(t, do(t, mux, http.MethodGet, "/api/stats", "an", ""))
	d, _ := b["dictation"].(map[string]any)
	if d["sentences"] != 1.0 || d["rate"] != 0.8 {
		t.Fatalf("stats: %v", b)
	}

	if r := do(t, mux, http.MethodGet, "/api/stats", "", ""); r.code != http.StatusUnauthorized {
		t.Fatalf("no session: %d", r.code)
	}
}
