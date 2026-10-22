package gemini

import (
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/ai"
)

func TestPracticePromptAsksForTheSteps(t *testing.T) {
	t.Parallel()
	prompt := practicePrompt(ai.PracticeRequest{Level: "A1", Title: "Music", Sentences: []string{"I like music."},
		TargetWords: []string{"music", "song"}})
	for _, want := range []string{
		"listeningQuestions", "readingGuideVi", "grammarStep", "readAloud", "speakQuestions", "writingHelp",
		`kind "tfn"`, "The words the lesson teaches most: music, song.",
		"in Vietnamese (prompt and options)", // beginners hear the dialogue, they do not read English questions
	} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
	if strings.Contains(prompt, "in simple English") {
		t.Error("A1 questions asked in English")
	}

	b1 := practicePrompt(ai.PracticeRequest{Level: "B1", Sentences: []string{"I like music."}})
	if !strings.Contains(b1, "in simple English") || strings.Contains(b1, "in Vietnamese (prompt and options)") {
		t.Error("B1 questions not asked in simple English")
	}
	if strings.Contains(b1, "The words the lesson teaches most") {
		t.Error("target words line without target words")
	}
}

func TestPracticePromptGrammarChoices(t *testing.T) {
	t.Parallel()
	choices := []ai.GrammarChoice{
		{ID: "a1-present-simple", TitleEn: "Present simple", Pattern: "I play / she plays"},
		{ID: "a1-to-be", TitleEn: "The verb to be", Pattern: "I am / you are"},
	}
	prompt := practicePrompt(ai.PracticeRequest{Level: "A1", GrammarChoices: choices})
	for _, want := range []string{
		"grammarPointId", "- a1-present-simple: Present simple (I play / she plays)", "- a1-to-be: The verb to be (I am / you are)",
		"the main grammar point the lesson sentences use",
	} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q", want)
		}
	}

	// With a grammar point the choices are not listed.
	focused := practicePrompt(ai.PracticeRequest{Level: "A1", GrammarChoices: choices,
		GrammarFocus: &ai.GrammarFocus{TitleEn: "Present simple", Pattern: "I play / she plays"}})
	if strings.Contains(focused, "a1-to-be") || strings.Contains(focused, "return its id") {
		t.Error("choices listed with a grammar focus")
	}
	if !strings.Contains(focused, "the lesson's grammar point, Present simple (I play / she plays)") {
		t.Error("grammar step not on the focus")
	}
}

func TestPracticeSchemaHasTheSteps(t *testing.T) {
	t.Parallel()
	props := practiceSchema["properties"].(map[string]any)
	for _, k := range []string{"listeningQuestions", "readingGuideVi", "grammarStep", "readAloud", "speakQuestions", "writingHelp", "grammarPointId"} {
		if _, ok := props[k]; !ok {
			t.Errorf("practice schema missing %s", k)
		}
	}
	kind := props["listeningQuestions"].(map[string]any)["items"].(map[string]any)["properties"].(map[string]any)["kind"].(map[string]any)
	if enum := kind["enum"].([]string); strings.Join(enum, ",") != "gist,detail,tfn" {
		t.Errorf("listening kinds = %v", enum)
	}
}

func TestSpeakPromptListsTheAnswers(t *testing.T) {
	t.Parallel()
	prompt := speakPrompt(ai.SpeakCheckRequest{Level: "A2", Answers: []ai.SpeakAnswer{
		{Question: "What music do you like?", Said: "I like pop music"},
		{Question: "Who is your friend?", Said: "she name Lan"},
	}})
	for _, want := range []string{
		"CEFR level A2",
		"0. Question: What music do you like?\n   Answer: I like pop music",
		"1. Question: Who is your friend?\n   Answer: she name Lan",
		"recognition slip", // speech recognition errors are not mistakes
	} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
}

func TestGradeSchemaHasMistakes(t *testing.T) {
	t.Parallel()
	props := gradeSchema["properties"].(map[string]any)
	m, ok := props["mistakes"].(map[string]any)
	if !ok {
		t.Fatal("grade schema missing mistakes")
	}
	types := m["items"].(map[string]any)["properties"].(map[string]any)["type"].(map[string]any)["enum"].([]string)
	if len(types) != 11 || types[len(types)-1] != "other" {
		t.Errorf("mistake types = %v", types)
	}
	required := strings.Join(gradeSchema["required"].([]string), ",")
	if !strings.Contains(required, "mistakes") {
		t.Errorf("required = %s", required)
	}
	if prompt := gradePrompt(ai.GradeRequest{Level: "A1", Text: "I very like music."}); !strings.Contains(prompt, "mistakes: every correction") {
		t.Error("grade prompt does not ask for mistakes")
	}
}
