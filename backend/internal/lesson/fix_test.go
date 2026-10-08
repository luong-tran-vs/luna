package lesson

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/ai"
)

// flagged returns checkable after a check that flags sentence 0, annotation 1, question 0 and
// translation 0.
func (e *env) flagged(t *testing.T) Lesson {
	t.Helper()
	l := e.checkable(t)
	e.ai.review = ai.ReviewResult{
		Sentences:    []ai.ReviewIssue{{Index: 0, NoteVi: "Câu sai."}},
		Annotations:  []ai.ReviewIssue{{Index: 1, NoteVi: "Nghĩa sai."}},
		Translations: []ai.ReviewIssue{{Index: 0, NoteVi: "Dịch sai."}},
		Answers:      []ai.ReviewAnswer{{Index: 0, ChoiceIndex: intp(1)}, {Index: 1, ChoiceIndex: intp(1)}},
	}
	got, err := e.svc.Check(t.Context(), l.ID)
	if err != nil || got.Review == nil || len(got.Review.Flags) != 4 {
		t.Fatalf("Check: %+v, %v", got.Review, err)
	}
	return got
}

func flagAreas(r *Review) []string {
	if r == nil {
		return nil
	}
	var out []string
	for _, f := range r.Flags {
		out = append(out, string(f.Area))
	}
	return out
}

