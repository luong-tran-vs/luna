package writing

import (
	"errors"
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/job"
)

// submitted returns a submitted, pending writing of u1 for l1 and its grade job.
func (e *env) submitted(t *testing.T) (Writing, job.Job) {
	t.Helper()
	w, err := e.svc.Submit(t.Context(), "u1", "l1", "My family have four people.")
	if err != nil {
		t.Fatal(err)
	}
	jobs := e.jobs.all()
	return w, jobs[len(jobs)-1]
}

func TestProcessGrade(t *testing.T) {
	t.Parallel()
	e := newEnv()
	w, j := e.submitted(t)

	if err := e.svc.ProcessGrade(t.Context(), j); err != nil {
		t.Fatal(err)
	}
	req := e.grader.req
	if e.grader.calls != 1 || req.Level != "A1" || req.LessonText != "Tom has a big family." ||
		req.Prompt != "Write about your family." || req.Text != "My family have four people." {
		t.Fatalf("AI request = %+v (%d calls)", req, e.grader.calls)
	}
	got, _ := e.repo.Get(t.Context(), w.ID)
	g := got.Grade
	if g == nil || g.Status != GradeDone || g.Seen || !g.GradedAt.Equal(e.now) || g.OverallVi != "Khá tốt." ||
		g.CorrectedText != "My family has four members." {
		t.Fatalf("grade = %+v", g)
	}
	names := make([]string, len(g.Criteria))
	for i, c := range g.Criteria {
		names[i] = c.Name
	}
	if strings.Join(names, ",") != "task,grammar,vocabulary,coherence" || g.Criteria[1].Score != 3 || g.Criteria[1].CommentVi != "Sai thì." {
		t.Fatalf("criteria = %+v", g.Criteria)
	}
	if a := Average(g); a == nil || *a != 3.8 {
		t.Fatalf("average = %v", a)
	}

	// A writing that is no longer pending is left alone.
	if err := e.svc.ProcessGrade(t.Context(), j); err != nil || e.grader.calls != 1 {
		t.Fatalf("second run: %v, %d calls", err, e.grader.calls)
	}
}

func TestProcessGradeErrors(t *testing.T) {
	t.Parallel()

	bad := func(mutate func(*ai.Grade)) ai.Grade {
		g := goodGrade()
		mutate(&g)
		return g
	}
	tests := []struct {
		name      string
		grade     ai.Grade
		err       error
		wantErr   error
		permanent bool
	}{
		{name: "score 0", grade: bad(func(g *ai.Grade) { g.Task.Score = 0 }), wantErr: ErrUnusableGrade},
		{name: "score 6", grade: bad(func(g *ai.Grade) { g.Coherence.Score = 6 }), wantErr: ErrUnusableGrade},
		{name: "empty comment", grade: bad(func(g *ai.Grade) { g.Grammar.CommentVi = " " }), wantErr: ErrUnusableGrade},
		{name: "empty correction", grade: bad(func(g *ai.Grade) { g.CorrectedText = "" }), wantErr: ErrUnusableGrade},
		{name: "quota", err: ai.ErrQuota, wantErr: ai.ErrQuota},
		{name: "not configured", err: ai.ErrNotConfigured, wantErr: ai.ErrNotConfigured, permanent: true},
		{name: "invalid key", err: ai.ErrInvalidKey, wantErr: ai.ErrInvalidKey, permanent: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := newEnv()
			e.grader.grade, e.grader.err = tt.grade, tt.err
			w, j := e.submitted(t)
			err := e.svc.ProcessGrade(t.Context(), j)
			if !errors.Is(err, tt.wantErr) || job.IsPermanent(err) != tt.permanent {
				t.Fatalf("err = %v (permanent %v), want %v (permanent %v)", err, job.IsPermanent(err), tt.wantErr, tt.permanent)
			}
			if got, _ := e.repo.Get(t.Context(), w.ID); got.Grade.Status != GradePending {
				t.Fatalf("grade = %+v, want still pending", got.Grade)
			}
		})
	}

	e := newEnv()
	if err := e.svc.ProcessGrade(t.Context(), job.Job{Type: job.TypeGrade, TargetID: "missing"}); !job.IsPermanent(err) {
		t.Fatalf("missing writing: %v", err)
	}
}

func TestCheckGradeCapsTexts(t *testing.T) {
	t.Parallel()
	g := goodGrade()
	g.CorrectedText = strings.Repeat("a", 5000)
	g.Task.CommentVi = "  " + strings.Repeat("b", 700)
	out, err := CheckGrade(g)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.CorrectedText) != 4000 || len([]rune(out.Criteria[0].CommentVi)) != 600 {
		t.Fatalf("lengths %d %d", len(out.CorrectedText), len([]rune(out.Criteria[0].CommentVi)))
	}
}

func TestJobFailedReasons(t *testing.T) {
	t.Parallel()
	for err, want := range map[error]string{
		ai.ErrQuota:           "AI hết lượt, vui lòng chấm lại sau",
		ai.ErrNotConfigured:   "AI chưa được cấu hình",
		ai.ErrInvalidKey:      "Khoá API của AI không hợp lệ",
		ErrUnusableGrade:      "AI trả về kết quả không dùng được",
		errors.New("timeout"): "Không chấm được bài",
	} {
		e := newEnv()
		w, j := e.submitted(t)
		e.svc.JobFailed(t.Context(), j, err)
		got, _ := e.repo.Get(t.Context(), w.ID)
		if got.Grade.Status != GradeFailed || got.Grade.Error != want || got.Grade.Seen || !got.Grade.GradedAt.Equal(e.now) {
			t.Errorf("%v: grade = %+v, want %q", err, got.Grade, want)
		}
		if got.Text != "My family have four people." || got.Status != StatusSubmitted {
			t.Errorf("%v: writing changed: %+v", err, got)
		}
	}
}

