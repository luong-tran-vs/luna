package lesson

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
)

func intp(i int) *int { return &i }

// agreeing answers both questions of checkable like the stored keys.
var agreeing = []ai.ReviewAnswer{{Index: 0, ChoiceIndex: intp(0)}, {Index: 1, ChoiceIndex: intp(1)}}

// checkable returns an annotated lesson with a practice (one translation) and two questions whose
// stored answers are option 0 and option 1.
func (e *env) checkable(t *testing.T) Lesson {
	t.Helper()
	l := e.withPractice(t)
	if _, err := e.lessons.update(l.ID, func(l *Lesson) bool {
		l.Extras.Questions = []Question{
			{Prompt: "Where did we go?", Options: []string{"To the park", "Home", "To school", "To work"}, AnswerIndex: 0, ExplanationVi: "Câu 1"},
			{Prompt: "What was the weather?", Options: []string{"Rainy", "Sunny", "Cold", "Windy"}, AnswerIndex: 1, ExplanationVi: "Câu 3"},
		}
		return true
	}); err != nil {
		t.Fatal(err)
	}
	got, _ := e.lessons.Get(t.Context(), l.ID)
	return got
}

func TestCheckFlagsEveryKind(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.checkable(t)
	if len(l.Annotations) != 3 || l.Practice == nil || len(l.Practice.Translations) != 1 {
		t.Fatalf("fixture = %d annotations, practice %+v", len(l.Annotations), l.Practice)
	}
	e.ai.review = ai.ReviewResult{
		Sentences:    []ai.ReviewIssue{{Index: 2, NoteVi: "Câu sai."}, {Index: 99, NoteVi: "ngoài phạm vi"}, {Index: -1}},
		Annotations:  []ai.ReviewIssue{{Index: 1, NoteVi: "Nghĩa sai."}, {Index: 3, NoteVi: "ngoài phạm vi"}},
		Translations: []ai.ReviewIssue{{Index: 0}, {Index: 1, NoteVi: "ngoài phạm vi"}},
		Answers: []ai.ReviewAnswer{
			{Index: 0, ChoiceIndex: intp(1), NoteVi: "Nhà."},
			{Index: 1, ChoiceIndex: intp(1), Ambiguous: true, NoteVi: "Cả hai đúng."},
			{Index: 5, ChoiceIndex: intp(0)},
		},
	}
	e.ai.reviewHook = nil

	got, err := e.svc.Check(t.Context(), l.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Review == nil || !got.Review.CheckedAt.Equal(e.now) || !got.Review.VerifiedAt.IsZero() {
		t.Fatalf("review = %+v", got.Review)
	}
	want := []Flag{
		{Area: AreaSentence, Index: 2, Kind: FlagWrong, NoteVi: "Câu sai."},
		{Area: AreaAnnotation, Index: 1, Kind: FlagWrong, NoteVi: "Nghĩa sai."},
		{Area: AreaQuestion, Index: 0, Kind: FlagMismatch, NoteVi: "AI chọn \"Home\", khác đáp án đã lưu. Nhà."},
		{Area: AreaQuestion, Index: 1, Kind: FlagAmbiguous, NoteVi: "Cả hai đúng."},
		{Area: AreaTranslation, Index: 0, Kind: FlagWrong},
	}
	// A wrong item with no note gets a default one.
	if len(got.Review.Flags) != len(want) {
		t.Fatalf("flags = %+v", got.Review.Flags)
	}
	for i, w := range want {
		g := got.Review.Flags[i]
		if g.Area != w.Area || g.Index != w.Index || g.Kind != w.Kind || g.Confirmed || (w.NoteVi != "" && g.NoteVi != w.NoteVi) || g.NoteVi == "" {
			t.Errorf("flag %d = %+v, want %+v", i, g, w)
		}
	}
	if n := e.ai.reviewCalls; n != 1 {
		t.Errorf("AI calls = %d, want 1", n)
	}
}

func TestCheckRequestHidesTheKeys(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.checkable(t)
	if _, err := e.svc.Check(t.Context(), l.ID); err != nil {
		t.Fatal(err)
	}
	req := e.ai.reviewReq
	if req.Level != "B1" || req.Title != "Park" || len(req.Sentences) != 3 || len(req.Annotations) != 3 ||
		len(req.Questions) != 2 || len(req.Translations) != 1 {
		t.Fatalf("request = %+v", req)
	}
	if req.Annotations[2].Index != 2 || req.Annotations[2].Text != "sunny" || req.Annotations[2].SentenceIndex != 2 {
		t.Errorf("annotation = %+v", req.Annotations[2])
	}
	if q := req.Questions[1]; q.Index != 1 || q.Prompt != "What was the weather?" || len(q.Options) != 4 {
		t.Errorf("question = %+v", q)
	}
	if tr := req.Translations[0]; tr.Vi != "Tôi đi công viên." || tr.En != "I go to the park." {
		t.Errorf("translation = %+v", tr)
	}
}

func TestCheckUncheckedQuestion(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.checkable(t)
	// Question 0 is missing from the answers, question 1 has no choice. Question 1 would match the
	// key, so only the missing reply flags it.
	e.ai.review = ai.ReviewResult{Answers: []ai.ReviewAnswer{{Index: 1}}}
	got, err := e.svc.Check(t.Context(), l.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Review.Flags) != 2 || got.Review.Flags[0].Kind != FlagUnchecked || got.Review.Flags[1].Kind != FlagUnchecked ||
		got.Review.Flags[0].Index != 0 || got.Review.Flags[1].Index != 1 {
		t.Fatalf("flags = %+v", got.Review.Flags)
	}
}

func TestCheckAgreeingReviewHasNoFlags(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.checkable(t)
	e.ai.review = ai.ReviewResult{Answers: []ai.ReviewAnswer{{Index: 0, ChoiceIndex: intp(0)}, {Index: 1, ChoiceIndex: intp(1)}}}
	got, err := e.svc.Check(t.Context(), l.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Review == nil || got.Review.Flags == nil || len(got.Review.Flags) != 0 {
		t.Fatalf("review = %+v", got.Review)
	}
	// Checking again replaces the review, also a verified one.
	e.ai.review = ai.ReviewResult{Sentences: []ai.ReviewIssue{{Index: 0, NoteVi: "x"}}, Answers: e.ai.review.Answers}
	got, err = e.svc.Check(t.Context(), l.ID)
	if err != nil || len(got.Review.Flags) != 1 || !got.Review.VerifiedAt.IsZero() {
		t.Fatalf("second check: %+v, %v", got.Review, err)
	}
}

func TestCheckNeedsAnnotations(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.create(t)
	if _, err := e.svc.Check(t.Context(), l.ID); !errors.Is(err, ErrAnnotationNotDone) {
		t.Fatalf("err = %v", err)
	}
	if _, err := e.svc.Check(t.Context(), "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing: %v", err)
	}
	if e.ai.reviewCalls != 0 {
		t.Errorf("AI calls = %d", e.ai.reviewCalls)
	}
}

func TestCheckAIFailureLeavesLessonUnchanged(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.checkable(t)
	e.ai.review = ai.ReviewResult{Sentences: []ai.ReviewIssue{{Index: 0, NoteVi: "x"}}, Answers: agreeing}
	if _, err := e.svc.Check(t.Context(), l.ID); err != nil {
		t.Fatal(err)
	}
	before, _ := e.lessons.Get(t.Context(), l.ID)

	for _, aiErr := range []error{ai.ErrNotConfigured, ai.ErrInvalidKey, ai.ErrQuota, errors.New("boom")} {
		e.ai.reviewErr = aiErr
		_, err := e.svc.Check(t.Context(), l.ID)
		if !errors.Is(err, aiErr) {
			t.Errorf("err = %v, want %v", err, aiErr)
		}
		if aiErr.Error() == "boom" && !errors.Is(err, errReviewAI) {
			t.Errorf("other error is not wrapped: %v", err)
		}
		after, _ := e.lessons.Get(t.Context(), l.ID)
		if after.Review == nil || len(after.Review.Flags) != 1 || !after.Review.CheckedAt.Equal(before.Review.CheckedAt) {
			t.Fatalf("review changed: %+v", after.Review)
		}
	}
}

func TestCheckStaleReplyIsDropped(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.checkable(t)
	e.ai.review = ai.ReviewResult{Sentences: []ai.ReviewIssue{{Index: 0, NoteVi: "x"}}, Answers: agreeing}
	// The text is replaced while the AI works.
	e.ai.reviewHook = func() {
		_, _ = e.lessons.update(l.ID, func(l *Lesson) bool { l.Revision++; return true })
	}
	got, err := e.svc.Check(t.Context(), l.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Review != nil || got.Revision != l.Revision+1 {
		t.Fatalf("lesson = revision %d review %+v", got.Revision, got.Review)
	}
}

func TestConfirmFlag(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.checkable(t)
	if _, err := e.svc.ConfirmFlag(t.Context(), l.ID, AreaQuestion, 0); !errors.Is(err, ErrFlagNotFound) {
		t.Fatalf("not checked: %v", err)
	}
	e.ai.review = ai.ReviewResult{
		Sentences: []ai.ReviewIssue{{Index: 1, NoteVi: "x"}},
		Answers:   []ai.ReviewAnswer{{Index: 0, ChoiceIndex: intp(2)}, {Index: 1, ChoiceIndex: intp(1)}},
	}
	if _, err := e.svc.Check(t.Context(), l.ID); err != nil {
		t.Fatal(err)
	}

	got, err := e.svc.ConfirmFlag(t.Context(), l.ID, AreaQuestion, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Review.Flags) != 2 || got.Review.Flags[0].Confirmed || !got.Review.Flags[1].Confirmed {
		t.Fatalf("flags = %+v", got.Review.Flags)
	}
	if got.Review.CheckedAt.IsZero() {
		t.Error("CheckedAt lost")
	}
	// Confirming again is fine.
	if _, err := e.svc.ConfirmFlag(t.Context(), l.ID, AreaQuestion, 0); err != nil {
		t.Errorf("again: %v", err)
	}
	for _, c := range []struct {
		area  FlagArea
		index int
	}{{AreaQuestion, 1}, {AreaSentence, 0}, {"nonsense", 0}, {AreaTranslation, 0}} {
		if _, err := e.svc.ConfirmFlag(t.Context(), l.ID, c.area, c.index); !errors.Is(err, ErrFlagNotFound) {
			t.Errorf("%v: err = %v", c, err)
		}
	}
	if _, err := e.svc.ConfirmFlag(t.Context(), "missing", AreaQuestion, 0); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing: %v", err)
	}
}

func TestVerify(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.checkable(t)
	if _, err := e.svc.Verify(t.Context(), l.ID); !errors.Is(err, ErrNotChecked) {
		t.Fatalf("not checked: %v", err)
	}
	e.ai.review = ai.ReviewResult{
		Sentences: []ai.ReviewIssue{{Index: 1, NoteVi: "x"}, {Index: 2, NoteVi: "y"}},
		Answers:   []ai.ReviewAnswer{{Index: 0, ChoiceIndex: intp(0)}, {Index: 1, ChoiceIndex: intp(1)}},
	}
	if _, err := e.svc.Check(t.Context(), l.ID); err != nil {
		t.Fatal(err)
	}
	_, err := e.svc.Verify(t.Context(), l.ID)
	var fe *FlagsError
	if !errors.As(err, &fe) || fe.Count != 2 {
		t.Fatalf("open flags: %v", err)
	}
	if _, err := e.svc.ConfirmFlag(t.Context(), l.ID, AreaSentence, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.Verify(t.Context(), l.ID); !errors.As(err, &fe) || fe.Count != 1 {
		t.Fatalf("one open flag: %v", err)
	}
	if _, err := e.svc.ConfirmFlag(t.Context(), l.ID, AreaSentence, 2); err != nil {
		t.Fatal(err)
	}
	e.now = e.now.Add(time.Hour)
	got, err := e.svc.Verify(t.Context(), l.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Review.VerifiedAt.Equal(e.now) || len(got.Review.Flags) != 2 {
		t.Fatalf("review = %+v", got.Review)
	}
	if _, err := e.svc.Verify(t.Context(), "missing"); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing: %v", err)
	}
}

func TestVerifyWithoutFlags(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.checkable(t)
	e.ai.review = ai.ReviewResult{Answers: []ai.ReviewAnswer{{Index: 0, ChoiceIndex: intp(0)}, {Index: 1, ChoiceIndex: intp(1)}}}
	if _, err := e.svc.Check(t.Context(), l.ID); err != nil {
		t.Fatal(err)
	}
	got, err := e.svc.Verify(t.Context(), l.ID)
	if err != nil || got.Review.VerifiedAt.IsZero() {
		t.Fatalf("verify: %+v, %v", got.Review, err)
	}
}

// --- the review goes away with the content ---

func (e *env) checked(t *testing.T) Lesson {
	t.Helper()
	l := e.checkable(t)
	e.ai.review = ai.ReviewResult{Sentences: []ai.ReviewIssue{{Index: 0, NoteVi: "x"}}, Answers: agreeing}
	got, err := e.svc.Check(t.Context(), l.ID)
	if err != nil || got.Review == nil {
		t.Fatalf("Check: %+v, %v", got.Review, err)
	}
	return got
}

func TestContentWritesDropTheReview(t *testing.T) {
	t.Parallel()
	cases := map[string]func(e *env, l Lesson) error{
		"ReplaceExtras": func(e *env, l Lesson) error {
			return e.lessons.ReplaceExtras(t.Context(), l.ID, Extras{WritingPrompt: "Write more."}, false)
		},
		"ReplaceAnnotations": func(e *env, l Lesson) error {
			return e.lessons.ReplaceAnnotations(t.Context(), l.ID, nil)
		},
		"Update with new content": func(e *env, l Lesson) error {
			_, err := e.svc.Update(t.Context(), l.ID, Input{Title: "Park", Content: "A new text.", TopicID: "topic-b1", Source: "s", License: "l"})
			return err
		},
		"SaveAnnotations": func(e *env, l Lesson) error {
			_, err := e.lessons.SaveAnnotations(t.Context(), l.ID, l.Revision, nil, Extras{})
			return err
		},
		"SavePractice": func(e *env, l Lesson) error {
			_, err := e.lessons.SavePractice(t.Context(), l.ID, l.Revision, l.PracticeVersion, Practice{})
			return err
		},
	}
	for name, write := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			e := newEnv(t)
			l := e.checked(t)
			if err := write(e, l); err != nil {
				t.Fatal(err)
			}
			got, _ := e.lessons.Get(t.Context(), l.ID)
			if got.Review != nil {
				t.Errorf("review kept: %+v", got.Review)
			}
		})
	}
}

