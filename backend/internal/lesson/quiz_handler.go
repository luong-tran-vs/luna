package lesson

import (
	"net/http"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

// --- JSON shapes (contracts/reading-quiz-api.md) ---

type quizQuestionJSON struct {
	Prompt  string   `json:"prompt"`
	Options []string `json:"options"`
}

type answerJSON struct {
	QuestionIndex int    `json:"questionIndex"`
	Choice        int    `json:"choice"`
	Correct       bool   `json:"correct"`
	AnswerIndex   int    `json:"answerIndex"`
	ExplanationVi string `json:"explanationVi"`
}

type quizJSON struct {
	Version   int                `json:"version"`
	Questions []quizQuestionJSON `json:"questions"`
	Answers   []answerJSON       `json:"answers"`
}

type answerInputJSON struct {
	Version       int `json:"version"`
	QuestionIndex int `json:"questionIndex"`
	Choice        int `json:"choice"`
}

type answerResultJSON struct {
	Answer   answerJSON `json:"answer"`
	Answered int        `json:"answered"`
	Total    int        `json:"total"`
	Correct  int        `json:"correct"`
}

func toQuizJSON(q *QuizView) *quizJSON {
	if q == nil {
		return nil
	}
	out := &quizJSON{
		Version:   q.Version,
		Questions: make([]quizQuestionJSON, len(q.Questions)),
		Answers:   make([]answerJSON, len(q.Answers)),
	}
	for i, x := range q.Questions {
		out.Questions[i] = quizQuestionJSON(x)
	}
	for i, a := range q.Answers {
		out.Answers[i] = answerJSON(a)
	}
	return out
}

// answer checks one answer of the session's learner (F15).
func (h *ReadingHandler) answer(w http.ResponseWriter, r *http.Request) {
	var in answerInputJSON
	if httpx.DecodeJSON(w, r, &in) != nil {
		return
	}
	p, _ := httpx.PrincipalFrom(r.Context())
	res, err := h.reader.Answer(r.Context(), p.UserID, r.PathValue("id"), AnswerInput(in))
	if err != nil {
		h.writeError(w, r, err, "not_found")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, answerResultJSON{
		Answer: answerJSON(res.Answer), Answered: res.Answered, Total: res.Total, Correct: res.Correct,
	})
}

// resetAnswers deletes the session learner's answers to the lesson's current questions so the
// quiz can be taken again; lesson progress is kept.
func (h *ReadingHandler) resetAnswers(w http.ResponseWriter, r *http.Request) {
	p, _ := httpx.PrincipalFrom(r.Context())
	if err := h.reader.ResetAnswers(r.Context(), p.UserID, r.PathValue("id")); err != nil {
		h.writeError(w, r, err, "not_found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
