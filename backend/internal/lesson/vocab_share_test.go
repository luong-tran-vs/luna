package lesson

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
)

// fakeVocab records what lessons share with the topics and the word bank.
type fakeVocab struct {
	mu    sync.Mutex
	calls []vocabCall
	fail  string // topic whose call fails
}

type vocabCall struct {
	topic string
	level Level
	words []LessonWord
}

func (f *fakeVocab) LessonWords(_ context.Context, topicID string, level Level, words []LessonWord) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if topicID == f.fail {
		return 0, errors.New("boom")
	}
	f.calls = append(f.calls, vocabCall{topicID, level, slices.Clone(words)})
	return len(words), nil
}

func TestLessonWords(t *testing.T) {
	t.Parallel()
	got := lessonWords([]Annotation{
		{Lemma: "Give up", MeaningVi: "từ bỏ"}, {Lemma: "park", MeaningVi: "công viên"},
		{Lemma: "give  up", MeaningVi: "bỏ"}, {Lemma: " "},
	}, []string{"Give Up"})
	want := []LessonWord{{"give up", "từ bỏ", true}, {"park", "công viên", false}}
	if !slices.Equal(got, want) {
		t.Fatalf("words = %+v", got)
	}
}

func TestUpdateAnnotationsSharesWords(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.annotated(t)
	v := &fakeVocab{}
	e.svc.Vocab = v
	if _, err := e.svc.UpdateAnnotations(t.Context(), l.ID, []AnnotationInput{
		{Text: "park", Lemma: "park", MeaningVi: "công viên"},
	}); err != nil {
		t.Fatal(err)
	}
	if len(v.calls) != 1 || v.calls[0].topic != "topic-b1" || v.calls[0].level != "B1" ||
		!slices.Equal(v.calls[0].words, []LessonWord{{Lemma: "park", MeaningVi: "công viên"}}) {
		t.Fatalf("calls = %+v", v.calls)
	}
}

func TestShareAllWords(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	v := &fakeVocab{fail: "broken"}
	e.svc.Vocab = v
	add := func(topic string, level Level, status Status, lemmas ...string) {
		var anns []Annotation
		for _, l := range lemmas {
			anns = append(anns, Annotation{Text: l, Lemma: l, MeaningVi: "nghĩa " + l})
		}
		if _, err := e.lessons.Create(t.Context(), Lesson{TopicID: topic, Level: level, AnnotationStatus: status, Annotations: anns}); err != nil {
			t.Fatal(err)
		}
	}
	add("t1", "C1", StatusDone, "negotiate", "leverage")
	add("t1", "C1", StatusDone, "leverage", "concede")
	add("t1", "B2", StatusDone, "deal")
	add("t2", "C1", StatusRunning, "ignored")
	add("broken", "C1", StatusDone, "lost")

	n, err := e.svc.ShareAllWords(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	// One call a topic and level; the failing topic does not stop the others.
	if n != 4 || len(v.calls) != 2 {
		t.Fatalf("added = %d, calls = %+v", n, v.calls)
	}
	for _, c := range v.calls {
		var lemmas []string
		for _, w := range c.words {
			lemmas = append(lemmas, w.Lemma)
		}
		slices.Sort(lemmas)
		switch c.level {
		case "C1":
			if !slices.Equal(lemmas, []string{"concede", "leverage", "negotiate"}) {
				t.Fatalf("C1 words = %v", lemmas)
			}
		case "B2":
			if !slices.Equal(lemmas, []string{"deal"}) {
				t.Fatalf("B2 words = %v", lemmas)
			}
		}
	}

	e.svc.Vocab = nil
	if n, err := e.svc.ShareAllWords(t.Context()); err != nil || n != 0 {
		t.Fatalf("without sink = %d, %v", n, err)
	}
}
