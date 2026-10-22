package lesson

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/ai"
)

// simplifyingAI is the fake AI with the Simplifier of the Gemini client.
type simplifyingAI struct {
	*fakeAI
	texts []ai.SimplifiedText
	err   error
	reqs  []ai.SimplifyRequest
}

func (s *simplifyingAI) SimplifyWords(_ context.Context, req ai.SimplifyRequest) ([]ai.SimplifiedText, error) {
	s.reqs = append(s.reqs, req)
	return s.texts, s.err
}

func TestHardWords(t *testing.T) {
	t.Parallel()
	got := hardWords("I feel anxious and abandoned. Anxious Minh is here.", "A1", []string{"anxious"})
	if !slices.Equal(got, []string{"abandoned"}) {
		t.Errorf("hardWords = %q (target words and names are allowed)", got)
	}
	if got := hardWords("I feel anxious.", "B2", nil); len(got) != 0 {
		t.Errorf("nothing is above B2: %q", got)
	}
}

func TestGenerateSimplifiesHardWords(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	fake := &simplifyingAI{fakeAI: e.ai}
	e.svc.AI = fake
	e.ai.drafts = []ai.LessonDraft{
		{Title: "Anxious", Content: "She is anxious. " + text(97)},
		{Title: "Abandon", Content: "We abandon it. " + text(97)},
		{Title: "Easy", Content: text(100)},
	}
	fake.texts = []ai.SimplifiedText{
		{Index: 0, Text: "She is sad. " + text(97)},
		{Index: 1, Text: "We abandon them. " + text(97)}, // still hard: kept as written
		{Index: 7, Text: "out of range"},
	}
	res, err := e.svc.Generate(t.Context(), "topic-a1", validGenerate())
	if err != nil {
		t.Fatal(err)
	}
	if len(fake.reqs) != 1 || len(fake.reqs[0].Texts) != 2 || fake.reqs[0].Level != "A1" ||
		!slices.Equal(fake.reqs[0].Texts[0].Words, []string{"anxious"}) {
		t.Fatalf("one request for the two hard drafts: %+v", fake.reqs)
	}
	if d := res.Drafts[0]; !strings.HasPrefix(d.Content, "She is sad.") || len(d.HardWords) != 0 || d.Words != 100 {
		t.Errorf("draft 0 = %+v", d)
	}
	if d := res.Drafts[1]; !strings.HasPrefix(d.Content, "We abandon it.") || !slices.Equal(d.HardWords, []string{"abandon"}) {
		t.Errorf("draft 1 = %+v", d)
	}
	if d := res.Drafts[2]; len(d.HardWords) != 0 {
		t.Errorf("draft 2 = %+v", d)
	}

	// A failed request keeps the drafts and their warnings.
	fake.err, fake.reqs = errors.New("down"), nil
	res, err = e.svc.Generate(t.Context(), "topic-a1", validGenerate())
	if err != nil || !slices.Equal(res.Drafts[0].HardWords, []string{"anxious"}) || !strings.HasPrefix(res.Drafts[0].Content, "She is anxious.") {
		t.Fatalf("after a failure: %+v, %v", res.Drafts[0], err)
	}
}

func TestSimplifyRejectsTooShortOrLosingTargets(t *testing.T) {
	t.Parallel()
	s := &Service{Deps: Deps{AI: &simplifyingAI{fakeAI: &fakeAI{}, texts: []ai.SimplifiedText{
		{Index: 0, Text: "Short."},
		{Index: 1, Text: "We like it. " + text(97)},
	}}}}
	drafts := []Draft{
		{Content: "She is anxious. " + text(97), Words: 100},
		{Content: "We abandon the piano. " + text(96), Words: 100, index: 1},
	}
	s.simplifyDrafts(t.Context(), "A1", drafts, [][]string{nil, {"piano"}})
	if !strings.HasPrefix(drafts[0].Content, "She is anxious.") || !strings.HasPrefix(drafts[1].Content, "We abandon the piano.") {
		t.Errorf("drafts = %q / %q", drafts[0].Content, drafts[1].Content)
	}
}
