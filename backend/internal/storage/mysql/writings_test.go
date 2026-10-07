package mysql

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/writing"
)

var writingsT0 = time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)

func writingsIn(u, l, text string, at time.Time) writing.Writing {
	return writing.Writing{UserID: u, LessonID: l, LessonRevision: 2, LessonTitle: "My family",
		Prompt: "Write about your family.", Text: text, SubmittedAt: at}
}

func writingsDone(seen bool) writing.Grade {
	return writing.Grade{
		Status: writing.GradeDone, OverallVi: "Khá tốt.", CorrectedText: "My family has four members.",
		GradedAt: writingsT0.Add(time.Hour), Seen: seen,
		Criteria: []writing.Criterion{
			{Name: writing.CriterionTask, Score: 4, CommentVi: "Đúng đề."},
			{Name: writing.CriterionGrammar, Score: 3, CommentVi: "Sai thì."},
			{Name: writing.CriterionVocabulary, Score: 4, CommentVi: "Đủ từ."},
			{Name: writing.CriterionCoherence, Score: 4, CommentVi: "Mạch lạc."},
		},
	}
}

func TestWritingsDraftLifecycle(t *testing.T) {
	t.Parallel()
	r := NewWritings(testDB(t))
	ctx := t.Context()

	if _, ok, err := r.GetByLesson(ctx, "u1", "l1"); err != nil || ok {
		t.Fatalf("GetByLesson empty = %v, %v", ok, err)
	}
	w, err := r.SaveDraft(ctx, "u1", "l1", "Hello", writingsT0)
	if err != nil {
		t.Fatal(err)
	}
	if len(w.ID) != 24 || w.Status != writing.StatusDraft || w.Text != "Hello" || w.Grade != nil ||
		!w.CreatedAt.Equal(writingsT0) || !w.UpdatedAt.Equal(writingsT0) || !w.SubmittedAt.IsZero() {
		t.Fatalf("draft = %+v", w)
	}
	w2, err := r.SaveDraft(ctx, "u1", "l1", "Hello world", writingsT0.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if w2.ID != w.ID || w2.Text != "Hello world" || !w2.CreatedAt.Equal(writingsT0) || !w2.UpdatedAt.Equal(writingsT0.Add(time.Minute)) {
		t.Fatalf("second draft = %+v", w2)
	}
	got, ok, err := r.GetByLesson(ctx, "u1", "l1")
	if err != nil || !ok || got.ID != w.ID || got.Text != "Hello world" {
		t.Fatalf("GetByLesson = %+v %v %v", got, ok, err)
	}
	// another learner and another lesson are separate writings
	o, err := r.SaveDraft(ctx, "u2", "l1", "x", writingsT0)
	if err != nil || o.ID == w.ID {
		t.Fatalf("other learner = %+v %v", o, err)
	}
	if _, ok, _ := r.GetByLesson(ctx, "u1", "l2"); ok {
		t.Fatal("lesson l2 has a writing")
	}
	g, err := r.Get(ctx, w.ID)
	if err != nil || g.ID != w.ID {
		t.Fatalf("Get = %+v %v", g, err)
	}
	if _, err := r.Get(ctx, "nope"); !errors.Is(err, writing.ErrNotFound) {
		t.Fatalf("Get unknown = %v", err)
	}
	if _, err := r.Get(ctx, newID()); !errors.Is(err, writing.ErrNotFound) {
		t.Fatalf("Get missing = %v", err)
	}
}

func TestWritingsSubmitFromDraftAndOnce(t *testing.T) {
	t.Parallel()
	r := NewWritings(testDB(t))
	ctx := t.Context()

	d, err := r.SaveDraft(ctx, "u1", "l1", "draft", writingsT0)
	if err != nil {
		t.Fatal(err)
	}
	at := writingsT0.Add(time.Hour)
	w, err := r.Submit(ctx, writingsIn("u1", "l1", "final text", at))
	if err != nil {
		t.Fatal(err)
	}
	if w.ID != d.ID || w.Status != writing.StatusSubmitted || w.Text != "final text" || w.LessonRevision != 2 ||
		w.LessonTitle != "My family" || w.Prompt != "Write about your family." ||
		!w.SubmittedAt.Equal(at) || !w.UpdatedAt.Equal(at) || !w.CreatedAt.Equal(writingsT0) {
		t.Fatalf("submitted = %+v", w)
	}
	if w.Grade == nil || w.Grade.Status != writing.GradePending || !w.Grade.Seen || len(w.Grade.Criteria) != 0 || !w.Grade.GradedAt.IsZero() {
		t.Fatalf("grade = %+v", w.Grade)
	}
	if _, err := r.Submit(ctx, writingsIn("u1", "l1", "again", at.Add(time.Hour))); !errors.Is(err, writing.ErrSubmitted) {
		t.Fatalf("second Submit = %v", err)
	}
	if _, err := r.SaveDraft(ctx, "u1", "l1", "edit", at.Add(time.Hour)); !errors.Is(err, writing.ErrSubmitted) {
		t.Fatalf("SaveDraft after Submit = %v", err)
	}
	got, _, _ := r.GetByLesson(ctx, "u1", "l1")
	if got.Text != "final text" || !got.UpdatedAt.Equal(at) {
		t.Fatalf("changed after refused writes: %+v", got)
	}
}

func TestWritingsSubmitWithoutDraft(t *testing.T) {
	t.Parallel()
	r := NewWritings(testDB(t))
	at := writingsT0.Add(time.Hour)
	w, err := r.Submit(t.Context(), writingsIn("u1", "l1", "straight in", at))
	if err != nil {
		t.Fatal(err)
	}
	if len(w.ID) != 24 || w.Status != writing.StatusSubmitted || !w.CreatedAt.Equal(at) || !w.SubmittedAt.Equal(at) {
		t.Fatalf("submitted = %+v", w)
	}
}

func TestWritingsSubmitConcurrentOnlyOnce(t *testing.T) {
	t.Parallel()
	r := NewWritings(testDB(t))
	var wg sync.WaitGroup
	results := make([]error, 6)
	for i := range results {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, results[i] = r.Submit(t.Context(), writingsIn("u1", "l1", "text", writingsT0.Add(time.Duration(i)*time.Second)))
		}()
	}
	wg.Wait()
	okCount := 0
	for _, err := range results {
		switch {
		case err == nil:
			okCount++
		case !errors.Is(err, writing.ErrSubmitted):
			t.Fatalf("unexpected error %v", err)
		}
	}
	if okCount != 1 {
		t.Fatalf("%d submits succeeded, want 1", okCount)
	}
}

