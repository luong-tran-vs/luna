package mysql

import (
	"errors"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/toeic"
)

func toeicGroup(key string, answer int) toeic.Group {
	at := time.Date(2026, 11, 19, 7, 0, 0, 0, time.UTC)
	return toeic.Group{Key: key, Part: 5, Level: 600, Status: toeic.StatusDraft, Source: "content/toeic/part5/a.json",
		CreatedAt: at, UpdatedAt: at, Questions: []toeic.Question{{No: 1, Prompt: "It was ___.", Options: []string{"a", "b", "c", "d"},
			Answer: answer, ExplainVi: "Vì.", Tags: []string{"word-form"}}}}
}

func TestToeicGroupsAndTests(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	db := testDB(t)
	groups := NewToeicGroups(db)
	created, replaced, err := groups.Upsert(ctx, []toeic.Group{toeicGroup("p5-a", 1), toeicGroup("p5-b", 2)})
	if err != nil || len(created) != 2 || len(replaced) != 0 {
		t.Fatalf("insert = %v %v %v", created, replaced, err)
	}
	if err := groups.SetStatus(ctx, "p5-a", toeic.StatusPublished); err != nil {
		t.Fatal(err)
	}
	if err := groups.SetStatus(ctx, "nope", toeic.StatusPublished); !errors.Is(err, toeic.ErrNotFound) {
		t.Fatalf("unknown = %v", err)
	}
	g := toeicGroup("p5-a", 3)
	g.Status = toeic.StatusDraft
	if _, replaced, err := groups.Upsert(ctx, []toeic.Group{g}); err != nil || len(replaced) != 1 {
		t.Fatalf("replace = %v %v", replaced, err)
	}
	got, err := groups.ByKeys(ctx, []string{"p5-a"})
	if err != nil || len(got) != 1 || got[0].Status != toeic.StatusPublished || got[0].Version != 2 || got[0].Questions[0].Answer != 3 {
		t.Fatalf("replaced = %+v, %v", got, err)
	}
	if pub, _ := groups.Published(ctx, 5); len(pub) != 1 {
		t.Fatalf("published = %d", len(pub))
	}

	images := NewToeicImages(db)
	img := toeic.Image{GroupKey: "p1-a", Slot: toeic.SlotPhoto, MIME: "image/jpeg", Data: []byte{1, 2}, Origin: "gemini", CreatedAt: time.Now()}
	_ = images.Save(ctx, img)
	img.Data = []byte{3}
	if err := images.Save(ctx, img); err != nil {
		t.Fatal(err)
	}
	if got, err := images.Get(ctx, "p1-a", toeic.SlotPhoto); err != nil || len(got.Data) != 1 {
		t.Fatalf("image = %+v, %v", got, err)
	}
	if has, _ := images.Has(ctx, []string{"p1-a", "p1-b"}); len(has["p1-a"]) != 1 || len(has["p1-b"]) != 0 {
		t.Fatalf("has = %v", has)
	}

	tests := NewToeicTests(db)
	test := toeic.Test{Key: "luna-01", Title: "Luna Test 01", Kind: toeic.KindFull, Order: 2, Groups: []string{"p5-a"}, Status: toeic.StatusDraft,
		CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := tests.Save(ctx, test); err != nil {
		t.Fatal(err)
	}
	test.Groups = append(test.Groups, "p5-b")
	_ = tests.Save(ctx, test)
	_ = tests.SetStatus(ctx, "luna-01", toeic.StatusPublished)
	if got, err := tests.Get(ctx, "luna-01"); err != nil || len(got.Groups) != 2 || got.Status != toeic.StatusPublished {
		t.Fatalf("test = %+v, %v", got, err)
	}
	if list, _ := tests.List(ctx, true); len(list) != 1 {
		t.Fatalf("published tests = %d", len(list))
	}
}

func TestToeicAttempts(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	repo := NewToeicAttempts(testDB(t))
	u := newID()
	at := time.Date(2026, 11, 19, 7, 0, 0, 0, time.UTC)
	a, err := repo.Create(ctx, toeic.Attempt{UserID: u, Mode: toeic.ModePractice, Source: toeic.Source{Kind: toeic.SourceTest, TestKey: "luna-01"},
		GroupKeys: []string{"p5-a", "p5-b"}, QuestionIDs: []string{"p5-a#1", "p5-b#1"}, Numbers: map[string]int{"p5-a#1": 101},
		Answers: map[string]int{}, Status: toeic.AttemptInProgress, StartedAt: at, SavedAt: at, TimeLimitMs: 60000, RemainingMs: 60000})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.InProgress(ctx, u, "luna-01"); err != nil {
		t.Fatalf("in progress = %v", err)
	}
	a.Answers["p5-a#1"] = 2
	a.RemainingMs = 5000
	if err := repo.Save(ctx, a); err != nil {
		t.Fatal(err)
	}
	a.Status, a.SubmittedAt = toeic.AttemptSubmitted, at.Add(time.Minute)
	a.Result = &toeic.Result{Correct: 1, ByQuestion: map[string]string{"p5-a#1": "right"}}
	if err := repo.Submit(ctx, a); err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(ctx, a); !errors.Is(err, toeic.ErrAttemptSubmitted) {
		t.Fatalf("save after submit = %v", err)
	}
	got, err := repo.Get(ctx, u, a.ID)
	if err != nil || got.Answers["p5-a#1"] != 2 || got.RemainingMs != 5000 || got.Result.Correct != 1 || got.Numbers["p5-a#1"] != 101 ||
		got.SubmittedAt.IsZero() {
		t.Fatalf("get = %+v, %v", got, err)
	}
	if _, err := repo.Get(ctx, newID(), a.ID); !errors.Is(err, toeic.ErrNotFound) {
		t.Fatalf("another user = %v", err)
	}
	with, err := repo.WithGroup(ctx, "p5-b")
	if err != nil || len(with) != 1 {
		t.Fatalf("with group = %d, %v", len(with), err)
	}
	if err := repo.UpdateResult(ctx, a.ID, toeic.Result{Correct: 0, Wrong: 1}); err != nil {
		t.Fatal(err)
	}
	answered, _ := repo.AnsweredQuestions(ctx, u)
	if !answered["p5-b#1"] {
		t.Fatalf("answered = %v", answered)
	}
}

func TestToeicMistakesAndGoals(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	db := testDB(t)
	repo := NewToeicMistakes(db)
	u := newID()
	at := time.Date(2026, 11, 19, 7, 0, 0, 0, time.UTC)
	m := toeic.Mistake{UserID: u, QuestionID: "p5-a#1", GroupKey: "p5-a", Part: 5, Tags: []string{"word-form"}, LastWrongAt: at, AddedAt: at}
	_ = repo.Upsert(ctx, m)
	m.Streak = 1
	if err := repo.Upsert(ctx, m); err != nil {
		t.Fatal(err)
	}
	if n, _ := repo.Count(ctx, u); n != 1 {
		t.Fatalf("count = %d", n)
	}
	if got, err := repo.Get(ctx, u, "p5-a#1"); err != nil || got.Streak != 1 || got.Tags[0] != "word-form" {
		t.Fatalf("get = %+v, %v", got, err)
	}
	_ = repo.Delete(ctx, u, "p5-a#1")
	if _, err := repo.Get(ctx, u, "p5-a#1"); !errors.Is(err, toeic.ErrNotFound) {
		t.Fatalf("deleted = %v", err)
	}

	goals := NewToeicGoals(db)
	_ = goals.Save(ctx, toeic.Goal{UserID: u, Target: 600, UpdatedAt: at})
	_ = goals.Save(ctx, toeic.Goal{UserID: u, Target: 750, ExamDate: "2027-01-15", UpdatedAt: at})
	if g, err := goals.Get(ctx, u); err != nil || g.Target != 750 || g.ExamDate != "2027-01-15" {
		t.Fatalf("goal = %+v, %v", g, err)
	}
}
