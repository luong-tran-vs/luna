package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/luongtran/luna/backend/internal/ai"
)

var grammarExerciseSchema = map[string]any{
	"type": "OBJECT",
	"properties": map[string]any{
		"kind":          map[string]any{"type": "STRING", "enum": []string{"choice", "fill", "reorder", "fix"}},
		"promptVi":      map[string]any{"type": "STRING"},
		"text":          map[string]any{"type": "STRING"},
		"options":       stringArray,
		"answerIndex":   map[string]any{"type": "INTEGER"},
		"answers":       stringArray,
		"words":         stringArray,
		"sentence":      map[string]any{"type": "STRING"},
		"explanationVi": map[string]any{"type": "STRING"},
	},
	"required": []string{"kind", "promptVi", "text", "explanationVi"},
}

// grammarLessonSchema forces every part of a grammar lesson (F20); the grammar service drops the
// exercises that fail its checks.
var grammarLessonSchema = map[string]any{
	"type": "OBJECT",
	"properties": map[string]any{
		"objective":   map[string]any{"type": "STRING"},
		"explanation": stringArray,
		"usage":       stringArray,
		"structures": map[string]any{
			"type": "ARRAY",
			"items": map[string]any{
				"type": "OBJECT",
				"properties": map[string]any{
					"label":   map[string]any{"type": "STRING"},
					"pattern": map[string]any{"type": "STRING"},
					"example": map[string]any{"type": "STRING"},
				},
				"required": []string{"label", "pattern", "example"},
			},
		},
		"examples": map[string]any{
			"type": "ARRAY",
			"items": map[string]any{
				"type": "OBJECT",
				"properties": map[string]any{
					"en": map[string]any{"type": "STRING"},
					"vi": map[string]any{"type": "STRING"},
				},
				"required": []string{"en", "vi"},
			},
		},
		"mistakes": map[string]any{
			"type": "ARRAY",
			"items": map[string]any{
				"type": "OBJECT",
				"properties": map[string]any{
					"wrong":  map[string]any{"type": "STRING"},
					"right":  map[string]any{"type": "STRING"},
					"noteVi": map[string]any{"type": "STRING"},
				},
				"required": []string{"wrong", "right", "noteVi"},
			},
		},
		"practice": map[string]any{"type": "ARRAY", "items": grammarExerciseSchema},
		"mastery":  map[string]any{"type": "ARRAY", "items": grammarExerciseSchema},
	},
	"required": []string{"objective", "explanation", "usage", "structures", "examples", "mistakes", "practice", "mastery"},
}

// GrammarLesson sends one generateContent request for the whole grammar lesson of a syllabus
// point: explanation, structures, examples, mistakes, practice and the mastery check. The grammar
// service filters and checks the answer.
func (c *Client) GrammarLesson(ctx context.Context, req ai.GrammarLessonRequest) (ai.GrammarLessonContent, error) {
	text, err := c.generate(ctx, "grammar_lesson", grammarLessonPrompt(req), grammarLessonSchema, 0.5,
		slog.String("level", req.Level), slog.String("point", req.TitleEn))
	if err != nil {
		return ai.GrammarLessonContent{}, err
	}
	var out ai.GrammarLessonContent
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return ai.GrammarLessonContent{}, fmt.Errorf("gemini: decode grammar lesson: %w", err)
	}
	return out, nil
}

