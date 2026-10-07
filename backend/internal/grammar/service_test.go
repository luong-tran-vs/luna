package grammar

import (
	"errors"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
)

const point = "a1-to-be"

// published generates and publishes the lesson of point.
func (e *env) published(t *testing.T) Lesson {
	t.Helper()
	if _, _, err := e.svc.Generate(t.Context(), point, false); err != nil {
		t.Fatal(err)
	}
	l, err := e.svc.Publish(t.Context(), point, false)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func (e *env) attempt(t *testing.T, kind string, correct, total int, wrong ...string) (Progress, bool) {
	t.Helper()
	p, passed, err := e.svc.RecordAttempt(t.Context(), "u1", point, Attempt{Kind: kind, Correct: correct, Total: total, Wrong: wrong})
	if err != nil {
		t.Fatalf("attempt: %v", err)
	}
	return p, passed
}

func TestGenerateCreatesADraft(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l, created, err := e.svc.Generate(t.Context(), point, false)
	if err != nil || !created {
		t.Fatalf("Generate = %v, %v", created, err)
	}
	if l.Status != StatusDraft || l.Edited || !l.CreatedAt.Equal(testNow) || !l.PublishedAt.IsZero() {
		t.Errorf("lesson = %+v", l)
	}
	if e.ai.calls != 1 || e.ai.req.Level != "A1" || e.ai.req.TitleEn == "" || e.ai.req.Pattern == "" || len(e.ai.req.Examples) == 0 {
		t.Errorf("AI called %d times with %+v", e.ai.calls, e.ai.req)
	}
	if l.Content.Practice[0].ID != "p1" || l.Content.Mastery[0].ID != "m1" {
		t.Errorf("ids = %v %v", ids(l.Content.Practice), ids(l.Content.Mastery))
	}
	if got, _ := e.svc.AdminGet(t.Context(), point); got.Status != StatusDraft || len(got.Content.Practice) != 8 {
		t.Errorf("stored = %+v", got)
	}
}

func TestGenerateRefusesToOverwriteWithoutForce(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	// An untouched draft is replaced freely.
	if _, _, err := e.svc.Generate(t.Context(), point, false); err != nil {
		t.Fatal(err)
	}
	if _, created, err := e.svc.Generate(t.Context(), point, false); err != nil || created {
		t.Fatalf("regenerate a draft: %v, created %v", err, created)
	}

	if _, err := e.svc.Save(t.Context(), point, Normalize(validContent())); err != nil {
		t.Fatal(err)
	}
	calls := e.ai.calls
	if _, _, err := e.svc.Generate(t.Context(), point, false); !errors.Is(err, ErrExists) {
		t.Fatalf("edited without force: %v", err)
	}
	if e.ai.calls != calls {
		t.Error("the AI must not be called when the lesson is kept")
	}
	e.now = testNow.Add(time.Hour)
	l, created, err := e.svc.Generate(t.Context(), point, true)
	if err != nil || created || l.Edited || l.Status != StatusDraft || !l.CreatedAt.Equal(testNow) {
		t.Fatalf("force: %+v %v %v", l, created, err)
	}

	e.published(t)
	if _, _, err := e.svc.Generate(t.Context(), point, false); !errors.Is(err, ErrExists) {
		t.Fatalf("published without force: %v", err)
	}
	l, _, err = e.svc.Generate(t.Context(), point, true)
	if err != nil || l.Status != StatusDraft || !l.PublishedAt.IsZero() {
		t.Fatalf("force over published: %+v %v", l, err)
	}
}

func TestGenerateErrors(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	if _, _, err := e.svc.Generate(t.Context(), "nope", false); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown point: %v", err)
	}
	for _, aiErr := range []error{ai.ErrNotConfigured, ai.ErrInvalidKey, ai.ErrQuota} {
		e.ai.err = aiErr
		if _, _, err := e.svc.Generate(t.Context(), point, false); !errors.Is(err, aiErr) {
			t.Errorf("%v became %v", aiErr, err)
		}
	}
	e.ai.err = errors.New("boom")
	if _, _, err := e.svc.Generate(t.Context(), point, false); !errors.Is(err, errAI) {
		t.Errorf("other AI error: %v", err)
	}
	e.ai.err = nil
	e.ai.content = ai.GrammarLessonContent{Objective: "x"}
	if _, _, err := e.svc.Generate(t.Context(), point, false); !errors.Is(err, ErrUnusable) {
		t.Errorf("unusable content: %v", err)
	}
	if _, err := e.svc.AdminGet(t.Context(), point); !errors.Is(err, ErrNotFound) {
		t.Errorf("a failed generation must store nothing: %v", err)
	}
}

