package grammar

import (
	"fmt"
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/ai"
)

func validContent() Content {
	return FilterGenerated(fromAI(aiContent()))
}

func TestFilterGeneratedKeepsGoodContentAndAssignsIDs(t *testing.T) {
	t.Parallel()
	c := validContent()
	if f := Validate(c); f != nil {
		t.Fatalf("valid content rejected: %v", f)
	}
	if got := ids(c.Practice); got[0] != "p1" || got[7] != "p8" {
		t.Errorf("practice ids = %v", got)
	}
	if got := ids(c.Mastery); got[0] != "m1" || got[5] != "m6" {
		t.Errorf("mastery ids = %v", got)
	}
}

func TestValidateReportsEveryKindOfProblem(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("x", 401)
	tests := []struct {
		name   string
		mutate func(c *Content)
		key    string
	}{
		{"empty objective", func(c *Content) { c.Objective = "" }, "content.objective"},
		{"long objective", func(c *Content) { c.Objective = long }, "content.objective"},
		{"no explanation", func(c *Content) { c.Explanation = nil }, "content.explanation"},
		{"seven explanations", func(c *Content) { c.Explanation = []string{"a", "b", "c", "d", "e", "f", "g"} }, "content.explanation"},
		{"empty paragraph", func(c *Content) { c.Explanation = []string{""} }, "content.explanation[0]"},
		{"long paragraph", func(c *Content) { c.Explanation = []string{strings.Repeat("x", 1201)} }, "content.explanation[0]"},
		{"no usage", func(c *Content) { c.Usage = nil }, "content.usage"},
		{"empty usage", func(c *Content) { c.Usage = []string{" "} }, "content.usage[0]"},
		{"no structures", func(c *Content) { c.Structures = nil }, "content.structures"},
		{"structure without pattern", func(c *Content) { c.Structures[0].Pattern = "" }, "content.structures[0].pattern"},
		{"too few examples", func(c *Content) { c.Examples = c.Examples[:2] }, "content.examples"},
		{"example without vi", func(c *Content) { c.Examples[1].Vi = "" }, "content.examples[1].vi"},
		{"too many mistakes", func(c *Content) { c.Mistakes = make([]Mistake, 7) }, "content.mistakes"},
		{"mistake without note", func(c *Content) { c.Mistakes[0].NoteVi = "" }, "content.mistakes[0].noteVi"},
		{"too few practice", func(c *Content) { c.Practice = c.Practice[:5] }, "content.practice"},
		{"too many practice", func(c *Content) { c.Practice = append(c.Practice, c.Practice[:5]...) }, "content.practice"},
		{"too few mastery", func(c *Content) { c.Mastery = c.Mastery[:4] }, "content.mastery"},
		{"unknown kind", func(c *Content) { c.Practice[2].Kind = "essay" }, "content.practice[2].kind"},
		{"no explanationVi", func(c *Content) { c.Practice[0].ExplanationVi = "" }, "content.practice[0].explanationVi"},
		{"no text", func(c *Content) { c.Mastery[1].Text = "" }, "content.mastery[1].text"},
		{"three options", func(c *Content) { c.Practice[0].Options = []string{"a", "b", "c"} }, "content.practice[0].options"},
		{"duplicate options", func(c *Content) { c.Practice[0].Options = []string{"is", "IS", "are", "be"} }, "content.practice[0].options"},
		{"empty option", func(c *Content) { c.Practice[0].Options[3] = "" }, "content.practice[0].options"},
		{"answer out of range", func(c *Content) { c.Practice[0].AnswerIndex = 4 }, "content.practice[0].answerIndex"},
		{"negative answer", func(c *Content) { c.Practice[0].AnswerIndex = -1 }, "content.practice[0].answerIndex"},
		{"fill without blank", func(c *Content) { c.Practice[1].Text = "I am a student." }, "content.practice[1].text"},
		{"fill with two blanks", func(c *Content) { c.Practice[1].Text = "I ___ a ___." }, "content.practice[1].text"},
		{"fill without answer", func(c *Content) { c.Practice[1].Answers = nil }, "content.practice[1].answers"},
		{"fill with empty answer", func(c *Content) { c.Practice[1].Answers = []string{""} }, "content.practice[1].answers"},
		{"reorder without sentence", func(c *Content) { c.Practice[2].Sentence = "" }, "content.practice[2].sentence"},
		{"reorder sentence too short", func(c *Content) {
			c.Practice[2].Sentence = "Hello there"
			c.Practice[2].Words = []string{"Hello", "there"}
		}, "content.practice[2].sentence"},
		{"reorder sentence too long", func(c *Content) {
			c.Practice[2].Sentence = strings.Repeat("go ", 15)
			c.Practice[2].Words = strings.Fields(c.Practice[2].Sentence)
		}, "content.practice[2].sentence"},
		{"reorder missing word", func(c *Content) { c.Practice[2].Words = []string{"teacher", "a", "She"} }, "content.practice[2].words"},
		{"reorder repeated word short", func(c *Content) {
			c.Practice[2].Sentence = "She is is happy"
			c.Practice[2].Words = []string{"She", "is", "happy"}
		}, "content.practice[2].words"},
		{"reorder three distractors", func(c *Content) {
			c.Practice[2].Words = append(c.Practice[2].Words, "x", "y")
		}, "content.practice[2].words"},
		{"reorder tile with space", func(c *Content) { c.Practice[2].Words[0] = "a b" }, "content.practice[2].words"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := validContent()
			tt.mutate(&c)
			fields := Validate(c)
			if _, ok := fields[tt.key]; !ok {
				t.Fatalf("want error at %q, got %v", tt.key, fields)
			}
		})
	}
}

