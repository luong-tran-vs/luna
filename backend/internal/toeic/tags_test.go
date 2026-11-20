package toeic

import (
	"testing"

	"github.com/luongtran/luna/backend/internal/grammar"
)

func TestTags(t *testing.T) {
	t.Parallel()
	seen := map[string]bool{}
	syllabus := grammar.Default()
	for _, tag := range Tags {
		if seen[tag.Code] {
			t.Errorf("duplicate tag %q", tag.Code)
		}
		seen[tag.Code] = true
		if tag.LabelVi == "" || len(tag.Parts) == 0 {
			t.Errorf("tag %q has no label or part", tag.Code)
		}
		for _, p := range tag.Parts {
			if !p.Valid() {
				t.Errorf("tag %q: part %d", tag.Code, p)
			}
		}
		for _, id := range tag.GrammarPoints {
			if _, ok := syllabus.Get(id); !ok {
				t.Errorf("tag %q: grammar point %q is not in the syllabus", tag.Code, id)
			}
		}
	}
	if !TagAllowed("word-form", 5) || TagAllowed("word-form", 7) || TagAllowed("nope", 5) {
		t.Fatal("TagAllowed")
	}
	if !TagAllowed("sentence-insert", 6) || TagAllowed("sentence-insert", 5) {
		t.Fatal("sentence-insert is Part 6/7 only")
	}
}