func TestGenerateFiltersBrokenExercises(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.ai.content.Practice = append(e.ai.content.Practice, mod(goodChoice("bad"), func(x *ai.GrammarExercise) { x.AnswerIndex = 9 }))
	l, _, err := e.svc.Generate(t.Context(), point, false)
	if err != nil || len(l.Content.Practice) != 8 {
		t.Fatalf("Generate = %d practice, %v", len(l.Content.Practice), err)
	}
}

func TestSaveValidatesAndKeepsStatus(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	if _, err := e.svc.Save(t.Context(), point, validContent()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("save without a lesson: %v", err)
	}
	e.published(t)

	c := validContent()
	c.Objective = "Bạn có thể dùng to be."
	c.Practice[0], c.Practice[1] = c.Practice[1], c.Practice[0] // ids stay with their exercises
	e.now = testNow.Add(time.Hour)
	l, err := e.svc.Save(t.Context(), point, c)
	if err != nil {
		t.Fatal(err)
	}
	if !l.Edited || l.Status != StatusPublished || l.Content.Objective != "Bạn có thể dùng to be." || l.Content.Practice[0].ID != "p2" {
		t.Errorf("saved = %+v", l)
	}
	if !l.UpdatedAt.Equal(e.now) || !l.PublishedAt.Equal(testNow) {
		t.Errorf("times = %v %v", l.UpdatedAt, l.PublishedAt)
	}

	c.Practice[3].Options = []string{"a", "b"}
	_, err = e.svc.Save(t.Context(), point, c)
	var verr *ValidationError
	if !errors.As(err, &verr) || verr.Fields["content.practice[3].options"] == "" {
		t.Fatalf("invalid content: %v", err)
	}
	if got, _ := e.svc.AdminGet(t.Context(), point); got.Content.Practice[0].ID != "p2" || got.UpdatedAt.After(e.now) || got.Content.Practice[3].Kind == KindChoice && len(got.Content.Practice[3].Options) != 4 {
		t.Error("a rejected save must change nothing")
	}
}

func TestPublishAndUnpublish(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	if _, err := e.svc.Publish(t.Context(), point, false); !errors.Is(err, ErrNotFound) {
		t.Fatalf("publish without a lesson: %v", err)
	}
	if _, err := e.svc.Unpublish(t.Context(), "nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unpublish unknown: %v", err)
	}
	l := e.published(t)
	if l.Status != StatusPublished || !l.PublishedAt.Equal(testNow) {
		t.Fatalf("published = %+v", l)
	}
	e.now = testNow.Add(time.Hour)
	if l, _ = e.svc.Publish(t.Context(), point, false); !l.PublishedAt.Equal(testNow) {
		t.Errorf("publishing again moved publishedAt to %v", l.PublishedAt)
	}
	l, err := e.svc.Unpublish(t.Context(), point)
	if err != nil || l.Status != StatusDraft || !l.PublishedAt.IsZero() {
		t.Fatalf("unpublish = %+v, %v", l, err)
	}
	if _, err := e.svc.Get(t.Context(), "u1", point); !errors.Is(err, ErrNotFound) {
		t.Errorf("learner can still open it: %v", err)
	}
}

