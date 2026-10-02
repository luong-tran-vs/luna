package lesson

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/luongtran/luna/backend/internal/ai"
)

type askEnv struct {
	r       *Reader
	lessons *fakeLessons
	asks    *fakeAsks
	ai      *fakeAI
	lesson  Lesson
}

// newAskEnv is a reader over readingContent (sentence 1: "He gave up smoking.") at revision 3.
func newAskEnv(t *testing.T) *askEnv {
	t.Helper()
	e := &askEnv{lessons: newFakeLessons(), asks: newFakeAsks(), ai: &fakeAI{
		explanation: ai.Explanation{Lemma: "give up", MeaningVi: "bỏ (thói quen)", NoteVi: "Ở đây là bỏ hút thuốc."},
	}}
	e.lesson, _ = e.lessons.Create(context.Background(), Lesson{
		Title: "Park", Level: "B1", TopicID: "topic-a1", Content: readingContent, Revision: 3,
		Sentences: toSentences(SplitSentences(readingContent)),
	})
	e.r = NewReader(e.lessons, readingDict, newFakeTopics(), newFakeAnswers(), e.asks, e.ai, "")
	return e
}

func (e *askEnv) ask(t *testing.T, text string, sentence int) (LookupResult, bool, error) {
	t.Helper()
	return e.r.Ask(t.Context(), e.lesson.ID, text, sentence)
}

// --- US1 ---

func TestAskCallsAIAndStores(t *testing.T) {
	t.Parallel()
	e := newAskEnv(t)

	res, cached, err := e.ask(t, "gave up", 1)
	if err != nil || cached {
		t.Fatalf("ask = %+v cached %v, %v", res, cached, err)
	}
	if res.Source != SourceAI || res.Text != "gave up" || res.Lemma != "give up" || len(res.Meanings) != 1 ||
		res.Meanings[0].Text != "bỏ (thói quen)" || res.Note != "Ở đây là bỏ hút thuốc." {
		t.Fatalf("result = %+v", res)
	}
	calls, req := e.ai.explains()
	if calls != 1 || req.Text != "gave up" || req.Sentence != "He gave up smoking." || req.Level != "B1" {
		t.Fatalf("AI calls %d request %+v", calls, req)
	}
	stored, ok, _ := e.asks.Get(t.Context(), AskKey{LessonID: e.lesson.ID, Revision: 3, SentenceIndex: 1, Text: "gave up"})
	if !ok || stored.MeaningVi != "bỏ (thói quen)" {
		t.Fatalf("stored = %+v %v", stored, ok)
	}
}

func TestAskEmptyLemmaAndIPA(t *testing.T) {
	t.Parallel()
	e := newAskEnv(t)
	e.ai.explanation = ai.Explanation{MeaningVi: "đã đi", NoteVi: "Quá khứ của go."}
	res, _, err := e.ask(t, "went", 0)
	if err != nil || res.Lemma != "went" {
		t.Fatalf("empty lemma = %+v, %v", res, err)
	}
	e.ai.explanation = ai.Explanation{Lemma: "go", MeaningVi: "đi", NoteVi: "n"}
	res, _, _ = e.ask(t, "went", 3)
	if res.Lemma != "go" || res.IPA == "" {
		t.Fatalf("IPA from the dictionary: %+v", res)
	}
}

func TestAskValidation(t *testing.T) {
	t.Parallel()
	e := newAskEnv(t)
	for _, c := range []struct {
		text     string
		sentence int
		field    string
	}{
		{"  ", 1, "text"},
		{strings.Repeat("a", 101), 1, "text"},
		{"one two three four five six seven", 1, "text"},
		{"banana", 1, "text"},
		{"went", 1, "text"}, // in the lesson, but not in that sentence
		{"gave up", 9, "sentenceIndex"},
		{"gave up", -1, "sentenceIndex"},
	} {
		_, _, err := e.ask(t, c.text, c.sentence)
		var verr *ValidationError
		if !errors.As(err, &verr) || verr.Fields[c.field] == "" {
			t.Errorf("%q in %d: %v, want field %s", c.text, c.sentence, err, c.field)
		}
	}
	if calls, _ := e.ai.explains(); calls != 0 {
		t.Fatalf("AI called for invalid input: %d", calls)
	}
	if _, _, err := e.r.Ask(t.Context(), "missing", "x", 0); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing lesson: %v", err)
	}
}

// --- US2 ---

func TestAskUsesStoredResult(t *testing.T) {
	t.Parallel()
	e := newAskEnv(t)
	if _, _, err := e.ask(t, "gave up", 1); err != nil {
		t.Fatal(err)
	}
	res, cached, err := e.ask(t, "  Gave   UP ", 1)
	if err != nil || !cached || res.Meanings[0].Text != "bỏ (thói quen)" || res.Note == "" {
		t.Fatalf("second ask = %+v cached %v, %v", res, cached, err)
	}
	if calls, _ := e.ai.explains(); calls != 1 {
		t.Fatalf("AI calls = %d, want 1", calls)
	}

	// Another sentence is another key.
	if _, cached, _ := e.ask(t, "we went", 3); cached {
		t.Fatal("sentence 3 used another sentence's result")
	}
	if calls, _ := e.ai.explains(); calls != 2 {
		t.Fatalf("AI calls = %d, want 2", calls)
	}
}

