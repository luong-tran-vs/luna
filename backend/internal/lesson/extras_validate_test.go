package lesson

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/job"
)

func validExtras() ExtrasInput {
	return ExtrasInput{
		Questions: []Question{
			{Prompt: "Where did we go?", Options: []string{"Park", "Home", "School", "Work"}, AnswerIndex: 0, ExplanationVi: "Câu 1."},
		},
		GrammarNote:   &GrammarNote{Title: "Quá khứ đơn", BodyVi: "Việc đã xong.", Examples: []string{"We went to the park."}},
		WritingPrompt: "Write about a park.",
	}
}

func TestValidateExtrasFields(t *testing.T) {
	t.Parallel()

	// q adds a second, valid question changed by mutate.
	q := func(mutate func(*Question)) ExtrasInput {
		in := validExtras()
		x := Question{Prompt: "When did we go?", Options: []string{"Park", "Home", "School", "Work"}, ExplanationVi: "Câu 2."}
		mutate(&x)
		in.Questions = append(in.Questions, x)
		return in
	}
	tests := []struct {
		name  string
		in    ExtrasInput
		field string
		msg   string
	}{
		{"empty prompt", q(func(x *Question) { x.Prompt = " " }), "questions.1.prompt", "Vui lòng nhập câu hỏi"},
		{"duplicate prompt", q(func(x *Question) { x.Prompt = "Where did we go?" }), "questions.1.prompt", "Câu hỏi bị trùng"},
		{"empty option", q(func(x *Question) { x.Options[2] = "" }), "questions.1.options.2", "Vui lòng nhập lựa chọn"},
		{"same option", q(func(x *Question) { x.Options[3] = " park " }), "questions.1.options.3", "Lựa chọn bị trùng"},
		{"no answer", q(func(x *Question) { x.AnswerIndex = -1 }), "questions.1.answerIndex", "Chọn đáp án đúng"},
		{"no explanation", q(func(x *Question) { x.ExplanationVi = "" }), "questions.1.explanationVi", "Vui lòng nhập giải thích"},
		{"too many", func() ExtrasInput {
			in := validExtras()
			for i := range 5 {
				in.Questions = append(in.Questions, Question{
					Prompt: "Q" + strings.Repeat("?", i+1), Options: []string{"a", "b", "c", "d"}, ExplanationVi: "e",
				})
			}
			return in
		}(), "questions", "Tối đa 5 câu hỏi"},
		{"example not in lesson", func() ExtrasInput {
			in := validExtras()
			in.GrammarNote.Examples = []string{"We went to the zoo."}
			return in
		}(), "grammarNote.examples.0", "Ví dụ phải có trong bài"},
		{
			"no example", func() ExtrasInput { in := validExtras(); in.GrammarNote.Examples = nil; return in }(),
			"grammarNote.examples", "Cần 1–3 ví dụ",
		},
		{
			"no title", func() ExtrasInput { in := validExtras(); in.GrammarNote.Title = ""; return in }(),
			"grammarNote.title", "Vui lòng nhập tiêu đề",
		},
		{
			"long prompt", func() ExtrasInput { in := validExtras(); in.WritingPrompt = strings.Repeat("a", 501); return in }(),
			"writingPrompt", "Tối đa 500 ký tự",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := ValidateExtras(tt.in, sampleContent)
			var verr *ValidationError
			if !errors.As(err, &verr) || verr.Fields[tt.field] != tt.msg {
				t.Fatalf("err = %v, want %s: %q", err, tt.field, tt.msg)
			}
		})
	}
}

func TestValidateExtrasAllowsEmpty(t *testing.T) {
	t.Parallel()
	x, err := ValidateExtras(ExtrasInput{}, sampleContent)
	if err != nil || len(x.Questions) != 0 || x.GrammarNote != nil {
		t.Fatalf("empty extras = %+v, %v", x, err)
	}
	x, err = ValidateExtras(validExtras(), sampleContent)
	if err != nil || len(x.Questions) != 1 || x.GrammarNote == nil {
		t.Fatalf("valid extras = %+v, %v", x, err)
	}
}

