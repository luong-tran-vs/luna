package grammar

import (
	"errors"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
)

// flaggedEnv generates a lesson whose check flags p1 (mismatch), p2 (ambiguous) and leaves m1
// without an answer.
func flaggedEnv(t *testing.T) (*env, Lesson) {
	t.Helper()
	e := newEnv(t)
	e.ai.solve = func(req ai.SolveRequest) ([]ai.Solution, error) {
		var out []ai.Solution
		for _, s := range solveLikeFixtures(req) {
			switch s.ExerciseID {
			case "p1":
				s.ChoiceIndex = intp(3)
			case "p2":
				s.Ambiguous = true
			case "m1":
				continue
			}
			out = append(out, s)
		}
		return out, nil
	}
	l, _, err := e.svc.Generate(t.Context(), point, false)
	if err != nil {
		t.Fatal(err)
	}
	return e, l
}

func TestCheckMarksMissingSolutionsUnchecked(t *testing.T) {
	t.Parallel()
	_, l := flaggedEnv(t)
	got := flagByID(l)
	if len(got) != 3 || got["p1"].Kind != CheckMismatch || got["p2"].Kind != CheckAmbiguous {
		t.Fatalf("checks = %+v", l.Checks)
	}
	if c := got["m1"]; c.Kind != CheckUnchecked || c.NoteVi != "AI không trả lời câu này, hãy tự kiểm tra." {
		t.Errorf("m1 = %+v", c)
	}
	for _, c := range l.Checks {
		if c.Confirmed {
			t.Errorf("%s confirmed by a fresh check", c.ExerciseID)
		}
	}
}

func TestConfirmVerifyAndTheFlagGate(t *testing.T) {
	t.Parallel()
	e, _ := flaggedEnv(t)

	if _, err := e.svc.ConfirmCheck(t.Context(), point, "p5"); !errors.Is(err, ErrNotFound) {
		t.Errorf("confirm without a check: %v", err)
	}
	l, err := e.svc.ConfirmCheck(t.Context(), point, "p1")
	if err != nil || !flagByID(l)["p1"].Confirmed || flagByID(l)["p2"].Confirmed {
		t.Fatalf("confirm = %+v, %v", l.Checks, err)
	}
	sum, _ := e.svc.AdminList(t.Context())
	if sum[0].Flags != 2 || sum[0].Verified {
		t.Errorf("summary = %+v", sum[0])
	}
	// Only the unconfirmed ones block publishing.
	var fe *FlagsError
	if _, err := e.svc.Publish(t.Context(), point, false); !errors.As(err, &fe) || fe.Count != 2 {
		t.Fatalf("publish = %v", err)
	}

	e.now = e.now.Add(time.Hour)
	l, err = e.svc.Verify(t.Context(), point)
	if err != nil || !l.VerifiedAt.Equal(e.now) {
		t.Fatalf("verify = %+v, %v", l, err)
	}
	for _, c := range l.Checks {
		if !c.Confirmed {
			t.Errorf("%s not confirmed after verify", c.ExerciseID)
		}
	}
	if sum, _ = e.svc.AdminList(t.Context()); sum[0].Flags != 0 || !sum[0].Verified {
		t.Errorf("summary = %+v", sum[0])
	}
	if p, err := e.svc.Publish(t.Context(), point, false); err != nil || p.Status != StatusPublished {
		t.Errorf("publish after verify = %+v, %v", p, err)
	}

	// A new check starts over.
	e.ai.solve = nil
	if l, err = e.svc.Check(t.Context(), point); err != nil || !l.VerifiedAt.IsZero() || len(l.Checks) != 0 {
		t.Errorf("recheck = %+v, %v", l, err)
	}
}

func TestVerifyNeedsACheck(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.ai.solve = func(ai.SolveRequest) ([]ai.Solution, error) { return nil, errBoom }
	if _, _, err := e.svc.Generate(t.Context(), point, false); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.Verify(t.Context(), point); !errors.Is(err, ErrNotChecked) {
		t.Errorf("verify = %v", err)
	}
}

func TestSaveClearsVerification(t *testing.T) {
	t.Parallel()
	e, l := flaggedEnv(t)
	if _, err := e.svc.Verify(t.Context(), point); err != nil {
		t.Fatal(err)
	}
	saved, err := e.svc.Save(t.Context(), point, l.Content)
	if err != nil || !saved.VerifiedAt.IsZero() || !saved.CheckedAt.IsZero() || len(saved.Checks) != 0 {
		t.Errorf("save = %+v, %v", saved, err)
	}
}

