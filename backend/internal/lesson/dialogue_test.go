package lesson

import (
	"reflect"
	"testing"
)

// sentencesOf splits content the way a saved lesson is split.
func sentencesOf(content string) []Sentence {
	return toSentences(SplitSentences(content))
}

func TestDialogueTurns(t *testing.T) {
	t.Parallel()
	content := "Minh: Hi, Anna. How are you?\nAnna: I'm fine, thanks!\r\n\nMinh: Great. Mary Jane: Me too.\nMary Jane: Me too."
	// "Mary Jane: Me too." inside Minh's line is just text; the next line is a third speaker.
	got := DialogueTurns(content, sentencesOf(content))
	want := []TextTurn{
		{Speaker: "Minh", Text: "Hi, Anna. How are you?", Sentences: []int{0, 1}},
		{Speaker: "Anna", Text: "I'm fine, thanks!", Sentences: []int{2}},
		{Speaker: "Minh", Text: "Great. Mary Jane: Me too.", Sentences: []int{3, 4}},
		{Speaker: "Mary Jane", Text: "Me too.", Sentences: []int{5}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("turns = %+v", got)
	}
}

func TestDialogueTurnsNotADialogue(t *testing.T) {
	t.Parallel()
	for name, content := range map[string]string{
		"reading text":        "We went to the park. It was sunny.\n\nWe had lunch there.",
		"one line":            "Minh: Hello there.",
		"one speaker":         "Minh: Hello.\nMinh: Anyone here?",
		"narration line":      "Minh: Hello.\nThey shake hands.\nAnna: Hi.",
		"lowercase name":      "minh: Hello.\nanna: Hi.",
		"note in a reading":   "Note: this is a text.\nIt has two lines.",
		"line runs into next": "Minh: Hello Anna\nAnna: Hi.",
		"too many speakers":   "A: One.\nB: Two.\nC: Three.\nD: Four.\nE: Five.",
		"empty":               "",
	} {
		if got := DialogueTurns(content, sentencesOf(content)); got != nil {
			t.Errorf("%s: turns = %+v", name, got)
		}
	}
}

func TestDialogueTurnsNeedTheSavedSentences(t *testing.T) {
	t.Parallel()
	content := "Minh: Hello.\nAnna: Hi."
	// Sentences that no longer match the content (edited elsewhere) give no turns.
	if got := DialogueTurns(content, toSentences([]string{"Minh: Hello.", "Anna: Hey."})); got != nil {
		t.Fatalf("turns = %+v", got)
	}
}
