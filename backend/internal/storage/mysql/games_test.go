package mysql

import (
	"errors"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/games"
)

func TestGameBestsRaise(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	repo := NewGameBests(testDB(t))
	u := newID()
	at := time.Date(2026, 11, 13, 7, 0, 0, 0, time.UTC)

	if ok, err := repo.Raise(ctx, u, games.GamePicture, 1200, at); err != nil || !ok {
		t.Fatalf("first = %v, %v", ok, err)
	}
	if ok, err := repo.Raise(ctx, u, games.GamePicture, 900, at.Add(time.Hour)); err != nil || ok {
		t.Fatalf("lower = %v, %v", ok, err)
	}
	if ok, err := repo.Raise(ctx, u, games.GamePicture, 1200, at.Add(time.Hour)); err != nil || ok {
		t.Fatalf("same = %v, %v", ok, err)
	}
	if ok, err := repo.Raise(ctx, u, games.GamePicture, 2100, at.Add(2*time.Hour)); err != nil || !ok {
		t.Fatalf("higher = %v, %v", ok, err)
	}
	got, err := repo.List(ctx, u)
	if err != nil || len(got) != 1 || got[0].Score != 2100 || !got[0].At.Equal(at.Add(2*time.Hour)) {
		t.Fatalf("list = %+v, %v", got, err)
	}
}

func TestDailyChallenges(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	repo := NewDailyChallenges(testDB(t))
	u := newID()
	at := time.Date(2026, 11, 13, 7, 0, 0, 0, time.UTC)
	r := games.DailyResult{UserID: u, Day: "2026-11-13", Score: 920, FinishedAt: at,
		Answers: []games.DailyAnswer{{Kind: "picture", Right: true}, {Kind: "verb"}}}
	if err := repo.Save(ctx, r); err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(ctx, r); !errors.Is(err, games.ErrExists) {
		t.Fatalf("second save = %v", err)
	}
	_ = repo.Save(ctx, games.DailyResult{UserID: u, Day: "2026-11-12", Score: 800, FinishedAt: at})
	got, err := repo.Get(ctx, u, "2026-11-13")
	if err != nil || got.Score != 920 || len(got.Answers) != 2 || !got.Answers[0].Right || got.Answers[1].Kind != "verb" {
		t.Fatalf("get = %+v, %v", got, err)
	}
	if _, err := repo.Get(ctx, u, "2026-11-01"); !errors.Is(err, games.ErrNotFound) {
		t.Fatalf("missing = %v", err)
	}
	list, err := repo.List(ctx, u)
	if err != nil || len(list) != 2 || list[0].Day != "2026-11-13" {
		t.Fatalf("list = %+v, %v", list, err)
	}
}

func TestGrammarSpotItems(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	repo := NewGrammarSpotItems(testDB(t))
	at := time.Date(2026, 11, 13, 7, 0, 0, 0, time.UTC)
	created, err := repo.Create(ctx, []games.SpotItem{
		{PointID: "a1-to-be", Level: "A1", Sentence: "She are happy.", Wrong: "are", Right: "is", Distractors: []string{"am", "be"},
			RuleVi: "She đi với is.", Status: games.SpotDraft, CreatedAt: at, UpdatedAt: at},
		{PointID: "a1-to-be", Level: "A1", Sentence: "I is tired.", Wrong: "is", Right: "am", Distractors: []string{"are", "be"},
			RuleVi: "I đi với am.", Status: games.SpotApproved, CreatedAt: at, UpdatedAt: at},
	})
	if err != nil || len(created) != 2 || created[0].ID == "" {
		t.Fatalf("create = %+v, %v", created, err)
	}
	approved, err := repo.Approved(ctx, nil)
	if err != nil || len(approved) != 1 || approved[0].Wrong != "is" || len(approved[0].Distractors) != 2 {
		t.Fatalf("approved = %+v, %v", approved, err)
	}
	counts, _ := repo.Counts(ctx)
	if c := counts["a1-to-be"]; c.Approved != 1 || c.Draft != 1 {
		t.Fatalf("counts = %+v", counts)
	}
	it := created[0]
	it.Status = games.SpotApproved
	if err := repo.Update(ctx, it); err != nil {
		t.Fatal(err)
	}
	if got, _ := repo.Get(ctx, it.ID); got.Status != games.SpotApproved {
		t.Fatalf("updated = %+v", got)
	}
	if err := repo.Delete(ctx, it.ID); err != nil {
		t.Fatal(err)
	}
	if err := repo.Delete(ctx, it.ID); !errors.Is(err, games.ErrNotFound) {
		t.Fatalf("delete twice = %v", err)
	}
	if list, _ := repo.ListByPoint(ctx, "a1-to-be"); len(list) != 1 {
		t.Fatalf("list = %+v", list)
	}
}