func TestInfoAndStatusKeepTheReview(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.checked(t)
	if _, err := e.svc.Update(t.Context(), l.ID, Input{Title: "New title", Content: l.Content, TopicID: "topic-b1", Source: "s", License: "l"}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.lessons.SetStatus(t.Context(), l.ID, l.Revision, "practice", StatusFailed, "x"); err != nil {
		t.Fatal(err)
	}
	got, _ := e.lessons.Get(t.Context(), l.ID)
	if got.Review == nil || got.Title != "New title" {
		t.Fatalf("lesson = %+v", got)
	}
}

func TestListShowsReviewCounts(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.checked(t)
	flagsOf := func() Summary {
		items, err := e.svc.List(t.Context(), Filter{})
		if err != nil || len(items) == 0 {
			t.Fatalf("List: %v, %v", items, err)
		}
		for _, it := range items {
			if it.ID == l.ID {
				return it
			}
		}
		t.Fatal("lesson not listed")
		return Summary{}
	}
	if s := flagsOf(); s.Flags != 1 || !s.Checked || s.Verified {
		t.Fatalf("summary = %+v", s)
	}
	if _, err := e.svc.ConfirmFlag(t.Context(), l.ID, AreaSentence, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.Verify(t.Context(), l.ID); err != nil {
		t.Fatal(err)
	}
	if s := flagsOf(); s.Flags != 0 || !s.Checked || !s.Verified {
		t.Fatalf("summary = %+v", s)
	}
}

// --- API ---

func (a *api) checkedLesson(t *testing.T) string {
	t.Helper()
	l := a.env.checkable(t)
	return l.ID
}

func TestCheckEndpoints(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	id := a.checkedLesson(t)
	base := "/api/admin/lessons/" + id

	// Not checked yet: review is null and the other calls say so.
	rec := a.do(t, http.MethodGet, base, "admin", "")
	lessonJSON := decodeBody(t, rec)["lesson"].(map[string]any)
	if v, ok := lessonJSON["review"]; !ok || v != nil || lessonJSON["checked"] != false {
		t.Fatalf("new lesson review = %v (present %v)", v, ok)
	}
	if rec := a.do(t, http.MethodPost, base+"/check/verify", "admin", ""); rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "not_checked") {
		t.Fatalf("verify before check: %d %s", rec.Code, rec.Body)
	}
	if rec := a.do(t, http.MethodPost, base+"/check/confirm", "admin", `{"area":"question","index":0}`); rec.Code != http.StatusNotFound {
		t.Fatalf("confirm before check: %d %s", rec.Code, rec.Body)
	}

	a.env.ai.review = ai.ReviewResult{
		Sentences: []ai.ReviewIssue{{Index: 1, NoteVi: "Câu lạ."}},
		Answers:   []ai.ReviewAnswer{{Index: 0, ChoiceIndex: intp(2)}, {Index: 1, ChoiceIndex: intp(1)}},
	}
	rec = a.do(t, http.MethodPost, base+"/check", "admin", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("check: %d %s", rec.Code, rec.Body)
	}
	got := decodeBody(t, rec)["lesson"].(map[string]any)
	review := got["review"].(map[string]any)
	flags := review["flags"].([]any)
	if review["checkedAt"] == "" || review["verifiedAt"] != nil || len(flags) != 2 || got["flags"] != float64(2) || got["checked"] != true || got["verified"] != false {
		t.Fatalf("lesson = %v", got)
	}
	f0 := flags[0].(map[string]any)
	if f0["area"] != "sentence" || f0["index"] != float64(1) || f0["kind"] != "wrong" || f0["noteVi"] != "Câu lạ." || f0["confirmed"] != false {
		t.Errorf("flag = %v", f0)
	}

	rec = a.do(t, http.MethodPost, base+"/check/verify", "admin", "")
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "flags_unresolved") || decodeBody(t, rec)["count"] != float64(2) {
		t.Fatalf("verify with open flags: %d %s", rec.Code, rec.Body)
	}
	if rec := a.do(t, http.MethodPost, base+"/check/confirm", "admin", `{"area":"sentence","index":7}`); rec.Code != http.StatusNotFound {
		t.Fatalf("confirm unknown: %d", rec.Code)
	}
	for _, body := range []string{`{"area":"sentence","index":1}`, `{"area":"question","index":0}`} {
		if rec := a.do(t, http.MethodPost, base+"/check/confirm", "admin", body); rec.Code != http.StatusOK {
			t.Fatalf("confirm %s: %d %s", body, rec.Code, rec.Body)
		}
	}
	rec = a.do(t, http.MethodPost, base+"/check/verify", "admin", "")
	got = decodeBody(t, rec)["lesson"].(map[string]any)
	review = got["review"].(map[string]any)
	if rec.Code != http.StatusOK || review["verifiedAt"] == nil || got["flags"] != float64(0) || got["verified"] != true {
		t.Fatalf("verify: %d %s", rec.Code, rec.Body)
	}

	rec = a.do(t, http.MethodGet, "/api/admin/lessons", "admin", "")
	rows := decodeBody(t, rec)["lessons"].([]any)
	row := rows[0].(map[string]any)
	if row["flags"] != float64(0) || row["checked"] != true || row["verified"] != true {
		t.Errorf("list row = %v", row)
	}
}

