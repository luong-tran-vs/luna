package lesson

import (
	"net/http"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

type extrasInputJSON struct {
	Questions     []questionJSON   `json:"questions"`
	GrammarNote   *grammarNoteJSON `json:"grammarNote"`
	WritingPrompt string           `json:"writingPrompt"`
}

func (in extrasInputJSON) toInput() ExtrasInput {
	out := ExtrasInput{Questions: make([]Question, len(in.Questions)), WritingPrompt: in.WritingPrompt}
	for i, q := range in.Questions {
		out.Questions[i] = Question(q)
	}
	if in.GrammarNote != nil {
		n := GrammarNote(*in.GrammarNote)
		out.GrammarNote = &n
	}
	return out
}

// updateExtras saves the questions, grammar note and writing prompt of a lesson (F15).
func (h *Handler) updateExtras(w http.ResponseWriter, r *http.Request) {
	var in extrasInputJSON
	if httpx.DecodeJSON(w, r, &in) != nil {
		return
	}
	l, err := h.svc.UpdateExtras(r.Context(), r.PathValue("id"), in.toInput())
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeLesson(w, r, http.StatusOK, l)
}
