package mysql

import (
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/auth"
	"github.com/luongtran/luna/backend/internal/export"
	"github.com/luongtran/luna/backend/internal/lesson"
	"github.com/luongtran/luna/backend/internal/vocab"
)

func TestExportKey(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]string{
		"id": "_id", "user_id": "userId", "lesson_revision": "lessonRevision", "grade_seen": "gradeSeen", "text": "text",
	} {
		if got := exportKey(in); got != want {
			t.Errorf("exportKey(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestExportOrderCoversEveryAllowedTable(t *testing.T) {
	t.Parallel()
	for c := range exportOrder {
		if !export.Allowed(c) {
			t.Errorf("%s is ordered but not allowed", c)
		}
	}
	if len(exportOrder) != 9 {
		t.Errorf("exportOrder has %d tables, export.Allowed has 9", len(exportOrder))
	}
}

func TestExportAccountAndDocs(t *testing.T) {
	t.Parallel()
	db := testDB(t)
	ctx := t.Context()
	r := NewExport(db)

	created := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	u, err := NewUsers(db).Create(ctx, auth.User{Email: "a@b.c", PasswordHash: "secret-hash", Role: auth.RoleLearner, CreatedAt: created})
	if err != nil {
		t.Fatal(err)
	}

	acc, err := r.Account(ctx, u.ID)
	if err != nil || acc.ID != u.ID || acc.Email != "a@b.c" || acc.Role != "learner" || !acc.CreatedAt.Equal(created) {
		t.Fatalf("Account = %+v, %v", acc, err)
	}
	if _, err := r.Account(ctx, newID()); err != export.ErrNotFound {
		t.Fatalf("unknown account err = %v", err)
	}

	lessonID := newID()
	cards := NewCards(db)
	for _, lemma := range []string{"go", "run"} {
		if _, err := cards.Create(ctx, vocab.Card{
			UserID: u.ID, Text: lemma, Lemma: lemma, MeaningVi: "nghĩa " + lemma, LessonID: lessonID,
			Source: vocab.SourceAI, CreatedAt: created,
			Schedule: vocab.Schedule{Due: created.Add(time.Hour), Reps: 2, State: vocab.StateReview},
		}); err != nil {
			t.Fatal(err)
		}
	}
	other := vocab.Card{UserID: newID(), Text: "other", Lemma: "other", MeaningVi: "x", Source: vocab.SourceManual, CreatedAt: created}
	if _, err := cards.Create(ctx, other); err != nil {
		t.Fatal(err)
	}

	docs, err := r.UserDocs(ctx, export.CollCards, u.ID)
	if err != nil || len(docs) != 2 {
		t.Fatalf("UserDocs = %v, %v", docs, err)
	}
	d := docs[0]
	if d["lemma"] != "go" || d["userId"] != u.ID || d["lessonId"] != lessonID || d["meaningVi"] != "nghĩa go" {
		t.Fatalf("doc = %v", d)
	}
	if d["_id"] == nil || d["id"] != nil || d["user_id"] != nil {
		t.Fatalf("keys not converted: %v", d)
	}
	if d["createdAt"] != "2026-03-04T05:06:07Z" || d["due"] != "2026-03-04T06:06:07Z" {
		t.Fatalf("times: createdAt=%v due=%v", d["createdAt"], d["due"])
	}
	if _, has := d["lastReview"]; has {
		t.Fatalf("a NULL column must leave its key out: %v", d)
	}
	if docs[1]["lemma"] != "run" {
		t.Fatalf("not oldest first: %v", docs[1]["lemma"])
	}

	// Every allowed table can be read (empty here); anything else cannot.
	for _, c := range []string{
		export.CollReviewLogs, export.CollGoals, export.CollLessonProgress, export.CollStudyDays,
		export.CollDictationResults, export.CollReadingAnswers, export.CollWritings, export.CollGrammarProgress,
	} {
		got, err := r.UserDocs(ctx, c, u.ID)
		if err != nil || got == nil || len(got) != 0 {
			t.Errorf("UserDocs(%s) = %v, %v", c, got, err)
		}
	}
	for _, c := range []string{"users", "sessions", "lessons", "cards; DROP TABLE cards", ""} {
		if _, err := r.UserDocs(ctx, c, u.ID); err != export.ErrCollection {
			t.Errorf("UserDocs(%q) err = %v, want ErrCollection", c, err)
		}
	}
}

func TestExportLessonsAndFlags(t *testing.T) {
	t.Parallel()
	db := testDB(t)
	ctx := t.Context()
	r := NewExport(db)

	l, err := NewLessons(db).Create(ctx, lesson.Lesson{
		Title: "A day", Content: "We went.", Level: "A1", Revision: 1, ExtrasEditedByAdmin: true,
		Sentences: []lesson.Sentence{{Index: 0, Text: "We went."}}, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := r.Lessons(ctx, []string{l.ID, newID()})
	if err != nil || len(got) != 1 {
		t.Fatalf("Lessons = %v, %v", got, err)
	}
	d := got[0]
	if d["_id"] != l.ID || d["title"] != "A day" || d["extrasEdited"] != true {
		t.Fatalf("doc = %v", d)
	}
	if s, ok := d["sentences"].([]any); !ok || len(s) != 1 {
		t.Fatalf("JSON column not decoded: %T %v", d["sentences"], d["sentences"])
	}
	if none, err := r.Lessons(ctx, nil); err != nil || none == nil || len(none) != 0 {
		t.Fatalf("Lessons(nil) = %v, %v", none, err)
	}
}