func TestValidateAcceptsLimits(t *testing.T) {
	t.Parallel()
	c := validContent()
	c.Mistakes = nil // mistakes are optional
	c.Practice = append(c.Practice, c.Practice[:4]...)
	c.Mastery = append(c.Mastery, c.Mastery[:4]...)
	c.Practice[2].Words = []string{"teacher", "a", "She", "is", "x", "y"} // two distractors
	if f := Validate(Normalize(c)); f != nil {
		t.Fatalf("limits rejected: %v", f)
	}
}

func TestFilterGeneratedDropsBrokenExercises(t *testing.T) {
	t.Parallel()
	raw := aiContent()
	raw.Practice = append(raw.Practice,
		mod(goodChoice("bad1"), func(e *ai.GrammarExercise) { e.Options = []string{"a", "b", "c"} }),
		mod(goodChoice("bad2"), func(e *ai.GrammarExercise) { e.AnswerIndex = 7 }),
		mod(goodChoice("bad3"), func(e *ai.GrammarExercise) { e.Options = []string{"x", "x", "y", "z"} }),
		mod(goodFill("bad4"), func(e *ai.GrammarExercise) { e.Text = "no blank here bad4" }),
		mod(goodFill("bad5"), func(e *ai.GrammarExercise) { e.Answers = nil }),
		mod(goodReorder("bad6"), func(e *ai.GrammarExercise) { e.Words = []string{"She"} }),
		mod(goodChoice("bad7"), func(e *ai.GrammarExercise) { e.ExplanationVi = "" }),
	)
	c := FilterGenerated(fromAI(raw))
	if len(c.Practice) != 8 {
		t.Fatalf("practice = %d, want the 8 good ones", len(c.Practice))
	}
	for _, e := range c.Practice {
		if strings.Contains(e.Text, "bad") {
			t.Errorf("broken exercise kept: %+v", e)
		}
	}
	if f := Validate(c); f != nil {
		t.Errorf("filtered content invalid: %v", f)
	}
}

func TestFilterGeneratedMasteryDiffersFromPractice(t *testing.T) {
	t.Parallel()
	raw := aiContent()
	raw.Mastery = append(exercises("p", 2), raw.Mastery...) // two copies of practice questions
	c := FilterGenerated(fromAI(raw))
	if len(c.Mastery) != 6 {
		t.Fatalf("mastery = %d, want 6", len(c.Mastery))
	}
	seen := map[string]bool{}
	for _, e := range c.Practice {
		seen[strings.ToLower(e.Text)] = true
	}
	for _, e := range c.Mastery {
		if seen[strings.ToLower(e.Text)] {
			t.Errorf("mastery repeats practice: %q", e.Text)
		}
	}
}

func TestFilterGeneratedCutsListsAndDropsEmptyItems(t *testing.T) {
	t.Parallel()
	raw := aiContent()
	for range 10 {
		raw.Examples = append(raw.Examples, ai.GrammarExample{En: "x", Vi: "y"})
	}
	raw.Examples = append(raw.Examples, ai.GrammarExample{Vi: "no english"})
	raw.Practice = append(raw.Practice, exercises("q", 8)...)
	c := FilterGenerated(fromAI(raw))
	if len(c.Examples) != 12 || len(c.Practice) != 12 {
		t.Errorf("examples = %d, practice = %d, want 12 and 12", len(c.Examples), len(c.Practice))
	}
	if f := Validate(c); f != nil {
		t.Errorf("invalid: %v", f)
	}
}

func TestFilterGeneratedTooLittleLeftIsInvalid(t *testing.T) {
	t.Parallel()
	raw := aiContent()
	for i := range raw.Mastery {
		raw.Mastery[i].Kind = "choice"
		raw.Mastery[i].Options = []string{"a"} // every choice is broken
	}
	c := FilterGenerated(fromAI(raw))
	if f := Validate(c); f["content.mastery"] == "" {
		t.Fatalf("want a mastery count error, got %v", f)
	}
}

func TestNormalizeClearsFieldsOfOtherKinds(t *testing.T) {
	t.Parallel()
	c := Normalize(Content{Practice: []Exercise{{Kind: KindFill, Text: " I ___ ", Options: []string{"x"}, AnswerIndex: 2, Answers: []string{" am "}}}})
	e := c.Practice[0]
	if e.Options != nil || e.AnswerIndex != 0 || e.Answers[0] != "am" || e.Text != "I ___" || e.ID != "p1" {
		t.Errorf("exercise = %+v", e)
	}
	if c.Explanation == nil || c.Mastery == nil || c.Structures == nil {
		t.Error("lists must never be nil")
	}
}

func TestNormalizeIDsAreByPosition(t *testing.T) {
	t.Parallel()
	c := Normalize(Content{Practice: make([]Exercise, 3), Mastery: make([]Exercise, 2)})
	if got := fmt.Sprint(ids(c.Practice), ids(c.Mastery)); got != "[p1 p2 p3] [m1 m2]" {
		t.Errorf("ids = %s", got)
	}
}
