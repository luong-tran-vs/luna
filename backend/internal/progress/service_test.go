package progress

import (
	"errors"
	"strings"
	"testing"
	"time"
)

var testNow = time.Date(2026, 9, 29, 8, 0, 0, 0, time.UTC)

func newTestService() (*Service, *fakeLessons) {
	lessons := &fakeLessons{lessons: map[string]lessonInfo{"l1": {revision: 1, sentences: 3}}}
	return NewService(newFakeRepo(), lessons, func() time.Time { return testNow }), lessons
}

func TestRecordValidation(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		in    Input
		field string
	}{
		"index negative":  {Input{SentenceIndex: -1, Typed: "a", CorrectWords: 1, TotalWords: 1}, "sentenceIndex"},
		"index too large": {Input{SentenceIndex: 3, Typed: "a", CorrectWords: 1, TotalWords: 1}, "sentenceIndex"},
		"typed empty":     {Input{Typed: "   ", CorrectWords: 0, TotalWords: 1}, "typed"},
		"typed too long":  {Input{Typed: strings.Repeat("á", 1001), CorrectWords: 0, TotalWords: 1}, "typed"},
		"total zero":      {Input{Typed: "a", CorrectWords: 0, TotalWords: 0}, "totalWords"},
		"total too large": {Input{Typed: "a", CorrectWords: 0, TotalWords: 1001}, "totalWords"},
		"correct < 0":     {Input{Typed: "a", CorrectWords: -1, TotalWords: 1}, "correctWords"},
		"correct > total": {Input{Typed: "a", CorrectWords: 3, TotalWords: 2}, "correctWords"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			svc, _ := newTestService()
			_, err := svc.Record(t.Context(), "u1", "l1", tc.in)
			var verr *ValidationError
			if !errors.As(err, &verr) || verr.Fields[tc.field] == "" {
				t.Fatalf("err = %v, want field %q", err, tc.field)
			}
		})
	}

	// Exactly 1000 characters is accepted.
	svc, _ := newTestService()
	in := Input{Typed: strings.Repeat("á", 1000), CorrectWords: 0, TotalWords: 1000}
	if _, err := svc.Record(t.Context(), "u1", "l1", in); err != nil {
		t.Fatalf("1000 chars: %v", err)
	}
}

func TestRecordUnknownLesson(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService()
	if _, err := svc.Record(t.Context(), "u1", "nope", Input{Typed: "a", TotalWords: 1}); !errors.Is(err, ErrLessonNotFound) {
		t.Fatalf("record: %v", err)
	}
	if _, err := svc.Summary(t.Context(), "u1", "nope"); !errors.Is(err, ErrLessonNotFound) {
		t.Fatalf("summary: %v", err)
	}
}

func TestRecordAndSummary(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService()
	ctx := t.Context()

	s, err := svc.Record(ctx, "u1", "l1", Input{SentenceIndex: 2, Typed: " we went ", CorrectWords: 1, TotalWords: 2})
	if err != nil {
		t.Fatal(err)
	}
	if s.SentenceCount != 3 || s.CheckedCount != 1 || s.CorrectWords != 1 || s.TotalWords != 2 || s.Rate != 0.5 || s.Completed {
		t.Fatalf("summary = %+v", s)
	}
	if r := s.Results[0]; r.Typed != "we went" || !r.CheckedAt.Equal(testNow) {
		t.Fatalf("result = %+v", r)
	}

	// Checking the same sentence again replaces its result.
	s, _ = svc.Record(ctx, "u1", "l1", Input{SentenceIndex: 2, Typed: "we went home", CorrectWords: 3, TotalWords: 3})
	if s.CheckedCount != 1 || s.CorrectWords != 3 || s.TotalWords != 3 {
		t.Fatalf("after re-check = %+v", s)
	}

	// Wrong sentences still count as checked.
	_, _ = svc.Record(ctx, "u1", "l1", Input{SentenceIndex: 0, Typed: "x", CorrectWords: 0, TotalWords: 4})
	s, _ = svc.Record(ctx, "u1", "l1", Input{SentenceIndex: 1, Typed: "y", CorrectWords: 2, TotalWords: 3})
	if !s.Completed || s.CheckedCount != 3 || s.CorrectWords != 5 || s.TotalWords != 10 || s.Rate != 0.5 {
		t.Fatalf("completed summary = %+v", s)
	}
	for i, r := range s.Results {
		if r.SentenceIndex != i {
			t.Fatalf("results not sorted: %+v", s.Results)
		}
	}

	// Another learner sees nothing.
	other, err := svc.Summary(ctx, "u2", "l1")
	if err != nil || other.CheckedCount != 0 || other.Rate != 0 || other.Completed || len(other.Results) != 0 {
		t.Fatalf("other user = %+v, %v", other, err)
	}
}

func TestSummaryIgnoresOldRevision(t *testing.T) {
	t.Parallel()
	svc, lessons := newTestService()
	ctx := t.Context()
	_, _ = svc.Record(ctx, "u1", "l1", Input{SentenceIndex: 2, Typed: "a", CorrectWords: 1, TotalWords: 1})

	// The admin edits the content: new revision with fewer sentences.
	lessons.set("l1", 2, 2)
	s, err := svc.Summary(ctx, "u1", "l1")
	if err != nil || s.CheckedCount != 0 || s.SentenceCount != 2 || len(s.Results) != 0 {
		t.Fatalf("after edit = %+v, %v", s, err)
	}
	s, _ = svc.Record(ctx, "u1", "l1", Input{SentenceIndex: 1, Typed: "b", CorrectWords: 1, TotalWords: 1})
	if s.CheckedCount != 1 {
		t.Fatalf("after new record = %+v", s)
	}
}

func TestSummaryEmptyLesson(t *testing.T) {
	t.Parallel()
	svc, lessons := newTestService()
	lessons.set("empty", 1, 0)
	s, err := svc.Summary(t.Context(), "u1", "empty")
	if err != nil || s.Completed || s.Results == nil {
		t.Fatalf("empty lesson = %+v, %v", s, err)
	}
}
