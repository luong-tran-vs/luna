package ai

// The parts of the six-step lesson (added 2026-10-22, design/de-xuat-quy-trinh-bai-hoc.png) that
// come with the vocabulary practice, in the same request: listening questions on the dialogue,
// a guiding question for the reading, the grammar step, the speaking step and the help for the
// writing.

// ListenQuestion is a question on the dialogue of the Listening step. Kind is "gist" (what the
// dialogue is about), "detail" (a fact said in it) or "tfn": a statement that is true, false or
// not said, answered with AnswerIndex 0, 1 or 2.
type ListenQuestion struct {
	Kind   string `json:"kind"`
	Prompt string `json:"prompt"`
	// PromptVi is the Vietnamese of an English prompt; empty when Prompt is Vietnamese already.
	PromptVi string   `json:"promptVi"`
	Options  []string `json:"options"`
	// AnswerIndex points into Options; for "tfn": 0 true, 1 false, 2 not said.
	AnswerIndex   int    `json:"answerIndex"`
	ExplanationVi string `json:"explanationVi"`
	// TurnIndex is the dialogue turn that holds the answer, -1 for a question on the whole dialogue.
	TurnIndex int `json:"turnIndex"`
}

// Noticing is a lesson sentence with the words that show the grammar point.
type Noticing struct {
	Sentence   string   `json:"sentence"`
	Highlights []string `json:"highlights"`
}

// FormRow is one line of the rule table: who, the form, the negative.
type FormRow struct {
	Subject  string `json:"subject"`
	Form     string `json:"form"`
	Negative string `json:"negative"`
}

// MistakePair is a mistake Vietnamese learners make and its correction.
type MistakePair struct {
	Wrong string `json:"wrong"`
	Right string `json:"right"`
}

// GrammarStep is the grammar step of a lesson: notice the pattern in the lesson's sentences,
// answer why, read the rule in one card, then do a few exercises.
type GrammarStep struct {
	TitleVi       string            `json:"titleVi"`
	Noticing      []Noticing        `json:"noticing"`
	QuestionVi    string            `json:"questionVi"`
	Options       []string          `json:"options"`
	AnswerIndex   int               `json:"answerIndex"`
	ExplanationVi string            `json:"explanationVi"`
	WhenVi        string            `json:"whenVi"`
	Table         []FormRow         `json:"table"`
	Mistakes      []MistakePair     `json:"mistakes"`
	NoteVi        string            `json:"noteVi"`
	Exercises     []GrammarExercise `json:"exercises"`
}

// ReadAloud is a sentence of the lesson to repeat, with a pronunciation tip in Vietnamese.
type ReadAloud struct {
	Text  string `json:"text"`
	TipVi string `json:"tipVi"`
}

// SpeakQuestion is a question the learner answers aloud about themselves, with a frame to fill
// ("I like ___ music.").
type SpeakQuestion struct {
	Question   string `json:"question"`
	QuestionVi string `json:"questionVi"`
	Frame      string `json:"frame"`
}

// WritingHelp is what the Writing step shows next to its prompt: the requirement on the lesson's
// grammar (checked loosely by the words in AnyOf), an outline and a model answer.
type WritingHelp struct {
	GrammarLabelVi string   `json:"grammarLabelVi"`
	AnyOf          []string `json:"anyOf"`
	OutlineVi      []string `json:"outlineVi"`
	ModelAnswer    string   `json:"modelAnswer"`
}