func TestSuggestFixSendsTheFlaggedItem(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.flagged(t)
	e.ai.fix = ai.FixResult{MeaningVi: "  bỏ (thói quen)  ", NoteVi: " Sửa nghĩa. ", Text: "ignored"}

	got, err := e.svc.SuggestFix(t.Context(), l.ID, AreaAnnotation, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got.MeaningVi != "bỏ (thói quen)" || got.NoteVi != "Sửa nghĩa." || got.Text != "" || got.Area != AreaAnnotation || got.Index != 1 {
		t.Fatalf("suggestion = %+v", got)
	}
	req := e.ai.fixReq
	if req.Area != "annotation" || req.Problem != "Nghĩa sai." || req.Annotation == nil ||
		req.Annotation.Text != l.Annotations[1].Text || len(req.Sentences) != len(l.Sentences) {
		t.Fatalf("request = %+v", req)
	}

	e.ai.fix = ai.FixResult{Question: ai.FixQuestion{Prompt: " Where? ", Options: []string{" Park ", "Home"}, AnswerIndex: 0}}
	got, err = e.svc.SuggestFix(t.Context(), l.ID, AreaQuestion, 0)
	if err != nil || got.Question == nil || got.Question.Prompt != "Where?" || got.Question.Options[0] != "Park" {
		t.Fatalf("question suggestion = %+v, %v", got, err)
	}
	if e.ai.fixReq.Question == nil || e.ai.fixReq.Question.AnswerIndex != 0 {
		t.Fatalf("question request = %+v", e.ai.fixReq)
	}
	// Nothing was saved.
	if now, _ := e.lessons.Get(t.Context(), l.ID); len(now.Review.Flags) != 4 {
		t.Fatalf("flags = %+v", now.Review.Flags)
	}
}

func TestSuggestFixErrors(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.flagged(t)
	for _, c := range []struct {
		area  FlagArea
		index int
	}{{AreaSentence, 99}, {AreaAnnotation, -1}, {AreaTranslation, 1}, {"poem", 0}} {
		if _, err := e.svc.SuggestFix(t.Context(), l.ID, c.area, c.index); !errors.Is(err, ErrFixTarget) {
			t.Errorf("%s %d: %v", c.area, c.index, err)
		}
	}
	e.ai.fixErr = ai.ErrQuota
	if _, err := e.svc.SuggestFix(t.Context(), l.ID, AreaSentence, 0); !errors.Is(err, ai.ErrQuota) {
		t.Fatalf("quota: %v", err)
	}
	e.ai.fixErr = errors.New("boom")
	if _, err := e.svc.SuggestFix(t.Context(), l.ID, AreaSentence, 0); !errors.Is(err, errReviewAI) {
		t.Fatalf("AI failure: %v", err)
	}
}

func TestApplyFixKeepsTheOtherFlags(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.flagged(t)

	got, err := e.svc.ApplyFix(t.Context(), l.ID, FixInput{Area: AreaAnnotation, Index: 1, MeaningVi: "bỏ (thói quen)"})
	if err != nil {
		t.Fatal(err)
	}
	if a := got.Annotations[1]; a.MeaningVi != "bỏ (thói quen)" || !a.EditedByAdmin {
		t.Fatalf("annotation = %+v", a)
	}
	if areas := flagAreas(got.Review); strings.Join(areas, ",") != "sentence,question,translation" {
		t.Fatalf("flags after annotation fix = %v", areas)
	}

	q := Question{Prompt: "Where did we go?", Options: []string{"To the park", "Home", "To school", "To work"}, AnswerIndex: 0, ExplanationVi: "Câu 1 nói vậy."}
	if got, err = e.svc.ApplyFix(t.Context(), l.ID, FixInput{Area: AreaQuestion, Index: 0, Question: &q}); err != nil {
		t.Fatal(err)
	}
	if got.Extras.Questions[0].ExplanationVi != "Câu 1 nói vậy." || len(got.Extras.Questions) != 2 {
		t.Fatalf("questions = %+v", got.Extras.Questions)
	}

	if got, err = e.svc.ApplyFix(t.Context(), l.ID, FixInput{Area: AreaTranslation, Index: 0, Vi: "Tôi đến công viên.", En: "I go to the park."}); err != nil {
		t.Fatal(err)
	}
	if tr := got.Practice.Translations[0]; tr.Vi != "Tôi đến công viên." || len(tr.Distractors) == 0 {
		t.Fatalf("translation = %+v", tr)
	}
	if areas := flagAreas(got.Review); strings.Join(areas, ",") != "sentence" {
		t.Fatalf("flags left = %v", areas)
	}
}

func TestApplyFixSentenceRewritesTheContent(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.flagged(t)
	got, err := e.svc.ApplyFix(t.Context(), l.ID, FixInput{Area: AreaSentence, Index: 1, Text: "He gave up smoking last year."})
	if err != nil {
		t.Fatal(err)
	}
	if got.Content != "We went to the park. He gave up smoking last year. It was a sunny day." {
		t.Fatalf("content = %q", got.Content)
	}
	// Like any content edit: annotated again, check dropped.
	if got.Revision != l.Revision+1 || got.AnnotationStatus != StatusRunning || got.Review != nil {
		t.Fatalf("lesson = rev %d, %s, review %+v", got.Revision, got.AnnotationStatus, got.Review)
	}
	if _, err := e.svc.ApplyFix(t.Context(), l.ID, FixInput{Area: AreaSentence, Index: 0, Text: "  "}); !errors.Is(err, ErrFixTarget) {
		t.Fatalf("empty sentence: %v", err)
	}
}

func TestReplaceSentenceKeepsTheLayout(t *testing.T) {
	t.Parallel()
	content := "Minh: Hi,\n  Anna. How are you?\nAnna: I'm fine.\n\nMinh: Good. Good."
	sentences := toSentences(SplitSentences(content))
	got, ok := replaceSentence(content, sentences, 4, "Minh: Great.")
	if !ok || got != "Minh: Hi,\n  Anna. How are you?\nAnna: I'm fine.\n\nMinh: Good. Minh: Great." {
		t.Fatalf("got %q, %v", got, ok)
	}
	got, ok = replaceSentence(content, sentences, 0, "Minh: Hello, Anna.")
	if !ok || !strings.HasPrefix(got, "Minh: Hello, Anna. How are you?\n") {
		t.Fatalf("got %q, %v", got, ok)
	}
	if _, ok := replaceSentence("Something else.", sentences, 0, "x"); ok {
		t.Fatal("replaced a sentence that is not in the content")
	}
}

func TestFixEndpoints(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	l := a.env.flagged(t)
	base := "/api/admin/lessons/" + l.ID + "/check/"
	a.env.ai.fix = ai.FixResult{En: "I go to the park.", Vi: "Tôi đến công viên.", NoteVi: "Sửa động từ."}

	rec := a.do(t, http.MethodPost, base+"suggest", "admin", `{"area":"translation","index":0}`)
	want := `{"suggestion":{"area":"translation","index":0,"vi":"Tôi đến công viên.","en":"I go to the park.","noteVi":"Sửa động từ."}}`
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != want {
		t.Fatalf("suggest: %d %s", rec.Code, rec.Body)
	}
	if rec := a.do(t, http.MethodPost, base+"suggest", "admin", `{"area":"translation","index":5}`); rec.Code != http.StatusNotFound {
		t.Fatalf("suggest missing: %d", rec.Code)
	}
	a.env.ai.fixErr = ai.ErrNotConfigured
	if rec := a.do(t, http.MethodPost, base+"suggest", "admin", `{"area":"sentence","index":0}`); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("suggest without AI: %d", rec.Code)
	}

	rec = a.do(t, http.MethodPost, base+"apply", "admin", `{"area":"question","index":1,"question":{"prompt":"What was the weather?","options":["Rainy","Sunny","Cold","Windy"],"answerIndex":1,"explanationVi":"Trời nắng."}}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"explanationVi":"Trời nắng."`) {
		t.Fatalf("apply: %d %s", rec.Code, rec.Body)
	}
	if rec := a.do(t, http.MethodPost, base+"apply", "admin", `{"area":"annotation","index":0,"meaningVi":""}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("apply empty meaning: %d %s", rec.Code, rec.Body)
	}
	for _, p := range []string{"suggest", "apply"} {
		if rec := a.do(t, http.MethodPost, base+p, "learner", `{}`); rec.Code != http.StatusForbidden {
			t.Errorf("learner %s: %d", p, rec.Code)
		}
	}
}