func TestWritingsSetGradeAndMarkSeen(t *testing.T) {
	t.Parallel()
	r := NewWritings(testDB(t))
	ctx := t.Context()
	w, err := r.Submit(ctx, writingsIn("u1", "l1", "text", writingsT0))
	if err != nil {
		t.Fatal(err)
	}
	if err := r.SetGrade(ctx, w.ID, writingsDone(false)); err != nil {
		t.Fatal(err)
	}
	got, _ := r.Get(ctx, w.ID)
	want := writingsDone(false)
	if got.Grade == nil || got.Grade.Status != want.Status || got.Grade.OverallVi != want.OverallVi ||
		got.Grade.CorrectedText != want.CorrectedText || got.Grade.Seen || !got.Grade.GradedAt.Equal(want.GradedAt) ||
		len(got.Grade.Criteria) != 4 || got.Grade.Criteria[1] != want.Criteria[1] || got.Grade.Criteria[3] != want.Criteria[3] {
		t.Fatalf("grade = %+v", got.Grade)
	}
	if err := r.MarkSeen(ctx, w.ID); err != nil {
		t.Fatal(err)
	}
	got, _ = r.Get(ctx, w.ID)
	if !got.Grade.Seen || got.Grade.Status != writing.GradeDone {
		t.Fatalf("after MarkSeen = %+v", got.Grade)
	}
	// a failed grade with an error and no criteria
	if err := r.SetGrade(ctx, w.ID, writing.Grade{Status: writing.GradeFailed, Error: "AI down"}); err != nil {
		t.Fatal(err)
	}
	got, _ = r.Get(ctx, w.ID)
	if got.Grade.Status != writing.GradeFailed || got.Grade.Error != "AI down" || got.Grade.Seen || len(got.Grade.Criteria) != 0 {
		t.Fatalf("failed grade = %+v", got.Grade)
	}
	for _, id := range []string{newID(), "bad"} {
		if err := r.SetGrade(ctx, id, writingsDone(false)); !errors.Is(err, writing.ErrNotFound) {
			t.Fatalf("SetGrade(%q) = %v", id, err)
		}
		if err := r.MarkSeen(ctx, id); !errors.Is(err, writing.ErrNotFound) {
			t.Fatalf("MarkSeen(%q) = %v", id, err)
		}
	}
}

