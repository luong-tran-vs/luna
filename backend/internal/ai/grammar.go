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
}

// GrammarExercise is one generated exercise of kind "choice", "fill" or "reorder"; only the fields
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
// Sentence for a reorder. Ambiguous is true when, to the AI, more than one answer is right or none is.
type Solution struct {
	ExerciseID  string `json:"exerciseId"`
	ChoiceIndex *int   `json:"choiceIndex,omitempty"`
	Answer      string `json:"answer"`
	Sentence    string `json:"sentence"`
	Ambiguous   bool   `json:"ambiguous"`
	NoteVi      string `json:"noteVi"`
}

// SolveGrammarExercises always returns ErrNotConfigured.
func (Disabled) SolveGrammarExercises(context.Context, SolveRequest) ([]Solution, error) {
	return nil, ErrNotConfigured
}