func TestRegrade(t *testing.T) {
	t.Parallel()
	e := newEnv()
	w, j := e.submitted(t)

	if _, err := e.svc.Regrade(t.Context(), "u1", w.ID); !errors.Is(err, ErrNotFailed) {
		t.Fatalf("pending: %v", err)
	}
	e.svc.JobFailed(t.Context(), j, ai.ErrQuota)
	if _, err := e.svc.Regrade(t.Context(), "u2", w.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other user: %v", err)
	}
	before := len(e.jobs.all())
	got, err := e.svc.Regrade(t.Context(), "u1", w.ID)
	if err != nil || got.Grade.Status != GradePending || !got.Grade.Seen || got.Grade.Error != "" {
		t.Fatalf("regrade = %+v, %v", got.Grade, err)
	}
	if jobs := e.jobs.all(); len(jobs) != before+1 || jobs[len(jobs)-1].TargetID != w.ID {
		t.Fatalf("jobs = %+v", jobs)
	}
	_ = e.svc.ProcessGrade(t.Context(), e.jobs.all()[len(e.jobs.all())-1])
	if _, err := e.svc.Regrade(t.Context(), "u1", w.ID); !errors.Is(err, ErrNotFailed) {
		t.Fatalf("done: %v", err)
	}
}

// lastJob is the last queued job.
func (e *env) lastJob() job.Job {
	jobs := e.jobs.all()
	return jobs[len(jobs)-1]
}

func TestResubmitUsesTheSecondGrading(t *testing.T) {
	t.Parallel()
	e := newEnv()
	w, j := e.submitted(t)
	if w.Gradings != 1 {
		t.Fatalf("first gradings = %d", w.Gradings)
	}
	if _, err := e.svc.Resubmit(t.Context(), "u1", w.ID, "My family has four people now."); !errors.Is(err, ErrGrading) {
		t.Fatalf("while grading: %v", err)
	}
	_ = e.svc.ProcessGrade(t.Context(), j)

	if _, err := e.svc.Resubmit(t.Context(), "u1", w.ID, "Too short"); err == nil {
		t.Fatal("short text accepted")
	}
	if _, err := e.svc.Resubmit(t.Context(), "u2", w.ID, "My family has four people now."); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other user: %v", err)
	}
	before := len(e.jobs.all())
	got, err := e.svc.Resubmit(t.Context(), "u1", w.ID, "  My family has four people now.  ")
	if err != nil || got.Gradings != 2 || got.Text != "My family has four people now." || got.Grade.Status != GradePending {
		t.Fatalf("resubmit = %+v, %v", got, err)
	}
	if len(e.jobs.all()) != before+1 || e.lastJob().TargetID != w.ID {
		t.Fatalf("jobs = %+v", e.jobs.all())
	}
	stored, _ := e.repo.Get(t.Context(), w.ID)
	if stored.Text != got.Text || stored.Gradings != 2 {
		t.Fatalf("stored = %+v", stored)
	}

	// The second grading was the last: no resubmit, no regrade after a failure.
	e.svc.JobFailed(t.Context(), e.lastJob(), ai.ErrQuota)
	if _, err := e.svc.Regrade(t.Context(), "u1", w.ID); !errors.Is(err, ErrNoGradings) {
		t.Fatalf("regrade after two gradings: %v", err)
	}
	if _, err := e.svc.Resubmit(t.Context(), "u1", w.ID, "My family has four people again."); !errors.Is(err, ErrNoGradings) {
		t.Fatalf("third submit: %v", err)
	}
}

func TestRegradeCountsAsTheSecondGrading(t *testing.T) {
	t.Parallel()
	e := newEnv()
	w, j := e.submitted(t)
	e.svc.JobFailed(t.Context(), j, ai.ErrQuota)
	got, err := e.svc.Regrade(t.Context(), "u1", w.ID)
	if err != nil || got.Gradings != 2 {
		t.Fatalf("regrade = %+v, %v", got, err)
	}
	_ = e.svc.ProcessGrade(t.Context(), e.lastJob())
	if _, err := e.svc.Resubmit(t.Context(), "u1", w.ID, "My family has four people now."); !errors.Is(err, ErrNoGradings) {
		t.Fatalf("resubmit after regrade: %v", err)
	}
}

func TestWritingsBeforeTheCountUsedOneGrading(t *testing.T) {
	t.Parallel()
	if got := GradingsUsed(Writing{Status: StatusSubmitted}); got != 1 {
		t.Fatalf("legacy = %d", got)
	}
	e := newEnv()
	w, j := e.submitted(t)
	_ = e.svc.ProcessGrade(t.Context(), j)
	_, _ = e.repo.update(w.ID, func(w *Writing) { w.Gradings = 0 })
	if got, err := e.svc.Resubmit(t.Context(), "u1", w.ID, "My family has four people now."); err != nil || got.Gradings != 2 {
		t.Fatalf("legacy resubmit = %+v, %v", got, err)
	}
}
