package lesson

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

type speakAnswerJSON struct {
	Question string `json:"question"`
	Said     string `json:"said"`
}

type speakResultJSON struct {
	Said      string `json:"said"`
	Corrected string `json:"corrected"`
	NoteVi    string `json:"noteVi"`
}

// checkSpeaking is POST /api/lessons/{id}/speak/check: {"answers": [{question, said}]} →
// {"results": [{said, corrected, noteVi}]} in the same order.
func (h *ReadingHandler) checkSpeaking(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Answers []speakAnswerJSON `json:"answers"`
	}
	if httpx.DecodeJSON(w, r, &in) != nil {
		return
	}
	answers := make([]ai.SpeakAnswer, len(in.Answers))
	for i, a := range in.Answers {
		answers[i] = ai.SpeakAnswer(a)
	}
	p, _ := httpx.PrincipalFrom(r.Context())
	res, err := h.reader.CheckSpeaking(r.Context(), p.UserID, r.PathValue("id"), answers, p.Role == httpx.RoleAdmin)
	var verr *ValidationError
	switch {
	case err == nil:
		out := make([]speakResultJSON, len(res))
		for i, s := range res {
			out[i] = speakResultJSON(s)
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"results": out})
	case errors.Is(err, ErrSpeakLimit):
		httpx.WriteError(w, http.StatusTooManyRequests, "speak_limit",
			fmt.Sprintf("Mỗi bài chỉ nhận xét phần nói %d lần mỗi ngày.", SpeakChecksPerDay))
	case errors.Is(err, ai.ErrNotConfigured), errors.Is(err, ai.ErrInvalidKey):
		httpx.WriteError(w, http.StatusServiceUnavailable, "ai_not_configured", "AI chưa sẵn sàng để nhận xét phần nói.")
	case errors.Is(err, ai.ErrQuota):
		httpx.WriteError(w, http.StatusTooManyRequests, "ai_quota", "Đã hết lượt AI, vui lòng thử lại sau.")
	case errors.As(err, &verr), errors.Is(err, ErrNotFound):
		h.writeError(w, r, err, "not_found")
	default:
		h.log.WarnContext(r.Context(), "speaking check failed", slog.Any("error", err))
		httpx.WriteError(w, http.StatusBadGateway, "ai_failed", "Chưa nhận xét được, vui lòng thử lại.")
	}
}
