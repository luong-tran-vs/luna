package games

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

func presentSimple() PointSource {
	return PointSource{
		PointID: "a1-present-simple", Level: "A1", TitleVi: "Hiện tại đơn", TitleEn: "Present simple", Pattern: "S + V(s/es)",
		HintVi: "Thói quen, sự thật.", Examples: []string{"I play tennis."},
		Exercises: []SourceExercise{
			{ID: "p1", Kind: "choice", Text: "Hoa says to ___.", Options: []string{"they", "them", "their"}, AnswerIndex: 1, ExplanationVi: "Sau giới từ dùng tân ngữ."},
			{ID: "p2", Kind: "choice", Text: "She ___ to school every day.", Options: []string{"goes", "go", "going", "went"}, AnswerIndex: 0, ExplanationVi: "She + V-s."},
			{ID: "p3", Kind: "choice", Text: "Which sentence is right?", Options: []string{"a", "b", "c"}, AnswerIndex: 0},
			{ID: "p4", Kind: "choice", Text: "play___ football", Options: []string{"s", "es", "ing"}, AnswerIndex: 0},
			{ID: "m1", Kind: "fix", Text: "He play football.", Sentence: "He plays football.", ExplanationVi: "He + V-s."},
			{ID: "m2", Kind: "fix", Text: "They goes home.", Sentence: "They go home.", ExplanationVi: "They + V."},
			{ID: "m3", Kind: "fix", Text: "We is not at the park.", Sentence: "We are never in a park.", ExplanationVi: "x"},
		},
		Mistakes: []PointMistake{
			{Wrong: "I happy.", Right: "I am happy.", NoteVi: "Cần be."},
			{Wrong: "I am go to school.", Right: "I go to school.", NoteVi: "Bỏ am."},
		},
	}
}

func pastSimple() PointSource {
	return PointSource{
		PointID: "a2-past-simple", Level: "A2", TitleVi: "Quá khứ đơn",
		Exercises: []SourceExercise{
			{ID: "p1", Kind: "choice", Text: "Yesterday I ___ a cake.", Options: []string{"made", "make", "makes"}, AnswerIndex: 0},
			{ID: "p2", Kind: "choice", Text: "She ___ home late last night.", Options: []string{"came", "come", "comes"}, AnswerIndex: 0},
			{ID: "p3", Kind: "choice", Text: "We ___ a film on Sunday.", Options: []string{"saw", "see", "sees"}, AnswerIndex: 0},
		},
	}
}

func TestTokenizeAndDiff(t *testing.T) {
	t.Parallel()
	want := []string{`"`, "Hoa", "says", "to", "them", ",", `"`, "–", "he", "aren't", "well-known", "."}
	if got := tokenize(`"Hoa says to them," – he aren't well-known.`); !slices.Equal(got, want) {
		t.Fatalf("tokens = %q", got)
	}
	a, b := tokenize("She do not likes it."), tokenize("She does not like it.")
	wa, wb := diffSpan(a, b)
	if wa != (Span{1, 4}) || wb != (Span{1, 4}) {
		t.Fatalf("diff = %v %v", wa, wb)
	}
}

