package lesson

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/job"
)

// fakeGlosses keeps the glosses of each lesson with their revision.
type fakeGlosses struct {
	mu   sync.Mutex
	rows map[string]struct {
		revision int
		glosses  []Gloss
	}
}

func newFakeGlosses() *fakeGlosses {
	return &fakeGlosses{rows: map[string]struct {
		revision int
		glosses  []Gloss
	}{}}
}

func (f *fakeGlosses) Get(_ context.Context, id string) (int, []Gloss, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.rows[id]
	return r.revision, r.glosses, ok, nil
}

func (f *fakeGlosses) Save(_ context.Context, id string, revision int, gs []Gloss) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows[id] = struct {
		revision int
		glosses  []Gloss
	}{revision, gs}
	return nil
}

func (f *fakeGlosses) Revisions(context.Context) (map[string]int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[string]int{}
	for id, r := range f.rows {
		out[id] = r.revision
	}
	return out, nil
}

// fakeGlossAI answers with glosses, or fails with err, and counts the requests.
type fakeGlossAI struct {
	mu      sync.Mutex
	glosses []ai.Gloss
	err     error
	calls   int
}

func (f *fakeGlossAI) Glosses(context.Context, ai.GlossRequest) ([]ai.Gloss, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	return f.glosses, f.err
}

func TestCleanGlosses(t *testing.T) {
	t.Parallel()
	sentences := []string{"It is a great way to relax after school.", "Music makes my life fun."}
	got := CleanGlosses([]ai.Gloss{
		{Text: "relax", Lemma: "Relax", MeaningVi: " thư giãn ", SentenceIndex: 0, POS: "verb",
			Phrase: "a great way to relax", PhraseVi: "một cách tuyệt vời để thư giãn"},
		{Text: "relax", Lemma: "relax", MeaningVi: "nghỉ", SentenceIndex: 0},                   // the same word again
		{Text: "life", Lemma: "life", MeaningVi: "cuộc sống", SentenceIndex: 1, Phrase: "x y"}, // a phrase not in the sentence
		{Text: "piano", Lemma: "piano", MeaningVi: "đàn", SentenceIndex: 1},                    // not in that sentence
		{Text: "fun", Lemma: "fun", MeaningVi: "", SentenceIndex: 1},                           // no meaning
		{Text: "school", Lemma: "school", MeaningVi: "trường", SentenceIndex: 7},               // no such sentence
	}, sentences)
	want := []Gloss{
		{Text: "relax", Lemma: "relax", MeaningVi: "thư giãn", SentenceIndex: 0, POS: "verb",
			Phrase: "a great way to relax", PhraseVi: "một cách tuyệt vời để thư giãn"},
		{Text: "life", Lemma: "life", MeaningVi: "cuộc sống", SentenceIndex: 1},
	}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("got %+v", got)
	}
}

func TestProcessGlosses(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	glosses, glossAI := newFakeGlosses(), &fakeGlossAI{glosses: []ai.Gloss{
		{Text: "park", Lemma: "park", MeaningVi: "công viên", SentenceIndex: 0, Phrase: "went to the park", PhraseVi: "đã đi công viên"},
	}}
	e.svc.Glosses, e.svc.GlossAI = glosses, glossAI
	l := e.annotated(t)

	// Annotating queues the glosses, which one AI request writes.
	queued := jobsOf(e, job.TypeGlosses)
	if len(queued) != 1 || queued[0].LessonID != l.ID {
		t.Fatalf("glosses jobs = %+v", queued)
	}
	if err := e.svc.ProcessGlosses(t.Context(), queued[0]); err != nil {
		t.Fatal(err)
	}
	rev, gs, ok, _ := glosses.Get(t.Context(), l.ID)
	if !ok || rev != l.Revision || len(gs) != 1 || gs[0].MeaningVi != "công viên" || glossAI.calls != 1 {
		t.Fatalf("saved %d %+v %v, calls %d", rev, gs, ok, glossAI.calls)
	}

	// Already written for this revision: not queued again at start.
	if n, err := e.svc.QueueMissingGlosses(t.Context()); err != nil || n != 0 {
		t.Fatalf("queued again %d, %v", n, err)
	}
}

