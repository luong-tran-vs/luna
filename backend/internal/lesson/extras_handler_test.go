package lesson

import (
	"net/http"
	"testing"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/job"
)

// --- F15: admin JSON and retry of a done annotation ---

func TestAdminLessonJSONHasExtras(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	l := a.createLesson(t)
	id := l["id"].(string)

	if qs, ok := l["questions"].([]any); !ok || len(qs) != 0 || l["grammarNote"] != nil || l["writingPrompt"] != "" ||
		l["extrasEditedByAdmin"] != false || l["quizVersion"] != float64(0) {
		t.Fatalf("new lesson extras = %v %v %v %v %v", l["questions"], l["grammarNote"], l["writingPrompt"],
			l["extrasEditedByAdmin"], l["quizVersion"])
	}

	a.env.ai.result = []ai.Annotation{{Text: "went", Lemma: "go", MeaningVi: "đã đi"}}
	a.env.ai.extras = ai.LessonExtras{
		Questions:   []ai.Question{parkQuestion},
		GrammarNote: &ai.GrammarNote{Title: "Quá khứ đơn", BodyVi: "Việc đã xong.", Examples: []string{"We went to the park."}},
	}
	if err := a.env.svc.ProcessAnnotate(t.Context(), job.Job{Type: job.TypeAnnotate, LessonID: id, Revision: 1}); err != nil {
		t.Fatal(err)
	}
	rec := a.do(t, http.MethodGet, "/api/admin/lessons/"+id, "admin", "")
	got := decodeBody(t, rec)["lesson"].(map[string]any)
	q := got["questions"].([]any)[0].(map[string]any)
	if q["prompt"] != "Where did we go?" || q["answerIndex"] != float64(0) || q["explanationVi"] == "" ||
		len(q["options"].([]any)) != 4 {
		t.Fatalf("question = %v", q)
	}
	note := got["grammarNote"].(map[string]any)
	if note["title"] != "Quá khứ đơn" || len(note["examples"].([]any)) != 1 || got["quizVersion"] != float64(1) {
		t.Fatalf("lesson = %v", got)
	}

	// A done annotation can be run again (F15: old lessons get their questions).
	rec = a.do(t, http.MethodPost, "/api/admin/lessons/"+id+"/retry?job=annotate", "admin", "")
	if rec.Code != http.StatusAccepted || decodeBody(t, rec)["lesson"].(map[string]any)["annotationStatus"] != "running" {
		t.Fatalf("retry done: %d %s", rec.Code, rec.Body)
	}
	rec = a.do(t, http.MethodPost, "/api/admin/lessons/"+id+"/retry?job=annotate", "admin", "")
	if rec.Code != http.StatusConflict || decodeBody(t, rec)["error"] != "annotation_running" {
		t.Fatalf("retry running: %d %s", rec.Code, rec.Body)
	}
}
