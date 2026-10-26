package grammar

import (
	"errors"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
)

func flagByID(l Lesson) map[string]Check {
	out := map[string]Check{}
	for _, c := range l.Checks {
		out[c.ExerciseID] = c
	}
	return out
}

func intp(n int) *int { return &n }

func TestGenerateRunsTheCheck(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l, _, err := e.svc.Generate(t.Context(), point, false)
	if err != nil {
		t.Fatal(err)
	}
	if e.ai.calls != 1 || e.ai.solveCalls != 1 {
		t.Errorf("AI calls = %d + %d, want 1 + 1", e.ai.calls, e.ai.solveCalls)
	}
	if len(l.Checks) != 0 || !l.CheckedAt.Equal(testNow) {
		t.Errorf("checks = %v at %v", l.Checks, l.CheckedAt)
	}
	if got := len(e.ai.solveReq.Exercises); got != 14 {
		t.Errorf("solve request has %d exercises, want 14", got)
	}
	if stored, _ := e.svc.AdminGet(t.Context(), point); !stored.CheckedAt.Equal(testNow) {
		t.Errorf("stored CheckedAt = %v", stored.CheckedAt)
	}
}

func TestCheckFlagsMismatchesAndAmbiguity(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.ai.solve = func(req ai.SolveRequest) ([]ai.Solution, error) {
		out := solveLikeFixtures(req)
		for i := range out {
			switch out[i].ExerciseID {
			case "p1": // choice: another option
				out[i].ChoiceIndex = intp(2)
			case "p2": // fill: a word that is not accepted
				out[i].Answer = "are"
			case "p3": // reorder: a different sentence
				out[i].Sentence = "A teacher is she"
			case "m1":
				out[i].Ambiguous, out[i].NoteVi = true, "Hai đáp án đều đúng."
			case "m2": // fill: differs only by case and spaces: fine
				out[i].Answer = "  AM "
			case "m3": // reorder: differs only by case, spaces and the final stop: fine
				out[i].Sentence = "she  is a Teacher."
			}
		}
		return out, nil
	}
	l, _, err := e.svc.Generate(t.Context(), point, false)
	if err != nil {
		t.Fatal(err)
	}
	got := flagByID(l)
	if len(got) != 4 {
		t.Fatalf("checks = %+v", l.Checks)
	}
	for id, kind := range map[string]CheckKind{"p1": CheckMismatch, "p2": CheckMismatch, "p3": CheckMismatch, "m1": CheckAmbiguous} {
		if got[id].Kind != kind || got[id].NoteVi == "" {
			t.Errorf("%s = %+v, want %s with a note", id, got[id], kind)
		}
	}
	if got["m1"].NoteVi != "Hai đáp án đều đúng." {
		t.Errorf("m1 note = %q", got["m1"].NoteVi)
	}
	if sum, _ := e.svc.AdminList(t.Context()); sum[0].Flags != 4 || !sum[0].Checked {
		t.Errorf("summary = %+v", sum[0])
	}
}

