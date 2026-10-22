package lesson

import "github.com/luongtran/luna/backend/internal/grammar"

// JSON shapes of the six-step lesson (added 2026-10-22), part of GET /api/lessons/{id}/practice.

type stepsJSON struct {
	ListeningQuestions []listenQuestionJSON `json:"listeningQuestions"`
	ReadingGuideVi     string               `json:"readingGuideVi"`
	Grammar            *grammarStepJSON     `json:"grammar"`
	ReadAloud          []readAloudJSON      `json:"readAloud"`
	SpeakQuestions     []speakQuestionJSON  `json:"speakQuestions"`
	Writing            *writingHelpJSON     `json:"writing"`
}

type listenQuestionJSON struct {
	Kind          string   `json:"kind"`
	Prompt        string   `json:"prompt"`
	PromptVi      string   `json:"promptVi"`
	Options       []string `json:"options"`
	AnswerIndex   int      `json:"answerIndex"`
	ExplanationVi string   `json:"explanationVi"`
	TurnIndex     int      `json:"turnIndex"`
}

type noticingJSON struct {
	Sentence   string   `json:"sentence"`
	Highlights []string `json:"highlights"`
}

type formRowJSON struct {
	Subject  string `json:"subject"`
	Form     string `json:"form"`
	Negative string `json:"negative"`
}

type mistakePairJSON struct {
	Wrong string `json:"wrong"`
	Right string `json:"right"`
}

// stepExerciseJSON mirrors the exercises of the grammar pages (lu-grammar-exercise).
type stepExerciseJSON struct {
	ID            string   `json:"id"`
	Kind          string   `json:"kind"`
	PromptVi      string   `json:"promptVi,omitempty"`
	Text          string   `json:"text,omitempty"`
	Options       []string `json:"options,omitempty"`
	AnswerIndex   *int     `json:"answerIndex,omitempty"`
	Answers       []string `json:"answers,omitempty"`
	Words         []string `json:"words,omitempty"`
	Sentence      string   `json:"sentence,omitempty"`
	ExplanationVi string   `json:"explanationVi,omitempty"`
}

type grammarStepJSON struct {
	TitleVi       string             `json:"titleVi"`
	Noticing      []noticingJSON     `json:"noticing"`
	QuestionVi    string             `json:"questionVi"`
	Options       []string           `json:"options"`
	AnswerIndex   int                `json:"answerIndex"`
	ExplanationVi string             `json:"explanationVi"`
	WhenVi        string             `json:"whenVi"`
	Table         []formRowJSON      `json:"table"`
	Mistakes      []mistakePairJSON  `json:"mistakes"`
	NoteVi        string             `json:"noteVi"`
	Exercises     []stepExerciseJSON `json:"exercises"`
}

type readAloudJSON struct {
	Text  string `json:"text"`
	TipVi string `json:"tipVi"`
}

type speakQuestionJSON struct {
	Question   string `json:"question"`
	QuestionVi string `json:"questionVi"`
	Frame      string `json:"frame"`
}

type writingHelpJSON struct {
	GrammarLabelVi string   `json:"grammarLabelVi"`
	AnyOf          []string `json:"anyOf"`
	OutlineVi      []string `json:"outlineVi"`
	ModelAnswer    string   `json:"modelAnswer"`
}

type dictationJSON struct {
	Text  string `json:"text"`
	Lemma string `json:"lemma"`
}

func toStepsJSON(s *Steps) *stepsJSON {
	if s == nil {
		return nil
	}
	out := &stepsJSON{
		ListeningQuestions: make([]listenQuestionJSON, len(s.ListeningQuestions)),
		ReadingGuideVi:     s.ReadingGuideVi,
		ReadAloud:          make([]readAloudJSON, len(s.ReadAloud)),
		SpeakQuestions:     make([]speakQuestionJSON, len(s.SpeakQuestions)),
	}
	for i, q := range s.ListeningQuestions {
		out.ListeningQuestions[i] = listenQuestionJSON(q)
	}
	for i, r := range s.ReadAloud {
		out.ReadAloud[i] = readAloudJSON(r)
	}
	for i, q := range s.SpeakQuestions {
		out.SpeakQuestions[i] = speakQuestionJSON(q)
	}
	if g := s.Grammar; g != nil {
		gj := &grammarStepJSON{
			TitleVi: g.TitleVi, QuestionVi: g.QuestionVi, Options: nonNil(g.Options), AnswerIndex: g.AnswerIndex,
			ExplanationVi: g.ExplanationVi, WhenVi: g.WhenVi, NoteVi: g.NoteVi,
			Noticing: make([]noticingJSON, len(g.Noticing)), Table: make([]formRowJSON, len(g.Table)),
			Mistakes: make([]mistakePairJSON, len(g.Mistakes)), Exercises: make([]stepExerciseJSON, len(g.Exercises)),
		}
		for i, n := range g.Noticing {
			gj.Noticing[i] = noticingJSON(n)
		}
		for i, r := range g.Table {
			gj.Table[i] = formRowJSON(r)
		}
		for i, m := range g.Mistakes {
			gj.Mistakes[i] = mistakePairJSON(m)
		}
		for i, e := range g.Exercises {
			gj.Exercises[i] = stepExerciseJSON{
				ID: e.ID, Kind: string(e.Kind), PromptVi: e.PromptVi, Text: e.Text, Options: e.Options,
				Answers: e.Answers, Words: e.Words, Sentence: e.Sentence, ExplanationVi: e.ExplanationVi,
			}
			if e.Kind == grammar.KindChoice {
				idx := e.AnswerIndex
				gj.Exercises[i].AnswerIndex = &idx
			}
		}
		out.Grammar = gj
	}
	if w := s.Writing; w != nil {
		out.Writing = &writingHelpJSON{
			GrammarLabelVi: w.GrammarLabelVi, AnyOf: nonNil(w.AnyOf), OutlineVi: nonNil(w.OutlineVi), ModelAnswer: w.ModelAnswer,
		}
	}
	return out
}

func toDictationJSON(items []DictationItem) []dictationJSON {
	out := make([]dictationJSON, len(items))
	for i, it := range items {
		out[i] = dictationJSON(it)
	}
	return out
}
