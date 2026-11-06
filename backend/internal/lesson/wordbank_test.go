package lesson

import (
	"context"
	"errors"
	"testing"
	"time"
)

// fakeBank is a word bank with "give up" (IPA and picture) and "study" (IPA only).
type fakeBank struct{}

func (fakeBank) Words(_ context.Context, lemmas []string) (map[string]BankWord, error) {
	all := map[string]BankWord{
		"give up": {IPA: "/ɡɪv ʌp/", HasImage: true},
		"study":   {IPA: "/ˈstʌd.i/"},
	}
	out := map[string]BankWord{}
	for _, l := range lemmas {
		if w, ok := all[l]; ok {
			out[l] = w
		}
	}
	return out, nil
}

func (fakeBank) Image(_ context.Context, lemma string) (WordImage, error) {
	if lemma == "give up" {
		return WordImage{Lemma: lemma, MIME: "image/jpeg", Data: []byte("bank")}, nil
	}
	return WordImage{}, ErrImageNotFound
}

func TestVocabularyUsesWordBank(t *testing.T) {
	t.Parallel()
	r, id := newVocabularyEnv(t, StatusDone, []Annotation{
		{Text: "went", Lemma: "go", MeaningVi: "đi", SentenceIndex: 0},
		{Text: "gave up", Lemma: "give up", MeaningVi: "bỏ", SentenceIndex: 1},
		{Text: "studies", Lemma: "study", MeaningVi: "học", SentenceIndex: 2},
	})
	images := newFakeImageStore()
	if err := images.Save(t.Context(), WordImage{LessonID: id, Lemma: "go", MIME: "image/jpeg", Data: []byte("own"), CreatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	r.WithImages(images).WithWordBank(fakeBank{})

	items, _, err := r.Vocabulary(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	// IPA: the bank's first, then the dictionary's. Pictures: the lesson's own or the bank's.
	got := map[string]VocabItem{}
	for _, it := range items {
		got[it.Lemma] = it
	}
	if g := got["go"]; g.IPA != "/ɡoʊ/" || !g.HasImage {
		t.Fatalf("go = %+v", g)
	}
	if g := got["give up"]; g.IPA != "/ɡɪv ʌp/" || !g.HasImage {
		t.Fatalf("give up = %+v", g)
	}
	if g := got["study"]; g.IPA != "/ˈstʌd.i/" || g.HasImage {
		t.Fatalf("study = %+v", g)
	}

	if img, err := r.Image(t.Context(), id, "go"); err != nil || string(img.Data) != "own" {
		t.Fatalf("own image = %q, %v", img.Data, err)
	}
	if img, err := r.Image(t.Context(), id, "Give Up"); err != nil || string(img.Data) != "bank" {
		t.Fatalf("bank image = %q, %v", img.Data, err)
	}
	if _, err := r.Image(t.Context(), id, "study"); !errors.Is(err, ErrImageNotFound) {
		t.Fatalf("no image: %v", err)
	}
}

func TestAdminImagesShowWordBankPictures(t *testing.T) {
	t.Parallel()
	e := newImageEnv(t)
	e.svc.Bank = fakeBank{}
	l := e.annotated(t)

	st, err := e.svc.Images(t.Context(), l.ID)
	if err != nil {
		t.Fatal(err)
	}
	if st.Count != 0 || st.BankCount != 1 {
		t.Fatalf("count = %d, bank count = %d", st.Count, st.BankCount)
	}
	if w := st.Words[1]; w.Lemma != "give up" || w.HasImage || !w.FromBank {
		t.Fatalf("give up = %+v", w)
	}
	if w := st.Words[0]; w.Lemma != "go" || w.FromBank {
		t.Fatalf("go = %+v", w)
	}
	img, err := e.svc.AdminImage(t.Context(), l.ID, "Give up")
	if err != nil || string(img.Data) != "bank" {
		t.Fatalf("admin image = %q, %v", img.Data, err)
	}
	if _, err := e.svc.AdminImage(t.Context(), l.ID, SceneKey); !errors.Is(err, ErrImageNotFound) {
		t.Fatalf("scene falls back to the bank: %v", err)
	}

	// The lesson's own picture wins over the bank's.
	if err := e.svc.UploadImage(t.Context(), l.ID, "give up", testPNG(40, 30)); err != nil {
		t.Fatal(err)
	}
	st, _ = e.svc.Images(t.Context(), l.ID)
	if w := st.Words[1]; !w.HasImage || w.FromBank || st.Count != 1 || st.BankCount != 0 {
		t.Fatalf("after upload: %+v, counts %d/%d", w, st.Count, st.BankCount)
	}
	if img, _ := e.svc.AdminImage(t.Context(), l.ID, "give up"); string(img.Data) == "bank" {
		t.Fatal("admin image is still the bank's")
	}
}