func TestCheckAcceptsExpandedFillAnswers(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c := aiContent()
	c.Practice[1] = mod(goodFill("x"), func(x *ai.GrammarExercise) { x.Text = "She ___ here."; x.Answers = []string{"is not"} })
	e.ai.content = c
	e.ai.solve = func(req ai.SolveRequest) ([]ai.Solution, error) {
		out := solveLikeFixtures(req)
		for i := range out {
			if out[i].ExerciseID == "p2" {
				out[i].Answer = "isn’t"
			}
		}
		return out, nil
	}
	l, _, err := e.svc.Generate(t.Context(), point, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(l.Checks) != 0 {
		t.Errorf("checks = %+v", l.Checks)
	}
}

func TestGenerateSurvivesACheckFailure(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.ai.solve = func(ai.SolveRequest) ([]ai.Solution, error) { return nil, errors.New("boom") }
	l, created, err := e.svc.Generate(t.Context(), point, false)
	if err != nil || !created {
		t.Fatalf("Generate = %v, %v", created, err)
	}
	if !l.CheckedAt.IsZero() || len(l.Checks) != 0 {
		t.Errorf("lesson = %+v", l)
	}
	if stored, err := e.svc.AdminGet(t.Context(), point); err != nil || len(stored.Content.Practice) != 8 || !stored.CheckedAt.IsZero() {
		t.Errorf("stored = %+v, %v", stored, err)
	}
}

func TestCheckErrorsLeaveTheLessonAlone(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	if _, err := e.svc.Check(t.Context(), point); !errors.Is(err, ErrNotFound) {
		t.Errorf("no lesson: %v", err)
	}
	if _, err := e.svc.Check(t.Context(), "nope"); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown point: %v", err)
	}
	if _, _, err := e.svc.Generate(t.Context(), point, false); err != nil {
		t.Fatal(err)
	}
	before, _ := e.svc.AdminGet(t.Context(), point)

	e.ai.solve = func(ai.SolveRequest) ([]ai.Solution, error) { return nil, ai.ErrQuota }
	if _, err := e.svc.Check(t.Context(), point); !errors.Is(err, ai.ErrQuota) {
		t.Errorf("quota: %v", err)
	}
	e.ai.solve = func(ai.SolveRequest) ([]ai.Solution, error) { return nil, errors.New("boom") }
	if _, err := e.svc.Check(t.Context(), point); !errors.Is(err, errAI) {
		t.Errorf("failure: %v", err)
	}
	after, _ := e.svc.AdminGet(t.Context(), point)
	if !after.CheckedAt.Equal(before.CheckedAt) || len(after.Checks) != len(before.Checks) {
		t.Errorf("lesson changed: %+v -> %+v", before, after)
	}
}

func TestCheckReplacesOldFlags(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.ai.solve = func(req ai.SolveRequest) ([]ai.Solution, error) {
		out := solveLikeFixtures(req)
		out[0].Ambiguous = true
		return out, nil
	}
	if l, _, _ := e.svc.Generate(t.Context(), point, false); len(l.Checks) != 1 {
		t.Fatalf("checks = %+v", l.Checks)
	}
	e.ai.solve = nil
	e.now = e.now.Add(time.Hour)
	l, err := e.svc.Check(t.Context(), point)
	if err != nil || len(l.Checks) != 0 || !l.CheckedAt.Equal(e.now) {
		t.Errorf("Check = %+v, %v", l, err)
	}
}

func TestSaveClearsTheFlags(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.ai.solve = func(req ai.SolveRequest) ([]ai.Solution, error) {
		out := solveLikeFixtures(req)
		out[0].Ambiguous = true
		return out, nil
	}
	l, _, _ := e.svc.Generate(t.Context(), point, false)
	if len(l.Checks) == 0 {
		t.Fatal("no flags")
	}
	l, err := e.svc.Save(t.Context(), point, l.Content)
	if err != nil {
		t.Fatal(err)
	}
	if len(l.Checks) != 0 || !l.CheckedAt.IsZero() {
		t.Errorf("after save: %+v %v", l.Checks, l.CheckedAt)
	}
	if stored, _ := e.svc.AdminGet(t.Context(), point); len(stored.Checks) != 0 || !stored.CheckedAt.IsZero() {
		t.Errorf("stored: %+v %v", stored.Checks, stored.CheckedAt)
	}
}

func TestPublishIsBlockedByFlagsUntilAcknowledged(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.ai.solve = func(req ai.SolveRequest) ([]ai.Solution, error) {
		out := solveLikeFixtures(req)
		out[0].Ambiguous, out[1].Ambiguous = true, true
		return out, nil
	}
	if _, _, err := e.svc.Generate(t.Context(), point, false); err != nil {
		t.Fatal(err)
	}
	var fe *FlagsError
	if _, err := e.svc.Publish(t.Context(), point, false); !errors.As(err, &fe) || fe.Count != 2 {
		t.Fatalf("Publish = %v", err)
	}
	if stored, _ := e.svc.AdminGet(t.Context(), point); stored.Status != StatusDraft {
		t.Errorf("status = %s", stored.Status)
	}
	l, err := e.svc.Publish(t.Context(), point, true)
	if err != nil || l.Status != StatusPublished {
		t.Fatalf("acknowledged Publish = %+v, %v", l, err)
	}
}

