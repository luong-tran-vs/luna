package writing

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/job"
)

func words(n int) string { return strings.TrimSpace(strings.Repeat("word ", n)) }

func fieldMsg(t *testing.T, err error, field string) string {
	t.Helper()
	var verr *ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("err = %v, want ValidationError", err)
	}
	return verr.Fields[field]
}

func TestGetLessonWriting(t *testing.T) {
	t.Parallel()
	e := newEnv()

	v, err := e.svc.Get(t.Context(), "u1", "l1")
	if err != nil || v.Prompt != "Write about your family." || v.Level != "A1" || !v.CanWrite || v.Writing != nil {
		t.Fatalf("l1 = %+v, %v", v, err)
	}
	v, err = e.svc.Get(t.Context(), "u1", "l2")
	if err != nil || v.Prompt != DefaultPrompt || v.CanWrite {
		t.Fatalf("l2 = %+v, %v", v, err)
	}
	if _, err := e.svc.Get(t.Context(), "u1", "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing lesson: %v", err)
	}
}

func TestSaveDraft(t *testing.T) {
	t.Parallel()
	e := newEnv()

	if _, err := e.svc.SaveDraft(t.Context(), "u1", "l1", "My family"); err != nil {
		t.Fatal(err)
	}
	w, err := e.svc.SaveDraft(t.Context(), "u1", "l1", "My family has four people")
	if err != nil || w.Text != "My family has four people" || w.Status != StatusDraft {
		t.Fatalf("draft = %+v, %v", w, err)
	}
	if v, _ := e.svc.Get(t.Context(), "u1", "l1"); v.Writing == nil || v.Writing.Text != "My family has four people" {
		t.Fatalf("reloaded = %+v", v.Writing)
	}
	if _, err := e.svc.SaveDraft(t.Context(), "u1", "l1", ""); err != nil {
		t.Fatalf("empty draft: %v", err)
	}
	if msg := fieldMsg(t, func() error {
		_, err := e.svc.SaveDraft(t.Context(), "u1", "l1", strings.Repeat("a", 4001))
		return err
	}(), "text"); msg == "" {
		t.Fatal("no message for a long draft")
	}
	if _, err := e.svc.SaveDraft(t.Context(), "u1", "l2", "x"); !errors.Is(err, ErrLocked) {
		t.Fatalf("locked: %v", err)
	}
	// Another learner's draft is separate.
	if v, _ := e.svc.Get(t.Context(), "u2", "l1"); v.Writing != nil {
		t.Fatalf("u2 sees u1's draft: %+v", v.Writing)
	}
}

func TestSubmitLimits(t *testing.T) {
	t.Parallel()
	for n, want := range map[int]string{4: "Bài viết cần ít nhất 5 từ", 401: "Bài viết tối đa 400 từ"} {
		e := newEnv()
		_, err := e.svc.Submit(t.Context(), "u1", "l1", words(n))
		if msg := fieldMsg(t, err, "text"); msg != want {
			t.Errorf("%d words: %q, want %q", n, msg, want)
		}
	}
	for _, n := range []int{5, 400} {
		e := newEnv()
		if _, err := e.svc.Submit(t.Context(), "u1", "l1", words(n)); err != nil {
			t.Errorf("%d words: %v", n, err)
		}
	}
}

func TestSubmit(t *testing.T) {
	t.Parallel()
	e := newEnv()
	_, _ = e.svc.SaveDraft(t.Context(), "u1", "l1", "draft")

	w, err := e.svc.Submit(t.Context(), "u1", "l1", "  My family has four people.  ")
	if err != nil {
		t.Fatal(err)
	}
	if w.Status != StatusSubmitted || w.Text != "My family has four people." || w.Prompt != "Write about your family." ||
		w.LessonTitle != "My family" || w.LessonRevision != 2 || !w.SubmittedAt.Equal(e.now) {
		t.Fatalf("submitted = %+v", w)
	}
	if w.Grade == nil || w.Grade.Status != GradePending {
		t.Fatalf("grade = %+v", w.Grade)
	}
	jobs := e.jobs.all()
	if len(jobs) != 1 || jobs[0].Type != job.TypeGrade || jobs[0].TargetID != w.ID || jobs[0].LessonID != "" || e.notify != 1 {
		t.Fatalf("jobs = %+v notify %d", jobs, e.notify)
	}

	if _, err := e.svc.Submit(t.Context(), "u1", "l1", words(10)); !errors.Is(err, ErrSubmitted) {
		t.Fatalf("second submit: %v", err)
	}
	if _, err := e.svc.SaveDraft(t.Context(), "u1", "l1", "change"); !errors.Is(err, ErrSubmitted) {
		t.Fatalf("draft after submit: %v", err)
	}
	if len(e.jobs.all()) != 1 {
		t.Fatalf("jobs after second submit = %d", len(e.jobs.all()))
	}
}