func TestAskConcurrentCallsShareOneAICall(t *testing.T) {
	t.Parallel()
	e := newAskEnv(t)
	gate := make(chan struct{})
	e.ai.explainGate = gate

	var wg sync.WaitGroup
	results := make([]LookupResult, 10)
	errs := make([]error, 10)
	for i := range 10 {
		wg.Go(func() {
			results[i], _, errs[i] = e.r.Ask(context.Background(), e.lesson.ID, "gave up", 1)
		})
	}
	// Let every goroutine reach the shared call before the AI answers.
	for {
		if calls, _ := e.ai.explains(); calls > 0 {
			break
		}
	}
	close(gate)
	wg.Wait()

	if calls, _ := e.ai.explains(); calls != 1 {
		t.Fatalf("AI calls = %d, want 1", calls)
	}
	for i := range 10 {
		if errs[i] != nil || results[i].Meanings[0].Text != "bỏ (thói quen)" {
			t.Fatalf("goroutine %d: %+v, %v", i, results[i], errs[i])
		}
	}
	if e.asks.count() != 1 {
		t.Fatalf("stored = %d", e.asks.count())
	}
}

func TestAskNewRevisionAsksAgain(t *testing.T) {
	t.Parallel()
	e := newAskEnv(t)
	_, _, _ = e.ask(t, "gave up", 1)
	l, _ := e.lessons.Get(t.Context(), e.lesson.ID)
	l.Revision = 4
	_ = e.lessons.ReplaceContent(t.Context(), l)
	if _, cached, err := e.ask(t, "gave up", 1); err != nil || cached {
		t.Fatalf("after a new revision: cached %v, %v", cached, err)
	}
	if calls, _ := e.ai.explains(); calls != 2 {
		t.Fatalf("AI calls = %d, want 2", calls)
	}
}

func TestLookupReturnsStoredAsk(t *testing.T) {
	t.Parallel()
	e := newAskEnv(t)
	if _, err := e.r.Lookup(t.Context(), e.lesson.ID, "smoking", 1); err == nil {
		t.Fatal("smoking is in the fake dictionary? the test needs an unknown word")
	}
	e.ai.explanation = ai.Explanation{Lemma: "smoking", MeaningVi: "việc hút thuốc", NoteVi: "Danh động từ."}
	_, _, _ = e.ask(t, "smoking", 1)

	res, err := e.r.Lookup(t.Context(), e.lesson.ID, "Smoking", 1)
	if err != nil || res.Source != SourceAI || res.Meanings[0].Text != "việc hút thuốc" || res.Note != "Danh động từ." {
		t.Fatalf("lookup = %+v, %v", res, err)
	}
	// Another sentence: not the stored answer.
	if _, err := e.r.Lookup(t.Context(), e.lesson.ID, "smoking", 0); !errors.Is(err, ErrLookupNotFound) {
		t.Fatalf("other sentence: %v", err)
	}
	// The dictionary answer for "went" stays when nothing was asked for it.
	if res, _ := e.r.Lookup(t.Context(), e.lesson.ID, "went", 0); res.Source != SourceDictionary || res.Note != "" {
		t.Fatalf("went = %+v", res)
	}
}

// --- US3 ---

func TestAskErrorsAreNotStored(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name string
		res  ai.Explanation
		err  error
		want error
	}{
		{"not configured", ai.Explanation{}, ai.ErrNotConfigured, ai.ErrNotConfigured},
		{"quota", ai.Explanation{}, ai.ErrQuota, ai.ErrQuota},
		{"empty meaning", ai.Explanation{Lemma: "give up", MeaningVi: " "}, nil, ErrUnusableExplanation},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			e := newAskEnv(t)
			e.ai.explanation, e.ai.explainErr = c.res, c.err
			if _, _, err := e.ask(t, "gave up", 1); !errors.Is(err, c.want) {
				t.Fatalf("err = %v, want %v", err, c.want)
			}
			if e.asks.count() != 0 {
				t.Fatal("a failed answer was stored")
			}
			// A later ask tries the AI again.
			e.ai.explanation, e.ai.explainErr = ai.Explanation{MeaningVi: "bỏ"}, nil
			if _, cached, err := e.ask(t, "gave up", 1); err != nil || cached {
				t.Fatalf("retry: cached %v, %v", cached, err)
			}
		})
	}
}

func TestLookupStoredAskRules(t *testing.T) {
	t.Parallel()
	e := newAskEnv(t)
	e.ai.explanation = ai.Explanation{Lemma: "smoking", MeaningVi: "việc hút thuốc", NoteVi: "n"}
	_, _, _ = e.ask(t, "smoking", 1)

	// No sentence given: the stored answer belongs to a sentence, so it is not used.
	if _, err := e.r.Lookup(t.Context(), e.lesson.ID, "smoking", -1); !errors.Is(err, ErrLookupNotFound) {
		t.Fatalf("without sentence: %v", err)
	}

	// Annotations still come first.
	l, _ := e.lessons.Get(t.Context(), e.lesson.ID)
	l.Annotations = []Annotation{{Text: "smoking", Lemma: "smoke", MeaningVi: "hút thuốc", SentenceIndex: 1}}
	_ = e.lessons.ReplaceContent(t.Context(), l)
	if res, _ := e.r.Lookup(t.Context(), e.lesson.ID, "smoking", 1); res.Meanings[0].Text != "hút thuốc" || res.Note != "" {
		t.Fatalf("annotation first: %+v", res)
	}

	// A new revision ignores the stored answer.
	l.Annotations, l.Revision = nil, 4
	_ = e.lessons.ReplaceContent(t.Context(), l)
	if _, err := e.r.Lookup(t.Context(), e.lesson.ID, "smoking", 1); !errors.Is(err, ErrLookupNotFound) {
		t.Fatalf("old revision: %v", err)
	}
}
