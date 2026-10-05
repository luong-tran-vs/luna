package mysql

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/export"
	"github.com/luongtran/luna/backend/internal/grammar"
)

func sampleGrammarLesson() grammar.Lesson {
	t := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	return grammar.Lesson{
		PointID: "a1-to-be", Status: grammar.StatusPublished, Edited: true,
		CreatedAt: t, UpdatedAt: t.Add(time.Hour), PublishedAt: t.Add(2 * time.Hour),
		CheckedAt: t.Add(90 * time.Minute), VerifiedAt: t.Add(100 * time.Minute),
		Checks: []grammar.Check{
			{ExerciseID: "p1", Kind: grammar.CheckMismatch, NoteVi: "AI chọn đáp án khác"},
			{ExerciseID: "m1", Kind: grammar.CheckAmbiguous, NoteVi: "mơ hồ", Confirmed: true},
		},
		Content: grammar.Content{
			Objective:   "Bạn có thể dùng to be.",
			Explanation: []string{"một", "hai"},
			Usage:       []string{"dùng khi …"},
			Structures:  []grammar.Structure{{Label: "Khẳng định", Pattern: "S + am/is/are", Example: "I am here."}},
			Examples:    []grammar.Example{{En: "I am a student.", Vi: "Tôi là học sinh."}},
			Mistakes:    []grammar.Mistake{{Wrong: "She are", Right: "She is", NoteVi: "chia động từ"}},
			Practice: []grammar.Exercise{
				{ID: "p1", Kind: grammar.KindChoice, PromptVi: "Chọn", Text: "She ___ a teacher.",
					Options: []string{"am", "is", "are", "be"}, AnswerIndex: 1, ExplanationVi: "vì she"},
				{ID: "p2", Kind: grammar.KindFill, Text: "I ___ ok.", Answers: []string{"am"}, ExplanationVi: "vì I"},
			},
			Mastery: []grammar.Exercise{
				{ID: "m1", Kind: grammar.KindReorder, Text: "Tôi là học sinh.", Sentence: "I am a student",
					Words: []string{"I", "am", "a", "student", "is"}, ExplanationVi: "trật tự"},
			},
		},
	}
}

func TestGrammarLessonsSaveGetList(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	repo := NewGrammarLessons(testDB(t))

	if _, err := repo.Get(ctx, "a1-to-be"); !errors.Is(err, grammar.ErrNotFound) {
		t.Fatalf("missing Get err = %v", err)
	}
	if got, err := repo.List(ctx); err != nil || got == nil || len(got) != 0 {
		t.Fatalf("empty List = %v, %v", got, err)
	}

	in := sampleGrammarLesson()
	if err := repo.Save(ctx, in); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Get(ctx, in.PointID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, in) {
		t.Fatalf("round trip:\n got %+v\nwant %+v", got, in)
	}

	// A draft: no published time, empty lists read back non-nil.
	draft := grammar.Lesson{PointID: "a1-articles", Status: grammar.StatusDraft, CreatedAt: in.CreatedAt, UpdatedAt: in.UpdatedAt}
	if err := repo.Save(ctx, draft); err != nil {
		t.Fatal(err)
	}
	d, err := repo.Get(ctx, "a1-articles")
	if err != nil || !d.PublishedAt.IsZero() || d.Status != grammar.StatusDraft || d.Content.Practice == nil || d.Content.Explanation == nil {
		t.Fatalf("draft = %+v, %v", d, err)
	}
	if d.Checks == nil || len(d.Checks) != 0 || !d.CheckedAt.IsZero() {
		t.Fatalf("unchecked draft: checks = %#v, checkedAt = %v", d.Checks, d.CheckedAt)
	}
	// List reports the flags and whether the check ran.
	list0, err := repo.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range list0 {
		switch s.PointID {
		case "a1-to-be":
			if s.Flags != 1 || !s.Checked || !s.Verified {
				t.Fatalf("checked summary = %+v", s)
			}
		case "a1-articles":
			if s.Flags != 0 || s.Checked || s.Verified {
				t.Fatalf("unchecked summary = %+v", s)
			}
		}
	}
	// A check that ran and found nothing: checked, no flags, empty array.
	clean := draft
	clean.PointID, clean.CheckedAt = "a1-clean", in.CheckedAt
	if err := repo.Save(ctx, clean); err != nil {
		t.Fatal(err)
	}
	if c, err := repo.Get(ctx, "a1-clean"); err != nil || c.Checks == nil || len(c.Checks) != 0 || !c.CheckedAt.Equal(in.CheckedAt) {
		t.Fatalf("clean = %+v, %v", c, err)
	}

	// Save replaces: edit, unpublish.
	in.Status, in.Edited, in.PublishedAt, in.Content.Objective = grammar.StatusDraft, false, time.Time{}, "Mới"
	in.UpdatedAt = in.UpdatedAt.Add(time.Minute)
	if err := repo.Save(ctx, in); err != nil {
		t.Fatal(err)
	}
	got, err = repo.Get(ctx, in.PointID)
	if err != nil || !reflect.DeepEqual(got, in) {
		t.Fatalf("after replace: %+v, %v", got, err)
	}

	list, err := repo.List(ctx)
	if err != nil || len(list) != 3 {
		t.Fatalf("List = %v, %v", list, err)
	}
	byID := map[string]grammar.LessonSummary{}
	for _, s := range list {
		byID[s.PointID] = s
	}
	if s := byID["a1-to-be"]; s.Status != grammar.StatusDraft || s.Edited || !s.UpdatedAt.Equal(in.UpdatedAt) || !s.PublishedAt.IsZero() {
		t.Fatalf("summary = %+v", s)
	}
}