func TestSubmitDefaultPromptAndLock(t *testing.T) {
	t.Parallel()
	e := newEnv()
	if _, err := e.svc.Submit(t.Context(), "u1", "l2", words(10)); !errors.Is(err, ErrLocked) {
		t.Fatalf("locked: %v", err)
	}
	e.steps.set("u1", "l2", true)
	w, err := e.svc.Submit(t.Context(), "u1", "l2", words(10))
	if err != nil || w.Prompt != DefaultPrompt {
		t.Fatalf("default prompt = %+v, %v", w, err)
	}
}

func TestSubmittedPort(t *testing.T) {
	t.Parallel()
	e := newEnv()
	if ok, _ := e.svc.Submitted(t.Context(), "u1", "l1"); ok {
		t.Fatal("submitted before submitting")
	}
	_, _ = e.svc.SaveDraft(t.Context(), "u1", "l1", "draft")
	if ok, _ := e.svc.Submitted(t.Context(), "u1", "l1"); ok {
		t.Fatal("a draft counts as submitted")
	}
	_, _ = e.svc.Submit(t.Context(), "u1", "l1", words(10))
	if ok, _ := e.svc.Submitted(t.Context(), "u1", "l1"); !ok {
		t.Fatal("not submitted after submitting")
	}
}

// --- US2: list, detail, seen, unseen ---

func TestListDetailSeenAndUnseen(t *testing.T) {
	t.Parallel()
	e := newEnv()
	w1, j1 := e.submitted(t)
	e.steps.set("u1", "l2", true)
	e.now = e.now.Add(24 * time.Hour)
	w2, err := e.svc.Submit(t.Context(), "u1", "l2", words(10))
	if err != nil {
		t.Fatal(err)
	}
	_, _ = e.svc.SaveDraft(t.Context(), "u2", "l1", "x") // locked for u2
	e.steps.set("u2", "l1", true)
	_, _ = e.svc.SaveDraft(t.Context(), "u2", "l1", "draft of u2")

	if c, _ := e.svc.Unseen(t.Context(), "u1"); c.Unseen != 0 || c.Pending != 2 || c.Latest != nil {
		t.Fatalf("pending = %+v", c)
	}
	_ = e.svc.ProcessGrade(t.Context(), j1)

	list, err := e.svc.List(t.Context(), "u1")
	if err != nil || len(list) != 2 || list[0].ID != w2.ID || list[1].ID != w1.ID {
		t.Fatalf("list = %+v, %v", list, err)
	}
	if list[1].GradeStatus != GradeDone || list[1].Average == nil || *list[1].Average != 3.8 || list[1].Seen ||
		list[1].LessonTitle != "My family" || list[0].GradeStatus != GradePending || list[0].Average != nil {
		t.Fatalf("summaries = %+v %+v", list[0], list[1])
	}
	if l, _ := e.svc.List(t.Context(), "u2"); len(l) != 0 {
		t.Fatalf("u2 list (only a draft) = %+v", l)
	}

	c, _ := e.svc.Unseen(t.Context(), "u1")
	if c.Unseen != 1 || c.Pending != 1 || c.Latest == nil || c.Latest.ID != w1.ID || c.Latest.Status != GradeDone {
		t.Fatalf("unseen = %+v", c)
	}

	if _, err := e.svc.Detail(t.Context(), "u2", w1.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("u2 opens u1's writing: %v", err)
	}
	if err := e.svc.MarkSeen(t.Context(), "u2", w1.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("u2 marks u1's writing: %v", err)
	}
	d, err := e.svc.Detail(t.Context(), "u1", w1.ID)
	if err != nil || d.Grade.Status != GradeDone {
		t.Fatalf("detail = %+v, %v", d, err)
	}
	if err := e.svc.MarkSeen(t.Context(), "u1", w1.ID); err != nil {
		t.Fatal(err)
	}
	if c, _ := e.svc.Unseen(t.Context(), "u1"); c.Unseen != 0 || c.Latest != nil {
		t.Fatalf("after seen = %+v", c)
	}
}

func TestSubmittedBeatsLocked(t *testing.T) {
	t.Parallel()
	e := newEnv()
	_, _ = e.svc.Submit(t.Context(), "u1", "l1", words(10))
	e.steps.set("u1", "l1", false) // the lesson is over
	if _, err := e.svc.Submit(t.Context(), "u1", "l1", words(10)); !errors.Is(err, ErrSubmitted) {
		t.Fatalf("submit after the lesson: %v", err)
	}
	if _, err := e.svc.SaveDraft(t.Context(), "u1", "l1", "x"); !errors.Is(err, ErrSubmitted) {
		t.Fatalf("draft after the lesson: %v", err)
	}
}
