package lesson

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/luongtran/luna/backend/internal/dictionary"
)

// fakeDict is a tiny dictionary; Resolve uses the real base-form logic over it.
type fakeDict map[string]dictionary.Entry

func (f fakeDict) Lookup(_ context.Context, word string) (dictionary.Entry, bool, error) {
	e, ok := f[word]
	return e, ok, nil
}

func (f fakeDict) Resolve(ctx context.Context, word string) (dictionary.Entry, bool, error) {
	return dictionary.Resolve(ctx, f, word)
}

func entry(word, ipa string, meanings ...string) dictionary.Entry {
	e := dictionary.Entry{Word: word, IPA: ipa}
	for _, m := range meanings {
		e.Meanings = append(e.Meanings, dictionary.Meaning{POS: "V", Text: m})
	}
	return e
}

var readingDict = fakeDict{
	"go":      entry("go", "/ɡəʊ/", "Đi, đi đến."),
	"went":    entry("went", "", "động từ quá khứ của go."),
	"park":    entry("park", "/pɑːk/", "Công viên.", "Bãi đỗ xe."),
	"study":   entry("study", "/ˈstʌdi/", "Học."),
	"give up": entry("give up", "", "Từ bỏ."),
	"we":      entry("we", "/wiː/", "Chúng tôi."),
}

const readingContent = "We went to the park. He gave up smoking.\n\nShe studies every day. We went home."

func newReaderEnv(t *testing.T) (*Reader, Lesson) {
	t.Helper()
	lessons := newFakeLessons()
	l, _ := lessons.Create(context.Background(), Lesson{
		Title: "Park", Level: "A1", TopicID: "topic-a1", Content: readingContent, Revision: 3,
		Sentences: toSentences(SplitSentences(readingContent)),
		Annotations: []Annotation{
			{Text: "went", Lemma: "go", MeaningVi: "đã đi (tới công viên)", SentenceIndex: 0},
			{Text: "went", Lemma: "go", MeaningVi: "đã về", SentenceIndex: 3},
			{Text: "gave up", Lemma: "give up", MeaningVi: "đã bỏ", SentenceIndex: 1},
		},
		AnnotationError: "secret", AudioError: "secret",
	})
	return NewReader(lessons, readingDict, newFakeTopics(), newFakeAnswers()), l
}

// --- reading view (foundation) ---

func TestReadingView(t *testing.T) {
	t.Parallel()
	r, l := newReaderEnv(t)

	v, err := r.View(t.Context(), "u1", l.ID)
	if err != nil {
		t.Fatal(err)
	}
	if v.Title != "Park" || len(v.Sentences) != 4 {
		t.Fatalf("view = %+v", v)
	}
	if want := [][]int{{0, 1}, {2, 3}}; !slices.EqualFunc(v.Paragraphs, want, slices.Equal) {
		t.Errorf("paragraphs = %v, want %v", v.Paragraphs, want)
	}
	lemmas := map[string]string{"went": "go", "park": "park", "studies": "study", "we": "we"}
	for word, lemma := range lemmas {
		if v.Lemmas[word] != lemma {
			t.Errorf("lemmas[%q] = %q, want %q", word, v.Lemmas[word], lemma)
		}
	}
	if _, ok := v.Lemmas["smoking"]; ok {
		t.Error("unknown word should have no lemma entry")
	}
	if len(v.Phrases) != 1 || v.Phrases[0] != (Phrase{Text: "gave up", Lemma: "give up"}) {
		t.Errorf("phrases = %+v", v.Phrases)
	}

	if _, err := r.View(t.Context(), "u1", "missing"); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing lesson: %v", err)
	}
}

func TestParagraphsFallback(t *testing.T) {
	t.Parallel()
	// Sentences that do not match a re-split of the content collapse into one paragraph.
	got := paragraphs("One. Two.\n\nThree.", 5)
	if want := [][]int{{0, 1, 2, 3, 4}}; !slices.EqualFunc(got, want, slices.Equal) {
		t.Fatalf("paragraphs = %v, want %v", got, want)
	}
}

// --- lookup (US1) ---

func TestLookupPrefersAnnotationOfSameSentence(t *testing.T) {
	t.Parallel()
	r, l := newReaderEnv(t)

	got, err := r.Lookup(t.Context(), l.ID, "went", 3)
	if err != nil {
		t.Fatal(err)
	}
	want := LookupResult{Source: SourceAI, Text: "went", Lemma: "go", IPA: "/ɡəʊ/", Meanings: []LookupMeaning{{Text: "đã về"}}}
	if !lookupEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}

	// Another sentence without its own annotation still uses the annotation of the same word.
	got, _ = r.Lookup(t.Context(), l.ID, "Went", 2)
	if got.Source != SourceAI || got.Lemma != "go" {
		t.Fatalf("other sentence: %+v", got)
	}
}

func TestLookupByAnnotationLemma(t *testing.T) {
	t.Parallel()
	r, l := newReaderEnv(t)

	got, err := r.Lookup(t.Context(), l.ID, "go", 0)
	if err != nil || got.Source != SourceAI || got.Meanings[0].Text != "đã đi (tới công viên)" {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestLookupFallsBackToDictionary(t *testing.T) {
	t.Parallel()
	r, l := newReaderEnv(t)

	got, err := r.Lookup(t.Context(), l.ID, "studies", 2)
	if err != nil {
		t.Fatal(err)
	}
	want := LookupResult{Source: SourceDictionary, Text: "studies", Lemma: "study", IPA: "/ˈstʌdi/", Meanings: []LookupMeaning{{POS: "V", Text: "Học."}}}
	if !lookupEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}

	if got, _ := r.Lookup(t.Context(), l.ID, "park", 0); len(got.Meanings) != 2 {
		t.Fatalf("park meanings = %+v", got.Meanings)
	}
}

func TestLookupNotFound(t *testing.T) {
	t.Parallel()
	r, l := newReaderEnv(t)

	if _, err := r.Lookup(t.Context(), l.ID, "smoking", 1); !errors.Is(err, ErrLookupNotFound) {
		t.Fatalf("err = %v, want ErrLookupNotFound", err)
	}
	if _, err := r.Lookup(t.Context(), "missing", "went", 0); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing lesson: %v", err)
	}
}

// --- phrases (US3) ---

func TestLookupPhrase(t *testing.T) {
	t.Parallel()
	r, l := newReaderEnv(t)

	got, err := r.Lookup(t.Context(), l.ID, "gave  up", 1)
	if err != nil || got.Source != SourceAI || got.Lemma != "give up" || got.Text != "gave up" {
		t.Fatalf("by text: %+v, %v", got, err)
	}
	got, err = r.Lookup(t.Context(), l.ID, "give up", 0)
	if err != nil || got.Source != SourceAI || got.Meanings[0].Text != "đã bỏ" {
		t.Fatalf("by lemma: %+v, %v", got, err)
	}
	if _, err := r.Lookup(t.Context(), l.ID, "to the", 0); !errors.Is(err, ErrLookupNotFound) {
		t.Fatalf("unknown phrase: %v", err)
	}
}

func lookupEqual(a, b LookupResult) bool {
	return a.Source == b.Source && a.Text == b.Text && a.Lemma == b.Lemma && a.IPA == b.IPA &&
		slices.Equal(a.Meanings, b.Meanings)
}