func TestPublishRefusesAnInvalidLesson(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c := Normalize(validContent())
	c.Practice = c.Practice[:2]
	e.lessons.lessons[point] = Lesson{PointID: point, Status: StatusDraft, Content: c}
	_, err := e.svc.Publish(t.Context(), point, false)
	var verr *ValidationError
	if !errors.As(err, &verr) || verr.Fields["content.practice"] == "" {
		t.Fatalf("publish invalid: %v", err)
	}
	if e.lessons.lessons[point].Status != StatusDraft {
		t.Error("an invalid lesson must stay a draft")
	}
}

func TestAdminListFollowsTheSyllabus(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	if items, err := e.svc.AdminList(t.Context()); err != nil || len(items) != 0 {
		t.Fatalf("empty list = %v, %v", items, err)
	}
	all := Default().All()
	for _, id := range []string{all[2].ID, all[0].ID} {
		e.lessons.lessons[id] = Lesson{PointID: id, Status: StatusDraft}
	}
	e.lessons.lessons["removed-point"] = Lesson{PointID: "removed-point"}
	items, _ := e.svc.AdminList(t.Context())
	if len(items) != 2 || items[0].PointID != all[0].ID || items[1].PointID != all[2].ID {
		t.Errorf("list = %+v", items)
	}
}

func TestLearnerListAndNext(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	a1 := Default().ByLevel("A1")

	ov, err := e.svc.List(t.Context(), "u1", "A1")
	if err != nil || len(ov.Points) != len(a1) || ov.Next != "" {
		t.Fatalf("empty overview: %d points, next %q, %v", len(ov.Points), ov.Next, err)
	}
	if p := ov.Points[0]; p.Available || p.Status != ProgressNew || p.ID != a1[0].ID {
		t.Errorf("first point = %+v", p)
	}

	// Publish the 1st and 3rd A1 points; a draft on the 2nd must stay hidden.
	for i, status := range map[int]Status{0: StatusPublished, 1: StatusDraft, 2: StatusPublished} {
		e.lessons.lessons[a1[i].ID] = Lesson{PointID: a1[i].ID, Status: status}
	}
	ov, _ = e.svc.List(t.Context(), "u1", "A1")
	if !ov.Points[0].Available || ov.Points[1].Available || !ov.Points[2].Available || ov.Next != a1[0].ID {
		t.Fatalf("availability wrong, next %q", ov.Next)
	}

	// Mastering the 1st moves next to the 3rd.
	e.progress.recs[[2]string{"u1", a1[0].ID}] = Progress{UserID: "u1", PointID: a1[0].ID, Status: ProgressMastered, BestMastery: 90}
	e.progress.recs[[2]string{"u2", a1[2].ID}] = Progress{UserID: "u2", PointID: a1[2].ID, Status: ProgressMastered}
	ov, _ = e.svc.List(t.Context(), "u1", "A1")
	if ov.Next != a1[2].ID || ov.Points[0].Status != ProgressMastered || ov.Points[0].BestMastery != 90 || ov.Points[2].Status != ProgressNew {
		t.Errorf("after mastering: next %q, points %+v", ov.Next, ov.Points[:3])
	}

	all, _ := e.svc.List(t.Context(), "u1", "")
	if len(all.Points) != len(Default().All()) {
		t.Errorf("no level: %d points", len(all.Points))
	}
	if b1, _ := e.svc.List(t.Context(), "u1", "B1"); b1.Next != "" {
		t.Errorf("B1 has no published lesson, next = %q", b1.Next)
	}
	var verr *ValidationError
	if _, err := e.svc.List(t.Context(), "u1", "Z9"); !errors.As(err, &verr) || verr.Fields["level"] == "" {
		t.Errorf("bad level: %v", err)
	}
}

