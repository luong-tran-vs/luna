package llm

import (
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/ai"
)

func TestToeicPrompt(t *testing.T) {
	t.Parallel()
	got := toeicPrompt(ai.ToeicGroupsRequest{
		Part: 3, Count: 4, Level: 750, Topic: "a delayed delivery",
		Tags:     []ai.ToeicTag{{Code: "detail", Label: "detail"}, {Code: "graphic", Label: "question about the graphic"}},
		Existing: []string{"Hi, I'd like to pick up the brochures."},
	})
	for _, want := range []string{
		"senior item writer", "Write exactly 4 new groups of Part 3.", "TOEIC 750 (upper-intermediate)",
		`Set "level" to 750`, "PART 3 - CONVERSATIONS", "detail (detail), graphic (question about the graphic)",
		"Exactly one key", "annual = \"hằng năm\"", "FINAL CHECK", "ADMIN'S FOCUS", "a delayed delivery",
		"pick up the brochures", `"Look at the graphic."`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("prompt misses %q", want)
		}
	}
	if strings.Contains(toeicPrompt(ai.ToeicGroupsRequest{Part: 5, Count: 10, Level: 600}), "ADMIN'S FOCUS") {
		t.Error("no topic: the focus section must be left out")
	}
}

func TestToeicPromptPart7AsksForQuestions(t *testing.T) {
	t.Parallel()
	got := toeicPrompt(ai.ToeicGroupsRequest{Part: 7, Questions: 10, Level: 900})
	for _, want := range []string{"exactly 10 questions in total", "add up to EXACTLY 10", "TOEIC 900 (advanced)", "cross-reference"} {
		if !strings.Contains(got, want) {
			t.Errorf("part 7 prompt misses %q", want)
		}
	}
	if strings.Contains(got, "%!") {
		t.Error("format verb left in the prompt")
	}
}

func TestToeicPromptHasEveryPartAndLevel(t *testing.T) {
	t.Parallel()
	for part := 1; part <= 7; part++ {
		for _, level := range []int{450, 600, 750, 900} {
			got := toeicPrompt(ai.ToeicGroupsRequest{Part: part, Count: 3, Questions: 10, Level: level})
			if !strings.Contains(got, "PART ") || strings.Contains(got, "%!") {
				t.Fatalf("part %d level %d: bad prompt", part, level)
			}
		}
	}
}