func TestCandidates(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c := candidatesOf(presentSimple(), nil, e.svc.newRand())
	byKey := map[string]*spotCandidate{}
	for _, w := range c.wrong {
		byKey[w.key] = w
	}
	// (a) "Hoa says to ___": a wrong option shown, the three options to fix it.
	p1 := byKey["a1-present-simple/p1"]
	if p1 == nil {
		t.Fatal("no choice sentence")
	}
	shown := strings.Join(p1.q.Tokens[p1.q.Wrong.From:p1.q.Wrong.To], " ")
	if shown != "they" && shown != "their" {
		t.Fatalf("shown = %q", shown)
	}
	opts := slices.Sorted(slices.Values(p1.q.Options))
	if !slices.Equal(opts, []string{"their", "them", "they"}) || p1.q.Options[p1.q.Answer] != "them" || p1.q.TopicVi != "Hiện tại đơn" {
		t.Fatalf("p1 = %+v", p1.q)
	}
	// p2 has four options: the right one, the wrong one shown and one more.
	if p2 := byKey["a1-present-simple/p2"]; p2 == nil || len(p2.q.Options) != 3 || p2.q.Options[p2.q.Answer] != "goes" ||
		!slices.Contains(p2.q.Options, p2.q.Tokens[p2.q.Wrong.From]) {
		t.Fatalf("p2 = %+v", p2)
	}
	// No blank, or a blank inside a word: left out.
	if byKey["a1-present-simple/p3"] != nil || byKey["a1-present-simple/p4"] != nil {
		t.Fatal("unusable choice kept")
	}
	// (b) a fix differing by one word, with a third option from another sentence.
	m1 := byKey["a1-present-simple/m1"]
	if m1 == nil || strings.Join(m1.q.Tokens, " ") != "He play football ." || m1.q.Options[m1.q.Answer] != "plays" ||
		!slices.Contains(m1.q.Options, "play") || len(m1.q.Options) != 3 || m1.q.Source != SourceFix {
		t.Fatalf("m1 = %+v", m1)
	}
	// Four differing words, a missing word (nothing to tap) or an extra one (no word to fix it with):
	// left out.
	for _, k := range []string{"m3", "mistake0", "mistake1"} {
		if byKey["a1-present-simple/"+k] != nil {
			t.Fatalf("unusable %s kept", k)
		}
	}
	if byKey["a1-present-simple/m2"] == nil {
		t.Fatal("one-word fix dropped")
	}
	// Right sentences: the choices filled in, the corrections and the examples.
	rights := map[string]bool{}
	for _, r := range c.right {
		if r.q.Wrong != nil || len(r.q.Options) != 0 {
			t.Fatalf("right sentence with a mistake: %+v", r.q)
		}
		rights[strings.Join(r.q.Tokens, " ")] = true
	}
	for _, s := range []string{"Hoa says to them .", "He plays football .", "I play tennis ."} {
		if !rights[s] {
			t.Fatalf("missing right sentence %q in %v", s, rights)
		}
	}

	// (c) approved extra sentences.
	items := []SpotItem{{ID: "x1", PointID: "a1-present-simple", Sentence: "My dog eat bones.", Wrong: "eat", Right: "eats",
		Distractors: []string{"eating", "ate"}, RuleVi: "r", Status: SpotApproved}}
	c = candidatesOf(presentSimple(), items, e.svc.newRand())
	found := false
	for _, w := range c.wrong {
		if w.key == "spot/x1" {
			found = true
			if w.q.Options[w.q.Answer] != "eats" || w.q.Tokens[w.q.Wrong.From] != "eat" {
				t.Fatalf("spot = %+v", w.q)
			}
		}
	}
	if !found {
		t.Fatal("spot item not used")
	}
}

func TestGrammarRound(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.grammar.sources = []PointSource{presentSimple()}
	if _, err := e.svc.GrammarRound(t.Context(), "u1"); !errors.Is(err, ErrNotEnough) {
		t.Fatalf("too few sentences: %v", err)
	}
	e.grammar.sources = append(e.grammar.sources, pastSimple())
	for i, s := range []string{"My dog eat bones.", "Tom like milk.", "She watch TV every night.", "The sun rise in the east."} {
		w := strings.Fields(s)[1]
		if i >= 2 {
			w = strings.Fields(s)[2]
			if i == 2 {
				w = strings.Fields(s)[1]
			}
		}
		e.spots.items = append(e.spots.items, SpotItem{ID: string(rune('a' + i)), PointID: "a1-present-simple", Sentence: s, Wrong: w,
			Right: w + "s", Distractors: []string{w + "ing", w + "ed"}, RuleVi: "r", Status: SpotApproved})
	}
	e.spots.items = append(e.spots.items, SpotItem{ID: "draft", PointID: "a1-present-simple", Sentence: "Ann swim well.", Wrong: "swim",
		Right: "swims", Distractors: []string{"swam", "swimming"}, RuleVi: "r", Status: SpotDraft})
	e.grammar.started["u1"] = []string{"a1-present-simple"}

	qs, err := e.svc.GrammarRound(t.Context(), "u1")
	if err != nil || len(qs) != spotQuestions {
		t.Fatalf("round = %d, %v", len(qs), err)
	}
	right, started := 0, 0
	for _, q := range qs {
		if q.Wrong == nil {
			right++
		} else if len(q.Options) != 3 || q.Answer < 0 || q.Wrong.To <= q.Wrong.From {
			t.Fatalf("question = %+v", q)
		}
		if q.PointID == "a1-present-simple" {
			started++
		}
		if strings.Contains(strings.Join(q.Tokens, " "), "swim") {
			t.Fatal("a draft was played")
		}
	}
	if right != spotRight {
		t.Fatalf("%d right sentences", right)
	}
	if started < 7 {
		t.Fatalf("only %d sentences of the started point", started)
	}
}