func TestGlossesFailureIsRecordedOnce(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	glosses := newFakeGlosses()
	e.svc.Glosses, e.svc.GlossAI = glosses, &fakeGlossAI{err: ai.ErrInvalidKey}
	l := e.annotated(t)
	j := jobsOf(e, job.TypeGlosses)[0]
	err := e.svc.ProcessGlosses(t.Context(), j)
	if !errors.Is(err, ai.ErrInvalidKey) {
		t.Fatalf("err = %v", err)
	}
	e.svc.JobFailed(t.Context(), j, err)
	// An empty list for the revision: the popup falls back to the dictionary, nothing is queued again.
	if rev, gs, ok, _ := glosses.Get(t.Context(), l.ID); !ok || rev != l.Revision || len(gs) != 0 {
		t.Fatalf("recorded %d %+v %v", rev, gs, ok)
	}
	if n, _ := e.svc.QueueMissingGlosses(t.Context()); n != 0 {
		t.Fatalf("queued again %d", n)
	}
}

func TestQueueMissingGlosses(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	old := e.annotated(t) // annotated before glosses were on: none queued
	e.create(t)           // not annotated yet: nothing to write
	glosses := newFakeGlosses()
	e.svc.Glosses, e.svc.GlossAI = glosses, &fakeGlossAI{}
	if n, err := e.svc.QueueMissingGlosses(t.Context()); err != nil || n != 1 {
		t.Fatalf("queued %d, %v", n, err)
	}
	if jobs := jobsOf(e, job.TypeGlosses); len(jobs) != 1 || jobs[0].LessonID != old.ID {
		t.Fatalf("jobs = %+v", jobs)
	}
	// Glosses of an older revision count as none.
	_ = glosses.Save(t.Context(), old.ID, old.Revision-1, []Gloss{})
	if n, _ := e.svc.QueueMissingGlosses(t.Context()); n != 1 {
		t.Fatalf("stale revision: queued %d", n)
	}
}

func TestLookupGivesTheMeaningInTheLesson(t *testing.T) {
	t.Parallel()
	r, l := newReaderEnv(t)
	glosses := newFakeGlosses()
	r.WithGlosses(glosses)
	_ = glosses.Save(t.Context(), l.ID, l.Revision, []Gloss{
		{Text: "park", Lemma: "park", MeaningVi: "công viên", SentenceIndex: 0, Phrase: "went to the park", PhraseVi: "đã đi công viên"},
		{Text: "went", Lemma: "go", MeaningVi: "đi", SentenceIndex: 0, Phrase: "We went", PhraseVi: "Chúng tôi đã đi"},
	})

	// A word the annotations do not cover: its gloss, then the dictionary's other meanings.
	got, err := r.Lookup(t.Context(), l.ID, "park", 0)
	if err != nil {
		t.Fatal(err)
	}
	if got.Source != SourceAI || got.Context == nil || *got.Context != (LookupContext{"công viên", "went to the park", "đã đi công viên"}) ||
		len(got.Meanings) != 1 || got.Meanings[0].Text != "công viên" || len(got.Others) != 1 || got.Others[0].Text != "Bãi đỗ xe." {
		t.Fatalf("park = %+v (context %+v)", got, got.Context)
	}

	// An annotated word: the annotation's meaning (the admin may have edited it), the gloss's phrase.
	got, _ = r.Lookup(t.Context(), l.ID, "went", 0)
	if got.Context == nil || got.Context.MeaningVi != "đã đi (tới công viên)" || got.Context.Phrase != "We went" ||
		len(got.Others) != 1 || got.Others[0].Text != "Đi, đi đến." {
		t.Fatalf("went = %+v (context %+v)", got, got.Context)
	}

	// Glosses of another revision are not used: the dictionary answers.
	_ = glosses.Save(t.Context(), l.ID, l.Revision+1, nil)
	if got, _ = r.Lookup(t.Context(), l.ID, "park", 0); got.Source != SourceDictionary || got.Context != nil {
		t.Fatalf("stale = %+v", got)
	}
}