func grammarLessonPrompt(req ai.GrammarLessonRequest) string {
	var b strings.Builder
	fmt.Fprintf(&b, `You write a grammar lesson for Vietnamese learners of English at CEFR level %[1]s.
The grammar point to teach, and only this point: %[2]s (%[3]s). Pattern: %[4]s. When it is used: %[5]s.
Every English sentence you write (examples, structures, exercises) must use only vocabulary and grammar suitable for CEFR %[1]s,
apart from the point itself. Explain in simple, short Vietnamese, without grammar jargon a learner at this level would not know.
Return a JSON object with:

1. objective: one Vietnamese sentence starting with "Bạn có thể".
2. explanation: 2 to 4 short Vietnamese paragraphs that explain the point step by step.
3. usage: 2 to 5 short Vietnamese sentences saying when to use it.
4. structures: 1 to 6 patterns that together cover every form of the point a learner at this level needs: affirmative, negative,
   Yes/No question with its short answers, a question with a question word when the point has one, and the contracted forms.
   Each has a Vietnamese label, the pattern and one English example.
   When the form changes with the subject, name every subject in each pattern (for example "I am", "he / she / it is", "you / we / they are"),
   in the explanation too: never say a form goes with only one subject when it goes with several.
5. examples: 4 to 8 English sentences that show the point, each with its Vietnamese meaning. Cover the different forms (not only affirmative)
   and subjects. Take inspiration from the course examples below, but do not only copy them.
6. mistakes: 3 to 5 mistakes Vietnamese learners often make: the wrong English sentence, the right one, and noteVi, a short Vietnamese note on why.
   Start from the common mistakes listed below when there are any (you may reword the note), then add others.
7. practice: exactly 8 exercises for practice.
8. mastery: exactly 6 exercises for the mastery check. They must be different sentences from the practice exercises.

Exercises, in both practice and mastery, mix the four kinds (in practice at least 2 of each kind; in mastery at least 1 of each kind)
and the forms of the point (affirmative, negative, questions, short answers) with different subjects.
Every exercise has kind, promptVi (a short Vietnamese instruction), text, and explanationVi, a Vietnamese explanation of why the answer is right.
- choice: text is the English question or a sentence with one blank written "___"; options has exactly 4 different English options; answerIndex (0 to 3) is the only correct one.
- fill: text is an English sentence with exactly one blank written "___"; answers lists the accepted words for the blank (at least 1, lower case).
- reorder: text is the exact Vietnamese translation of sentence: the same meaning, person, tense and sentence type
  (a question in Vietnamese only for an English question, an answer only for an answer); sentence is the correct English sentence (3 to 14 words);
  words are all the words of sentence, plus at most 2 extra wrong words.
- fix: text is an English sentence (3 to 14 words) with exactly one mistake on this point, preferably one Vietnamese learners make;
  sentence is the corrected sentence, changing as few words as possible; answers lists other full corrected sentences that are also right (may be empty).
  promptVi asks the learner to find and correct the mistake.
Whenever an exercise has Vietnamese, it must say exactly what the right English answer says.
Plain text only, no markdown.

Course examples:
`, req.Level, req.TitleEn, req.TitleVi, req.Pattern, req.HintVi)
	for _, e := range req.Examples {
		fmt.Fprintf(&b, "- %s\n", e)
	}
	if len(req.CommonMistakes) > 0 {
		b.WriteString("\nCommon mistakes of Vietnamese learners with this point (wrong → right: why, in Vietnamese):\n")
		for _, m := range req.CommonMistakes {
			fmt.Fprintf(&b, "- %s → %s: %s\n", m.Wrong, m.Right, m.NoteVi)
		}
	}
	return b.String()
}

var grammarSolveSchema = map[string]any{
	"type": "ARRAY",
	"items": map[string]any{
		"type": "OBJECT",
		"properties": map[string]any{
			"exerciseId":  map[string]any{"type": "STRING"},
			"choiceIndex": map[string]any{"type": "INTEGER"},
			"answer":      map[string]any{"type": "STRING"},
			"sentence":    map[string]any{"type": "STRING"},
			"ambiguous":   map[string]any{"type": "BOOLEAN"},
			"meaningOff":  map[string]any{"type": "BOOLEAN"},
			"noteVi":      map[string]any{"type": "STRING"},
		},
		"required": []string{"exerciseId", "ambiguous"},
	},
}

// SolveGrammarExercises sends one low-temperature request that answers every exercise of a lesson
// without seeing the keys (F21).
func (c *Client) SolveGrammarExercises(ctx context.Context, req ai.SolveRequest) ([]ai.Solution, error) {
	text, err := c.generate(ctx, "grammar_solve", grammarSolvePrompt(req), grammarSolveSchema, 0.1,
		slog.String("level", req.Level), slog.String("point", req.TitleEn), slog.Int("exercises", len(req.Exercises)))
	if err != nil {
		return nil, err
	}
	var out []ai.Solution
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return nil, fmt.Errorf("gemini: decode grammar solutions: %w", err)
	}
	return out, nil
}

func grammarSolvePrompt(req ai.SolveRequest) string {
	var b strings.Builder
	fmt.Fprintf(&b, `You are a careful English teacher. Solve every exercise below on your own; you have no answer key.
The exercises belong to a grammar lesson for Vietnamese learners at CEFR level %s on the point: %s (%s).
Do not guess what the author intended and do not pick the most likely-looking option: decide what is really correct English.
Return a JSON array with exactly one object per exercise, in the same order, each with exerciseId (copy the id) and:
- choice: choiceIndex, the 0-based index of the one right option.
- fill: answer, the single word or phrase that fills the blank "___".
- reorder: sentence, the English sentence you build by arranging the words (use only the given words; some may be extra and unused).
- fix: sentence, the text with its mistake corrected, changing as few words as possible.
- ambiguous: true when, in your honest opinion, more than one option or answer is correct, or none is, or the exercise is unclear or has an error; otherwise false. When ambiguous is true, noteVi is one short Vietnamese sentence that says why. Do not mark an exercise ambiguous just to be safe.
- meaningOff: true when the Vietnamese of the exercise (the text of a reorder, or Vietnamese in promptVi or text) does not say exactly what your answer says:
  a different meaning, person or tense, or a question for a statement or the other way round (for example "Bạn bao nhiêu tuổi?" for "I am ten."); otherwise false.
  When meaningOff is true, noteVi is one short Vietnamese sentence that says what differs.
Plain text only, no markdown.

Exercises (JSON):
`, req.Level, req.TitleEn, req.Pattern)
	raw, _ := json.Marshal(req.Exercises)
	b.Write(raw)
	b.WriteString("\n")
	return b.String()
}