func TestPublishIsNotBlockedWhenNeverChecked(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.ai.solve = func(ai.SolveRequest) ([]ai.Solution, error) { return nil, errors.New("boom") }
	if _, _, err := e.svc.Generate(t.Context(), point, false); err != nil {
		t.Fatal(err)
	}
	if l, err := e.svc.Publish(t.Context(), point, false); err != nil || l.Status != StatusPublished {
		t.Errorf("Publish = %+v, %v", l, err)
	}
}

func TestLearnerGetExpandsFillAnswersWithoutChangingTheLesson(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c := aiContent()
	c.Practice[1] = mod(goodFill("x"), func(x *ai.GrammarExercise) { x.Text = "She ___ here."; x.Answers = []string{"is not"} })
	e.ai.content = c
	e.ai.solve = func(req ai.SolveRequest) ([]ai.Solution, error) {
		out := solveLikeFixtures(req)
		out[1].Answer = "is not"
		return out, nil
	}
	e.published(t)

	d, err := e.svc.Get(t.Context(), "u1", point)
	if err != nil {
		t.Fatal(err)
	}
	got := d.Content.Practice[1].Answers
	if len(got) < 2 || got[0] != "is not" || got[1] != "isn't" {
		t.Errorf("learner answers = %q", got)
	}
	if stored, _ := e.svc.AdminGet(t.Context(), point); len(stored.Content.Practice[1].Answers) != 1 {
		t.Errorf("stored answers = %q", stored.Content.Practice[1].Answers)
	}
}

func TestReportValidatesInput(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ok := ReportInput{ExerciseID: "p3", Reason: ReasonTypo}

	if err := e.svc.Report(t.Context(), "u1", point, ok); !errors.Is(err, ErrNotFound) {
		t.Errorf("no lesson: %v", err)
	}
	if _, _, err := e.svc.Generate(t.Context(), point, false); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.Report(t.Context(), "u1", point, ok); !errors.Is(err, ErrNotFound) {
		t.Errorf("draft: %v", err)
	}
	if _, err := e.svc.Publish(t.Context(), point, false); err != nil {
		t.Fatal(err)
	}

	long := make([]rune, 301)
	for i := range long {
		long[i] = 'a'
	}
	cases := map[string]ReportInput{
		"exerciseId": {ExerciseID: "p99", Reason: ReasonTypo},
		"reason":     {ExerciseID: "p3", Reason: "nope"},
		"note":       {ExerciseID: "p3", Reason: ReasonTypo, Note: string(long)},
	}
	for field, in := range cases {
		var verr *ValidationError
		if err := e.svc.Report(t.Context(), "u1", point, in); !errors.As(err, &verr) || verr.Fields[field] == "" {
			t.Errorf("%s: %v", field, err)
		}
	}
	if len(e.reports.rows) != 0 {
		t.Errorf("stored %d bad reports", len(e.reports.rows))
	}
	if err := e.svc.Report(t.Context(), "u1", point, ReportInput{ExerciseID: "m2", Reason: ReasonOther, Note: string(long[:300])}); err != nil {
		t.Errorf("valid mastery report: %v", err)
	}
}