func TestLearnerGet(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	if _, err := e.svc.Get(t.Context(), "u1", point); !errors.Is(err, ErrNotFound) {
		t.Fatalf("no lesson: %v", err)
	}
	if _, _, err := e.svc.Generate(t.Context(), point, false); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.Get(t.Context(), "u1", point); !errors.Is(err, ErrNotFound) {
		t.Fatalf("draft must be hidden: %v", err)
	}
	if _, err := e.svc.Get(t.Context(), "u1", "nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown point: %v", err)
	}
	e.published(t)
	d, err := e.svc.Get(t.Context(), "u1", point)
	if err != nil {
		t.Fatal(err)
	}
	if d.Point.ID != point || d.LessonCount != 3 || len(d.Content.Mastery) != 6 {
		t.Errorf("detail = %+v", d)
	}
	if d.Progress.Status != ProgressNew || d.Progress.Weak == nil || d.Progress.PracticeAttempts != 0 {
		t.Errorf("new learner progress = %+v", d.Progress)
	}
}

func TestRecordPracticeAttempt(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.published(t)
	e.now = testNow.Add(time.Minute)
	p, passed := e.attempt(t, "practice", 5, 8, "p1", "p2", "p7")
	if passed {
		t.Error("practice never passes")
	}
	if p.Status != ProgressLearning || p.PracticeAttempts != 1 || p.LastPractice != 63 || p.MasteryAttempts != 0 ||
		len(p.Weak) != 3 || !p.UpdatedAt.Equal(e.now) || p.UserID != "u1" || p.PointID != point {
		t.Errorf("progress = %+v", p)
	}
	p, _ = e.attempt(t, "practice", 8, 8)
	if p.PracticeAttempts != 2 || p.LastPractice != 100 || p.Weak == nil || len(p.Weak) != 0 {
		t.Errorf("perfect round = %+v", p)
	}
	// Another learner is not touched.
	if _, err := e.progress.Get(t.Context(), "u2", point); !errors.Is(err, ErrNotFound) {
		t.Errorf("u2 progress: %v", err)
	}
}

func TestMasteryPassAndFail(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.published(t)

	p, passed := e.attempt(t, "mastery", 4, 6, "m1", "m2") // 67%
	if passed || p.Status != ProgressLearning || p.BestMastery != 67 || p.MasteryAttempts != 1 || !p.MasteredAt.IsZero() {
		t.Fatalf("fail = %v %+v", passed, p)
	}
	e.now = testNow.Add(time.Hour)
	p, passed = e.attempt(t, "mastery", 5, 6, "m3") // 83%, above 80
	if !passed || p.Status != ProgressMastered || p.BestMastery != 83 || p.MasteryAttempts != 2 || !p.MasteredAt.Equal(e.now) {
		t.Fatalf("pass = %v %+v", passed, p)
	}

	// Mastered is never lost, and the best score only goes up.
	e.now = testNow.Add(2 * time.Hour)
	p, passed = e.attempt(t, "mastery", 1, 6, "m1", "m2", "m3", "m4", "m5")
	if passed || p.Status != ProgressMastered || p.BestMastery != 83 || !p.MasteredAt.Equal(testNow.Add(time.Hour)) {
		t.Fatalf("after a bad round = %v %+v", passed, p)
	}
	p, _ = e.attempt(t, "practice", 2, 8, "p1", "p2", "p3", "p4", "p5", "p6")
	if p.Status != ProgressMastered || len(p.Weak) != 6 {
		t.Errorf("after practice = %+v", p)
	}
}

func TestMasteryThresholdIsExact(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.published(t)
	// With 6 questions 5 right is 83% (pass) and 4 right is 67% (fail); extend to 10 for exactly 80%.
	l := e.lessons.lessons[point]
	l.Content.Mastery = Normalize(Content{Mastery: make([]Exercise, 10)}).Mastery
	e.lessons.lessons[point] = l
	_, passed := e.attempt(t, "mastery", 8, 10, "m1", "m2")
	if !passed {
		t.Error("80% must pass")
	}
	_, passed = e.attempt(t, "mastery", 7, 10, "m1", "m2", "m3")
	if passed {
		t.Error("70% must not pass")
	}
}

