package lesson

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/job"
)

func grammarField(t *testing.T, err error) string {
	t.Helper()
	var verr *ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("error = %v, want ValidationError", err)
	}
	return verr.Fields["grammarPointId"]
}

func TestCreateGrammarPoint(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.create(t, func(in *Input) { in.TopicID, in.Level = "topic-a1", "A1"; in.GrammarPointID = "a1-to-be" })
	if l.GrammarPointID != "a1-to-be" {
		t.Fatalf("lesson = %+v", l)
	}
	in := Input{Title: "x", Content: sampleContent, TopicID: "topic-a1", Level: "A1", Source: "s", License: "l"}

	in.GrammarPointID = "nope"
	_, err := e.svc.Create(t.Context(), in)
	if got := grammarField(t, err); got != "Điểm ngữ pháp không tồn tại" {
		t.Errorf("unknown point: %q", got)
	}
	in.GrammarPointID = "b1-present-perfect" // B1 point on an A1 topic
	_, err = e.svc.Create(t.Context(), in)
	if got := grammarField(t, err); got != "Điểm ngữ pháp không thuộc trình độ của bài" {
		t.Errorf("wrong level: %q", got)
	}
}

func TestUpdateGrammarPoint(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.create(t, func(in *Input) { in.TopicID, in.Level = "topic-a1", "A1"; in.GrammarPointID = "a1-to-be" })
	in := Input{Title: "Park", Content: sampleContent, TopicID: "topic-a1", Level: "A1", Source: "s", License: "l", GrammarPointID: "a1-articles"}

	got, err := e.svc.Update(t.Context(), l.ID, in)
	if err != nil || got.GrammarPointID != "a1-articles" {
		t.Fatalf("change point: %+v %v", got, err)
	}
	in.GrammarPointID = ""
	if got, err = e.svc.Update(t.Context(), l.ID, in); err != nil || got.GrammarPointID != "" {
		t.Fatalf("clear point: %+v %v", got, err)
	}
	// A content change keeps the point too.
	in.GrammarPointID, in.Content = "a1-articles", "A new text. Another one."
	if got, err = e.svc.Update(t.Context(), l.ID, in); err != nil || got.GrammarPointID != "a1-articles" {
		t.Fatalf("content change: %+v %v", got, err)
	}
	// Moving to level B1 with an A1 point is refused.
	in.Level = "B1"
	_, err = e.svc.Update(t.Context(), l.ID, in)
	if grammarField(t, err) == "" {
		t.Fatal("level change should refuse the old point")
	}
}

func TestProcessAnnotatePassesGrammarFocus(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.ai.result = []ai.Annotation{{Text: "park", Lemma: "park", MeaningVi: "công viên", SentenceIndex: 0}}
	l := e.create(t, func(in *Input) { in.TopicID, in.Level = "topic-a1", "A1"; in.GrammarPointID = "a1-to-be" })
	if err := e.svc.ProcessAnnotate(t.Context(), jobFor(l, job.TypeAnnotate)); err != nil {
		t.Fatal(err)
	}
	g := e.ai.annotateReq.GrammarFocus
	if g == nil || g.TitleVi != "Động từ to be (am / is / are)" || g.TitleEn != "The verb to be" ||
		g.Pattern != "I am / you are / he, she, it is" || g.HintVi == "" {
		t.Fatalf("GrammarFocus = %+v", g)
	}

	plain := e.create(t, func(in *Input) { in.TopicID, in.Level = "topic-a1", "A1" })
	if err := e.svc.ProcessAnnotate(t.Context(), jobFor(plain, job.TypeAnnotate)); err != nil {
		t.Fatal(err)
	}
	if e.ai.annotateReq.GrammarFocus != nil {
		t.Fatalf("GrammarFocus without a point = %+v", e.ai.annotateReq.GrammarFocus)
	}
}