func TestReportsGroupResolveAndReopen(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.published(t)
	report := func(user, ex string, reason ReportReason, note string) {
		t.Helper()
		e.now = e.now.Add(time.Minute)
		if err := e.svc.Report(t.Context(), user, point, ReportInput{ExerciseID: ex, Reason: reason, Note: note}); err != nil {
			t.Fatal(err)
		}
	}
	report("u1", "p3", ReasonWrongAnswer, "note one")
	report("u2", "p3", ReasonTypo, "")
	report("u3", "p3", ReasonTypo, "note three")
	report("u4", "p3", ReasonOther, "note four")
	report("u1", "p1", ReasonAmbiguous, "other exercise") // newest group

	groups, err := e.svc.AdminReports(t.Context())
	if err != nil || len(groups) != 2 {
		t.Fatalf("groups = %+v, %v", groups, err)
	}
	if groups[0].ExerciseID != "p1" || groups[1].ExerciseID != "p3" {
		t.Errorf("order = %s, %s", groups[0].ExerciseID, groups[1].ExerciseID)
	}
	g := groups[1]
	if g.Count != 4 || g.Reasons[ReasonTypo] != 2 || g.Reasons[ReasonWrongAnswer] != 1 || g.PointID != point {
		t.Errorf("group = %+v", g)
	}
	if len(g.Notes) != 3 || g.Notes[0] != "note four" || g.Notes[1] != "note three" || g.Notes[2] != "note one" {
		t.Errorf("notes = %q, want the 3 newest non-empty", g.Notes)
	}
	if of, _ := e.svc.AdminReportsOf(t.Context(), point); len(of) != 2 {
		t.Errorf("of point = %d", len(of))
	}
	if of, _ := e.svc.AdminReportsOf(t.Context(), "a1-other"); len(of) != 0 {
		t.Errorf("of other point = %d", len(of))
	}

	n, err := e.svc.ResolveReports(t.Context(), point, "p3")
	if err != nil || n != 4 {
		t.Fatalf("Resolve = %d, %v", n, err)
	}
	if groups, _ = e.svc.AdminReports(t.Context()); len(groups) != 1 || groups[0].ExerciseID != "p1" {
		t.Errorf("after resolve = %+v", groups)
	}

	// Reporting again reopens the same row.
	report("u1", "p3", ReasonTypo, "again")
	groups, _ = e.svc.AdminReports(t.Context())
	if len(groups) != 2 || groups[0].ExerciseID != "p3" || groups[0].Count != 1 {
		t.Errorf("after reopen = %+v", groups)
	}
	if len(e.reports.rows) != 5 {
		t.Errorf("rows = %d, want 5 (upsert)", len(e.reports.rows))
	}

	if _, err := e.svc.ResolveReports(t.Context(), "nope", "p3"); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown point: %v", err)
	}
	var verr *ValidationError
	if _, err := e.svc.ResolveReports(t.Context(), point, " "); !errors.As(err, &verr) {
		t.Errorf("empty exercise: %v", err)
	}
	if n, err := e.svc.ResolveReports(t.Context(), point, "p8"); err != nil || n != 0 {
		t.Errorf("nothing to resolve = %d, %v", n, err)
	}
}

// Fixed 2026-10-26: a model wrote its notes into the sentence of a reorder, and the flag quoted them.
func TestCheckDoesNotQuoteAGarbledAnswer(t *testing.T) {
	t.Parallel()
	e := Exercise{ID: "p3", Kind: KindReorder, Sentence: "Where do you live?", Words: []string{"Where", "do", "you", "live?", "are"}}
	c, flagged := compareSolution(e, ai.Solution{ExerciseID: "p3", Sentence: "Where do you live? wait words: ['Where','do']"})
	if !flagged || c.Kind != CheckUnchecked || c.NoteVi != garbledNote {
		t.Fatalf("check = %+v", c)
	}
	if _, flagged := compareSolution(e, ai.Solution{ExerciseID: "p3", Sentence: "where do you live"}); flagged {
		t.Fatal("the stored sentence without its question mark was flagged")
	}
	if c, _ := compareSolution(e, ai.Solution{ExerciseID: "p3", Sentence: "Where are you live?"}); c.Kind != CheckMismatch {
		t.Fatalf("a real other answer = %+v", c)
	}
}