func TestCheckEndpointErrors(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	created := a.createLesson(t)
	base := "/api/admin/lessons/" + created["id"].(string) + "/check"

	if rec := a.do(t, http.MethodPost, base, "admin", ""); rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "annotation_not_done") {
		t.Fatalf("not annotated: %d %s", rec.Code, rec.Body)
	}
	id := a.checkedLesson(t)
	base = "/api/admin/lessons/" + id + "/check"
	for _, c := range []struct {
		err  error
		code int
		body string
	}{
		{ai.ErrNotConfigured, http.StatusServiceUnavailable, "ai_not_configured"},
		{ai.ErrInvalidKey, http.StatusServiceUnavailable, "ai_not_configured"},
		{ai.ErrQuota, http.StatusTooManyRequests, "ai_quota"},
		{errors.New("boom"), http.StatusBadGateway, "ai_failed"},
	} {
		a.env.ai.reviewErr = c.err
		if rec := a.do(t, http.MethodPost, base, "admin", ""); rec.Code != c.code || !strings.Contains(rec.Body.String(), c.body) {
			t.Errorf("%v: %d %s", c.err, rec.Code, rec.Body)
		}
	}
	a.env.ai.reviewErr = nil
	if rec := a.do(t, http.MethodPost, base, "learner", ""); rec.Code != http.StatusForbidden {
		t.Errorf("learner: %d", rec.Code)
	}
	if rec := a.do(t, http.MethodPost, base, "", ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("anonymous: %d", rec.Code)
	}
	if rec := a.do(t, http.MethodPost, "/api/admin/lessons/missing/check", "admin", ""); rec.Code != http.StatusNotFound {
		t.Errorf("missing: %d", rec.Code)
	}
	if rec := a.do(t, http.MethodPost, base+"/confirm", "admin", `{bad`); rec.Code != http.StatusBadRequest {
		t.Errorf("bad body: %d", rec.Code)
	}
}