func TestUpdateExtras(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.create(t)

	if _, err := e.svc.UpdateExtras(t.Context(), l.ID, validExtras()); !errors.Is(err, ErrAnnotationRunning) {
		t.Fatalf("while running: %v", err)
	}
	_ = e.svc.ProcessAnnotate(t.Context(), jobFor(l, job.TypeAnnotate)) // fails: no annotations
	e.svc.JobFailed(t.Context(), jobFor(l, job.TypeAnnotate), ErrNoValidAnnotations)

	got, err := e.svc.UpdateExtras(t.Context(), l.ID, validExtras())
	if err != nil || len(got.Extras.Questions) != 1 || !got.ExtrasEditedByAdmin || got.QuizVersion != 1 {
		t.Fatalf("first save = %+v, %v", got, err)
	}

	// Same questions, new grammar note: the question set (and the answers) stay.
	in := validExtras()
	in.GrammarNote.Title = "Thì quá khứ"
	if got, _ = e.svc.UpdateExtras(t.Context(), l.ID, in); got.QuizVersion != 1 || got.Extras.GrammarNote.Title != "Thì quá khứ" {
		t.Fatalf("grammar only = %+v", got)
	}
	// Another answer: a new question set.
	in.Questions[0].AnswerIndex = 1
	if got, _ = e.svc.UpdateExtras(t.Context(), l.ID, in); got.QuizVersion != 2 {
		t.Fatalf("answer changed: quizVersion = %d", got.QuizVersion)
	}
	if _, err := e.svc.UpdateExtras(t.Context(), "missing", in); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing: %v", err)
	}
}

func TestUpdateExtrasEndpoint(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	id := a.createLesson(t)["id"].(string)
	a.env.svc.JobFailed(t.Context(), job.Job{Type: job.TypeAnnotate, LessonID: id, Revision: 1}, ErrNoValidAnnotations)
	path := "/api/admin/lessons/" + id + "/extras"
	body := `{"questions":[{"prompt":"Where did we go?","options":["Park","Home","School","Work"],"answerIndex":0,` +
		`"explanationVi":"Câu 1."}],"grammarNote":{"title":"Quá khứ đơn","bodyVi":"Việc đã xong.",` +
		`"examples":["We went to the park."]},"writingPrompt":"Write."}`

	rec := a.do(t, http.MethodPut, path, "admin", body)
	l, _ := decodeBody(t, rec)["lesson"].(map[string]any)
	if rec.Code != http.StatusOK || l["extrasEditedByAdmin"] != true || len(l["questions"].([]any)) != 1 || l["writingPrompt"] != "Write." {
		t.Fatalf("save: %d %s", rec.Code, rec.Body)
	}

	bad := strings.Replace(body, `"examples":["We went to the park."]`, `"examples":["Not here."]`, 1)
	rec = a.do(t, http.MethodPut, path, "admin", bad)
	if fields, _ := decodeBody(t, rec)["fields"].(map[string]any); rec.Code != http.StatusBadRequest ||
		fields["grammarNote.examples.0"] != "Ví dụ phải có trong bài" {
		t.Fatalf("bad example: %d %s", rec.Code, rec.Body)
	}
	if rec := a.do(t, http.MethodPut, "/api/admin/lessons/missing/extras", "admin", body); rec.Code != http.StatusNotFound {
		t.Fatalf("missing: %d", rec.Code)
	}
	if rec := a.do(t, http.MethodPut, path, "learner", body); rec.Code != http.StatusForbidden {
		t.Fatalf("learner: %d", rec.Code)
	}
	a.env.svc.Retry(t.Context(), id, job.TypeAnnotate) //nolint:errcheck // puts the annotation back to running
	if rec := a.do(t, http.MethodPut, path, "admin", body); rec.Code != http.StatusConflict {
		t.Fatalf("running: %d %s", rec.Code, rec.Body)
	}
}
