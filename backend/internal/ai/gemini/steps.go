package gemini

import (
	"fmt"
	"strings"

	"github.com/luongtran/luna/backend/internal/ai"
)

// The parts of the six-step lesson (added 2026-10-22) asked in the practice request, so a lesson
// still costs one request for its practice.

func init() {
	props := practiceSchema["properties"].(map[string]any)
	for k, v := range stepsSchema {
		props[k] = v
	}
}

var stepsSchema = map[string]any{
	"listeningQuestions": map[string]any{
		"type": "ARRAY",
		"items": map[string]any{
			"type": "OBJECT",
			"properties": map[string]any{
				"kind":          map[string]any{"type": "STRING", "enum": []string{"gist", "detail", "tfn"}},
				"prompt":        map[string]any{"type": "STRING"},
				"promptVi":      map[string]any{"type": "STRING"},
				"options":       stringArray,
				"answerIndex":   map[string]any{"type": "INTEGER"},
				"explanationVi": map[string]any{"type": "STRING"},
				"turnIndex":     map[string]any{"type": "INTEGER"},
			},
			"required": []string{"kind", "prompt", "answerIndex", "explanationVi", "turnIndex"},
		},
	},
	"readingGuideVi": map[string]any{"type": "STRING"},
	"grammarStep": map[string]any{
		"type": "OBJECT",
		"properties": map[string]any{
			"titleVi": map[string]any{"type": "STRING"},
			"noticing": map[string]any{
				"type": "ARRAY",
				"items": map[string]any{
					"type": "OBJECT",
					"properties": map[string]any{
						"sentence":   map[string]any{"type": "STRING"},
						"highlights": stringArray,
					},
					"required": []string{"sentence", "highlights"},
				},
			},
			"questionVi":    map[string]any{"type": "STRING"},
			"options":       stringArray,
			"answerIndex":   map[string]any{"type": "INTEGER"},
			"explanationVi": map[string]any{"type": "STRING"},
			"whenVi":        map[string]any{"type": "STRING"},
			"table": map[string]any{
				"type": "ARRAY",
				"items": map[string]any{
					"type": "OBJECT",
					"properties": map[string]any{
						"subject":  map[string]any{"type": "STRING"},
						"form":     map[string]any{"type": "STRING"},
						"negative": map[string]any{"type": "STRING"},
					},
					"required": []string{"subject", "form", "negative"},
				},
			},
			"mistakes": map[string]any{
				"type": "ARRAY",
				"items": map[string]any{
					"type": "OBJECT",
					"properties": map[string]any{
						"wrong": map[string]any{"type": "STRING"},
						"right": map[string]any{"type": "STRING"},
					},
					"required": []string{"wrong", "right"},
				},
			},
			"noteVi":    map[string]any{"type": "STRING"},
			"exercises": map[string]any{"type": "ARRAY", "items": grammarExerciseSchema},
		},
		"required": []string{"titleVi", "noticing", "questionVi", "options", "answerIndex", "explanationVi",
			"whenVi", "table", "mistakes", "exercises"},
	},
	"readAloud": map[string]any{
		"type": "ARRAY",
		"items": map[string]any{
			"type": "OBJECT",
			"properties": map[string]any{
				"text":  map[string]any{"type": "STRING"},
				"tipVi": map[string]any{"type": "STRING"},
			},
			"required": []string{"text", "tipVi"},
		},
	},
	"speakQuestions": map[string]any{
		"type": "ARRAY",
		"items": map[string]any{
			"type": "OBJECT",
			"properties": map[string]any{
				"question":   map[string]any{"type": "STRING"},
				"questionVi": map[string]any{"type": "STRING"},
				"frame":      map[string]any{"type": "STRING"},
			},
			"required": []string{"question", "questionVi", "frame"},
		},
	},
	"writingHelp": map[string]any{
		"type": "OBJECT",
		"properties": map[string]any{
			"grammarLabelVi": map[string]any{"type": "STRING"},
			"anyOf":          stringArray,
			"outlineVi":      stringArray,
			"modelAnswer":    map[string]any{"type": "STRING"},
		},
		"required": []string{"grammarLabelVi", "anyOf", "outlineVi", "modelAnswer"},
	},
	"grammarPointId": map[string]any{"type": "STRING"},
}

// beginner reports whether questions for the level are written in Vietnamese, so that they test
// listening, not reading English.
func beginner(level string) bool { return level == "A1" || level == "A2" }