func TestCheckAsksAgainForUnusableAnswers(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		secondBad bool
	}{{"second answer fine", false}, {"second answer bad too", true}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e := newEnv(t)
			calls := 0
			e.ai.solve = func(req ai.SolveRequest) ([]ai.Solution, error) {
				calls++
				out := solveLikeFixtures(req)
				if calls == 2 && len(req.Exercises) != 1 {
					t.Errorf("retry asked %d exercises, want only p3", len(req.Exercises))
				}
				for i := range out {
					if out[i].ExerciseID == "p3" && (calls == 1 || tc.secondBad) {
						out[i].Sentence = "wait words are ['She','is'] so She is a teacher"
					}
				}
				return out, nil
			}
			l, _, err := e.svc.Generate(t.Context(), point, false)
			if err != nil {
				t.Fatal(err)
			}
			got := flagByID(l)
			if calls != 2 {
				t.Errorf("solve calls = %d, want 2", calls)
			}
			if tc.secondBad {
				if len(got) != 1 || got["p3"].Kind != CheckUnchecked || got["p3"].NoteVi != garbledNote {
					t.Errorf("checks = %+v", l.Checks)
				}
			} else if len(got) != 0 {
				t.Errorf("checks = %+v", l.Checks)
			}
		})
	}
}

func TestCheckReadsASentenceWrittenAsTheAnswer(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.ai.solve = func(req ai.SolveRequest) ([]ai.Solution, error) {
		out := solveLikeFixtures(req)
		for i := range out {
			if out[i].Sentence != "" {
				out[i].Answer, out[i].Sentence = out[i].Sentence, ""
			}
		}
		return out, nil
	}
	l, _, err := e.svc.Generate(t.Context(), point, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(l.Checks) != 0 || e.ai.solveCalls != 1 {
		t.Errorf("checks = %+v, solve calls = %d", l.Checks, e.ai.solveCalls)
	}
}

// Fixed 2026-10-26: a model wrote "We were ... summer. Correct error was -> were. Perfect." as a fix.
func TestCheckKeepsAStoredAnswerFollowedByNotes(t *testing.T) {
	t.Parallel()
	fix := Exercise{ID: "f1", Kind: KindFix, Text: "We was at the beach last summer.", Sentence: "We were at the beach last summer."}
	notes := ai.Solution{ExerciseID: "f1", Sentence: "We were at the beach last summer. Correct error was -> were. Perfect. No meaningOff."}
	if c, flagged := compareSolution(fix, keepLeadingAnswer(fix, notes)); flagged {
		t.Fatalf("check = %+v", c)
	}
	other := ai.Solution{ExerciseID: "f1", Sentence: "We are at the beach this summer. Wait, the error was was, so it is were."}
	if c, _ := compareSolution(fix, keepLeadingAnswer(fix, other)); c.Kind != CheckUnchecked {
		t.Fatalf("notes after another answer = %+v", c)
	}
	reorder := Exercise{ID: "r1", Kind: KindReorder, Sentence: "I am happy", Words: []string{"I", "am", "happy", "now"}}
	longer := ai.Solution{ExerciseID: "r1", Sentence: "I am happy now"}
	if c, _ := compareSolution(reorder, keepLeadingAnswer(reorder, longer)); c.Kind != CheckMismatch {
		t.Fatalf("a real longer answer = %+v", c)
	}
}

// Seen 2026-10-26 with gemini-3.6-flash: short notes after a right correction.
func TestCheckCutsShortNotesAfterAFinishedSentence(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ key, got string }{
		{"Does he like reading books?", `Does he like reading books? caps:false, meaningOff:false, noteVi:""`},
		{"See you on Monday.", "See you on Monday. assassination attempt? No."},
	} {
		fix := Exercise{ID: "f1", Kind: KindFix, Sentence: tc.key}
		if c, flagged := compareSolution(fix, keepLeadingAnswer(fix, ai.Solution{ExerciseID: "f1", Sentence: tc.got})); flagged {
			t.Errorf("%q: check = %+v", tc.got, c)
		}
	}
	fix := Exercise{ID: "f1", Kind: KindFix, Sentence: "See you on Monday."}
	if c, _ := compareSolution(fix, keepLeadingAnswer(fix, ai.Solution{ExerciseID: "f1", Sentence: "See you on Monday morning."})); c.Kind != CheckMismatch {
		t.Errorf("a real longer answer = %+v", c)
	}
}
