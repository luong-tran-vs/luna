package mysql

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/lesson"
)

func aiLookupResult(lessonID, text, meaning string) lesson.AskResult {
	return lesson.AskResult{
		AskKey: lesson.AskKey{LessonID: lessonID, Revision: 2, SentenceIndex: 1, Text: text},
		Lemma:  "go", MeaningVi: meaning, NoteVi: "ghi chú", CreatedAt: time.Date(2026, 9, 30, 8, 0, 0, 5000, time.UTC),
	}
}

func TestAILookupsPutGet(t *testing.T) {
	t.Parallel()
	r := NewAILookups(testDB(t))
	ctx := t.Context()
	l := newID()
	res := aiLookupResult(l, "went", "đã đi")

	if _, ok, err := r.Get(ctx, res.AskKey); err != nil || ok {
		t.Fatalf("Get before Put = %v, %v", ok, err)
	}
	put, err := r.Put(ctx, res)
	if err != nil || put != res {
		t.Fatalf("Put = %+v, %v", put, err)
	}
	got, ok, err := r.Get(ctx, res.AskKey)
	if err != nil || !ok {
		t.Fatalf("Get = %v, %v", ok, err)
	}
	if got.AskKey != res.AskKey || got.Lemma != "go" || got.MeaningVi != "đã đi" || got.NoteVi != "ghi chú" || !got.CreatedAt.Equal(res.CreatedAt) {
		t.Fatalf("got = %+v", got)
	}
}

func TestAILookupsKeyParts(t *testing.T) {
	t.Parallel()
	r := NewAILookups(testDB(t))
	ctx := t.Context()
	l := newID()
	res := aiLookupResult(l, "went", "a")
	if _, err := r.Put(ctx, res); err != nil {
		t.Fatal(err)
	}
	for name, key := range map[string]lesson.AskKey{
		"revision": {LessonID: l, Revision: 3, SentenceIndex: 1, Text: "went"},
		"sentence": {LessonID: l, Revision: 2, SentenceIndex: 0, Text: "went"},
		"text":     {LessonID: l, Revision: 2, SentenceIndex: 1, Text: "go"},
		"lesson":   {LessonID: newID(), Revision: 2, SentenceIndex: 1, Text: "went"},
	} {
		if _, ok, err := r.Get(ctx, key); err != nil || ok {
			t.Fatalf("%s: Get = %v, %v", name, ok, err)
		}
		other := res
		other.AskKey = key
		other.MeaningVi = name
		if stored, err := r.Put(ctx, other); err != nil || stored.MeaningVi != name {
			t.Fatalf("%s: Put = %+v, %v", name, stored, err)
		}
	}
}

func TestAILookupsPutReturnsTheStoredOne(t *testing.T) {
	t.Parallel()
	r := NewAILookups(testDB(t))
	ctx := t.Context()
	l := newID()
	first, _ := r.Put(ctx, aiLookupResult(l, "went", "first"))
	second, err := r.Put(ctx, aiLookupResult(l, "went", "second"))
	if err != nil || second.MeaningVi != "first" || !second.CreatedAt.Equal(first.CreatedAt) {
		t.Fatalf("second = %+v, %v", second, err)
	}
	got, _, _ := r.Get(ctx, first.AskKey)
	if got.MeaningVi != "first" {
		t.Fatalf("stored = %+v", got)
	}
}

func TestAILookupsConcurrentPutKeepsOne(t *testing.T) {
	t.Parallel()
	db := testDB(t)
	r := NewAILookups(db)
	ctx := t.Context()
	l := newID()
	const n = 10
	results := make([]lesson.AskResult, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := r.Put(ctx, aiLookupResult(l, "went", string(rune('a'+i))))
			if err != nil {
				t.Error(err)
			}
			results[i] = res
		}()
	}
	wg.Wait()
	for _, res := range results {
		if res.MeaningVi != results[0].MeaningVi {
			t.Fatalf("callers saw different winners: %q vs %q", res.MeaningVi, results[0].MeaningVi)
		}
	}
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM ai_lookups").Scan(&count); err != nil || count != 1 {
		t.Fatalf("rows = %d, %v", count, err)
	}
}

func TestAILookupsMalformedLessonID(t *testing.T) {
	t.Parallel()
	r := NewAILookups(testDB(t))
	ctx := t.Context()
	res := aiLookupResult("not-an-id", "went", "x")
	if _, ok, err := r.Get(ctx, res.AskKey); err != nil || ok {
		t.Fatalf("Get = %v, %v", ok, err)
	}
	if _, err := r.Put(ctx, res); !errors.Is(err, lesson.ErrNotFound) {
		t.Fatalf("Put = %v", err)
	}
}