func TestReplaceExerciseInPlace(t *testing.T) {
	t.Parallel()
	e, before := flaggedEnv(t)
	if _, err := e.svc.Verify(t.Context(), point); err != nil {
		t.Fatal(err)
	}
	e.now = e.now.Add(time.Hour)

	fix := Exercise{
		Kind: KindChoice, PromptVi: "Chọn", Text: " She ___ a nurse. ", Options: []string{"am", "is", "are", "be"},
		AnswerIndex: 1, ExplanationVi: "She đi với is.",
	}
	l, err := e.svc.ReplaceExercise(t.Context(), point, "p1", fix)
	if err != nil {
		t.Fatal(err)
	}
	if l.Content.Practice[0].ID != "p1" || l.Content.Practice[0].Text != "She ___ a nurse." || !l.Edited {
		t.Errorf("p1 = %+v", l.Content.Practice[0])
	}
	if len(l.Content.Practice) != len(before.Content.Practice) || l.Content.Practice[1].ID != "p2" {
		t.Errorf("other exercises moved: %v", ids(l.Content.Practice))
	}
	got := flagByID(l)
	if _, ok := got["p1"]; ok || len(got) != 2 || !got["p2"].Confirmed {
		t.Errorf("checks = %+v", l.Checks)
	}
	if !l.VerifiedAt.IsZero() || !l.CheckedAt.Equal(before.CheckedAt) {
		t.Errorf("verifiedAt %v checkedAt %v", l.VerifiedAt, l.CheckedAt)
	}
	// Mastery works too.
	if _, err := e.svc.ReplaceExercise(t.Context(), point, "m1", mod2(goodChoiceExercise(), "m")); err != nil {
		t.Errorf("mastery replace: %v", err)
	}
}

func goodChoiceExercise() Exercise {
	return Exercise{
		Kind: KindChoice, PromptVi: "Chọn", Text: "He ___ here.", Options: []string{"am", "is", "are", "be"},
		AnswerIndex: 1, ExplanationVi: "He đi với is.",
	}
}

func mod2(e Exercise, _ string) Exercise { return e }

func TestReplaceExerciseRejectsBadInput(t *testing.T) {
	t.Parallel()
	e, before := flaggedEnv(t)

	other := goodChoiceExercise()
	other.Kind = KindFill
	var verr *ValidationError
	if _, err := e.svc.ReplaceExercise(t.Context(), point, "p1", other); !errors.As(err, &verr) || verr.Fields["kind"] == "" {
		t.Errorf("kind change: %v", err)
	}
	bad := goodChoiceExercise()
	bad.Options = []string{"a", "b"}
	bad.ExplanationVi = ""
	if _, err := e.svc.ReplaceExercise(t.Context(), point, "p1", bad); !errors.As(err, &verr) ||
		verr.Fields["exercise.options"] == "" || verr.Fields["exercise.explanationVi"] == "" {
		t.Errorf("invalid: %v", err)
	}
	if _, err := e.svc.ReplaceExercise(t.Context(), point, "p99", goodChoiceExercise()); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown exercise: %v", err)
	}
	after, _ := e.svc.AdminGet(t.Context(), point)
	if after.Edited || len(after.Checks) != len(before.Checks) || after.Content.Practice[0].Text != before.Content.Practice[0].Text {
		t.Error("a rejected replace must change nothing")
	}
}

