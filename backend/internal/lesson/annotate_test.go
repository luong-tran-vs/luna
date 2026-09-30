package lesson

import (
	"errors"
	"testing"

	"github.com/luongtran/luna/backend/internal/ai"
)

var sampleSentences = []string{"We went to the park.", "He gave up smoking.", "It was a sunny day."}

func TestCleanAnnotations(t *testing.T) {
	t.Parallel()

	got, err := CleanAnnotations([]ai.Annotation{
		{Text: " went ", Lemma: "go", MeaningVi: " đã đi ", SentenceIndex: 0},
		{Text: "Gave Up", Lemma: "give up", MeaningVi: "bỏ", SentenceIndex: 1},      // case-insensitive
		{Text: "sunny", Lemma: "sunny", MeaningVi: "nắng", SentenceIndex: 0},        // wrong index, found in 2
		{Text: "banana", Lemma: "banana", MeaningVi: "chuối", SentenceIndex: 0},     // not in lesson
		{Text: "park", Lemma: "", MeaningVi: "công viên", SentenceIndex: 0},         // missing lemma
		{Text: "went", Lemma: "go", MeaningVi: "đi", SentenceIndex: 0},              // duplicate
		{Text: "smoking", Lemma: "smoke", MeaningVi: "hút thuốc", SentenceIndex: 9}, // index out of range
	}, sampleSentences)
	if err != nil {
		t.Fatalf("CleanAnnotations: %v", err)
	}

	want := []Annotation{
		{Text: "went", Lemma: "go", MeaningVi: "đã đi", SentenceIndex: 0},
		{Text: "Gave Up", Lemma: "give up", MeaningVi: "bỏ", SentenceIndex: 1},
		{Text: "sunny", Lemma: "sunny", MeaningVi: "nắng", SentenceIndex: 2},
		{Text: "smoking", Lemma: "smoke", MeaningVi: "hút thuốc", SentenceIndex: 1},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v\nwant %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("item %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestCleanAnnotationsNoneValid(t *testing.T) {
	t.Parallel()

	_, err := CleanAnnotations([]ai.Annotation{{Text: "banana", Lemma: "banana", MeaningVi: "chuối"}}, sampleSentences)
	if !errors.Is(err, ErrNoValidAnnotations) {
		t.Fatalf("err = %v, want ErrNoValidAnnotations", err)
	}
	if _, err := CleanAnnotations(nil, sampleSentences); !errors.Is(err, ErrNoValidAnnotations) {
		t.Fatalf("empty: err = %v", err)
	}
}

func TestFindSentence(t *testing.T) {
	t.Parallel()

	if i, ok := findSentence("GAVE UP", 7, sampleSentences); !ok || i != 1 {
		t.Fatalf("findSentence = %d, %v", i, ok)
	}
	if _, ok := findSentence("", 0, sampleSentences); ok {
		t.Fatal("empty text found")
	}
}
