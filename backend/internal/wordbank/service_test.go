package wordbank

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
)

func TestAddFillsFromDictionary(t *testing.T) {
	t.Parallel()
	e := newEnv()
	w, err := e.svc.Add(t.Context(), Input{Lemma: "  House "})
	if err != nil {
		t.Fatal(err)
	}
	if w.Lemma != "house" || w.IPA != "/haʊs/" || w.MeaningVi != "ngôi nhà; nhà ở" || !w.CreatedAt.Equal(e.now) {
		t.Fatalf("word = %+v", w)
	}
	// A phrase gets the joined IPA; a typed meaning or IPA is kept.
	w, err = e.svc.Add(t.Context(), Input{Lemma: "Good morning", MeaningVi: "chào buổi sáng"})
	if err != nil || w.IPA != "/ɡʊd ˈmɔːnɪŋ/" || w.MeaningVi != "chào buổi sáng" {
		t.Fatalf("phrase = %+v, %v", w, err)
	}
	w, err = e.svc.Add(t.Context(), Input{Lemma: "table", IPA: "/ˈteɪbl/"})
	if err != nil || w.IPA != "/ˈteɪbl/" || w.MeaningVi != "" {
		t.Fatalf("unknown word = %+v, %v", w, err)
	}
	if _, err := e.svc.Add(t.Context(), Input{Lemma: "HOUSE"}); !errors.Is(err, ErrExists) {
		t.Fatalf("duplicate: %v", err)
	}
}

func TestAddValidation(t *testing.T) {
	t.Parallel()
	e := newEnv()
	_, err := e.svc.Add(t.Context(), Input{Lemma: " ", MeaningVi: strings.Repeat("a", MaxMeaning+1), IPA: strings.Repeat("a", MaxIPA+1)})
	var verr *ValidationError
	if !errors.As(err, &verr) || verr.Fields["lemma"] == "" || verr.Fields["meaningVi"] == "" || verr.Fields["ipa"] == "" {
		t.Fatalf("err = %v", err)
	}
	if _, err := e.svc.Add(t.Context(), Input{Lemma: strings.Repeat("a", MaxLemma+1)}); !errors.As(err, &verr) || verr.Fields["lemma"] == "" {
		t.Fatalf("long lemma: %v", err)
	}
}

func TestUpdateAndDelete(t *testing.T) {
	t.Parallel()
	e := newEnv()
	if _, err := e.svc.Add(t.Context(), Input{Lemma: "house"}); err != nil {
		t.Fatal(err)
	}
	meaning, ipa := " căn nhà ", ""
	e.now = e.now.Add(time.Hour)
	w, err := e.svc.Update(t.Context(), "House", Details{MeaningVi: &meaning, IPA: &ipa})
	if err != nil || w.MeaningVi != "căn nhà" || w.IPA != "" || !w.UpdatedAt.Equal(e.now) {
		t.Fatalf("updated = %+v, %v", w, err)
	}
	// nil fields stay as they are.
	if w, _ = e.svc.Update(t.Context(), "house", Details{}); w.MeaningVi != "căn nhà" {
		t.Fatalf("unchanged = %+v", w)
	}
	if _, err := e.svc.Update(t.Context(), "nope", Details{}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing word: %v", err)
	}
	if err := e.svc.Delete(t.Context(), "HOUSE"); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.Delete(t.Context(), "house"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted twice: %v", err)
	}
}

func TestImport(t *testing.T) {
	t.Parallel()
	e := newEnv()
	meaning := "nhà riêng"
	if _, err := e.svc.Add(t.Context(), Input{Lemma: "house", MeaningVi: meaning}); err != nil {
		t.Fatal(err)
	}
	n, err := e.svc.Import(t.Context())
	if err != nil || n != 2 {
		t.Fatalf("import = %d, %v; want 2 (good morning, table)", n, err)
	}
	if w, _ := e.svc.Get(t.Context(), "good morning"); w.IPA != "/ɡʊd ˈmɔːnɪŋ/" {
		t.Fatalf("imported = %+v", w)
	}
	// An existing word is not touched; a second import adds nothing.
	if w, _ := e.svc.Get(t.Context(), "house"); w.MeaningVi != meaning {
		t.Fatalf("existing = %+v", w)
	}
	if n, err := e.svc.Import(t.Context()); err != nil || n != 0 {
		t.Fatalf("again = %d, %v", n, err)
	}
}

