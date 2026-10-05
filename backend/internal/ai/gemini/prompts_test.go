package gemini

import (
	"fmt"
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/ai"
)

func TestAnnotationRangeFollowsTheLevel(t *testing.T) {
	t.Parallel()
	for level, want := range map[string][2]int{"A1": {6, 12}, "A2": {6, 12}, "B1": {8, 16}, "B2": {8, 16}, "C1": {8, 25}, "C2": {8, 25}} {
		if lo, hi := annotationRange(level); lo != want[0] || hi != want[1] {
			t.Errorf("annotationRange(%s) = %d, %d, want %v", level, lo, hi, want)
		}
	}
	for _, l := range []string{"A1", "B1", "C1"} {
		lo, hi := annotationRange(l)
		prompt := annotatePrompt(ai.AnnotateRequest{Level: l, Sentences: []string{"We went home."}})
		if want := fmt.Sprintf("pick %d to %d words", lo, hi); !strings.Contains(prompt, want) {
			t.Errorf("%s prompt missing %q", l, want)
		}
	}
}

func TestAnnotatePromptAsksForBetterContent(t *testing.T) {
	t.Parallel()
	prompt := annotatePrompt(ai.AnnotateRequest{Level: "A1", Sentences: []string{"We went home."}})
	for _, want := range []string{
		"Skip very basic words",                                           // no "family" for an A1 learner
		"what a word or phrase of the lesson means as used there",         // a vocabulary-in-context question
		"At most one may ask for a plain detail",                          // not only lookups
		"simple inference or a reason",                                    // an inference question
		"you must add the reason",                                         // explanations give the reason
		"at most 60 words",                                                // short grammar note for beginners
		"Choose the simple present tense only if nothing else stands out", // not the same note every time
		"Do not say how many sentences or words",                          // no clash with the length hint
	} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
	if b1 := annotatePrompt(ai.AnnotateRequest{Level: "B1", Sentences: []string{"x"}}); !strings.Contains(b1, "at most 90 words") {
		t.Error("B1 grammar note should allow 90 words")
	}
}

func TestGeneratePromptAsksForFullLengthNaturalText(t *testing.T) {
	t.Parallel()
	prompt := generatePrompt(ai.GenerateRequest{
		Level: "A1", TopicName: "Đồ ăn", Count: 2, Words: 120, Kind: ai.KindReading,
		TargetWords: [][]string{{"rice", "dish"}},
	})
	for _, want := range []string{
		"between 108 and 132 words", "long end of that range", "shorter than 108 words is too short",
		"natural, connected text", "There are\nfour people in my family",
		"Leave a word out only if it cannot be used naturally at CEFR A1",
	} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q:\n%s", want, prompt)
		}
	}
}

func TestAnnotatePromptTeachesTheGrammarFocus(t *testing.T) {
	t.Parallel()
	g := &ai.GrammarFocus{TitleVi: "Động từ to be (am / is / are)", TitleEn: "The verb to be", Pattern: "I am / you are / he, she, it is"}
	prompt := annotatePrompt(ai.AnnotateRequest{Level: "A1", Sentences: []string{"I am Nam."}, GrammarFocus: g})
	for _, want := range []string{
		"teach exactly this grammar point", `exactly "Động từ to be (am / is / are)"`, "I am / you are / he, she, it is",
		"return an empty array", "at most 60 words", "4. writingPrompt",
	} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q:\n%s", want, prompt)
		}
	}
	if strings.Contains(prompt, "Choose the simple present tense") {
		t.Error("focused prompt must not let the AI pick another point")
	}
	if plain := annotatePrompt(ai.AnnotateRequest{Level: "A1", Sentences: []string{"x"}}); strings.Contains(plain, "teach exactly") {
		t.Error("prompt without a focus must keep the free choice")
	}
}

