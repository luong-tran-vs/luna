package wordbank

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

func TestCardWordsListsTheSavedWordsLackingAPicture(t *testing.T) {
	t.Parallel()
	e := newEnv()
	addWords(t, e, "house", "apple")
	if _, err := e.svc.GenerateImage(t.Context(), "apple", "", ""); err != nil {
		t.Fatal(err)
	}
	e.cards.words = []CardWord{
		{Lemma: "apple", MeaningVi: "quả táo", Learners: 3},
		{Lemma: "run", MeaningVi: "chạy", IPA: "/rʌn/", Learners: 2},
		{Lemma: "house", MeaningVi: "ngôi nhà", Learners: 1},
		// Typed in another case by one learner: the same word as "run".
		{Lemma: "Run", MeaningVi: "chạy", Learners: 1},
		{Lemma: "  ", Learners: 5},
	}
	view, err := e.svc.CardWords(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if view.Total != 3 || view.WithImage != 1 {
		t.Fatalf("view = %+v", view)
	}
	want := []CardWordMissing{
		{CardWord: CardWord{Lemma: "run", MeaningVi: "chạy", IPA: "/rʌn/", Learners: 3}},
		{CardWord: CardWord{Lemma: "house", MeaningVi: "ngôi nhà", Learners: 1}, InBank: true},
	}
	if !slices.Equal(view.Missing, want) {
		t.Errorf("missing = %+v, want %+v", view.Missing, want)
	}
}

func TestCardWordsBatchAddsTheWordsToTheBankAndDrawsThem(t *testing.T) {
	t.Parallel()
	e := newEnv()
	addWords(t, e, "house")
	e.cards.words = []CardWord{
		{Lemma: "run", MeaningVi: "chạy", IPA: "/rʌn/", Learners: 2},
		{Lemma: "house", MeaningVi: "nhà của học viên", Learners: 1},
	}
	st, err := e.svc.StartCardWordsBatch(t.Context(), EngineGemini, "")
	if err != nil {
		t.Fatal(err)
	}
	if !st.Running || !st.CardWords || st.Total != 2 {
		t.Fatalf("started = %+v", st)
	}
	st = waitBatch(t, e.svc)
	if got := savedLemmas(st); !slices.Equal(got, []string{"run", "house"}) {
		t.Fatalf("saved = %v", got)
	}
	// The new word got the meaning of the card, so the AI draws the right sense; the word already
	// in the bank keeps its own.
	run, err := e.svc.Get(t.Context(), "run")
	if err != nil || run.MeaningVi != "chạy" || run.IPA != "/rʌn/" || !run.HasImage() {
		t.Errorf("run = %+v, %v", run, err)
	}
	if house, _ := e.svc.Get(t.Context(), "house"); house.MeaningVi == "nhà của học viên" {
		t.Errorf("house = %+v", house)
	}
	if e.draw.reqs[0].Word != "run" || e.draw.reqs[0].MeaningVi != "chạy" {
		t.Errorf("first request = %+v", e.draw.reqs[0])
	}

	// Nothing is left: the next batch says so.
	if _, err := e.svc.StartCardWordsBatch(t.Context(), EngineGemini, ""); err == nil {
		t.Error("a batch started with every word drawn")
	}
}

func TestImageURLsGiveTheLearnersAddress(t *testing.T) {
	t.Parallel()
	e := newEnv()
	addWords(t, e, "house", "apple")
	if _, err := e.svc.GenerateImage(t.Context(), "house", "", ""); err != nil {
		t.Fatal(err)
	}
	urls, err := e.svc.ImageURLs(t.Context(), []string{"House", "apple", "unknown"})
	if err != nil {
		t.Fatal(err)
	}
	if len(urls) != 1 || !strings.HasPrefix(urls["House"], "/api/words/house/image?v=") {
		t.Errorf("urls = %v", urls)
	}
}

func TestCardWordsAPI(t *testing.T) {
	t.Parallel()
	mux, e := newAPI(t)
	addWords(t, e, "house")
	if _, err := e.svc.GenerateImage(t.Context(), "house", "", ""); err != nil {
		t.Fatal(err)
	}
	e.cards.words = []CardWord{{Lemma: "run", MeaningVi: "chạy", Learners: 2}, {Lemma: "house", Learners: 1}}

	rec := do(mux, http.MethodGet, "/api/admin/words/card-words", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(),
		`{"total":2,"withImage":1,"missingCount":1,"missing":[{"lemma":"run","meaningVi":"chạy","learners":2,"inBank":false}],"codexImages":false}`) {
		t.Fatalf("list = %d %s", rec.Code, rec.Body)
	}
	rec = do(mux, http.MethodPost, "/api/admin/words/card-words/image-batch", `{"engine":"gemini"}`)
	if rec.Code != http.StatusAccepted || !strings.Contains(rec.Body.String(), `"cardWords":true`) {
		t.Fatalf("start = %d %s", rec.Code, rec.Body)
	}
	waitBatch(t, e.svc)

	// Learners load the picture from their own address; the admin list is not theirs.
	asLearner := func(path string) int {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("X-Role", "learner")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec.Code
	}
	if code := asLearner("/api/words/house/image"); code != http.StatusOK {
		t.Errorf("learner picture = %d", code)
	}
	if code := asLearner("/api/admin/words/card-words"); code != http.StatusForbidden {
		t.Errorf("learner list = %d", code)
	}
}