// The AI may write its own title for the note; a lesson with a syllabus point always shows the
// syllabus's, and a lesson without one keeps what the AI wrote.
func TestProcessAnnotateForcesTheSyllabusTitle(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.ai.result = []ai.Annotation{{Text: "park", Lemma: "park", MeaningVi: "công viên", SentenceIndex: 0}}
	e.ai.extras = ai.LessonExtras{
		GrammarNote: &ai.GrammarNote{Title: "Thì hiện tại đơn với to be và động từ thường", BodyVi: "Giải thích.", Examples: []string{"We went to the park."}},
	}

	withPoint := e.create(t, func(in *Input) { in.TopicID, in.Level = "topic-a1", "A1"; in.GrammarPointID = "a1-to-be" })
	if err := e.svc.ProcessAnnotate(t.Context(), jobFor(withPoint, job.TypeAnnotate)); err != nil {
		t.Fatal(err)
	}
	got, _ := e.lessons.Get(t.Context(), withPoint.ID)
	if got.Extras.GrammarNote == nil || got.Extras.GrammarNote.Title != "Động từ to be (am / is / are)" {
		t.Fatalf("note = %+v, want the syllabus title", got.Extras.GrammarNote)
	}
	if got.Extras.GrammarNote.BodyVi != "Giải thích." {
		t.Errorf("the body must stay the AI's: %+v", got.Extras.GrammarNote)
	}

	plain := e.create(t, func(in *Input) { in.TopicID, in.Level = "topic-a1", "A1" })
	if err := e.svc.ProcessAnnotate(t.Context(), jobFor(plain, job.TypeAnnotate)); err != nil {
		t.Fatal(err)
	}
	got, _ = e.lessons.Get(t.Context(), plain.ID)
	if got.Extras.GrammarNote == nil || got.Extras.GrammarNote.Title != "Thì hiện tại đơn với to be và động từ thường" {
		t.Fatalf("a lesson without a point must keep the AI's title: %+v", got.Extras.GrammarNote)
	}
}

