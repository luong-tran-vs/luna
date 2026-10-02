package lesson

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/job"
	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

const practiceLessonID = "0123456789abcdef01234567"

// newPracticeAPI serves a lesson (revision 3, practice version 2) in topic-a1 at position 2.
func newPracticeAPI(t *testing.T, withPractice bool, guard httpx.Middleware) (*http.ServeMux, string) {
	t.Helper()
	lessons := newFakeLessons()
	l := Lesson{
		ID: practiceLessonID, Title: "Park", Level: "A1", TopicID: "topic-a1", Content: sampleContent, Revision: 3,
		Sentences:        toSentences(SplitSentences(sampleContent)),
		AnnotationStatus: StatusDone,
		Annotations: []Annotation{
			{Text: "went", Lemma: "go", MeaningVi: "đã đi", SentenceIndex: 0},
			{Text: "gave up", Lemma: "give up", MeaningVi: "từ bỏ", SentenceIndex: 1},
			{Text: "sunny", Lemma: "sunny", MeaningVi: "nắng", SentenceIndex: 2},
		},
		PracticeStatus: StatusRunning,
	}
	if withPractice {
		p, err := CleanPractice(samplePractice(), practiceWords(l))
		if err != nil {
			t.Fatal(err)
		}
		l.Practice, l.PracticeStatus, l.PracticeVersion = &p, StatusDone, 2
	}
	lessons.byID[l.ID] = l
	topics := newFakeTopics()
	topics.setRoadmap("topic-a1", "other", l.ID)

	r := NewReader(lessons, readingDict, topics, newFakeAnswers(), newFakeAsks(), &fakeAI{})
	mux := http.NewServeMux()
	NewReadingHandler(r, slog.New(slog.DiscardHandler)).Register(mux, httpx.RequireAuth(resolver), guard)
	return mux, l.ID
}

type practiceResponse struct {
	Status       string `json:"status"`
	LessonNumber int    `json:"lessonNumber"`
	ObjectiveVi  string `json:"objectiveVi"`
	GrammarTipVi string `json:"grammarTipVi"`
	Examples     []struct {
		Lemma    string `json:"lemma"`
		Sentence string `json:"sentence"`
	} `json:"examples"`
	Dialogue *struct {
		Speakers []string `json:"speakers"`
		Turns    []struct {
			Speaker   int    `json:"speaker"`
			Text      string `json:"text"`
			MeaningVi string `json:"meaningVi"`
		} `json:"turns"`
	} `json:"dialogue"`
	Fill *struct {
		Turns []struct {
			Speaker   int    `json:"speaker"`
			TurnIndex int    `json:"turnIndex"`
			MeaningVi string `json:"meaningVi"`
			Parts     []struct {
				Text  *string `json:"text"`
				Blank *int    `json:"blank"`
			} `json:"parts"`
		} `json:"turns"`
		Blanks []struct {
			Answer string `json:"answer"`
		} `json:"blanks"`
		WordBank []string `json:"wordBank"`
	} `json:"fill"`
	Translations []struct {
		Vi     string   `json:"vi"`
		Answer []string `json:"answer"`
		Tiles  []string `json:"tiles"`
	} `json:"translations"`
}

func getPractice(t *testing.T, mux http.Handler, id string) practiceResponse {
	t.Helper()
	rec := get(t, mux, "/api/lessons/"+id+"/practice", "learner")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d %s", rec.Code, rec.Body)
	}
	var out practiceResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestPracticeEndpoint(t *testing.T) {
	t.Parallel()
	mux, id := newPracticeAPI(t, true, allowAll)
	got := getPractice(t, mux, id)

	if rec := get(t, mux, "/api/lessons/"+id+"/practice", "learner"); strings.Contains(rec.Body.String(), "audio") {
		t.Fatalf("practice still mentions audio: %s", rec.Body)
	}
	if got.Status != "done" || got.LessonNumber != 2 || got.ObjectiveVi == "" || got.GrammarTipVi == "" {
		t.Fatalf("header = %+v", got)
	}
	if len(got.Examples) != 2 || got.Examples[0].Lemma != "go" || got.Examples[0].Sentence != "I go to school." {
		t.Fatalf("examples = %+v", got.Examples)
	}
	if got.Dialogue == nil || len(got.Dialogue.Turns) != 4 || got.Dialogue.Speakers[0] != "Minh" || got.Dialogue.Turns[0].MeaningVi == "" {
		t.Fatalf("dialogue = %+v", got.Dialogue)
	}
	if got.Fill == nil || len(got.Fill.Turns) != 4 || len(got.Fill.Blanks) != 4 {
		t.Fatalf("fill = %+v", got.Fill)
	}
	first := got.Fill.Turns[0].Parts
	if len(first) != 3 || first[0].Text == nil || *first[0].Text != "We " || first[1].Blank == nil || *first[1].Blank != 0 || first[1].Text != nil {
		t.Fatalf("first turn parts = %+v", first)
	}
	if got.Fill.Blanks[0].Answer != "went" || len(got.Fill.WordBank) != 4 {
		t.Fatalf("blanks %+v bank %q", got.Fill.Blanks, got.Fill.WordBank)
	}
	tr := got.Translations
	if len(tr) != 1 || strings.Join(tr[0].Answer, " ") != "I go to the park." || len(tr[0].Tiles) != 7 {
		t.Fatalf("translations = %+v", tr)
	}
}

