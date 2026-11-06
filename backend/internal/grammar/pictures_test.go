package grammar

import (
	"errors"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
)

// publishedLesson generates, checks and publishes the lesson of a point, without picture exercises.
func publishedLesson(t *testing.T, e *env, pointID string) Lesson {
	t.Helper()
	if _, _, err := e.svc.Generate(t.Context(), pointID, true); err != nil {
		t.Fatal(err)
	}
	l, err := e.svc.Publish(t.Context(), pointID, true)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func kindsOf(list []Exercise, kind Kind) int {
	n := 0
	for _, e := range list {
		if e.Kind == kind {
			n++
		}
	}
	return n
}

func TestAddPicturesKeepsTheLesson(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	before := publishedLesson(t, e, "a1-to-be")
	// An edit the admin made stays.
	edited := before.Content
	edited.Objective = "Bạn có thể giới thiệu bản thân."
	if _, err := e.svc.Save(t.Context(), "a1-to-be", edited); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.Check(t.Context(), "a1-to-be"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.Verify(t.Context(), "a1-to-be"); err != nil {
		t.Fatal(err)
	}

	l, added, err := e.svc.AddPictures(t.Context(), "a1-to-be")
	if err != nil {
		t.Fatal(err)
	}
	if added != PicturePractice+PictureMastery {
		t.Fatalf("added = %d", added)
	}
	req := e.ai.picturesReq
	if req.Practice != PicturePractice || req.Mastery != PictureMastery || req.TitleEn == "" || len(req.Existing) != 14 {
		t.Fatalf("request = %+v", req)
	}
	if l.Status != StatusPublished || l.Content.Objective != edited.Objective {
		t.Fatalf("lesson changed: status %s, objective %q", l.Status, l.Content.Objective)
	}
	if len(l.Content.Practice) != 11 || len(l.Content.Mastery) != 7 ||
		kindsOf(l.Content.Practice, KindPicture) != 3 || kindsOf(l.Content.Mastery, KindPicture) != 1 {
		t.Fatalf("practice %d, mastery %d", len(l.Content.Practice), len(l.Content.Mastery))
	}
	if !slices.Equal(ids(l.Content.Practice[:8]), ids(before.Content.Practice)) {
		t.Fatalf("old ids changed: %v", ids(l.Content.Practice))
	}
	if got := ids(l.Content.Practice[8:]); !slices.Equal(got, []string{"p9", "p10", "p11"}) {
		t.Fatalf("new ids = %v", got)
	}
	if l.Content.Mastery[6].ID != "m7" || l.Content.Mastery[6].Picture == "" {
		t.Fatalf("new mastery = %+v", l.Content.Mastery[6])
	}
	// Only the new exercises were checked, and they passed: the lesson stays verified.
	if n := len(e.ai.solveReq.Exercises); n != 4 {
		t.Fatalf("checked %d exercises", n)
	}
	if len(l.Checks) != 0 || l.VerifiedAt.IsZero() {
		t.Fatalf("checks %v, verified %v", l.Checks, l.VerifiedAt)
	}
	stored, _ := e.lessons.Get(t.Context(), "a1-to-be")
	if len(stored.Content.Practice) != 11 {
		t.Fatal("not saved")
	}

	// Once full, nothing more is asked.
	calls := e.ai.picturesCalls
	var verr *ValidationError
	if _, _, err := e.svc.AddPictures(t.Context(), "a1-to-be"); !errors.As(err, &verr) {
		t.Fatalf("second add: %v", err)
	}
	if e.ai.picturesCalls != calls {
		t.Fatal("asked the AI again")
	}
}

func TestAddPicturesFlagsWrongNewExercises(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	publishedLesson(t, e, "a1-to-be")
	if _, err := e.svc.Verify(t.Context(), "a1-to-be"); err != nil {
		t.Fatal(err)
	}
	e.ai.solve = func(req ai.SolveRequest) ([]ai.Solution, error) {
		var out []ai.Solution
		for _, x := range req.Exercises {
			zero := 0
			out = append(out, ai.Solution{ExerciseID: x.ID, ChoiceIndex: &zero})
		}
		return out, nil
	}
	l, _, err := e.svc.AddPictures(t.Context(), "a1-to-be")
	if err != nil {
		t.Fatal(err)
	}
	if len(l.Checks) != 4 || !l.VerifiedAt.IsZero() {
		t.Fatalf("checks %d, verified %v", len(l.Checks), l.VerifiedAt)
	}
}

func TestAddPicturesOnlyWhatIsMissing(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.ai.content.Practice = append(exercises("p", 10), pictureExercise("old"))
	if _, _, err := e.svc.Generate(t.Context(), "a1-to-be", true); err != nil {
		t.Fatal(err)
	}
	l, added, err := e.svc.AddPictures(t.Context(), "a1-to-be")
	if err != nil {
		t.Fatal(err)
	}
	// 11 practice exercises with one picture: one more fits the limit of 12.
	if e.ai.picturesReq.Practice != 1 || e.ai.picturesReq.Mastery != 1 || added != 2 || len(l.Content.Practice) != maxPractice {
		t.Fatalf("request %+v, added %d, practice %d", e.ai.picturesReq, added, len(l.Content.Practice))
	}
}

func TestAddPicturesRefuses(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	var verr *ValidationError
	if _, _, err := e.svc.Generate(t.Context(), "a1-articles", true); err != nil {
		t.Fatal(err)
	}
	if _, _, err := e.svc.AddPictures(t.Context(), "a1-articles"); !errors.As(err, &verr) {
		t.Fatalf("point that cannot be drawn: %v", err)
	}
	if _, _, err := e.svc.AddPictures(t.Context(), "a1-there-is-are"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("no lesson: %v", err)
	}

	publishedLesson(t, e, "a1-to-be")
	// Pictures that repeat a sentence of the lesson, or are broken, are left out.
	e.ai.pictures = func(ai.GrammarPicturesRequest) (ai.GrammarPictures, error) {
		dup := pictureExercise("x")
		dup.Text = e.ai.content.Practice[0].Text
		broken := pictureExercise("y")
		broken.Picture = ""
		return ai.GrammarPictures{Practice: []ai.GrammarExercise{dup, broken, goodChoice("z")}}, nil
	}
	if _, _, err := e.svc.AddPictures(t.Context(), "a1-to-be"); !errors.Is(err, ErrUnusable) {
		t.Fatalf("unusable: %v", err)
	}
	e.ai.pictures = func(ai.GrammarPicturesRequest) (ai.GrammarPictures, error) {
		return ai.GrammarPictures{}, ai.ErrQuota
	}
	if _, _, err := e.svc.AddPictures(t.Context(), "a1-to-be"); !errors.Is(err, ai.ErrQuota) {
		t.Fatalf("quota: %v", err)
	}
	if l, _ := e.lessons.Get(t.Context(), "a1-to-be"); kindsOf(l.Content.Practice, KindPicture) != 0 {
		t.Fatal("a failed add changed the lesson")
	}
}

func waitPictureBatch(t *testing.T, svc *Service) PictureBatchStatus {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if st := svc.PictureBatch(); !st.Running {
			return st
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("picture batch still running")
	return PictureBatchStatus{}
}

func TestPictureBatchAddsToEveryLessonOfTheLevel(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	for _, id := range []string{"a1-to-be", "a1-this-that", "a1-articles"} {
		publishedLesson(t, e, id)
	}
	// a1-articles cannot be drawn and a1-can has no lesson: both are left out.
	st, err := e.svc.StartPictureBatch(t.Context(), PictureBatchRequest{
		PointIDs: []string{"a1-to-be", "a1-this-that", "a1-articles", "a1-can", "a1-to-be"}, Key: "level:A1", Label: "các bài ngữ pháp A1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !st.Running || st.Total != 2 || st.Key != "level:A1" {
		t.Fatalf("start = %+v", st)
	}
	st = waitPictureBatch(t, e.svc)
	if len(st.Saved) != 2 || len(st.Failures) != 0 || st.Saved[0].Added != 4 || st.Saved[0].Title == "" || st.FinishedAt.IsZero() {
		t.Fatalf("finished = %+v", st)
	}
	for _, id := range []string{"a1-to-be", "a1-this-that"} {
		if l, _ := e.lessons.Get(t.Context(), id); kindsOf(l.Content.Practice, KindPicture) != 3 {
			t.Fatalf("%s has no pictures", id)
		}
	}
	var verr *ValidationError
	if _, err := e.svc.StartPictureBatch(t.Context(), PictureBatchRequest{PointIDs: []string{"a1-to-be"}}); !errors.As(err, &verr) {
		t.Fatalf("nothing left: %v", err)
	}
}

func TestPictureBatchStopsWhenTheAIIsOut(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	for _, id := range []string{"a1-to-be", "a1-this-that"} {
		publishedLesson(t, e, id)
	}
	e.ai.pictures = func(ai.GrammarPicturesRequest) (ai.GrammarPictures, error) {
		return ai.GrammarPictures{}, ai.ErrQuota
	}
	if _, err := e.svc.StartPictureBatch(t.Context(), PictureBatchRequest{PointIDs: []string{"a1-to-be", "a1-this-that"}}); err != nil {
		t.Fatal(err)
	}
	st := waitPictureBatch(t, e.svc)
	if len(st.Failures) != 1 || !errors.Is(st.StopErr, ai.ErrQuota) || e.ai.picturesCalls != 1 {
		t.Fatalf("status = %+v, calls %d", st, e.ai.picturesCalls)
	}
}

func TestPictureRoutes(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	a.generate(t)
	rec := a.do(t, http.MethodPost, adminPath+"/pictures", "admin", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("add: %d %s", rec.Code, rec.Body)
	}
	if !strings.Contains(rec.Body.String(), `"kind":"picture"`) {
		t.Fatalf("no picture exercise in %s", rec.Body)
	}
	if rec := a.do(t, http.MethodPost, adminPath+"/pictures", "learner", ""); rec.Code != http.StatusForbidden {
		t.Fatalf("learner: %d", rec.Code)
	}
	if rec := a.do(t, http.MethodPost, adminPath+"/pictures", "admin", ""); rec.Code != http.StatusBadRequest && rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("full: %d %s", rec.Code, rec.Body)
	}
	if rec := a.do(t, http.MethodGet, "/api/admin/grammar-picture-batch", "admin", ""); rec.Code != http.StatusOK ||
		decode(t, rec)["running"] != false {
		t.Fatalf("status: %d %s", rec.Code, rec.Body)
	}
	if rec := a.do(t, http.MethodPost, "/api/admin/grammar-picture-batch", "admin", `{"pointIds":["`+point+`"]}`); rec.Code == http.StatusAccepted {
		t.Fatalf("start with nothing to add: %d", rec.Code)
	}
	if rec := a.do(t, http.MethodPost, "/api/admin/grammar-picture-batch/stop", "admin", ""); rec.Code != http.StatusOK {
		t.Fatalf("stop: %d", rec.Code)
	}
}

func TestAdminListCountsPictureExercises(t *testing.T) {
	t.Parallel()
	e, store, _ := newImageEnv(t)
	publishedLesson(t, e, "a1-to-be")
	publishedLesson(t, e, "a1-articles")
	summary := func(id string) LessonSummary {
		t.Helper()
		list, err := e.svc.AdminList(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range list {
			if s.PointID == id {
				return s
			}
		}
		t.Fatalf("%s not listed", id)
		return LessonSummary{}
	}
	if s := summary("a1-to-be"); s.Pictures != 0 || s.PicturesComplete {
		t.Fatalf("before = %+v", s)
	}
	l, _, err := e.svc.AddPictures(t.Context(), "a1-to-be")
	if err != nil {
		t.Fatal(err)
	}
	if s := summary("a1-to-be"); s.Pictures != 4 || s.PicturesDrawn != 0 || !s.PicturesComplete {
		t.Fatalf("added = %+v", s)
	}
	if err := store.Save(t.Context(), ExampleImage{PointID: "a1-to-be", Key: PictureKey(l.Content.Practice[8]), MIME: "image/jpeg", Data: []byte{1}}); err != nil {
		t.Fatal(err)
	}
	if s := summary("a1-to-be"); s.PicturesDrawn != 1 {
		t.Fatalf("drawn = %+v", s)
	}
	if s := summary("a1-articles"); s.Pictures != 0 || s.PicturesComplete {
		t.Fatalf("not drawable = %+v", s)
	}
}