// stepsPrompt asks for the listening questions, the reading guide, the grammar step, the speaking
// step and the writing help.
func stepsPrompt(req ai.PracticeRequest) string {
	var b strings.Builder
	lang := "in simple English, with promptVi its Vietnamese translation"
	if beginner(req.Level) {
		lang = "in Vietnamese (prompt and options), with promptVi empty"
	}
	fmt.Fprintf(&b, `
6. listeningQuestions: exactly 4 questions on the dialogue of part 3, which the learner hears without
seeing it. Write the gist and detail questions %[1]s.
- 1 question of kind "gist": what the whole dialogue is about; 3 options; turnIndex -1.
- 2 questions of kind "detail": one fact said in one turn; 3 options; turnIndex is that turn (from 0).
- 1 question of kind "tfn": prompt is one short English statement about the dialogue and promptVi its
  Vietnamese translation; options empty; answerIndex 0 when the dialogue says it, 1 when the dialogue
  says the opposite, 2 when the dialogue does not talk about it; turnIndex the turn that shows it, -1
  when not said.
- Wrong options must be plausible (same topic), never silly. Put the right answer at different
  positions. explanationVi: one Vietnamese sentence quoting the words of the dialogue that give the answer.

7. readingGuideVi: one short Vietnamese question the learner keeps in mind while reading the lesson
sentences (for example "Em gái người viết thích gì?"); its answer is in the text.
`, lang)

	fmt.Fprintf(&b, `
8. grammarStep: a short grammar step on %[1]s, built on the lesson's own sentences.
- titleVi: the Vietnamese name of the point (for example "Hiện tại đơn").
- noticing: 3 or 4 sentences copied exactly from the lesson sentences or the dialogue that show the point;
  highlights are the words of each sentence that carry it (for example "plays", "doesn't like"), copied
  exactly from the sentence. Include sentences that contrast (for example "I like" and "She plays").
- questionVi, options (3, in Vietnamese), answerIndex, explanationVi: a question that makes the learner
  work out the rule from the highlighted words (for example "Vì sao có câu là like, có câu là plays?").
- whenVi: one Vietnamese sentence on when to use it.
- table: 2 to 4 rows of the forms: subject (for example "he / she / it"), form ("likes"), negative
  ("doesn't like"); write "—" in negative when it does not apply.
- mistakes: 2 mistakes Vietnamese learners make with it, with the correction.
- noteVi: one Vietnamese sentence on why Vietnamese learners make these mistakes.
- exercises: exactly 4 exercises on the point at CEFR %[2]s, one of each kind, in this order: fill
  (text with one "___" blank, answers: every accepted answer), choice (text with "___", 3 or 4 options,
  answerIndex), fix (text is a sentence with one mistake, sentence its correction), reorder (words: the
  words of sentence, shuffled, plus at most 2 extra words). promptVi tells what to do, in Vietnamese;
  explanationVi says why, in Vietnamese. Every exercise has exactly one right answer: no blank that
  another option, article or tense could also fill correctly.
`, grammarName(req), req.Level)
	if req.GrammarFocus == nil && len(req.GrammarChoices) > 0 {
		b.WriteString("  The lesson has no grammar point yet: pick the one from this list that its sentences use most clearly, " +
			"and return its id, copied exactly, in grammarPointId:\n")
		for _, g := range req.GrammarChoices {
			fmt.Fprintf(&b, "  - %s: %s (%s)\n", g.ID, g.TitleEn, g.Pattern)
		}
	}

	fmt.Fprintf(&b, `
9. readAloud: 2 sentences copied exactly from the lesson sentences or the dialogue, at most 12 words each,
with sounds Vietnamese learners find hard (final consonants such as -s, -ed, -st, consonant clusters,
th, v, z). tipVi: one Vietnamese sentence on how to say the hardest word, with its US IPA.

10. speakQuestions: 3 questions to the learner about their own life on the lesson's subject (for
example "What music do you like?"), simple enough for CEFR %[1]s, asked as a friend would ask.
The third one asks about another person (a friend, a family member) so the answer uses he or she.
questionVi is the Vietnamese translation. frame is an answer frame with "___" for the learner's own
words (for example "I like ___ music. My favorite singer is ___.").

11. writingHelp, for the writing prompt of this lesson (write about the lesson's subject and someone else):
- grammarLabelVi: a short Vietnamese requirement about the lesson's grammar (for example "1 câu nói về
  người khác (he / she …)"),
- anyOf: 4 to 8 lower-case English words, one of which a text meeting that requirement must contain
  (for example "he", "she", "my mother", "my friend"),
- outlineVi: 3 short Vietnamese hints, one per sentence of a simple plan,
- modelAnswer: a model text of 40 to 60 words at CEFR %[1]s that answers it, uses at least 4
  vocabulary items exactly as written in the list and the lesson's grammar point at least twice.
`, req.Level)
	if len(req.TargetWords) > 0 {
		fmt.Fprintf(&b, "The words the lesson teaches most: %s.\n", strings.Join(req.TargetWords, ", "))
	}
	return b.String()
}

// grammarName names the point of the grammar step for the prompt.
func grammarName(req ai.PracticeRequest) string {
	if g := req.GrammarFocus; g != nil {
		return fmt.Sprintf("the lesson's grammar point, %s (%s)", g.TitleEn, g.Pattern)
	}
	return "the main grammar point the lesson sentences use"
}