func TestPracticeEndpointWithoutPractice(t *testing.T) {
	t.Parallel()
	mux, id := newPracticeAPI(t, false, allowAll)
	rec := get(t, mux, "/api/lessons/"+id+"/practice", "learner")
	want := `{"status":"running","lessonNumber":2,"objectiveVi":"","examples":[],"dialogue":null,"fill":null,"grammarTipVi":"","translations":[]}`
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != want {
		t.Fatalf("status %d body %s", rec.Code, rec.Body)
	}
}

func TestPracticeEndpointStatusNone(t *testing.T) {
	t.Parallel()
	r, l := newReaderEnv(t)
	v, err := r.Practice(context.Background(), l.ID)
	if err != nil || v.Status != StatusNone || v.LessonNumber != 0 {
		t.Fatalf("view = %+v, err %v", v, err)
	}
	if got := practiceStatusJSON(v.Status); got != "none" {
		t.Fatalf("status = %q", got)
	}
}

func TestPracticeEndpointErrors(t *testing.T) {
	t.Parallel()
	mux, id := newPracticeAPI(t, true, allowAll)
	if rec := get(t, mux, "/api/lessons/missing/practice", "learner"); rec.Code != http.StatusNotFound {
		t.Errorf("missing: %d", rec.Code)
	}
	if rec := get(t, mux, "/api/lessons/"+id+"/practice", ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("anonymous: %d", rec.Code)
	}

	deny := func(http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			httpx.WriteError(w, http.StatusForbidden, "lesson_locked", "Bài này sẽ mở khi tới lượt")
		})
	}
	locked, id := newPracticeAPI(t, true, deny)
	rec := get(t, locked, "/api/lessons/"+id+"/practice", "learner")
	if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "lesson_locked") {
		t.Errorf("locked: %d %s", rec.Code, rec.Body)
	}
}

func TestRegeneratePracticeEndpoint(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	created := a.createLesson(t)
	id := created["id"].(string)
	if created["practiceStatus"] != "none" || created["practice"] != nil {
		t.Fatalf("new lesson practice = %v / %v", created["practiceStatus"], created["practice"])
	}
	path := "/api/admin/lessons/" + id + "/practice/regenerate"

	rec := a.do(t, http.MethodPost, path, "admin", "")
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "annotation_not_done") {
		t.Fatalf("not annotated: %d %s", rec.Code, rec.Body)
	}

	l, _ := a.env.lessons.Get(t.Context(), id)
	a.env.ai.result = sampleAnnotations
	if err := a.env.svc.ProcessAnnotate(t.Context(), jobFor(l, job.TypeAnnotate)); err != nil {
		t.Fatal(err)
	}
	// Annotating queued a practice: it is running.
	if rec := a.do(t, http.MethodPost, path, "admin", ""); rec.Code != http.StatusConflict ||
		!strings.Contains(rec.Body.String(), "practice_running") {
		t.Fatalf("running: %d %s", rec.Code, rec.Body)
	}
	a.env.ai.practice = samplePractice()
	if err := a.env.svc.ProcessPractice(t.Context(), jobFor(l, job.TypePractice)); err != nil {
		t.Fatal(err)
	}

	rec = a.do(t, http.MethodGet, "/api/admin/lessons/"+id, "admin", "")
	got := decodeBody(t, rec)["lesson"].(map[string]any)
	p, _ := got["practice"].(map[string]any)
	if got["practiceStatus"] != "done" || got["practiceError"] != "" || p == nil || p["objectiveVi"] == "" ||
		len(p["translations"].([]any)) != 1 || p["dialogue"].(map[string]any)["speakers"].([]any)[1] != "Anna" {
		t.Fatalf("admin lesson = %v", got)
	}

	rec = a.do(t, http.MethodPost, path, "admin", "")
	if rec.Code != http.StatusAccepted || decodeBody(t, rec)["lesson"].(map[string]any)["practiceStatus"] != "running" {
		t.Fatalf("regenerate: %d %s", rec.Code, rec.Body)
	}
	if rec := a.do(t, http.MethodPost, path, "learner", ""); rec.Code != http.StatusForbidden {
		t.Fatalf("learner: %d", rec.Code)
	}
	if rec := a.do(t, http.MethodPost, "/api/admin/lessons/missing/practice/regenerate", "admin", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("missing: %d", rec.Code)
	}
}
