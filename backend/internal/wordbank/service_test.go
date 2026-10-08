package wordbank

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
)

func TestAddFillsFromDictionary(t *testing.T) {
	t.Parallel()
	e := newEnv()
	w, _, err := e.svc.Add(t.Context(), Input{Lemma: "  House "})
	if err != nil {
		t.Fatal(err)
	}
	if w.Lemma != "house" || w.IPA != "/haʊs/" || w.MeaningVi != "ngôi nhà; nhà ở" || !w.CreatedAt.Equal(e.now) {
		t.Fatalf("word = %+v", w)
	}
	// A phrase gets the joined IPA; a typed meaning or IPA is kept.
	w, _, err = e.svc.Add(t.Context(), Input{Lemma: "Good morning", MeaningVi: "chào buổi sáng"})
	if err != nil || w.IPA != "/ɡʊd ˈmɔːnɪŋ/" || w.MeaningVi != "chào buổi sáng" {
		t.Fatalf("phrase = %+v, %v", w, err)
	}
	w, _, err = e.svc.Add(t.Context(), Input{Lemma: "table", IPA: "/ˈteɪbl/"})
	if err != nil || w.IPA != "/ˈteɪbl/" || w.MeaningVi != "" {
		t.Fatalf("unknown word = %+v, %v", w, err)
	}
	if _, _, err := e.svc.Add(t.Context(), Input{Lemma: "HOUSE"}); !errors.Is(err, ErrExists) {
		t.Fatalf("duplicate: %v", err)
	}
}

func TestAddValidation(t *testing.T) {
	t.Parallel()
	e := newEnv()
	_, _, err := e.svc.Add(t.Context(), Input{Lemma: " ", MeaningVi: strings.Repeat("a", MaxMeaning+1), IPA: strings.Repeat("a", MaxIPA+1)})
	var verr *ValidationError
	if !errors.As(err, &verr) || verr.Fields["lemma"] == "" || verr.Fields["meaningVi"] == "" || verr.Fields["ipa"] == "" {
		t.Fatalf("err = %v", err)
	}
	if _, _, err := e.svc.Add(t.Context(), Input{Lemma: strings.Repeat("a", MaxLemma+1)}); !errors.As(err, &verr) || verr.Fields["lemma"] == "" {
		t.Fatalf("long lemma: %v", err)
	}
}

