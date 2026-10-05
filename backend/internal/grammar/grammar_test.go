package grammar_test

import (
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/grammar"
)

func TestEmbeddedSyllabusIsValidAndComplete(t *testing.T) {
	t.Parallel()
	s := grammar.Default() // panics on an invalid embedded file
	if len(s.All()) == 0 {
		t.Fatal("empty syllabus")
	}
	for _, level := range grammar.Levels {
		n := len(s.ByLevel(level))
		if n < 6 {
			t.Errorf("level %s has only %d points", level, n)
		}
	}
	// The beginner levels are the ones lessons are written for: they need a full course.
	if n := len(s.ByLevel("A1")); n < 12 {
		t.Errorf("A1 has %d points, want at least 12", n)
	}
}

func TestPointsAreOrderedByLevelAndIDsMatchTheirLevel(t *testing.T) {
	t.Parallel()
	s := grammar.Default()
	rank := map[string]int{}
	for i, l := range grammar.Levels {
		rank[l] = i
	}
	prev := -1
	for _, p := range s.All() {
		if rank[p.Level] < prev {
			t.Errorf("%s (%s) comes after a higher level", p.ID, p.Level)
		}
		prev = rank[p.Level]
		if !strings.HasPrefix(p.ID, strings.ToLower(p.Level)+"-") {
			t.Errorf("id %q should start with %q", p.ID, strings.ToLower(p.Level)+"-")
		}
		if p.ID != strings.ToLower(p.ID) || strings.ContainsAny(p.ID, " _") {
			t.Errorf("id %q should be lowercase kebab-case", p.ID)
		}
	}
}

func TestGetAndByLevel(t *testing.T) {
	t.Parallel()
	s := grammar.Default()
	p, ok := s.Get("a1-to-be")
	if !ok || p.Level != "A1" || p.TitleEn == "" || len(p.Examples) == 0 {
		t.Fatalf("Get(a1-to-be) = %+v, %v", p, ok)
	}
	if _, ok := s.Get("nope"); ok {
		t.Error("Get(nope) found a point")
	}
	for _, p := range s.ByLevel("B1") {
		if p.Level != "B1" {
			t.Errorf("ByLevel(B1) returned %s", p.Level)
		}
	}
	if got := s.ByLevel("Z9"); len(got) != 0 {
		t.Errorf("ByLevel(Z9) = %v", got)
	}
}

func TestAllReturnsACopy(t *testing.T) {
	t.Parallel()
	s := grammar.Default()
	all := s.All()
	all[0].ID = "changed"
	if s.All()[0].ID == "changed" {
		t.Error("All() exposes the internal slice")
	}
}

func TestParseRejectsBrokenSyllabi(t *testing.T) {
	t.Parallel()
	good := `{"id":"a1-x","level":"A1","titleVi":"v","titleEn":"e","pattern":"p","hintVi":"h","examples":["x"]}`
	cases := map[string]string{
		"not json":      `{`,
		"no id":         `{"points":[{"level":"A1","titleVi":"v","titleEn":"e","pattern":"p","hintVi":"h","examples":["x"]}]}`,
		"unknown level": `{"points":[{"id":"x","level":"Z9","titleVi":"v","titleEn":"e","pattern":"p","hintVi":"h","examples":["x"]}]}`,
		"missing title": `{"points":[{"id":"a1-y","level":"A1","titleVi":"","titleEn":"e","pattern":"p","hintVi":"h","examples":["x"]}]}`,
		"no example":    `{"points":[{"id":"a1-y","level":"A1","titleVi":"v","titleEn":"e","pattern":"p","hintVi":"h","examples":[]}]}`,
		"duplicate id":  `{"points":[` + good + `,` + good + `]}`,
	}
	for name, data := range cases {
		if _, err := grammar.Parse([]byte(data)); err == nil {
			t.Errorf("%s: Parse accepted it", name)
		}
	}
	if _, err := grammar.Parse([]byte(`{"points":[` + good + `]}`)); err != nil {
		t.Errorf("valid syllabus rejected: %v", err)
	}
}