func TestGenerateGrammarPoint(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.ai.drafts = []ai.LessonDraft{{Title: "Hello", Content: text(100)}}
	res, err := e.svc.Generate(t.Context(), "topic-a1", GenerateInput{Level: "A1", Count: 1, Words: 100, Kind: "reading", GrammarPointID: "a1-to-be"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Drafts) != 1 || res.Drafts[0].GrammarPointID != "a1-to-be" {
		t.Fatalf("drafts = %+v", res.Drafts)
	}
	if req, _ := e.ai.lastRequest(); req.GrammarFocus == nil || req.GrammarFocus.TitleEn != "The verb to be" {
		t.Fatalf("request = %+v", req)
	}
	_, err = e.svc.Generate(t.Context(), "topic-a1", GenerateInput{Level: "A1", Count: 1, Words: 100, Kind: "reading", GrammarPointID: "b1-present-perfect"})
	if grammarField(t, err) == "" {
		t.Fatal("wrong level should be refused")
	}
}

func TestGrammarEndpoint(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	a.env.create(t, func(in *Input) { in.TopicID, in.Level = "topic-a1", "A1"; in.GrammarPointID = "a1-to-be" })
	a.env.create(t, func(in *Input) { in.TopicID, in.Level = "topic-a1", "A1"; in.GrammarPointID = "a1-to-be" })
	a.env.create(t, func(in *Input) { in.TopicID = "topic-b1"; in.GrammarPointID = "b1-present-perfect" })

	rec := a.do(t, http.MethodGet, "/api/admin/grammar?level=A1", "admin", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d %s", rec.Code, rec.Body)
	}
	pts := decodeBody(t, rec)["points"].([]any)
	first := pts[0].(map[string]any)
	if first["id"] != "a1-to-be" || first["level"] != "A1" || first["lessonCount"] != float64(2) ||
		first["titleEn"] == "" || first["hintVi"] == "" || len(first["examples"].([]any)) == 0 {
		t.Fatalf("first = %v", first)
	}
	if pts[1].(map[string]any)["lessonCount"] != float64(0) {
		t.Fatalf("second = %v", pts[1])
	}

	rec = a.do(t, http.MethodGet, "/api/admin/grammar?level=A1&topicId=topic-b1", "admin", "")
	if got := decodeBody(t, rec)["points"].([]any)[0].(map[string]any)["lessonCount"]; got != float64(0) {
		t.Fatalf("topic-filtered count = %v", got)
	}
	for _, q := range []string{"", "?level=Z9"} {
		rec = a.do(t, http.MethodGet, "/api/admin/grammar"+q, "admin", "")
		if rec.Code != http.StatusUnprocessableEntity && rec.Code != http.StatusBadRequest {
			t.Fatalf("level %q: status %d", q, rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "level") {
			t.Fatalf("level %q body %s", q, rec.Body)
		}
	}
	if rec = a.do(t, http.MethodGet, "/api/admin/grammar?level=A1", "learner", ""); rec.Code != http.StatusForbidden {
		t.Fatalf("learner: %d", rec.Code)
	}
}

func TestLessonEndpointsGrammarPoint(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	body := strings.Replace(validBody, `"topic-b1","level":"B1"`, `"topic-a1","level":"A1"`, 1)
	rec := a.do(t, http.MethodPost, "/api/admin/lessons", "admin",
		strings.Replace(body, `"license":"CC BY"`, `"license":"CC BY","grammarPointId":"a1-to-be"`, 1))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	l := decodeBody(t, rec)["lesson"].(map[string]any)
	if l["grammarPointId"] != "a1-to-be" || l["grammarPointTitle"] != "Động từ to be (am / is / are)" {
		t.Fatalf("lesson = %v", l)
	}
	rec = a.do(t, http.MethodGet, "/api/admin/lessons/"+l["id"].(string), "admin", "")
	got := decodeBody(t, rec)["lesson"].(map[string]any)
	if got["grammarPointId"] != "a1-to-be" {
		t.Fatalf("get = %v", got)
	}
	rec = a.do(t, http.MethodPut, "/api/admin/lessons/"+l["id"].(string), "admin",
		strings.Replace(body, `"license":"CC BY"`, `"license":"CC BY","grammarPointId":""`, 1))
	got = decodeBody(t, rec)["lesson"].(map[string]any)
	if got["grammarPointId"] != "" || got["grammarPointTitle"] != "" {
		t.Fatalf("cleared = %v", got)
	}
	rec = a.do(t, http.MethodPost, "/api/admin/lessons", "admin",
		strings.Replace(body, `"license":"CC BY"`, `"license":"CC BY","grammarPointId":"x"`, 1))
	if rec.Code == http.StatusCreated || !strings.Contains(rec.Body.String(), "grammarPointId") {
		t.Fatalf("bad point: %d %s", rec.Code, rec.Body)
	}
}

func TestUpdateEndpointGrammarPointAbsentEmptyOrSet(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	base := strings.Replace(validBody, `"topic-b1","level":"B1"`, `"topic-a1","level":"A1"`, 1)
	withPoint := func(v string) string {
		return strings.Replace(base, `"license":"CC BY"`, `"license":"CC BY","grammarPointId":`+v, 1)
	}
	rec := a.do(t, http.MethodPost, "/api/admin/lessons", "admin", withPoint(`"a1-to-be"`))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	path := "/api/admin/lessons/" + decodeBody(t, rec)["lesson"].(map[string]any)["id"].(string)
	point := func(body string) any {
		t.Helper()
		rec := a.do(t, http.MethodPut, path, "admin", body)
		if rec.Code != http.StatusOK {
			t.Fatalf("update: %d %s", rec.Code, rec.Body)
		}
		return decodeBody(t, rec)["lesson"].(map[string]any)["grammarPointId"]
	}
	if got := point(base); got != "a1-to-be" {
		t.Fatalf("absent should keep the point, got %v", got)
	}
	if got := point(withPoint(`"a1-articles"`)); got != "a1-articles" {
		t.Fatalf("valid value should change it, got %v", got)
	}
	if got := point(base); got != "a1-articles" {
		t.Fatalf("absent should keep the new point, got %v", got)
	}
	if got := point(withPoint(`""`)); got != "" {
		t.Fatalf("empty string should clear it, got %v", got)
	}
}
