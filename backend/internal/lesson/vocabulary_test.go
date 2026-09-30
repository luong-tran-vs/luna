package lesson

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"testing"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

func newVocabularyEnv(t *testing.T, status Status, anns []Annotation) (*Reader, string) {
	t.Helper()
	lessons := newFakeLessons()
	l, err := lessons.Create(context.Background(), Lesson{
		Title: "Park", Level: "B1", Content: readingContent, Revision: 1,
		Sentences: toSentences(SplitSentences(readingContent)), AnnotationStatus: status, Annotations: anns,
	})
	if err != nil {
		t.Fatal(err)
	}
	return NewReader(lessons, readingDict, newFakeTopics()), l.ID
}

func TestVocabulary(t *testing.T) {
	t.Parallel()
	r, id := newVocabularyEnv(t, StatusDone, []Annotation{
		{Text: "went", Lemma: "go", MeaningVi: "đã về", SentenceIndex: 3},
		{Text: "gave up", Lemma: "Give Up", MeaningVi: "đã bỏ", SentenceIndex: 1},
		{Text: "went", Lemma: "go", MeaningVi: "đã đi (tới công viên)", SentenceIndex: 0},
		{Text: "studies", Lemma: "study", MeaningVi: "học", SentenceIndex: 2},
		{Text: "smoking", Lemma: "smoke", MeaningVi: "hút thuốc", SentenceIndex: 1},
	})

	items, available, err := r.Vocabulary(t.Context(), id)
	if err != nil || !available {
		t.Fatalf("available = %v, %v", available, err)
	}
	want := []VocabItem{
		{Lemma: "go", Text: "went", MeaningVi: "đã đi (tới công viên)", IPA: "/ɡəʊ/", SentenceIndex: 0, Sentence: "We went to the park."},
		{Lemma: "give up", Text: "gave up", MeaningVi: "đã bỏ", IPA: "", SentenceIndex: 1, Sentence: "He gave up smoking."},
		{Lemma: "smoke", Text: "smoking", MeaningVi: "hút thuốc", IPA: "", SentenceIndex: 1, Sentence: "He gave up smoking."},
		{Lemma: "study", Text: "studies", MeaningVi: "học", IPA: "/ˈstʌdi/", SentenceIndex: 2, Sentence: "She studies every day."},
	}
	if len(items) != len(want) {
		t.Fatalf("items = %+v", items)
	}
	for i := range want {
		if items[i] != want[i] {
			t.Fatalf("item %d = %+v, want %+v", i, items[i], want[i])
		}
	}
}

func TestVocabularyUnavailable(t *testing.T) {
	t.Parallel()
	ann := []Annotation{{Text: "went", Lemma: "go", MeaningVi: "đi", SentenceIndex: 0}}
	for name, tc := range map[string]struct {
		status Status
		anns   []Annotation
	}{
		"running":        {StatusRunning, ann},
		"failed":         {StatusFailed, ann},
		"no annotations": {StatusDone, nil},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			r, id := newVocabularyEnv(t, tc.status, tc.anns)
			items, available, err := r.Vocabulary(t.Context(), id)
			if err != nil || available || len(items) != 0 {
				t.Fatalf("items = %+v, available = %v, %v", items, available, err)
			}
		})
	}
}

func TestVocabularyUnknownLesson(t *testing.T) {
	t.Parallel()
	r, _ := newVocabularyEnv(t, StatusDone, nil)
	if _, _, err := r.Vocabulary(t.Context(), "nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
}

func TestVocabularyEndpoint(t *testing.T) {
	t.Parallel()
	r, id := newVocabularyEnv(t, StatusDone, []Annotation{{Text: "went", Lemma: "go", MeaningVi: "đi", SentenceIndex: 0}})
	mux := newReadingMux(r)

	rec := get(t, mux, "/api/lessons/"+id+"/vocabulary", "learner")
	want := `{"available":true,"items":[{"lemma":"go","text":"went","meaningVi":"đi","ipa":"/ɡəʊ/","sentenceIndex":0,"sentence":"We went to the park."}]}` + "\n"
	if rec.Code != http.StatusOK || rec.Body.String() != want {
		t.Fatalf("vocabulary: %d %s", rec.Code, rec.Body)
	}

	empty, emptyID := newVocabularyEnv(t, StatusRunning, nil)
	if rec := get(t, newReadingMux(empty), "/api/lessons/"+emptyID+"/vocabulary", "learner"); rec.Body.String() != "{\"available\":false,\"items\":[]}\n" {
		t.Fatalf("unavailable: %s", rec.Body)
	}
	if rec := get(t, mux, "/api/lessons/nope/vocabulary", "learner"); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown: %d", rec.Code)
	}
	if rec := get(t, mux, "/api/lessons/"+id+"/vocabulary", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous: %d", rec.Code)
	}
}

func newReadingMux(r *Reader) *http.ServeMux {
	mux := http.NewServeMux()
	NewReadingHandler(r, slog.New(slog.DiscardHandler)).Register(mux, httpx.RequireAuth(resolver), allowAll)
	return mux
}
