package mysql

import (
	"errors"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/wordbank"
)

func TestWordBank(t *testing.T) {
	t.Parallel()
	r := NewWordBank(testDB(t))
	ctx := t.Context()
	at := time.Date(2026, 10, 7, 9, 0, 0, 0, time.UTC)
	// Lemmas unique to this run, so a shared test database does not get in the way.
	tag := newID()
	house, table := "house "+tag, "table "+tag

	if err := r.Create(ctx, wordbank.Word{Lemma: house, MeaningVi: "ngôi nhà", IPA: "/haʊs/", CreatedAt: at, UpdatedAt: at}); err != nil {
		t.Fatal(err)
	}
	if err := r.Create(ctx, wordbank.Word{Lemma: house, CreatedAt: at, UpdatedAt: at}); !errors.Is(err, wordbank.ErrExists) {
		t.Fatalf("duplicate: %v", err)
	}
	n, err := r.InsertMissing(ctx, []wordbank.Word{{Lemma: house, CreatedAt: at, UpdatedAt: at}, {Lemma: table, CreatedAt: at, UpdatedAt: at}})
	if err != nil || n != 1 {
		t.Fatalf("insert missing = %d, %v", n, err)
	}

	words, total, err := r.List(ctx, wordbank.ListQuery{Search: tag, Missing: wordbank.MissingIPA, Limit: 10})
	if err != nil || total != 1 || words[0].Lemma != table {
		t.Fatalf("missing ipa = %+v (%d), %v", words, total, err)
	}
	if words, total, _ = r.List(ctx, wordbank.ListQuery{Search: tag, Limit: 1, Skip: 1}); total != 2 || len(words) != 1 || words[0].Lemma != table {
		t.Fatalf("page 2 = %+v (%d)", words, total)
	}

	upd := wordbank.Word{Lemma: house, MeaningVi: "căn nhà", IPA: "/haʊs/", UpdatedAt: at.Add(time.Hour)}
	if err := r.Update(ctx, upd); err != nil {
		t.Fatal(err)
	}
	if err := r.Update(ctx, upd); err != nil { // same values: still found
		t.Fatalf("same update: %v", err)
	}
	if err := r.Update(ctx, wordbank.Word{Lemma: "nope " + tag}); !errors.Is(err, wordbank.ErrNotFound) {
		t.Fatalf("update missing: %v", err)
	}

	img := wordbank.Image{Lemma: house, MIME: "image/jpeg", Data: []byte{0xff, 0xd8, 1}, CreatedAt: at}
	if err := r.SaveImage(ctx, img); err != nil {
		t.Fatal(err)
	}
	img.Data = []byte{0xff, 0xd8, 2}
	if err := r.SaveImage(ctx, img); err != nil {
		t.Fatal(err)
	}
	if err := r.SaveImage(ctx, wordbank.Image{Lemma: "nope " + tag, MIME: "image/jpeg", Data: []byte{1}, CreatedAt: at}); !errors.Is(err, wordbank.ErrNotFound) {
		t.Fatalf("image of missing word: %v", err)
	}
	got, err := r.Get(ctx, house)
	if err != nil || got.MeaningVi != "căn nhà" || !got.ImageAt.Equal(at) {
		t.Fatalf("get = %+v, %v", got, err)
	}
	if g, err := r.Image(ctx, house); err != nil || g.Data[2] != 2 {
		t.Fatalf("image = %+v, %v", g, err)
	}
	if found, err := r.Find(ctx, []string{house, "nope " + tag}); err != nil || len(found) != 1 {
		t.Fatalf("find = %+v, %v", found, err)
	}

	if err := r.DeleteImage(ctx, house); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Image(ctx, house); !errors.Is(err, wordbank.ErrImageNotFound) {
		t.Fatalf("deleted image: %v", err)
	}
	if err := r.Delete(ctx, house); err != nil {
		t.Fatal(err)
	}
	if err := r.Delete(ctx, house); !errors.Is(err, wordbank.ErrNotFound) {
		t.Fatalf("deleted twice: %v", err)
	}
	_ = r.Delete(ctx, table)
}
