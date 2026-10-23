package llm

import (
	"fmt"
	"strings"
)

// levelProfiles say what learners of each CEFR level can do (after the CEFR global scale), so the AI
// judges what fits a level from more than the level's name.
var levelProfiles = map[string]string{
	"A1": "they understand and use familiar everyday expressions and very basic sentences about themselves and " +
		"their immediate needs (name, family, home, daily routine, simple shopping). They know only present tenses: " +
		"no past events or future plans.",
	"A2": "they talk in short simple sentences about routine matters and their surroundings: past events, plans, " +
		"shopping, work, travel and health, in simple terms.",
	"B1": "they deal with most situations while travelling, describe experiences, events, hopes and plans, and give " +
		"brief reasons and opinions on familiar topics such as work, study and free time.",
	"B2": "they discuss concrete and abstract topics, follow arguments, and explain a viewpoint with its advantages " +
		"and disadvantages.",
	"C1": "they use English flexibly for social, academic and work purposes, understand long demanding texts and " +
		"implied meaning, and write well-organised texts on complex subjects.",
	"C2": "they understand almost everything they read or hear, summarise and rebuild arguments, and express " +
		"themselves precisely with fine shades of meaning on complex subjects.",
}

func writeProfile(b *strings.Builder, level string) {
	if p, ok := levelProfiles[level]; ok {
		fmt.Fprintf(b, "\nWhat learners at %s can do: %s\n", level, p)
	}
}

// beginner reports whether the level is A1 or A2: questions are then written in Vietnamese, words kept
// everyday, and texts shaped for the grammar step (another person, a negative, a question).
func beginner(level string) bool { return level == "A1" || level == "A2" }

// modelWords is the length of a model writing, in words: the suggested length the writing page shows
// for the level (frontend SUGGESTED_WORDS).
func modelWords(level string) (lo, hi int) {
	switch level {
	case "A1":
		return 40, 60
	case "A2":
		return 50, 80
	case "B1":
		return 80, 120
	case "B2":
		return 120, 180
	default:
		return 150, 250
	}
}