func TestGrammarProgressSaveGetList(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	repo := NewGrammarProgress(testDB(t))
	u, other := newID(), newID()

	if _, err := repo.Get(ctx, u, "a1-to-be"); !errors.Is(err, grammar.ErrNotFound) {
		t.Fatalf("missing Get err = %v", err)
	}
	if got, err := repo.List(ctx, u); err != nil || got == nil || len(got) != 0 {
		t.Fatalf("empty List = %v, %v", got, err)
	}

	t0 := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	p := grammar.Progress{
		UserID: u, PointID: "a1-to-be", Status: grammar.ProgressLearning, PracticeAttempts: 2, LastPractice: 50,
		MasteryAttempts: 1, BestMastery: 60, Weak: []string{"p2", "p4"}, UpdatedAt: t0,
	}
	if err := repo.Save(ctx, p); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Get(ctx, u, "a1-to-be")
	if err != nil || !reflect.DeepEqual(got, p) {
		t.Fatalf("Get = %+v, %v\nwant %+v", got, err, p)
	}

	// Upsert: mastered, no weak items.
	p.Status, p.MasteryAttempts, p.BestMastery, p.MasteredAt, p.Weak, p.UpdatedAt = grammar.ProgressMastered, 2, 90, t0.Add(time.Hour), nil, t0.Add(time.Hour)
	if err := repo.Save(ctx, p); err != nil {
		t.Fatal(err)
	}
	got, err = repo.Get(ctx, u, "a1-to-be")
	if err != nil || got.Status != grammar.ProgressMastered || !got.MasteredAt.Equal(p.MasteredAt) || got.Weak == nil || len(got.Weak) != 0 || got.BestMastery != 90 {
		t.Fatalf("after upsert = %+v, %v", got, err)
	}

	if err := repo.Save(ctx, grammar.Progress{UserID: u, PointID: "a1-articles", Status: grammar.ProgressLearning, UpdatedAt: t0}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(ctx, grammar.Progress{UserID: other, PointID: "a1-to-be", Status: grammar.ProgressLearning, UpdatedAt: t0}); err != nil {
		t.Fatal(err)
	}
	list, err := repo.List(ctx, u)
	if err != nil || len(list) != 2 || list[0].PointID != "a1-articles" || list[1].PointID != "a1-to-be" {
		t.Fatalf("List = %+v, %v", list, err)
	}
	if _, err := repo.Get(ctx, other, "a1-articles"); !errors.Is(err, grammar.ErrNotFound) {
		t.Fatalf("other learner Get err = %v", err)
	}
}

func TestExportGrammarProgress(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	db := testDB(t)
	u := newID()
	t0 := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	repo := NewGrammarProgress(db)
	for _, id := range []string{"b1-x", "a1-y"} {
		if err := repo.Save(ctx, grammar.Progress{UserID: u, PointID: id, Status: grammar.ProgressLearning, Weak: []string{"p1"}, UpdatedAt: t0}); err != nil {
			t.Fatal(err)
		}
	}
	docs, err := NewExport(db).UserDocs(ctx, export.CollGrammarProgress, u)
	if err != nil || len(docs) != 2 {
		t.Fatalf("UserDocs = %v, %v", docs, err)
	}
	d := docs[0]
	if d["pointId"] != "a1-y" || d["userId"] != u || d["practiceAttempts"] != int64(0) || d["updatedAt"] != "2026-10-01T08:00:00Z" {
		t.Fatalf("doc = %v", d)
	}
	if w, ok := d["weak"].([]any); !ok || len(w) != 1 || w[0] != "p1" {
		t.Fatalf("weak = %v", d["weak"])
	}
	if _, has := d["masteredAt"]; has {
		t.Fatalf("NULL masteredAt must leave its key out: %v", d)
	}
}