func TestWritingsListNewestFirstWithoutText(t *testing.T) {
	t.Parallel()
	r := NewWritings(testDB(t))
	ctx := t.Context()
	if _, err := r.SaveDraft(ctx, "u1", "draft-lesson", "just a draft", writingsT0); err != nil {
		t.Fatal(err)
	}
	for i, l := range []string{"l1", "l2", "l3"} {
		if _, err := r.Submit(ctx, writingsIn("u1", l, "text "+l, writingsT0.Add(time.Duration(i)*time.Hour))); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := r.Submit(ctx, writingsIn("u2", "l1", "other", writingsT0.Add(9*time.Hour))); err != nil {
		t.Fatal(err)
	}
	list, err := r.List(ctx, "u1")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 || list[0].LessonID != "l3" || list[1].LessonID != "l2" || list[2].LessonID != "l1" {
		t.Fatalf("list = %+v", list)
	}
	for _, w := range list {
		if w.Text != "" || w.Grade == nil || w.LessonTitle != "My family" {
			t.Fatalf("list item = %+v", w)
		}
	}
	if none, err := r.List(ctx, "nobody"); err != nil || none == nil || len(none) != 0 {
		t.Fatalf("empty list = %v, %v", none, err)
	}
}

func TestWritingsUnseenAndStats(t *testing.T) {
	t.Parallel()
	r := NewWritings(testDB(t))
	ctx := t.Context()

	if c, err := r.Unseen(ctx, "u1"); err != nil || c.Unseen != 0 || c.Pending != 0 || c.Latest != nil {
		t.Fatalf("empty Unseen = %+v %v", c, err)
	}
	if n, avg, err := r.Stats(ctx, "u1", nil); err != nil || n != 0 || avg != nil {
		t.Fatalf("empty Stats = %d %v %v", n, avg, err)
	}
	if _, err := r.SaveDraft(ctx, "u1", "l0", "draft", writingsT0); err != nil {
		t.Fatal(err)
	}
	ids := map[string]string{}
	for i, l := range []string{"l1", "l2", "l3", "l4"} {
		w, err := r.Submit(ctx, writingsIn("u1", l, "text", writingsT0.Add(time.Duration(i)*time.Hour)))
		if err != nil {
			t.Fatal(err)
		}
		ids[l] = w.ID
	}
	// l1 done+seen, l2 done+unseen (older), l3 failed+unseen (newer), l4 pending
	g1 := writingsDone(true)
	g2 := writingsDone(false)
	g2.GradedAt = writingsT0.Add(2 * time.Hour)
	g2.Criteria[0].Score = 2 // mean 3.25
	g3 := writing.Grade{Status: writing.GradeFailed, Error: "x", GradedAt: writingsT0.Add(3 * time.Hour)}
	for l, g := range map[string]writing.Grade{"l1": g1, "l2": g2, "l3": g3} {
		if err := r.SetGrade(ctx, ids[l], g); err != nil {
			t.Fatal(err)
		}
	}
	c, err := r.Unseen(ctx, "u1")
	if err != nil {
		t.Fatal(err)
	}
	if c.Pending != 1 || c.Unseen != 2 || c.Latest == nil || c.Latest.ID != ids["l3"] || c.Latest.Status != writing.GradeFailed {
		t.Fatalf("Unseen = %+v", c)
	}
	if err := r.MarkSeen(ctx, ids["l3"]); err != nil {
		t.Fatal(err)
	}
	c, _ = r.Unseen(ctx, "u1")
	if c.Unseen != 1 || c.Latest == nil || c.Latest.ID != ids["l2"] || c.Latest.Status != writing.GradeDone {
		t.Fatalf("Unseen after seen = %+v", c)
	}
	if other, _ := r.Unseen(ctx, "u2"); other.Unseen != 0 || other.Pending != 0 {
		t.Fatalf("other learner Unseen = %+v", other)
	}

	n, avg, err := r.Stats(ctx, "u1", nil)
	if err != nil || n != 4 || avg == nil {
		t.Fatalf("Stats = %d %v %v", n, avg, err)
	}
	// mean of 3.75 (l1) and 3.25 (l2)
	if *avg < 3.499 || *avg > 3.501 {
		t.Fatalf("average = %v, want 3.5", *avg)
	}
	// since l2's submission: l2, l3 and l4, averaged over l2 alone
	since := writingsT0.Add(time.Hour)
	if n, avg, err := r.Stats(ctx, "u1", &since); err != nil || n != 3 || avg == nil || *avg < 3.249 || *avg > 3.251 {
		t.Fatalf("Stats since = %d %v %v", n, avg, err)
	}
	// only pending and failed: submitted counted, no average
	if _, err := r.Submit(ctx, writingsIn("u3", "l1", "t", writingsT0)); err != nil {
		t.Fatal(err)
	}
	if n, avg, err := r.Stats(ctx, "u3", nil); err != nil || n != 1 || avg != nil {
		t.Fatalf("pending Stats = %d %v %v", n, avg, err)
	}
}