func TestDeleteExerciseKeepsOtherIDsAndClosesReports(t *testing.T) {
	t.Parallel()
	e, _ := flaggedEnv(t)
	// 8 practice exercises: p1 and p2 are flagged. Deleting is allowed down to 6.
	if _, err := e.svc.Publish(t.Context(), point, true); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"p2", "p3"} {
		if err := e.svc.Report(t.Context(), "u1", point, ReportInput{ExerciseID: id, Reason: ReasonTypo}); err != nil {
			t.Fatal(err)
		}
	}

	l, err := e.svc.DeleteExercise(t.Context(), point, "p2")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"p1", "p3", "p4", "p5", "p6", "p7", "p8"}; !slices.Equal(ids(l.Content.Practice), want) {
		t.Errorf("ids = %v, want %v", ids(l.Content.Practice), want)
	}
	if got := flagByID(l); len(got) != 2 || got["p1"].ExerciseID == "" || got["m1"].ExerciseID == "" {
		t.Errorf("checks = %+v", l.Checks)
	}
	if !l.Edited || !l.VerifiedAt.IsZero() || l.CheckedAt.IsZero() {
		t.Errorf("edited %v verified %v checked %v", l.Edited, l.VerifiedAt, l.CheckedAt)
	}
	groups, _ := e.svc.AdminReports(t.Context())
	if len(groups) != 1 || groups[0].ExerciseID != "p3" {
		t.Errorf("open reports = %+v", groups)
	}

	// A new exercise added later (via the whole-content save) gets the smallest free id.
	if got := Normalize(Content{Practice: append(slices.Clone(l.Content.Practice), Exercise{})}).Practice; got[len(got)-1].ID != "p2" {
		t.Errorf("new id = %q", got[len(got)-1].ID)
	}

	// Down to the minimum, then refused without saving.
	if _, err := e.svc.DeleteExercise(t.Context(), point, "p3"); err != nil {
		t.Fatal(err)
	}
	var verr *ValidationError
	_, err = e.svc.DeleteExercise(t.Context(), point, "p4")
	if !errors.As(err, &verr) || verr.Fields["content.practice"] != "Cần ít nhất 6 bài luyện tập" {
		t.Fatalf("below minimum: %v", err)
	}
	if got, _ := e.svc.AdminGet(t.Context(), point); len(got.Content.Practice) != 6 {
		t.Errorf("practice = %d after a refused delete", len(got.Content.Practice))
	}
	if _, err := e.svc.DeleteExercise(t.Context(), point, "p2"); !errors.Is(err, ErrNotFound) {
		t.Errorf("already deleted: %v", err)
	}
	// Mastery minimum is 5 of 6.
	if _, err := e.svc.DeleteExercise(t.Context(), point, "m6"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.DeleteExercise(t.Context(), point, "m5"); !errors.As(err, &verr) || verr.Fields["content.mastery"] == "" {
		t.Errorf("mastery minimum: %v", err)
	}
}

func TestNormalizeKeepsValidIDs(t *testing.T) {
	t.Parallel()
	in := Content{Practice: []Exercise{
		{ID: "p3"}, {ID: "p3"}, {ID: ""}, {ID: "m1"}, {ID: "p01"}, {ID: " p7 "}, {ID: "px"}, {ID: "p1"},
	}}
	got := ids(Normalize(in).Practice)
	want := []string{"p3", "p2", "p4", "p5", "p6", "p7", "p8", "p1"}
	if !slices.Equal(got, want) {
		t.Errorf("ids = %v, want %v", got, want)
	}
	if got := ids(Normalize(Content{Mastery: []Exercise{{ID: "p1"}, {}}}).Mastery); !slices.Equal(got, []string{"m1", "m2"}) {
		t.Errorf("mastery ids = %v", got)
	}
}

