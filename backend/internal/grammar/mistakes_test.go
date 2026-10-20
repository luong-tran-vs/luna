package grammar_test

import (
	"testing"

	"github.com/luongtran/luna/backend/internal/grammar"
)

func TestEmbeddedMistakesCoverEveryA1Point(t *testing.T) {
	t.Parallel()
	// MistakesOf panics on an invalid embedded file (unknown point, missing sentence).
	for _, p := range grammar.Default().ByLevel("A1") {
		if len(grammar.MistakesOf(p.ID)) == 0 {
			t.Errorf("A1 point %s has no hand-written mistake", p.ID)
		}
	}
	got := grammar.MistakesOf("a1-to-be")
	if len(got) < 3 || got[0].Wrong != "I happy." || got[0].Right != "I am happy." {
		t.Fatalf("a1-to-be = %+v", got)
	}
	if len(grammar.MistakesOf("c2-register")) != 0 {
		t.Fatal("c2-register has mistakes")
	}
}

func TestParseMistakesRejectsBrokenLists(t *testing.T) {
	t.Parallel()
	s := grammar.Default()
	for name, data := range map[string]string{
		"not json":      `{`,
		"unknown point": `[{"points":["a1-nope"],"wrong":"a","right":"b","noteVi":"c"}]`,
		"no point":      `[{"points":[],"wrong":"a","right":"b","noteVi":"c"}]`,
		"same":          `[{"points":["a1-to-be"],"wrong":"a","right":"a","noteVi":"c"}]`,
		"no note":       `[{"points":["a1-to-be"],"wrong":"a","right":"b","noteVi":" "}]`,
	} {
		if _, err := grammar.ParseMistakes([]byte(data), s); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}