func TestGeneratePromptUsesTheGrammarFocus(t *testing.T) {
	t.Parallel()
	g := &ai.GrammarFocus{TitleVi: "Thì hiện tại đơn", TitleEn: "Present simple", Pattern: "I, you + verb", Example: "I get up at six."}
	prompt := generatePrompt(ai.GenerateRequest{Level: "A1", TopicName: "Food", Count: 1, Words: 100, GrammarFocus: g})
	for _, want := range []string{"at least 3 times", "Present simple", "I, you + verb", "I get up at six.", "stays at CEFR A1"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
	if strings.Contains(generatePrompt(ai.GenerateRequest{Level: "A1", TopicName: "Food", Count: 1, Words: 100}), "Grammar focus") {
		t.Error("no focus, no grammar requirement")
	}
}

func TestGrammarLessonPromptTeachesOnlyThePoint(t *testing.T) {
	t.Parallel()
	prompt := grammarLessonPrompt(ai.GrammarLessonRequest{
		Level: "A1", TitleVi: "Động từ to be", TitleEn: "Verb to be", Pattern: "S + am/is/are",
		HintVi: "Nói về danh tính", Examples: []string{"I am a student."},
	})
	for _, want := range []string{
		"CEFR level A1", "Verb to be (Động từ to be)", "S + am/is/are", "Nói về danh tính", "- I am a student.",
		"simple, short Vietnamese", "only vocabulary and grammar suitable for CEFR A1",
		"mix the three kinds", "explanationVi", "different sentences from the practice", "exactly 8 exercises", "exactly 6 exercises",
		"choice:", "fill:", "reorder:", `"___"`,
	} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
}

func TestGrammarSolvePromptHidesTheKeyAndAsksForHonestAmbiguity(t *testing.T) {
	t.Parallel()
	prompt := grammarSolvePrompt(ai.SolveRequest{
		Level: "A1", TitleEn: "Verb to be", Pattern: "S + be",
		Exercises: []ai.SolveExercise{{ID: "p1", Kind: "choice", Text: "She ___ a nurse.", Options: []string{"am", "is", "are", "be"}}},
	})
	for _, want := range []string{
		"careful English teacher", "you have no answer key", "do not pick the most likely-looking option",
		"choiceIndex", "answer", "sentence", "ambiguous", "noteVi", "Do not mark an exercise ambiguous just to be safe",
		`"id":"p1"`, "She ___ a nurse.",
	} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
	for _, key := range []string{"answerIndex", "explanationVi"} {
		if strings.Contains(prompt, key+`":`) {
			t.Errorf("prompt leaks %q", key)
		}
	}
}

func TestReviewPromptHidesTheKeysAndReportsOnlyWhatIsSure(t *testing.T) {
	t.Parallel()
	prompt := reviewPrompt(ai.ReviewRequest{
		Level: "A2", Title: "At the park",
		Sentences:    []string{"Mai: We went to the park.", "He gave up smoking."},
		Annotations:  []ai.ReviewAnnotation{{Index: 0, Text: "gave up", Lemma: "give up", MeaningVi: "từ bỏ", SentenceIndex: 1}},
		Questions:    []ai.ReviewQuestion{{Index: 0, Prompt: "Where did they go?", Options: []string{"Park", "Home"}}},
		Translations: []ai.ReviewTranslation{{Index: 0, Vi: "Tôi đi công viên.", En: "I go to the park."}},
	})
	for _, want := range []string{
		"careful English teacher", "CEFR level A2", `"At the park"`, "You have no answer key", "Do not guess what the author intended",
		"Report only what you are sure about", "Name: sentence", "Do not correct the spelling of names",
		"sentences:", "annotations:", "translations:", "answers:", "choiceIndex", "ambiguous", "noteVi",
		"Do not mark a question ambiguous just to be safe", "empty arrays",
		`{"index":1,"text":"He gave up smoking."}`, `"meaningVi":"từ bỏ"`, `"prompt":"Where did they go?"`, `"en":"I go to the park."`,
	} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
	for _, key := range []string{"answerIndex", "explanationVi"} {
		if strings.Contains(prompt, key+`":`) {
			t.Errorf("prompt leaks %q", key)
		}
	}
	// Lists with nothing in them are sent as [], never null.
	empty := reviewPrompt(ai.ReviewRequest{Level: "A1", Title: "x"})
	if strings.Contains(empty, "null") {
		t.Errorf("empty prompt has null:\n%s", empty)
	}
}