func TestUpdateAndDelete(t *testing.T) {
	t.Parallel()
	e := newEnv()
	if _, _, err := e.svc.Add(t.Context(), Input{Lemma: "house"}); err != nil {
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
	if _, _, err := e.svc.Add(t.Context(), Input{Lemma: "house", MeaningVi: meaning}); err != nil {
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
		if _, _, err := e.svc.Add(t.Context(), Input{Lemma: l}); err != nil {
			t.Fatal(err)
		}
	}
	p, err := e.svc.List(t.Context(), "", MissingNone, "", 1)
	if err != nil || p.Total != 3 || p.HasMore || p.Words[0].Lemma != "good" {
		t.Fatalf("all = %+v, %v", p, err)
	}
	if p, _ = e.svc.List(t.Context(), " NHÀ ", MissingNone, "", 1); p.Total != 1 || p.Words[0].Lemma != "house" {
		t.Fatalf("search meaning = %+v", p)
	}
	if p, _ = e.svc.List(t.Context(), "", MissingIPA, "", 1); p.Total != 1 || p.Words[0].Lemma != "table" {
		t.Fatalf("missing ipa = %+v", p)
	}
	var verr *ValidationError
	if _, err := e.svc.List(t.Context(), "", "colour", "", 0); !errors.As(err, &verr) || verr.Fields["missing"] == "" || verr.Fields["page"] == "" {
		t.Fatalf("bad query: %v", err)
	}
}

func TestImages(t *testing.T) {
	t.Parallel()
	e := newEnv()
	if _, _, err := e.svc.Add(t.Context(), Input{Lemma: "house"}); err != nil {
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
	if _, _, err := e.svc.Add(t.Context(), Input{Lemma: "house"}); err != nil {
		t.Fatal(err)
	}
	got, err := e.svc.Words(t.Context(), []string{"House", "table", ""})
	if err != nil || len(got) != 1 || got["house"].IPA != "/haʊs/" {
		t.Fatalf("words = %+v, %v", got, err)
	}
}

func TestAddToTopic(t *testing.T) {
	t.Parallel()
	e := newEnv()
	// A new word goes into the bank and the topic.
	w, added, err := e.svc.Add(t.Context(), Input{Lemma: " Window ", TopicID: "t1"})
	if err != nil || w.Lemma != "window" || added.InBank || added.Topic.Name != "Nhà cửa" {
		t.Fatalf("new word = %+v, %+v, %v", w, added, err)
	}
	if lists, _ := e.topics.All(t.Context()); !slices.Contains(lists[0].Words, "window") {
		t.Fatalf("topic words = %v", lists[0].Words)
	}
	// A word of the bank only goes into the topic.
	if _, _, err := e.svc.Add(t.Context(), Input{Lemma: "good"}); err != nil {
		t.Fatal(err)
	}
	if w, added, err = e.svc.Add(t.Context(), Input{Lemma: "GOOD", TopicID: "t2"}); err != nil || !added.InBank || w.MeaningVi != "tốt" {
		t.Fatalf("bank word = %+v, %+v, %v", w, added, err)
	}
	// A word the topic holds already (ignoring case) is refused, and the bank is left alone.
	if _, added, err = e.svc.Add(t.Context(), Input{Lemma: "house", TopicID: "t1"}); !errors.Is(err, ErrInTopic) || added.Topic.Name != "Nhà cửa" {
		t.Fatalf("in topic: %+v, %v", added, err)
	}
	if _, err := e.svc.Get(t.Context(), "house"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("bank got the refused word: %v", err)
	}
	var verr *ValidationError
	if _, _, err = e.svc.Add(t.Context(), Input{Lemma: "door", TopicID: "nope"}); !errors.As(err, &verr) || verr.Fields["topicId"] == "" {
		t.Fatalf("unknown topic: %v", err)
	}
}

func TestListByTopic(t *testing.T) {
	t.Parallel()
	e := newEnv()
	for _, l := range []string{"house", "table", "good morning", "door"} {
		if _, _, err := e.svc.Add(t.Context(), Input{Lemma: l}); err != nil {
			t.Fatal(err)
		}
	}
	p, err := e.svc.List(t.Context(), "", MissingNone, "", 1)
	if err != nil || p.Total != 4 {
		t.Fatalf("all = %+v, %v", p, err)
	}
	topics := map[string][]TopicRef{}
	for _, w := range p.Words {
		topics[w.Lemma] = w.Topics
	}
	// By name; a word in no topic has none.
	want := []TopicRef{{ID: "t2", Name: "Chào hỏi"}, {ID: "t1", Name: "Nhà cửa"}}
	if !slices.Equal(topics["house"], want) || len(topics["door"]) != 0 || len(topics["table"]) != 1 {
		t.Fatalf("topics = %+v", topics)
	}
	if p, _ = e.svc.List(t.Context(), "", MissingNone, "t1", 1); p.Total != 2 || p.Words[0].Lemma != "house" || p.Words[1].Lemma != "table" {
		t.Fatalf("topic t1 = %+v", p)
	}
	var verr *ValidationError
	if _, err := e.svc.List(t.Context(), "", MissingNone, "nope", 1); !errors.As(err, &verr) || verr.Fields["topicId"] == "" {
		t.Fatalf("unknown topic: %v", err)
	}
}

func TestFillMissing(t *testing.T) {
	t.Parallel()
	e := newEnv()
	inputs := []Input{
		{Lemma: "house"}, // meaning and IPA from the dictionary
		{Lemma: "table"}, // neither
		{Lemma: "lamp", MeaningVi: "cái đèn", TopicID: "t1"}, // IPA only
		{Lemma: "door"}, // not in the topic
	}
	for _, in := range inputs {
		if _, _, err := e.svc.Add(t.Context(), in); err != nil {
			t.Fatal(err)
		}
	}
	e.means.answers = []ai.WordMeaning{
		{Word: "TABLE", MeaningVi: " cái bàn ", IPA: "/ˈteɪbl/"},
		{Word: "lamp", MeaningVi: "đèn bàn", IPA: "/læmp/"}, // the meaning it has stays
		{Word: "house", MeaningVi: "không hỏi"},             // not asked, not saved
		{Word: "door", MeaningVi: "cửa"},                    // not in the topic
		{Word: "table", MeaningVi: "lần thứ hai"},           // answered twice: the first answer stays
	}
	e.now = e.now.Add(time.Hour)
	f, err := e.svc.FillMissing(t.Context(), "t1")
	if err != nil || f != (Filled{Asked: 2, Meanings: 1, IPAs: 2}) {
		t.Fatalf("filled = %+v, %v", f, err)
	}
	// One request with the topic's words, each marked with what it lacks.
	want := []ai.WordNeed{{Word: "lamp", IPA: true}, {Word: "table", Meaning: true, IPA: true}}
	if len(e.means.reqs) != 1 || e.means.reqs[0].TopicName != "Nhà cửa" || !slices.Equal(e.means.reqs[0].Words, want) {
		t.Fatalf("requests = %+v", e.means.reqs)
	}
	if w, _ := e.svc.Get(t.Context(), "table"); w.MeaningVi != "cái bàn" || w.IPA != "/ˈteɪbl/" || !w.UpdatedAt.Equal(e.now) {
		t.Fatalf("table = %+v", w)
	}
	if w, _ := e.svc.Get(t.Context(), "lamp"); w.MeaningVi != "cái đèn" || w.IPA != "/læmp/" {
		t.Fatalf("lamp = %+v", w)
	}
	for lemma, meaning := range map[string]string{"house": "ngôi nhà; nhà ở", "door": ""} {
		if w, _ := e.svc.Get(t.Context(), lemma); w.MeaningVi != meaning {
			t.Fatalf("%s = %+v", lemma, w)
		}
	}
	// Nothing left to fill: no request.
	if f, err = e.svc.FillMissing(t.Context(), "t1"); err != nil || f != (Filled{}) || len(e.means.reqs) != 1 {
		t.Fatalf("again = %+v, %v (%d requests)", f, err, len(e.means.reqs))
	}
}

func TestFillMissingErrors(t *testing.T) {
	t.Parallel()
	e := newEnv()
	if _, _, err := e.svc.Add(t.Context(), Input{Lemma: "table"}); err != nil {
		t.Fatal(err)
	}
	var verr *ValidationError
	if _, err := e.svc.FillMissing(t.Context(), "nope"); !errors.As(err, &verr) || verr.Fields["topicId"] == "" {
		t.Fatalf("unknown topic: %v", err)
	}
	e.means.err = ai.ErrQuota
	if _, err := e.svc.FillMissing(t.Context(), "t1"); !errors.Is(err, ai.ErrQuota) {
		t.Fatalf("quota: %v", err)
	}
	e.means.err = nil
	e.means.answers = []ai.WordMeaning{{Word: "table", MeaningVi: strings.Repeat("a", MaxMeaning+1)}}
	if _, err := e.svc.FillMissing(t.Context(), "t1"); !errors.Is(err, ErrMeaningsFailed) {
		t.Fatalf("no usable answer: %v", err)
	}
}