func TestList(t *testing.T) {
	t.Parallel()
	e := newEnv()
	for _, l := range []string{"house", "good", "table"} {
		if _, err := e.svc.Add(t.Context(), Input{Lemma: l}); err != nil {
			t.Fatal(err)
		}
	}
	p, err := e.svc.List(t.Context(), "", MissingNone, 1)
	if err != nil || p.Total != 3 || p.HasMore || p.Words[0].Lemma != "good" {
		t.Fatalf("all = %+v, %v", p, err)
	}
	if p, _ = e.svc.List(t.Context(), " NHÀ ", MissingNone, 1); p.Total != 1 || p.Words[0].Lemma != "house" {
		t.Fatalf("search meaning = %+v", p)
	}
	if p, _ = e.svc.List(t.Context(), "", MissingIPA, 1); p.Total != 1 || p.Words[0].Lemma != "table" {
		t.Fatalf("missing ipa = %+v", p)
	}
	var verr *ValidationError
	if _, err := e.svc.List(t.Context(), "", "colour", 0); !errors.As(err, &verr) || verr.Fields["missing"] == "" || verr.Fields["page"] == "" {
		t.Fatalf("bad query: %v", err)
	}
}

func TestImages(t *testing.T) {
	t.Parallel()
	e := newEnv()
	if _, err := e.svc.Add(t.Context(), Input{Lemma: "house"}); err != nil {
		t.Fatal(err)
	}
	// Upload: scaled down to a JPEG.
	w, err := e.svc.UploadImage(t.Context(), "House", testPNG(1000, 500))
	if err != nil || !w.HasImage() {
		t.Fatalf("upload = %+v, %v", w, err)
	}
	img, err := e.svc.Image(t.Context(), "house")
	if err != nil || img.MIME != "image/jpeg" || len(img.Data) < 3 || img.Data[0] != 0xff {
		t.Fatalf("image = %s, %v", img.MIME, err)
	}
	var verr *ValidationError
	if _, err := e.svc.UploadImage(t.Context(), "house", []byte("not an image")); !errors.As(err, &verr) || verr.Fields["image"] == "" {
		t.Fatalf("bad image: %v", err)
	}
	if _, err := e.svc.UploadImage(t.Context(), "table", testPNG(10, 10)); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown word: %v", err)
	}

	// Delete, then from a link.
	if w, err = e.svc.DeleteImage(t.Context(), "house"); err != nil || w.HasImage() {
		t.Fatalf("delete = %+v, %v", w, err)
	}
	if _, err := e.svc.Image(t.Context(), "house"); !errors.Is(err, ErrImageNotFound) {
		t.Fatalf("deleted image: %v", err)
	}
	e.fetch["https://x.test/a.png"] = testPNG(20, 20)
	e.fetch["https://x.test/a.txt"] = []byte("hello")
	if w, err = e.svc.ImportImage(t.Context(), "house", " https://x.test/a.png "); err != nil || !w.HasImage() {
		t.Fatalf("import = %+v, %v", w, err)
	}
	for _, url := range []string{"https://x.test/a.txt", "https://x.test/missing.png", ""} {
		if _, err := e.svc.ImportImage(t.Context(), "house", url); !errors.As(err, &verr) || verr.Fields["url"] == "" {
			t.Fatalf("import %q: %v", url, err)
		}
	}

	// Drawn by the AI with the word's meaning and the style.
	if w, err = e.svc.GenerateImage(t.Context(), "house", " màu nước "); err != nil || !w.HasImage() {
		t.Fatalf("generate = %+v, %v", w, err)
	}
	if r := e.draw.reqs[0]; r.Word != "house" || r.MeaningVi != "ngôi nhà; nhà ở" || r.Style != "màu nước" {
		t.Fatalf("request = %+v", r)
	}
	e.draw.err = ai.ErrQuota
	if _, err := e.svc.GenerateImage(t.Context(), "house", ""); !errors.Is(err, ai.ErrQuota) {
		t.Fatalf("quota: %v", err)
	}
	e.draw.err = errors.New("boom")
	if _, err := e.svc.GenerateImage(t.Context(), "house", ""); !errors.Is(err, ErrDrawFailed) {
		t.Fatalf("failed draw: %v", err)
	}
}

func TestWords(t *testing.T) {
	t.Parallel()
	e := newEnv()
	if _, err := e.svc.Add(t.Context(), Input{Lemma: "house"}); err != nil {
		t.Fatal(err)
	}
	got, err := e.svc.Words(t.Context(), []string{"House", "table", ""})
	if err != nil || len(got) != 1 || got["house"].IPA != "/haʊs/" {
		t.Fatalf("words = %+v, %v", got, err)
	}
}
