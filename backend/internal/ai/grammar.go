package ai

import "context"

// GrammarLessonRequest asks for the whole study page of one syllabus point (F20).
type GrammarLessonRequest struct {
	Level   string
	TitleVi string
	TitleEn string
	Pattern string
	HintVi  string
	// Examples are sentences from the syllabus that show the point.
	Examples []string
	// CommonMistakes are hand-written mistakes Vietnamese learners make with the point (added
	// 2026-10-20); the lesson's mistakes start from them.
	CommonMistakes []GrammarMistake
}

// GrammarExercise is one generated exercise of kind "choice", "fill", "reorder" or "fix"; only the fields
// of its kind are used.
type GrammarExercise struct {
	Kind          string   `json:"kind"`
	PromptVi      string   `json:"promptVi"`
	Text          string   `json:"text"`
	Options       []string `json:"options"`
	AnswerIndex   int      `json:"answerIndex"`
	Answers       []string `json:"answers"`
	Words         []string `json:"words"`
	Sentence      string   `json:"sentence"`
	ExplanationVi string   `json:"explanationVi"`
}

// GrammarStructure is one pattern of the point.
type GrammarStructure struct {
	Label   string `json:"label"`
	Pattern string `json:"pattern"`
	Example string `json:"example"`
}

// GrammarExample is an English sentence with its Vietnamese meaning.
type GrammarExample struct {
	En string `json:"en"`
	Vi string `json:"vi"`
}

// GrammarMistake is a common error with the fix.
type GrammarMistake struct {
	Wrong  string `json:"wrong"`
	Right  string `json:"right"`
	NoteVi string `json:"noteVi"`
}

// GrammarLessonContent is a generated grammar lesson before any checks; it mirrors
// grammar.Content.
type GrammarLessonContent struct {
	Objective   string             `json:"objective"`
	Explanation []string           `json:"explanation"`
	Usage       []string           `json:"usage"`
	Structures  []GrammarStructure `json:"structures"`
	Examples    []GrammarExample   `json:"examples"`
	Mistakes    []GrammarMistake   `json:"mistakes"`
	Practice    []GrammarExercise  `json:"practice"`
	Mastery     []GrammarExercise  `json:"mastery"`
}

// GrammarLesson always returns ErrNotConfigured.
func (Disabled) GrammarLesson(context.Context, GrammarLessonRequest) (GrammarLessonContent, error) {
	return GrammarLessonContent{}, ErrNotConfigured
}

// SolveRequest asks for an independent answer to every exercise of a grammar lesson (F21). The
// exercises carry no key on purpose.
type SolveRequest struct {
	Level     string
	TitleEn   string
	Pattern   string
	Exercises []SolveExercise
}

// SolveExercise is an exercise without its answer.
type SolveExercise struct {
	ID       string   `json:"id"`
	Kind     string   `json:"kind"`
	PromptVi string   `json:"promptVi"`
	Text     string   `json:"text"`
	Options  []string `json:"options,omitempty"`
	Words    []string `json:"words,omitempty"`
}

// Solution is the AI's own answer to one exercise: ChoiceIndex for a choice, Answer for a fill,
// Sentence for a reorder or a fix. Ambiguous is true when, to the AI, more than one answer is right
// or none is. MeaningOff is true when the Vietnamese of the exercise does not say the same thing as
// the right English answer (added 2026-10-20). Alternatives are the other right corrections of a fix,
// which a fix may have without being ambiguous (added 2026-10-27).
type Solution struct {
	ExerciseID   string   `json:"exerciseId"`
	ChoiceIndex  *int     `json:"choiceIndex,omitempty"`
	Answer       string   `json:"answer"`
	Sentence     string   `json:"sentence"`
	Alternatives []string `json:"alternatives,omitempty"`
	Ambiguous    bool     `json:"ambiguous"`
	MeaningOff   bool     `json:"meaningOff"`
	NoteVi       string   `json:"noteVi"`
}

// SolveGrammarExercises always returns ErrNotConfigured.
func (Disabled) SolveGrammarExercises(context.Context, SolveRequest) ([]Solution, error) {
	return nil, ErrNotConfigured
}