func TestEditEndpoints(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	a.env.ai.solve = func(req ai.SolveRequest) ([]ai.Solution, error) {
		var out []ai.Solution
		for _, s := range solveLikeFixtures(req) {
			if s.ExerciseID == "m1" {
				continue
			}
			out = append(out, s)
		}
		return out, nil
	}
	a.generate(t)

	// checks[].confirmed, verifiedAt, unchecked.
	get := func() map[string]any {
		rec := a.do(t, http.MethodGet, adminPath, "admin", "")
		return decode(t, rec)["lesson"].(map[string]any)
	}
	l := get()
	checks := l["checks"].([]any)
	if len(checks) != 1 || checks[0].(map[string]any)["kind"] != "unchecked" || checks[0].(map[string]any)["confirmed"] != false || l["verifiedAt"] != nil {
		t.Fatalf("lesson = %v", l)
	}

	rec := a.do(t, http.MethodPost, adminPath+"/checks/p5/confirm", "admin", "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("confirm without check: %d", rec.Code)
	}
	rec = a.do(t, http.MethodPost, adminPath+"/checks/m1/confirm", "admin", "")
	if rec.Code != http.StatusOK || decode(t, rec)["lesson"].(map[string]any)["checks"].([]any)[0].(map[string]any)["confirmed"] != true {
		t.Fatalf("confirm: %d %s", rec.Code, rec.Body)
	}
	row := decode(t, a.do(t, http.MethodGet, "/api/admin/grammar-lessons", "admin", ""))["lessons"].([]any)[0].(map[string]any)
	if row["flags"] != float64(0) || row["verified"] != false {
		t.Errorf("row = %v", row)
	}

	rec = a.do(t, http.MethodPost, adminPath+"/verify", "admin", "")
	if rec.Code != http.StatusOK || decode(t, rec)["lesson"].(map[string]any)["verifiedAt"] == nil {
		t.Fatalf("verify: %d %s", rec.Code, rec.Body)
	}
	if row = decode(t, a.do(t, http.MethodGet, "/api/admin/grammar-lessons", "admin", ""))["lessons"].([]any)[0].(map[string]any); row["verified"] != true {
		t.Errorf("row = %v", row)
	}

	// Replace an exercise.
	body := `{"kind":"choice","promptVi":"Chọn","text":"He ___ here.","options":["am","is","are","be"],"answerIndex":1,"explanationVi":"He đi với is."}`
	rec = a.do(t, http.MethodPut, adminPath+"/exercises/p1", "admin", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("replace: %d %s", rec.Code, rec.Body)
	}
	got := decode(t, rec)["lesson"].(map[string]any)
	first := got["content"].(map[string]any)["practice"].([]any)[0].(map[string]any)
	if first["id"] != "p1" || first["text"] != "He ___ here." || got["verifiedAt"] != nil || got["edited"] != true || got["checkedAt"] == nil {
		t.Errorf("lesson = %v", got)
	}
	rec = a.do(t, http.MethodPut, adminPath+"/exercises/p1", "admin", `{"kind":"fill","text":"x ___","answers":["a"],"explanationVi":"y"}`)
	if rec.Code != http.StatusBadRequest || decode(t, rec)["fields"].(map[string]any)["kind"] == nil {
		t.Errorf("kind change: %d %s", rec.Code, rec.Body)
	}
	rec = a.do(t, http.MethodPut, adminPath+"/exercises/p1", "admin", `{"kind":"choice","text":"x","options":["a"],"answerIndex":0,"explanationVi":"y"}`)
	if rec.Code != http.StatusBadRequest || decode(t, rec)["fields"].(map[string]any)["exercise.options"] == nil {
		t.Errorf("invalid: %d %s", rec.Code, rec.Body)
	}
	if rec = a.do(t, http.MethodPut, adminPath+"/exercises/zz", "admin", body); rec.Code != http.StatusNotFound {
		t.Errorf("unknown exercise: %d", rec.Code)
	}

	// Delete.
	rec = a.do(t, http.MethodDelete, adminPath+"/exercises/p2", "admin", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("delete: %d %s", rec.Code, rec.Body)
	}
	practice := decode(t, rec)["lesson"].(map[string]any)["content"].(map[string]any)["practice"].([]any)
	if len(practice) != 7 || practice[1].(map[string]any)["id"] != "p3" {
		t.Errorf("practice after delete has %d items", len(practice))
	}
	a.do(t, http.MethodDelete, adminPath+"/exercises/p3", "admin", "")
	rec = a.do(t, http.MethodDelete, adminPath+"/exercises/p4", "admin", "")
	if rec.Code != http.StatusBadRequest || decode(t, rec)["error"] != "validation_failed" ||
		decode(t, rec)["fields"].(map[string]any)["content.practice"] != "Cần ít nhất 6 bài luyện tập" {
		t.Errorf("below minimum: %d %s", rec.Code, rec.Body)
	}
	if rec = a.do(t, http.MethodDelete, adminPath+"/exercises/zz", "admin", ""); rec.Code != http.StatusNotFound {
		t.Errorf("delete unknown: %d", rec.Code)
	}
}

func TestVerifyEndpointNeedsACheck(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	a.env.ai.solve = func(ai.SolveRequest) ([]ai.Solution, error) { return nil, errBoom }
	a.generate(t)
	rec := a.do(t, http.MethodPost, adminPath+"/verify", "admin", "")
	got := decode(t, rec)
	if rec.Code != http.StatusConflict || got["error"] != "grammar_not_checked" || got["message"] != "Hãy kiểm tra bằng AI trước khi xác nhận." {
		t.Errorf("verify: %d %s", rec.Code, rec.Body)
	}
}

func TestEditEndpointsRequireAdmin(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	a.generate(t)
	for _, ep := range []struct{ method, path, body string }{
		{http.MethodPost, adminPath + "/checks/p1/confirm", ""},
		{http.MethodPost, adminPath + "/verify", ""},
		{http.MethodPut, adminPath + "/exercises/p1", `{}`},
		{http.MethodDelete, adminPath + "/exercises/p1", ""},
	} {
		if rec := a.do(t, ep.method, ep.path, "", ep.body); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s anonymous = %d", ep.method, ep.path, rec.Code)
		}
		if rec := a.do(t, ep.method, ep.path, "learner", ep.body); rec.Code != http.StatusForbidden {
			t.Errorf("%s %s learner = %d", ep.method, ep.path, rec.Code)
		}
	}
}
