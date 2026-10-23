package llm

import (
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/ai"
)

func TestCurriculumPrompts(t *testing.T) {
	outline := outlinePrompt(ai.OutlineRequest{
		Level: "A2", MinTopics: 7, MaxTopics: 10,
		Points:      []ai.CurriculumPoint{{ID: "a2-past-simple-regular", Title: "Past simple", Pattern: "V-ed"}},
		Suggestions: []ai.CurriculumIdea{{Name: "Du lịch", Why: "kể chuyến đi"}},
		Topics:      []string{"Du lịch", "Gia đình"}, Earlier: []string{"A1: Gia đình"},
		PreviousErrors: []string{"there are 5 topics, write 7 to 10"},
	})
	for _, want := range []string{"CEFR level A2", "7 to 10 topics", "- a2-past-simple-regular: Past simple (V-ed)",
		"- Du lịch: kể chuyến đi", "Existing topics: Du lịch; Gia đình", "lower levels' courses: A1: Gia đình", "there are 5 topics"} {
		if !strings.Contains(outline, want) {
			t.Errorf("outline prompt lacks %q", want)
		}
	}
	slots := slotsPrompt(ai.SlotsRequest{
		Level: "A2", Main: 3, Extra: 5, Avoid: []string{"My Trip"},
		Topic: ai.OutlineTopic{Name: "Du lịch", CanDoVi: "kể về chuyến đi."},
	})
	for _, want := range []string{"Topic: Du lịch", "Write 8 lessons: the first 3", "the other 5", "do not reuse them: My Trip"} {
		if !strings.Contains(slots, want) {
			t.Errorf("slots prompt lacks %q", want)
		}
	}
}

func TestSlotsPromptExtraOnly(t *testing.T) {
	p := slotsPrompt(ai.SlotsRequest{Level: "A1", Extra: 3, Avoid: []string{"My Song"}, Topic: ai.OutlineTopic{Name: "Âm nhạc"}})
	if !strings.Contains(p, "Write 3 more extra lessons") || strings.Contains(p, "main lessons") {
		t.Errorf("extra-only prompt:\n%s", p)
	}
	if !strings.Contains(p, "do not reuse them: My Song") {
		t.Error("the topic's titles are to be avoided")
	}
}