func TestRecordAttemptRejectsBadInput(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.published(t)
	tests := []struct {
		name string
		a    Attempt
		key  string
	}{
		{"unknown kind", Attempt{Kind: "exam", Correct: 8, Total: 8}, "kind"},
		{"wrong total for practice", Attempt{Kind: "practice", Correct: 5, Total: 6, Wrong: []string{"p1"}}, "total"},
		{"practice total used for mastery", Attempt{Kind: "mastery", Correct: 8, Total: 8}, "total"},
		{"negative correct", Attempt{Kind: "practice", Correct: -1, Total: 8, Wrong: make([]string, 9)}, "correct"},
		{"correct above total", Attempt{Kind: "practice", Correct: 9, Total: 8}, "correct"},
		{"wrong id of the other kind", Attempt{Kind: "practice", Correct: 7, Total: 8, Wrong: []string{"m1"}}, "wrong"},
		{"unknown wrong id", Attempt{Kind: "practice", Correct: 7, Total: 8, Wrong: []string{"p99"}}, "wrong"},
		{"duplicate wrong id", Attempt{Kind: "practice", Correct: 6, Total: 8, Wrong: []string{"p1", "p1"}}, "wrong"},
		{"too few wrong ids", Attempt{Kind: "practice", Correct: 5, Total: 8, Wrong: []string{"p1"}}, "wrong"},
		{"wrong ids with a perfect score", Attempt{Kind: "practice", Correct: 8, Total: 8, Wrong: []string{"p1"}}, "wrong"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := e.svc.RecordAttempt(t.Context(), "u1", point, tt.a)
			var verr *ValidationError
			if !errors.As(err, &verr) || verr.Fields[tt.key] == "" {
				t.Fatalf("want error at %q, got %v", tt.key, err)
			}
		})
	}
	if _, err := e.progress.Get(t.Context(), "u1", point); !errors.Is(err, ErrNotFound) {
		t.Error("rejected attempts must not be recorded")
	}
}

func TestRecordAttemptNeedsAPublishedLesson(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	a := Attempt{Kind: "practice", Correct: 8, Total: 8}
	if _, _, err := e.svc.RecordAttempt(t.Context(), "u1", point, a); !errors.Is(err, ErrNotFound) {
		t.Errorf("no lesson: %v", err)
	}
	if _, _, err := e.svc.Generate(t.Context(), point, false); err != nil {
		t.Fatal(err)
	}
	if _, _, err := e.svc.RecordAttempt(t.Context(), "u1", point, a); !errors.Is(err, ErrNotFound) {
		t.Errorf("draft: %v", err)
	}
	if _, _, err := e.svc.RecordAttempt(t.Context(), "u1", "nope", a); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown point: %v", err)
	}
}

func TestMasteredCount(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	for _, p := range []Progress{
		{UserID: "u1", PointID: "old", Status: ProgressMastered, MasteredAt: testNow.AddDate(0, 0, -10)},
		{UserID: "u1", PointID: "new", Status: ProgressMastered, MasteredAt: testNow},
		{UserID: "u1", PointID: "learning", Status: ProgressLearning, UpdatedAt: testNow},
		{UserID: "u2", PointID: "other", Status: ProgressMastered, MasteredAt: testNow},
	} {
		if err := e.progress.Save(t.Context(), p); err != nil {
			t.Fatal(err)
		}
	}
	if n, err := e.svc.MasteredCount(t.Context(), "u1", nil); err != nil || n != 2 {
		t.Fatalf("all = %d, %v; want 2", n, err)
	}
	since := testNow.AddDate(0, 0, -1)
	if n, err := e.svc.MasteredCount(t.Context(), "u1", &since); err != nil || n != 1 {
		t.Fatalf("since = %d, %v; want 1", n, err)
	}
	if n, _ := e.svc.MasteredCount(t.Context(), "u1", &testNow); n != 1 {
		t.Fatalf("since exactly then = %d; want 1", n)
	}
}
